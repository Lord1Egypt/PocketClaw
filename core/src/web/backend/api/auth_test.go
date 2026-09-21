package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/web/backend/middleware"
)

type fakePasswordStore struct {
	initialized bool
	password    string
	err         error
}

func (s *fakePasswordStore) IsInitialized(context.Context) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.initialized, nil
}

func (s *fakePasswordStore) SetPassword(_ context.Context, plain string) error {
	if s.err != nil {
		return s.err
	}
	s.password = plain
	s.initialized = true
	return nil
}

func (s *fakePasswordStore) VerifyPassword(_ context.Context, plain string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.initialized && plain == s.password, nil
}

func TestLauncherAuthLoginAndStatus(t *testing.T) {
	const password = "dashboard-test-password"
	const sess = "session-cookie-value"
	store := &fakePasswordStore{initialized: true, password: password}
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: sess,
		PasswordStore: store,
	})

	t.Run("status_unauthenticated", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/status", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d", rec.Code)
		}
		var body struct {
			Authenticated bool `json:"authenticated"`
			Initialized   bool `json:"initialized"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Authenticated {
			t.Fatalf("unexpected authenticated=true: %+v", body)
		}
	})

	t.Run("login_ok", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"`+password+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:12345"
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("login code = %d body=%s", rec.Code, rec.Body.String())
		}
		sessionCookie(t, rec.Result().Cookies())
	})

	t.Run("status_authenticated", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
		req.AddCookie(&http.Cookie{Name: middleware.LauncherDashboardCookieName, Value: sess})
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"authenticated":true`)) {
			t.Fatalf("body = %s", rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "token_help") {
			t.Fatalf("authenticated response should omit token_help: %s", rec.Body.String())
		}
	})
}

func TestLauncherAuthLogoutRevokesServerSession(t *testing.T) {
	store := &fakePasswordStore{initialized: true, password: "dashboard-test-password"}
	sessions := middleware.NewLauncherDashboardSessions(time.Hour)
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		Sessions:      sessions,
		PasswordStore: store,
	})

	login := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"dashboard-test-password"}`))
	login.Header.Set("Content-Type", "application/json")
	login.RemoteAddr = "127.0.0.1:12345"
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, login)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status=%d cookies=%#v", loginRec.Code, loginRec.Result().Cookies())
	}
	cookie := sessionCookie(t, loginRec.Result().Cookies())
	if !sessions.Valid(cookie.Value) {
		t.Fatal("issued login session is not active")
	}

	logout := httptest.NewRequest(http.MethodPost, "/api/auth/logout", strings.NewReader(`{}`))
	logout.Header.Set("Content-Type", "application/json")
	logout.AddCookie(cookie)
	logoutRec := httptest.NewRecorder()
	mux.ServeHTTP(logoutRec, logout)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status=%d body=%s", logoutRec.Code, logoutRec.Body.String())
	}
	if sessions.Valid(cookie.Value) {
		t.Fatal("logout did not revoke the server-side session")
	}
}

func TestLauncherAuthUninitializedStoreRequiresSetup(t *testing.T) {
	const sess = "session-cookie-value"
	store := &fakePasswordStore{}
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: sess,
		PasswordStore: store,
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Authenticated bool `json:"authenticated"`
		Initialized   bool `json:"initialized"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Initialized {
		t.Fatalf("initialized = true, want false before setup")
	}
	if body.Authenticated {
		t.Fatalf("unexpected authenticated=true: %+v", body)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"not-set-yet"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("login before setup code = %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(
		http.MethodPost,
		"/api/auth/setup",
		strings.NewReader(`{"password":"12345678","confirm":"12345678"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	// First-claim setup is loopback-only since PC-DEF-039. This test is about
	// the setup flow, not about where it may come from, so it takes the
	// legitimate local path; the origin rules have their own coverage.
	req.RemoteAddr = "127.0.0.1:51000"
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup code = %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"12345678"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login after setup code = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLauncherAuthSetupRequiresSessionWhenInitialized(t *testing.T) {
	const sess = "session-cookie-value"
	store := &fakePasswordStore{initialized: true, password: "old-password"}
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: sess,
		PasswordStore: store,
	})

	body := strings.NewReader(`{"password":"new-password","confirm":"new-password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("setup without session code = %d body=%s", rec.Code, rec.Body.String())
	}

	body = strings.NewReader(`{"password":"new-password","confirm":"new-password"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/setup", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: middleware.LauncherDashboardCookieName, Value: sess})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup with session code = %d body=%s", rec.Code, rec.Body.String())
	}
	if store.password != "new-password" {
		t.Fatalf("password = %q, want new-password", store.password)
	}
}

