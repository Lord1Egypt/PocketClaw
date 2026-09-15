package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/netbind"
	"github.com/sipeed/picoclaw/web/backend/api"
	"github.com/sipeed/picoclaw/web/backend/dashboardauth"
	"github.com/sipeed/picoclaw/web/backend/middleware"
)

// The fresh-install journey, in order, against the real wiring.
//
// PC-DEF-065. This exists because the project kept fixing one step and
// breaking the next, and every isolated test still passed: the store was
// correct, the handler's rules were correct, the exposure rule was correct, and
// a real first install could still not create its first password. Only the
// ordered path shows that, so this drives the actual HTTP server, the actual
// listener swap, the actual middleware and the actual bcrypt store -- not
// httptest handlers in isolation.
//
// A step here is a step a user takes. If a later step becomes impossible
// because of an earlier hardening change, this fails.

// journeyEnv is one fresh install: empty private auth dir, no password, a real
// loopback listener, and the same route wiring main.go builds.
type journeyEnv struct {
	store   *dashboardauth.Store
	runtime *launcherHTTPRuntime
	mux     *http.ServeMux
	baseURL string
	client  *http.Client
	claims  chan struct{}
}

func newJourneyEnv(t *testing.T, desiredPublic bool) *journeyEnv {
	t.Helper()

	// A fresh install has no dashboard credential anywhere.
	authDir := t.TempDir()
	store, err := dashboardauth.New(authDir)
	if err != nil {
		t.Fatalf("open a fresh private auth store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if initialized, err := store.IsInitialized(context.Background()); err != nil || initialized {
		t.Fatalf("a fresh install must start uninitialized: %v, %v", initialized, err)
	}

	sessions := middleware.NewLauncherDashboardSessions(12 * time.Hour)
	env := &journeyEnv{store: store, claims: make(chan struct{}, 4)}

	env.mux = http.NewServeMux()
	api.RegisterLauncherAuthRoutes(env.mux, api.LauncherAuthRouteOpts{
		Sessions:      sessions,
		PasswordStore: store,
		OnDashboardClaimed: func() {
			// Exactly what main.go wires: the claim re-applies the desired
			// exposure, which drains and replaces the listeners on its own
			// goroutine.
			env.runtime.ReconcileAfterDashboardClaimed()
			env.claims <- struct{}{}
		},
	})

	handler := middleware.LauncherDashboardAuth(middleware.LauncherDashboardAuthConfig{
		Sessions: sessions,
	}, env.mux)

	// PC-DEF-039: an unclaimed dashboard is loopback-only whatever was asked for.
	effectivePublic, narrowed := effectiveLauncherExposure(desiredPublic, false)
	if desiredPublic && !narrowed {
		t.Fatal("an unclaimed dashboard with Public Mode requested must be narrowed")
	}
	if effectivePublic {
		t.Fatal("an unclaimed dashboard must never be bound beyond loopback")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	initial := netbind.OpenResult{Listeners: []net.Listener{listener}, Port: port}

	env.runtime = newLauncherHTTPRuntime(handler, "127.0.0.1", false, desiredPublic, initial)
	// Re-binding to a real LAN address is not this test's subject and is not
	// available in every sandbox, so the swap stays on loopback -- the listener
	// is still genuinely closed and replaced, which is what broke the response.
	env.runtime.open = func(_ string, _ bool, port string) (netbind.OpenResult, error) {
		replacement, listenErr := net.Listen("tcp", "127.0.0.1:"+port)
		if listenErr != nil {
			return netbind.OpenResult{}, listenErr
		}
		return netbind.OpenResult{Listeners: []net.Listener{replacement}, Port: port}, nil
	}
	env.runtime.Start()
	t.Cleanup(env.runtime.Shutdown)

	env.baseURL = "http://127.0.0.1:" + port
	jar := &journeyCookieJar{}
	env.client = &http.Client{Timeout: 10 * time.Second, Jar: jar}
	return env
}

// journeyCookieJar keeps the session cookie the way a browser would, so the
// authenticated steps are authenticated the same way.
type journeyCookieJar struct {
	cookies []*http.Cookie
}

func (j *journeyCookieJar) SetCookies(_ *url.URL, cookies []*http.Cookie) {
	j.cookies = append(j.cookies, cookies...)
}

func (j *journeyCookieJar) Cookies(_ *url.URL) []*http.Cookie { return j.cookies }

func (e *journeyEnv) post(t *testing.T, path string, body any) (int, string) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := e.client.Post(e.baseURL+path, "application/json", bytes.NewReader(encoded))
	if err != nil {
		// A transport error here is a finding, not a flake: it is what a user
		// sees as "setup failed" when the response never arrives.
		t.Fatalf("POST %s: the response never arrived: %v", path, err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(payload)
}

func (e *journeyEnv) get(t *testing.T, path string) (int, string) {
	t.Helper()
	resp, err := e.client.Get(e.baseURL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(payload)
}

func (e *journeyEnv) status(t *testing.T) (authenticated, initialized bool) {
	t.Helper()
	code, body := e.get(t, "/api/auth/status")
	if code != http.StatusOK {
		t.Fatalf("GET /api/auth/status = %d: %s", code, body)
	}
	var parsed struct {
		Authenticated bool `json:"authenticated"`
		Initialized   bool `json:"initialized"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("status body %q: %v", body, err)
	}
	return parsed.Authenticated, parsed.Initialized
}

const journeyPassword = "first-owner-password"

// The blocker: a fresh, unclaimed dashboard must be able to create its first
// password from the device, with Public Mode already requested.
//
// This is the case that failed physically with "must be authenticated to change
// password" -- the password was written, the response was lost, and the retry
// was then refused because the dashboard had become initialized.
func TestFreshInstallJourneyCanCreateTheFirstPassword(t *testing.T) {
	env := newJourneyEnv(t, true)

	if _, initialized := env.status(t); initialized {
		t.Fatal("a fresh install reports an initialized dashboard")
	}

	code, body := env.post(t, "/api/auth/setup", map[string]string{
		"password": journeyPassword,
		"confirm":  journeyPassword,
	})
	if code != http.StatusOK {
		t.Fatalf("first password setup = %d: %s", code, body)
	}

	// The claim is the event Public Mode reconciliation hangs off.
	select {
	case <-env.claims:
	case <-time.After(5 * time.Second):
		t.Fatal("the first claim did not fire the reconciliation")
	}
	// And the reconciliation actually widens the bind, eventually: it runs
	// asynchronously because it replaces the listener that served the setup
	// request.
	deadline := time.Now().Add(5 * time.Second)
	for !env.runtime.PublicMode() {
		if time.Now().After(deadline) {
			t.Fatal("the claim did not apply the requested Public Mode")
		}
		time.Sleep(5 * time.Millisecond)
	}

	if _, initialized := env.status(t); !initialized {
		t.Fatal("the dashboard is still uninitialized after a successful setup")
	}
}

// Same journey with Public Mode off: nothing is re-bound, and setup still works.
func TestFreshInstallJourneyCreatesTheFirstPasswordWithoutPublicMode(t *testing.T) {
	env := newJourneyEnv(t, false)

	code, body := env.post(t, "/api/auth/setup", map[string]string{
		"password": journeyPassword,
		"confirm":  journeyPassword,
	})
	if code != http.StatusOK {
		t.Fatalf("first password setup = %d: %s", code, body)
	}
	if _, initialized := env.status(t); !initialized {
		t.Fatal("the dashboard is still uninitialized after a successful setup")
	}
}

// After the first claim the owner can log in with the password they just set,
// and the destination-preserving redirect still works.
func TestFreshInstallJourneyLoginAfterFirstClaim(t *testing.T) {
	env := newJourneyEnv(t, false)

	if code, body := env.post(t, "/api/auth/setup", map[string]string{
		"password": journeyPassword,
		"confirm":  journeyPassword,
	}); code != http.StatusOK {
		t.Fatalf("first password setup = %d: %s", code, body)
	}

	if code, body := env.post(t, "/api/auth/login", map[string]string{
		"password": journeyPassword,
	}); code != http.StatusOK {
		t.Fatalf("login with the password just created = %d: %s", code, body)
	}

	authenticated, initialized := env.status(t)
	if !authenticated || !initialized {
		t.Fatalf("after login: authenticated=%v initialized=%v", authenticated, initialized)
	}
}

// An initialized dashboard may not have its password changed without a session,
// and /launcher-setup must not be a reset bypass. PC-DEF-037.
func TestJourneyInitializedDashboardRefusesUnauthenticatedPasswordChange(t *testing.T) {
	env := newJourneyEnv(t, false)

	if code, _ := env.post(t, "/api/auth/setup", map[string]string{
		"password": journeyPassword,
		"confirm":  journeyPassword,
	}); code != http.StatusOK {
		t.Fatal("first password setup failed")
	}

	// A different client, with no session, is an anonymous caller.
	anonymous := &http.Client{Timeout: 10 * time.Second}
	encoded, _ := json.Marshal(map[string]string{
		"password": "attacker-chosen-password",
		"confirm":  "attacker-chosen-password",
	})
	resp, err := anonymous.Post(env.baseURL+"/api/auth/setup",
		"application/json", bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous password change = %d, want 401", resp.StatusCode)
	}

	// And the original password still works, so nothing was overwritten.
	ok, err := env.store.VerifyPassword(context.Background(), journeyPassword)
	if err != nil || !ok {
		t.Fatalf("the original password no longer verifies: %v, %v", ok, err)
	}
}

// A remote first claim is refused even though the dashboard is unclaimed.
// PC-DEF-039, asserted at the handler because RemoteAddr is the whole decision.
func TestJourneyRemoteFirstClaimIsRefused(t *testing.T) {
	authDir := t.TempDir()
	store, err := dashboardauth.New(authDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	mux := http.NewServeMux()
	api.RegisterLauncherAuthRoutes(mux, api.LauncherAuthRouteOpts{
		Sessions:      middleware.NewLauncherDashboardSessions(12 * time.Hour),
		PasswordStore: store,
	})

	for _, remote := range []string{"192.168.1.50:44444", "10.0.0.9:1234", "not-an-address"} {
		body, _ := json.Marshal(map[string]string{
			"password": journeyPassword, "confirm": journeyPassword,
		})
		request := httptest.NewRequest(http.MethodPost, "/api/auth/setup", bytes.NewReader(body))
		request.RemoteAddr = remote
		// Spoofing the forwarding headers must not help.
		request.Header.Set("X-Forwarded-For", "127.0.0.1")
		request.Header.Set("Host", "localhost")
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("first claim from %s = %d, want 403", remote, recorder.Code)
		}
		if initialized, _ := store.IsInitialized(context.Background()); initialized {
			t.Fatalf("a refused claim from %s still initialized the dashboard", remote)
		}
	}
}

// Restarting the service must not lose ownership, and must not re-offer setup.
func TestJourneySurvivesAServiceRestart(t *testing.T) {
	authDir := t.TempDir()
	store, err := dashboardauth.New(authDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetPassword(context.Background(), journeyPassword); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// A restart is a new process opening the same private directory.
	reopened, err := dashboardauth.New(authDir)
	if err != nil {
		t.Fatalf("reopen after restart: %v", err)
	}
	defer reopened.Close()

	initialized, err := reopened.IsInitialized(context.Background())
	if err != nil || !initialized {
		t.Fatalf("after restart initialized=%v err=%v; ownership was lost", initialized, err)
	}
	ok, err := reopened.VerifyPassword(context.Background(), journeyPassword)
	if err != nil || !ok {
		t.Fatalf("after restart the password no longer verifies: %v %v", ok, err)
	}

	// And exposure is no longer narrowed, because the dashboard has an owner.
	effectivePublic, narrowed := effectiveLauncherExposure(true, true)
	if !effectivePublic || narrowed {
		t.Fatalf("a claimed dashboard with Public Mode on: public=%v narrowed=%v",
			effectivePublic, narrowed)
	}
}
