package health

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"maps"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/status"
)

type Server struct {
	server     *http.Server
	mu         sync.RWMutex
	ready      bool
	checks     map[string]Check
	startTime  time.Time
	reloadFunc func() error
	authToken  string // optional bearer token for protected endpoints

	// statusProbe supplies the detailed Status snapshot. It is nil until the
	// gateway installs one, and a nil probe makes detail mode unavailable
	// rather than empty.
	statusProbe func() status.Snapshot

	// activeRequests reports how many agent turns are currently in flight.
	// The launcher reads it to avoid restarting the gateway mid-answer; nil
	// means the gateway did not supply one, which callers must treat as
	// "unknown", never as "idle".
	activeRequests func() int
}

type Check struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type StatusResponse struct {
	Status string           `json:"status"`
	Uptime string           `json:"uptime"`
	PID    int              `json:"pid,omitempty"`
	Checks map[string]Check `json:"checks,omitempty"`

	// ActiveRequests is the number of in-flight provider calls, and Busy is
	// whether that number is above zero.
	//
	// Both are pointers so "the gateway did not report" is distinguishable
	// from "the gateway reported zero". A restart decision must not read a
	// missing field as an idle gateway and interrupt someone's answer.
	//
	// This is not a count of agent turns. It is incremented around each
	// provider request, including background summarization, so it is the
	// right signal for "is it safe to restart" and the wrong one for any
	// user-facing "running" number. Status reports turns separately.
	ActiveRequests *int  `json:"active_requests,omitempty"`
	Busy           *bool `json:"busy,omitempty"`

	// Detail carries the Status snapshot, and only when a request both asked
	// for it and presented the gateway credential. It is omitted entirely
	// otherwise, which is what keeps the anonymous /health response
	// byte-identical to what it has always been.
	Detail *status.Snapshot `json:"detail,omitempty"`
}

func NewServer(host string, port int, token string) *Server {
	mux := http.NewServeMux()
	s := &Server{
		ready:     false,
		checks:    make(map[string]Check),
		startTime: time.Now(),
		authToken: token,
	}

	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/reload", s.reloadHandler)

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()
	return s.server.ListenAndServe()
}

func (s *Server) StartContext(ctx context.Context) error {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.server.Shutdown(context.Background())
	}
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	s.ready = false
	s.mu.Unlock()
	return s.server.Shutdown(ctx)
}

func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	s.ready = ready
	s.mu.Unlock()
}

func (s *Server) RegisterCheck(name string, checkFn func() (bool, string)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ok, msg := checkFn()
	s.checks[name] = Check{
		Name:      name,
		Status:    statusString(ok),
		Message:   msg,
		Timestamp: time.Now(),
	}
}

// SetReloadFunc sets the callback function for config reload.
func (s *Server) SetReloadFunc(fn func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadFunc = fn
}

func (s *Server) reloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed, use POST"})
		return
	}

	// Token check
	s.mu.RLock()
	requiredToken := s.authToken
	s.mu.RUnlock()

	if requiredToken != "" {
		given := extractBearerToken(r.Header.Get("Authorization"))
		if given == "" || subtle.ConstantTimeCompare([]byte(given), []byte(requiredToken)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
	}

	s.mu.Lock()
	reloadFunc := s.reloadFunc
	s.mu.Unlock()

	if reloadFunc == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "reload not configured"})
		return
	}

	if err := reloadFunc(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "reload triggered"})
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Detail is opt-in and authenticated. The basic response stays exactly as
	// it was — same fields, same anonymous access — because the launcher and
	// the Android host both poll it for liveness, and because widening what an
	// unauthenticated caller learns about the process is not a side effect a
	// new screen should have.
	detailRequested := r.URL.Query().Get("detail") == "1"
	if detailRequested && !s.authorized(r) {
		// Fail closed: an unauthorized detail request is refused, never
		// quietly downgraded to a basic response. Silently serving less than
		// was asked for would hide a broken credential until someone noticed
		// the screen was empty.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	uptime := time.Since(s.startTime)
	resp := StatusResponse{
		Status: "ok",
		Uptime: uptime.String(),
		PID:    os.Getpid(),
	}

	s.mu.RLock()
	probe := s.activeRequests
	statusProbe := s.statusProbe
	s.mu.RUnlock()
	if probe != nil {
		active := probe()
		busy := active > 0
		resp.ActiveRequests = &active
		resp.Busy = &busy
	}
	if detailRequested && statusProbe != nil {
		snapshot := statusProbe()
		resp.Detail = &snapshot
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// authorized reports whether a request carries the gateway bearer credential.
//
// This is the same token that already guards /reload, read from the same
// field: detail mode reuses the existing gateway credential rather than
// introducing a second one. When no token is configured the gateway has no
// credential to check against, and detail mode is refused rather than opened
// to everyone.
func (s *Server) authorized(r *http.Request) bool {
	s.mu.RLock()
	required := s.authToken
	s.mu.RUnlock()

	if required == "" {
		return false
	}
	given := extractBearerToken(r.Header.Get("Authorization"))
	if given == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(given), []byte(required)) == 1
}

// SetStatusProbe supplies the snapshot served in detail mode.
func (s *Server) SetStatusProbe(probe func() status.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusProbe = probe
}

// SetActiveRequestsProbe supplies the in-flight turn count reported by /health.
//
// It exists so the launcher can defer a configuration restart until the gateway
// is idle, rather than interrupting a running answer or a tool that is part way
// through a side effect.
func (s *Server) SetActiveRequestsProbe(probe func() int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeRequests = probe
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	ready := s.ready
	checks := make(map[string]Check)
	maps.Copy(checks, s.checks)
	s.mu.RUnlock()

	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(StatusResponse{
			Status: "not ready",
			Checks: checks,
		})
		return
	}

	for _, check := range checks {
		if check.Status == "fail" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(StatusResponse{
				Status: "not ready",
				Checks: checks,
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	uptime := time.Since(s.startTime)
	_ = json.NewEncoder(w).Encode(StatusResponse{
		Status: "ready",
		Uptime: uptime.String(),
		Checks: checks,
	})
}

// HandlerMux is the interface for registering HTTP handlers, used by
// RegisterOnMux so that callers can pass any mux implementation
// (e.g. *http.ServeMux or a custom dynamic mux).
type HandlerMux interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// RegisterOnMux registers /health, /ready and /reload handlers onto the given mux.
// This allows the health endpoints to be served by a shared HTTP server.
func (s *Server) RegisterOnMux(mux HandlerMux) {
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/reload", s.reloadHandler)
}

func statusString(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}

// extractBearerToken returns the token from an "Authorization: Bearer <t>" header,
// or the empty string if the header is missing or malformed.
func extractBearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) < len(prefix) {
		return ""
	}
	if header[:len(prefix)] != prefix {
		return ""
	}
	return header[len(prefix):]
}
