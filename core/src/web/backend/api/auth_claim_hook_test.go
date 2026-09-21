package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/web/backend/middleware"
)

// PC-DEF-040, reopened. A first claim is the authoritative "dashboard became owned"
// event, and it had no detector on this side: exposure was reconciled only when the
// Android app saw its embedded WebView leave /launcher-setup.
//
// These assert the hook fires exactly when it should, because both mistakes are
// serious: not firing leaves the owner with an unreachable LAN dashboard, and firing
// on the wrong request would re-expose one.

// claimRequest is setupRequest with a caller-chosen body; auth_test.go already owns
// a fixed-body helper of its own.
func claimRequest(body string, remoteAddr string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/setup", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = remoteAddr
	return request
}

func TestAFirstClaimFiresTheReconciliationHook(t *testing.T) {
	store := &fakePasswordStore{initialized: false}
	claims := 0
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie:      "session-cookie-value",
		PasswordStore:      store,
		OnDashboardClaimed: func() { claims++ },
	})

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, claimRequest(
		`{"password":"dashboard-test-password","confirm":"dashboard-test-password"}`,
		"127.0.0.1:54321",
	))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if claims != 1 {
		t.Fatalf("claim hook fired %d times, want exactly 1", claims)
	}
}

// A password change is not a claim: the dashboard already had an owner, and nothing
// about its exposure has changed.
func TestAPasswordChangeDoesNotFireTheHook(t *testing.T) {
	const sess = "session-cookie-value"
	store := &fakePasswordStore{initialized: true, password: "old-dashboard-password"}
	claims := 0
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie:      sess,
		PasswordStore:      store,
		OnDashboardClaimed: func() { claims++ },
	})

	request := claimRequest(
		`{"password":"new-dashboard-password","confirm":"new-dashboard-password"}`,
		"127.0.0.1:54321",
	)
	// The exported constant, not a literal: the cookie was renamed once already.
	request.AddCookie(&http.Cookie{
		Name:  middleware.LauncherDashboardCookieName,
		Value: sess,
	})

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if claims != 0 {
		t.Fatalf("claim hook fired %d times on a password change, want 0", claims)
	}
}

// PC-DEF-039 is upstream of the hook: an off-device first claim is refused, so the
// hook cannot be reached by a remote client and can never widen exposure for one.
func TestARemoteFirstClaimIsRefusedAndFiresNothing(t *testing.T) {
	store := &fakePasswordStore{initialized: false}
	claims := 0
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie:      "session-cookie-value",
		PasswordStore:      store,
		OnDashboardClaimed: func() { claims++ },
	})

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, claimRequest(
		`{"password":"dashboard-test-password","confirm":"dashboard-test-password"}`,
		"192.168.1.50:54321",
	))

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: a remote first claim must be refused",
			recorder.Code)
	}
	if claims != 0 {
		t.Fatalf("claim hook fired %d times for a remote claim, want 0", claims)
	}
}

// A rejected claim must not fire either, or a failed setup on a Public-Mode device
// would expose an unclaimed dashboard.
func TestAFailedClaimFiresNothing(t *testing.T) {
	claims := 0
	newMux := func(store *fakePasswordStore) *http.ServeMux {
		mux := http.NewServeMux()
		RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
			SessionCookie:      "session-cookie-value",
			PasswordStore:      store,
			OnDashboardClaimed: func() { claims++ },
		})
		return mux
	}

	cases := map[string]string{
		"mismatched confirmation": `{"password":"dashboard-test-password","confirm":"different"}`,
		"too short":               `{"password":"short","confirm":"short"}`,
		"empty":                   `{"password":"","confirm":""}`,
		"malformed json":          `{`,
	}

	for name, body := range cases {
		recorder := httptest.NewRecorder()
		newMux(&fakePasswordStore{initialized: false}).ServeHTTP(
			recorder, claimRequest(body, "127.0.0.1:54321"))
		if recorder.Code == http.StatusOK {
			t.Errorf("%s: status = 200, want a rejection", name)
		}
	}

	if claims != 0 {
		t.Fatalf("claim hook fired %d times across rejected claims, want 0", claims)
	}
}

// A host with no network-mode controller passes nil, and the claim must still work.
func TestAClaimWithNoHookStillSucceeds(t *testing.T) {
	store := &fakePasswordStore{initialized: false}
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
		PasswordStore: store,
	})

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, claimRequest(
		`{"password":"dashboard-test-password","confirm":"dashboard-test-password"}`,
		"127.0.0.1:54321",
	))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
