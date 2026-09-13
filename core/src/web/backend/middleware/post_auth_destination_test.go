package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// PC-DEF-059, second attempt.
//
// The first fix was frontend-only and did nothing on the device. The real path:
// PocketClaw's native Settings opens the WebView at
// `http://127.0.0.1:18800/models?lng=en`, and an unauthenticated request is
// answered with a server-side 302 to /launcher-login — before any JavaScript runs.
// These tests exercise that server redirect with the URLs the native app actually
// launches, which is what the previous round's route-level tests could not see.

// The two paths the native cards open. Spelled here as literals on purpose: they
// are the constants in lib/src/ui/models_settings_card.dart and
// telegram_settings_card.dart, and a test that derived them from the same place as
// the code could not catch a change to either.
const (
	nativeModelsPath   = "/models"
	nativeTelegramPath = "/channels/telegram"
)

func TestNativeSettingsDestinationsSurviveTheServerRedirect(t *testing.T) {
	for _, requested := range []string{nativeModelsPath, nativeTelegramPath} {
		got := launcherLoginRedirectTarget(requested)
		if !strings.HasPrefix(got, launcherLoginPath+"?") {
			t.Errorf("%s: redirect = %q, want the login path with a destination",
				requested, got)
			continue
		}
		if !strings.Contains(got, PostAuthDestinationParam+"=") {
			t.Errorf("%s: redirect = %q, missing the %s parameter",
				requested, got, PostAuthDestinationParam)
		}
	}
}

// The whole flow, through the real middleware: the WebView's request, with the
// language query the app appends, answered by a redirect that remembers where it
// was going.
func TestUnauthenticatedPageRequestRedirectsWithItsDestination(t *testing.T) {
	handler := LauncherDashboardAuth(
		LauncherDashboardAuthConfig{},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	cases := map[string]string{
		// Exactly what lib/main.dart's _webUrl builds.
		"/models?lng=en":            "/models",
		"/channels/telegram?lng=en": "/channels/telegram",
		"/models":                   "/models",
		"/channels/telegram":        "/channels/telegram",
		"/agent/skills":             "/agent/skills",
		"/config/raw":               "/config/raw",
	}

	for requestURL, wantDestination := range cases {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, requestURL, nil))

		if recorder.Code != http.StatusFound {
			t.Errorf("%s: status = %d, want 302", requestURL, recorder.Code)
			continue
		}
		location := recorder.Header().Get("Location")
		parsedQuery := destinationFromLocation(t, location)
		if parsedQuery != wantDestination {
			t.Errorf("%s: Location = %q, destination = %q, want %q",
				requestURL, location, parsedQuery, wantDestination)
		}
	}
}

// An ordinary unauthenticated visit is unchanged: home is where login already goes,
// so remembering it would add a parameter that changes nothing.
func TestHomeAndAuthPagesProduceABareLoginRedirect(t *testing.T) {
	for _, canonicalPath := range []string{
		"/", "/launcher-login", "/launcher-setup", "/not-a-route", "/models/extra",
	} {
		if got := launcherLoginRedirectTarget(canonicalPath); got != launcherLoginPath {
			t.Errorf("%s: redirect = %q, want the bare login path", canonicalPath, got)
		}
	}
}

// The API and the websocket keep their own rejection shapes: a 302 to an HTML page
// is not a useful answer to a fetch.
func TestApiAndWebsocketRejectionsAreUnchanged(t *testing.T) {
	handler := LauncherDashboardAuth(
		LauncherDashboardAuthConfig{},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/models", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("api status = %d, want 401", recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "" {
		t.Errorf("an API rejection must not redirect, got Location %q", location)
	}
}

