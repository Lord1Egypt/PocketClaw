package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/netbind"
)

// listenerDrainWait bounds how long a widening swap waits for requests already
// in flight on the old bind. Generous enough for an ordinary request, short
// enough that a streaming response cannot hold the swap open.
const listenerDrainWait = 10 * time.Second

type launcherListenerGroup struct {
	server   *http.Server
	listener net.Listener

	connectionsMu sync.Mutex
	connections   map[net.Conn]struct{}
}

func newLauncherListenerGroup(handler http.Handler, listener net.Listener) *launcherListenerGroup {
	group := &launcherListenerGroup{
		listener:    listener,
		connections: make(map[net.Conn]struct{}),
	}
	group.server = &http.Server{
		Handler: handler,
		ConnState: func(connection net.Conn, state http.ConnState) {
			group.connectionsMu.Lock()
			defer group.connectionsMu.Unlock()
			switch state {
			case http.StateNew, http.StateActive, http.StateIdle, http.StateHijacked:
				group.connections[connection] = struct{}{}
			case http.StateClosed:
				delete(group.connections, connection)
			}
		},
	}
	return group
}

func (g *launcherListenerGroup) serve() {
	logger.InfoC("web", fmt.Sprintf("Server listening on %s", g.listener.Addr().String()))
	if err := g.server.Serve(g.listener); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
		logger.ErrorC("web", fmt.Sprintf("Dashboard listener stopped unexpectedly on %s: %v", g.listener.Addr().String(), err))
	}
}

// drain stops accepting and lets in-flight requests finish before the server
// goes away.
//
// PC-DEF-065. Used when a swap only *widens* access, where nothing is being
// revoked and cutting a request off is pure damage. The first-claim
// reconciliation is exactly that case, and closing hard there killed the
// connection carrying the setup request: the password was written, the response
// never reached the browser, and the retry was refused because the dashboard
// had meanwhile become initialized. Bounded, so a long-lived stream cannot
// wedge the swap -- whatever is still open is closed by the caller afterwards.
func (g *launcherListenerGroup) drain(within time.Duration) {
	g.server.SetKeepAlivesEnabled(false)
	ctx, cancel := context.WithTimeout(context.Background(), within)
	defer cancel()
	if err := g.server.Shutdown(ctx); err != nil {
		logger.WarnC("web", fmt.Sprintf(
			"Dashboard listener did not drain within %s; closing it: %v", within, err))
	}
	g.close()
}

func (g *launcherListenerGroup) close() {
	g.server.SetKeepAlivesEnabled(false)
	_ = g.server.Close()
	_ = g.listener.Close()

	// net/http does not own hijacked WebSocket connections after the upgrade.
	// Close every connection associated with the old bind so a remote browser
	// cannot retain access after Public Mode is disabled. Its authenticated
	// session cookie remains valid and can reconnect through the new listener.
	g.connectionsMu.Lock()
	connections := make([]net.Conn, 0, len(g.connections))
	for connection := range g.connections {
		connections = append(connections, connection)
	}
	g.connections = make(map[net.Conn]struct{})
	g.connectionsMu.Unlock()
	for _, connection := range connections {
		_ = connection.Close()
	}
}

type launcherHTTPRuntime struct {
	mu sync.Mutex

	handler   http.Handler
	hostInput string
	port      string
	public    bool
	// desiredPublic is what the user asked for, which PC-DEF-039 may have
	// narrowed. Kept so the preference can be re-applied once the dashboard has
	// an owner; see ReconcileAfterDashboardClaimed.
	desiredPublic bool
	groups        []*launcherListenerGroup
	open          func(string, bool, string) (netbind.OpenResult, error)
}

func newLauncherHTTPRuntime(
	handler http.Handler,
	hostInput string,
	public bool,
	desiredPublic bool,
	initial netbind.OpenResult,
) *launcherHTTPRuntime {
	return &launcherHTTPRuntime{
		handler:       handler,
		hostInput:     hostInput,
		port:          initial.Port,
		public:        public,
		desiredPublic: desiredPublic,
		groups:        launcherListenerGroups(handler, initial.Listeners),
		open:          openLauncherListeners,
	}
}

