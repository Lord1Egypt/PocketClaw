package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/netbind"
)

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
func (r *launcherHTTPRuntime) ReconcileAfterDashboardClaimed() error {
	r.mu.Lock()
	desired := r.desiredPublic
	already := r.public
	r.mu.Unlock()

	if !desired || already {
		return nil
	}
	logger.InfoC("web",
		"Dashboard now has an owner; applying the requested Public Mode")
	return r.ApplyPublicMode(true)
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
	r.closeLocked()

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