// The security contract. This value goes into a Location a browser follows, and the
// path comes from the request, so anyone can ask for anything.
func TestHostileDestinationsAreRefused(t *testing.T) {
	for _, hostile := range []string{
		"https://evil.example/",
		"http://evil.example/models",
		"//evil.example/",
		"//evil.example",
		"/\\evil.example",
		"\\\\evil.example",
		"javascript:alert(1)",
		"/java\x00script",
		"/models\nLocation: https://evil.example",
		"/models\r\nSet-Cookie: a=b",
		"/../etc/passwd",
		"/models/../../launcher-setup",
		"/channels/..",
		"/channels/TELEGRAM",
		"/channels/tele gram",
		"/channels/telegram/extra",
		"/channels/" + strings.Repeat("a", 80),
		"/api/config",
		"models",
		"",
	} {
		if isSafePostAuthDestination(hostile) {
			t.Errorf("%q must not be an allowed destination", hostile)
		}
		if got := launcherLoginRedirectTarget(hostile); got != launcherLoginPath {
			t.Errorf("%q: redirect = %q, want the bare login path", hostile, got)
		}
	}
}

// Whatever a request asks for, the Location must stay on this origin.
func TestRedirectLocationIsAlwaysRelative(t *testing.T) {
	handler := LauncherDashboardAuth(
		LauncherDashboardAuthConfig{},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}),
	)

	for _, requestURL := range []string{
		"/models", "/channels/telegram", "/not-a-route",
		"/models?next=https://evil.example",
		"//evil.example/models",
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, requestURL, nil))
		location := recorder.Header().Get("Location")
		if location == "" {
			continue
		}
		if !strings.HasPrefix(location, "/") || strings.HasPrefix(location, "//") {
			t.Errorf("%s: Location = %q, must be a same-origin relative path",
				requestURL, location)
		}
	}
}

// A destination the request carried itself must not be reflected: the middleware
// decides where to return to from the path it rejected, not from a parameter.
func TestAnAttackerSuppliedNextOnTheRequestIsIgnored(t *testing.T) {
	handler := LauncherDashboardAuth(
		LauncherDashboardAuthConfig{},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}),
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet, "/models?next=https%3A%2F%2Fevil.example", nil))

	location := recorder.Header().Get("Location")
	if strings.Contains(location, "evil.example") {
		t.Fatalf("Location = %q, must not reflect a request-supplied destination", location)
	}
	if destinationFromLocation(t, location) != "/models" {
		t.Fatalf("Location = %q, want the rejected path as the destination", location)
	}
}

// The backend cannot import the frontend's allowlist, so it is duplicated — and
// this reads the TypeScript to prove the two have not drifted.
func TestPostAuthDestinationsMatchTheFrontendAllowlist(t *testing.T) {
	source := filepath.Join(
		"..", "..", "frontend", "src", "lib", "post-auth-destination.ts")
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("reading the frontend allowlist: %v", err)
	}

	block := regexp.MustCompile(
		`(?s)STATIC_DESTINATIONS[^=]*=\s*new Set\(\[(.*?)\]\)`).FindSubmatch(raw)
	if block == nil {
		t.Fatal("could not find STATIC_DESTINATIONS in the frontend allowlist; " +
			"if it was renamed, update this test rather than deleting it")
	}
	frontend := map[string]struct{}{}
	for _, match := range regexp.MustCompile(`"([^"]+)"`).FindAllSubmatch(block[1], -1) {
		frontend[string(match[1])] = struct{}{}
	}

	// "/" is in the frontend set because resolvePostAuthDestination returns it as
	// the fallback; the backend never emits it as a destination.
	delete(frontend, "/")

	if len(frontend) == 0 {
		t.Fatal("parsed no routes from the frontend allowlist")
	}
	if !sameKeys(frontend, postAuthDestinations) {
		t.Fatalf("allowlists have drifted.\nfrontend: %v\nbackend:  %v",
			sortedKeys(frontend), sortedKeys(postAuthDestinations))
	}
}

func destinationFromLocation(t *testing.T, location string) string {
	t.Helper()
	parsed, err := http.NewRequest(http.MethodGet, location, nil)
	if err != nil {
		t.Fatalf("parsing Location %q: %v", location, err)
	}
	return parsed.URL.Query().Get(PostAuthDestinationParam)
}

func sameKeys(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if _, ok := b[key]; !ok {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
