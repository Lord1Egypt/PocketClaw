package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// PC-DEF-022. PocketClaw does not expose self-update through the Core HTTP API.
//
// The inherited upstream route took a caller-supplied URL, downloaded it,
// extracted the archive and handed the result to selfupdate.Apply. It required
// a dashboard session and no PocketClaw UI ever called it, and on Android the
// apply step could not succeed because the install directory is read-only — but
// what remained was an authenticated arbitrary-URL fetch and archive extraction
// surface on a route the product does not use.
//
// Securing an unused self-update subsystem would have been the wrong repair.
// The route is gone. These pin that it stays gone, and that its removal did not
// quietly become something else: a 404 is the router's ordinary answer for an
// unknown API path, not a special case written for this endpoint.

func newRoutedHandler(t *testing.T) *http.ServeMux {
	t.Helper()
	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestUpdateRouteIsNotRegistered(t *testing.T) {
	mux := newRoutedHandler(t)
	for _, method := range []string{
		http.MethodPost, http.MethodGet, http.MethodPut,
		http.MethodPatch, http.MethodDelete,
	} {
		t.Run(method, func(t *testing.T) {
			_, pattern := mux.Handler(httptest.NewRequest(method, "/api/update", nil))
			if pattern != "" {
				t.Fatalf("%s /api/update resolved to registered pattern %q; the "+
					"self-update surface is back", method, pattern)
			}
		})
	}
}

// An authenticated caller is the threat model this route had, so the test uses
// the routed mux directly — past any auth wall — and still must not reach
// updater execution.
func TestAuthenticatedRequestCannotReachTheUpdater(t *testing.T) {
	mux := newRoutedHandler(t)
	body := strings.NewReader(`{"url":"https://example.invalid/release.zip","binary":"picoclaw"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/update", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST /api/update = %d, want %d; anything else means a handler ran",
			rec.Code, http.StatusNotFound)
	}
	// A handler that answered would have produced this package's JSON envelope.
	if strings.Contains(rec.Body.String(), `"status"`) {
		t.Fatalf("response carries an update handler's envelope: %s", rec.Body.String())
	}
}

// Removing a route must not leave a differently-named one doing the same thing.
func TestNoRouteAcceptsACallerSuppliedDownloadURL(t *testing.T) {
	mux := newRoutedHandler(t)
	for _, path := range []string{
		"/api/update", "/api/updates", "/api/self-update", "/api/selfupdate",
		"/api/system/update", "/api/upgrade", "/api/download",
	} {
		t.Run(path, func(t *testing.T) {
			_, pattern := mux.Handler(httptest.NewRequest(http.MethodPost, path, nil))
			if pattern != "" {
				t.Fatalf("POST %s resolved to %q; an arbitrary-download surface "+
					"must not reappear under another name", path, pattern)
			}
		})
	}
}

// The removal must not have widened the unauthenticated surface either. The
// route was behind the session wall; deleting it cannot be what puts something
// in front of one.
func TestRemovalAddedNoUnauthenticatedRoute(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(
		"..", "middleware", "launcher_dashboard_auth.go"))
	if err != nil {
		t.Fatalf("read auth middleware: %v", err)
	}
	if strings.Contains(string(source), "/api/update") {
		t.Fatal("the auth middleware names /api/update; the removed route must " +
			"not appear in the unauthenticated allowlist")
	}
}

// The handler file itself is gone. Asserting this keeps a future revert from
// restoring the surface with the router line still absent, which would leave
// dead code that reads as live product.
func TestUpdateHandlerSourceIsRemoved(t *testing.T) {
	if _, err := os.Stat("update.go"); !os.IsNotExist(err) {
		t.Fatalf("api/update.go exists again (stat err = %v); PC-DEF-022 removed it", err)
	}
	router, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	if strings.Contains(string(router), "registerUpdateRoutes") {
		t.Fatal("router.go still calls registerUpdateRoutes")
	}
}