// ReconcileAfterDashboardClaimed re-applies the desired exposure once the
// dashboard has an owner.
//
// PC-DEF-040, reopened. PC-DEF-039 narrows an unclaimed dashboard to loopback
// whatever the user asked for, so "Public Mode ON" and "bound to the LAN" diverge
// until something re-applies the preference. Nothing on this side did: the only
// thing that reconciled was the Android app noticing its embedded WebView navigate
// away from /launcher-setup, and any other route to a first claim left the listener
// on loopback with a manual Public Mode OFF→ON as the only recovery. That is what
// the owner reproduced.
//
// The claim itself is the authoritative event, and it happens here, so the decision
// belongs here too. It is called only after a **first** claim that POST
// /api/auth/setup already required to be loopback-only, so PC-DEF-039 is intact:
// this widens exposure only for a dashboard that a local owner has just taken
// ownership of.
//
// A no-op when the user never asked for LAN, and a no-op when the listener is
// already public, so it is safe to call on every successful claim.
//
// PC-DEF-065. It runs the apply on its own goroutine, and it has to. This is
// called from the request that performed the claim, and applying the exposure
// replaces the listener serving that request: doing it inline cut the response
// off before it reached the browser, so the password was set and the user was
// told setup had failed. Draining instead of closing is the other half of that
// fix, and draining inline would be worse still -- Shutdown waits for the
// in-flight handler, which is the caller, so it would block until its own
// timeout. Asynchronous plus draining is what makes the response arrive.
//
// The outcome is logged here rather than returned, because there is no longer a
// caller left to return it to.
func (r *launcherHTTPRuntime) ReconcileAfterDashboardClaimed() {
	r.mu.Lock()
	desired := r.desiredPublic
	already := r.public
	r.mu.Unlock()

	if !desired || already {
		return
	}
	logger.InfoC("web",
		"Dashboard now has an owner; applying the requested Public Mode")
	go func() {
		if err := r.ApplyPublicMode(true); err != nil {
			logger.WarnC("web", fmt.Sprintf(
				"Dashboard was claimed but the requested Public Mode could not be "+
					"applied; it stays reachable locally: %v", err))
		}
	}()
}

func launcherListenerGroups(handler http.Handler, listeners []net.Listener) []*launcherListenerGroup {
	groups := make([]*launcherListenerGroup, 0, len(listeners))
	for _, listener := range listeners {
		groups = append(groups, newLauncherListenerGroup(handler, listener))
	}
	return groups
}

func (r *launcherHTTPRuntime) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.startLocked()
}

func (r *launcherHTTPRuntime) startLocked() {
	for _, group := range r.groups {
		go group.serve()
	}
}

func (r *launcherHTTPRuntime) closeLocked() {
	for _, group := range r.groups {
		group.close()
	}
	r.groups = nil
}

// drainLocked replaces closeLocked when access is being widened rather than
// revoked, so a request in flight on the old bind still gets its response.
func (r *launcherHTTPRuntime) drainLocked() {
	for _, group := range r.groups {
		group.drain(listenerDrainWait)
	}
	r.groups = nil
}

func (r *launcherHTTPRuntime) PublicMode() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.public
}

// ApplyPublicMode replaces only the authenticated Dashboard listeners. The
// API handler, session store, gateway process, Telegram polling, and agent
// runtime remain in the same launcher process.
func (r *launcherHTTPRuntime) ApplyPublicMode(public bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if public == r.public {
		return nil
	}

	previousPublic := r.public
	// An explicit change is also a change of intent: otherwise turning Public Mode
	// off and then claiming a dashboard would re-widen it from a stale desire.
	r.desiredPublic = public
	// PC-DEF-065. Widening revokes nothing, so in-flight requests are drained
	// rather than cut off. Narrowing is a revocation and must not let a remote
	// client finish what it started, so that one still closes hard.
	if public && !previousPublic {
		r.drainLocked()
	} else {
		r.closeLocked()
	}

	result, err := r.open(r.hostInput, public, r.port)
	if err == nil {
		r.groups = launcherListenerGroups(r.handler, result.Listeners)
		r.public = public
		r.startLocked()
		logger.InfoC("web", fmt.Sprintf("Dashboard Public Mode applied: public=%t", public))
		return nil
	}

	rollback, rollbackErr := r.open(r.hostInput, previousPublic, r.port)
	if rollbackErr != nil {
		return fmt.Errorf("dashboard bind failed: %w; previous listener restore failed: %v", err, rollbackErr)
	}
	r.groups = launcherListenerGroups(r.handler, rollback.Listeners)
	r.public = previousPublic
	r.startLocked()
	return fmt.Errorf("dashboard bind failed; previous listener restored: %w", err)
}

func (r *launcherHTTPRuntime) Shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeLocked()
}