// Replaces TestLauncherAuthInitialSetupAllowsDirectSetup, which asserted that
// an uninitialized dashboard accepts a password from any client with no
// authorization whatsoever. That was the defect, written down as the contract:
// the session check in handleSetup sits inside `if initialized`, so before
// PC-DEF-039 whoever reached the port first became the owner of the agent --
// on a Public-Mode device, anyone on the LAN. The old test could never fail,
// because passing it only required the vulnerability to still be present.
//
// The contract is now loopback-vs-remote, and that is what these assert.

func setupRequest(remoteAddr string) *http.Request {
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/setup",
		strings.NewReader(`{"password":"12345678","confirm":"12345678"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	return req
}

func uninitializedAuthMux() (*http.ServeMux, *fakePasswordStore) {
	store := &fakePasswordStore{}
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
		PasswordStore: store,
	})
	return mux, store
}

func TestInitialSetupAllowedFromIPv4Loopback(t *testing.T) {
	mux, store := uninitializedAuthMux()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, setupRequest("127.0.0.1:51000"))

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !store.initialized {
		t.Fatal("the local owner could not claim an unclaimed dashboard")
	}
}

func TestInitialSetupAllowedFromIPv6Loopback(t *testing.T) {
	mux, store := uninitializedAuthMux()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, setupRequest("[::1]:51000"))

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !store.initialized {
		t.Fatal("IPv6 loopback is this device too")
	}
}

func TestInitialSetupDeniedFromLAN(t *testing.T) {
	for _, addr := range []string{
		"192.168.1.50:44444",
		"10.0.0.7:44444",
		"172.16.4.9:44444",
		"[2001:db8::1]:44444",
	} {
		t.Run(addr, func(t *testing.T) {
			mux, store := uninitializedAuthMux()

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, setupRequest(addr))

			if rec.Code == http.StatusOK {
				t.Fatalf("a LAN client claimed the dashboard from %s", addr)
			}
			if store.initialized {
				t.Fatalf("a password was set by a remote first-claim from %s", addr)
			}
		})
	}
}

// Host, Origin and X-Forwarded-For are request content. A LAN attacker sets
// them freely, so the decision must ignore them entirely.
func TestInitialSetupIgnoresSpoofableHeaders(t *testing.T) {
	headers := []struct {
		name  string
		key   string
		value string
	}{
		{"Host", "Host", "localhost"},
		{"X-Forwarded-For", "X-Forwarded-For", "127.0.0.1"},
		{"X-Real-IP", "X-Real-IP", "127.0.0.1"},
		{"Origin", "Origin", "http://localhost:18800"},
		{"Forwarded", "Forwarded", "for=127.0.0.1"},
	}
	for _, h := range headers {
		t.Run(h.name, func(t *testing.T) {
			mux, store := uninitializedAuthMux()

			req := setupRequest("192.168.1.50:44444")
			if h.key == "Host" {
				req.Host = h.value
			} else {
				req.Header.Set(h.key, h.value)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK || store.initialized {
				t.Fatalf("spoofed %s let a LAN client claim the dashboard", h.name)
			}
		})
	}
}

// An unparseable RemoteAddr is not loopback. Fail closed.
func TestInitialSetupDeniedWhenRemoteAddrIsUnparseable(t *testing.T) {
	for _, addr := range []string{"", "garbage", "127.0.0.1", "not:a:port"} {
		t.Run(addr, func(t *testing.T) {
			mux, store := uninitializedAuthMux()

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, setupRequest(addr))

			if rec.Code == http.StatusOK || store.initialized {
				t.Fatalf("an unparseable RemoteAddr %q was treated as loopback", addr)
			}
		})
	}
}

// The initialized contract is unchanged: loopback is not a substitute for a
// session once an owner exists.
func TestInitializedSetupStillRequiresASessionEvenFromLoopback(t *testing.T) {
	store := &fakePasswordStore{initialized: true}
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
		PasswordStore: store,
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, setupRequest("127.0.0.1:51000"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want %d: an existing owner must not be overwritten "+
			"just because the request came from this device", rec.Code, http.StatusUnauthorized)
	}
}

func TestLauncherAuthStoreUnavailableFailsClosed(t *testing.T) {
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
		StoreError:    errors.New("open auth store"),
	})

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "status", method: http.MethodGet, path: "/api/auth/status"},
		{name: "login", method: http.MethodPost, path: "/api/auth/login", body: `{"password":"password"}`},
		{name: "setup", method: http.MethodPost, path: "/api/auth/setup", body: `{"password":"12345678","confirm":"12345678"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestLauncherAuthLogoutRequiresPostAndJSON(t *testing.T) {
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil))
	if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
		t.Fatalf("GET logout: code = %d (expected 404 or 405)", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("wrong content-type: code = %d body=%s", rec2.Code, rec2.Body.String())
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/api/auth/logout", strings.NewReader(`{}`))
	req3.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("POST json logout: code = %d", rec3.Code)
	}
}

func TestLauncherAuthLoginRateLimit(t *testing.T) {
	store := &fakePasswordStore{initialized: true, password: "correct-password"}
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
		PasswordStore: store,
	})

	// 11 failing logins by wrong password; each consumes allow() slot after valid JSON.
	wrongBody := `{"password":"wrong"}`
	for i := 0; i < loginAttemptsPerIP; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(wrongBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.5.5:9999"
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("iter %d: want 401 got %d %s", i, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(wrongBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.5.5:9999"
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("11th attempt: want 429 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestLoginRateLimiterWindow(t *testing.T) {
	l := newLoginRateLimiter()
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return t0 }
	for i := 0; i < loginAttemptsPerIP; i++ {
		if !l.allow("ip") {
			t.Fatalf("want allow at %d", i)
		}
	}
	if l.allow("ip") {
		t.Fatal("want deny on 11th")
	}
	l.now = func() time.Time { return t0.Add(loginAttemptWindow + time.Second) }
	if !l.allow("ip") {
		t.Fatal("want allow after window")
	}
}

func TestReferrerPolicyMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := middleware.ReferrerPolicyNoReferrer(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q", got)
	}
}

func TestLauncherAuthLogoutEmptyBody(t *testing.T) {
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = http.NoBody
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestLauncherAuthLogoutRejectsTrailingJSON(t *testing.T) {
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", strings.NewReader(`{}{}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d %s", rec.Code, rec.Body.String())
	}
}

// sessionCookie returns the response's canonical session cookie, asserting that
// the only other cookie set is the deletion of the legacy name — never a second
// session, and never a legacy session.
func sessionCookie(t *testing.T, cookies []*http.Cookie) *http.Cookie {
	t.Helper()
	var session *http.Cookie
	for _, c := range cookies {
		switch c.Name {
		case middleware.LauncherDashboardCookieName:
			if session != nil {
				t.Fatalf("more than one session cookie: %#v", cookies)
			}
			session = c
		case "picoclaw_launcher_auth":
			if c.Value != "" || c.MaxAge >= 0 {
				t.Fatalf("the legacy cookie was issued rather than expired: %#v", c)
			}
		default:
			t.Fatalf("unexpected cookie %q: %#v", c.Name, cookies)
		}
	}
	if session == nil {
		t.Fatalf("no session cookie: %#v", cookies)
	}
	return session
}
