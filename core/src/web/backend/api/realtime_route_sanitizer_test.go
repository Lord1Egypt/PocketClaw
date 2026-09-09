package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Redaction is keyed on the route, so moving the route moves the thing that
// hides it. These feed real lines through the real sanitizer rather than
// asserting that a pattern string changed.
func TestUserVisibleLogRedactsTheCanonicalRealtimeRoute(t *testing.T) {
	cases := []struct {
		name  string
		line  string
		want  string
		exact bool
	}{
		{
			name: "a routine successful websocket poll is dropped entirely",
			line: "DBG http server.go:88 > GET /pocketclaw/ws 101",
			want: "",
		},
		{
			name: "a failed websocket request keeps its status but not its path",
			line: "WRN http server.go:88 > GET /pocketclaw/ws 403",
			want: "/internal realtime connection",
		},
		{
			name: "the route field is replaced, not merely renamed",
			line: "INF channels manager.go:1291 > Webhook handler registered channel=pocketclaw path=/pocketclaw/",
			want: "path=<internal>",
		},
		{
			name: "the realtime log component is not shown as itself",
			line: "INF pocketclaw pocketclaw.go:248 > Starting PocketClaw realtime channel",
			want: "INF realtime realtime.go:248",
		},
		{
			name: "media requests carry no legacy prefix",
			line: "DBG http server.go:88 > GET /pocketclaw/media/abc 200",
			want: "/pocketclaw/media/abc",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeUserVisibleLog(tc.line)
			if tc.want == "" {
				if strings.TrimSpace(got) != "" {
					t.Fatalf("line survived: %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("normalizeUserVisibleLog(%q) = %q, want it to contain %q", tc.line, got, tc.want)
			}
			if strings.Contains(got, "/pocketclaw/ws") {
				t.Errorf("the realtime socket path reached a user-visible log: %q", got)
			}
		})
	}
}

// The legacy arm stays: a log file written before the migration must not become
// less redacted because the route moved.
func TestUserVisibleLogStillRedactsPreMigrationLines(t *testing.T) {
	if got := normalizeUserVisibleLog("DBG http server.go:88 > GET /pico/ws 101"); strings.TrimSpace(got) != "" {
		t.Errorf("a pre-migration routine poll survived: %q", got)
	}
	got := normalizeUserVisibleLog(
		"INF channels manager.go:1291 > Webhook handler registered channel=pico path=/pico/")
	if !strings.Contains(got, "path=<internal>") {
		t.Errorf("a pre-migration route was not redacted: %q", got)
	}
	if !strings.Contains(got, "channel=pocketclaw") {
		t.Errorf("a pre-migration channel label was not normalized: %q", got)
	}
}

// Redaction must stay narrow. A sanitizer broad enough to hide the realtime
// route by hiding everything would pass every test above and be useless.
func TestUserVisibleLogLeavesUnrelatedLinesAlone(t *testing.T) {
	for _, line := range []string{
		"INF tools loader.go:80 > Tools loaded count=17",
		"DBG http server.go:88 > GET /api/agents 200",
		"INF telegram telegram.go:120 > Telegram channel initialized",
		"DBG http server.go:88 > GET /pocketclawish/ws 200",
		"INF picometer picophone.go:42 > unrelated component",
	} {
		if got := normalizeUserVisibleLog(line); got != line {
			t.Errorf("normalizeUserVisibleLog(%q) = %q, want it unchanged", line, got)
		}
	}
}

// Credential material stays redacted through the renamed patterns.
func TestUserVisibleLogStillRedactsCredentials(t *testing.T) {
	cases := map[string]string{
		"DBG http server.go:1 > authorization=Bearer sk-secret-value":     "sk-secret-value",
		"INF agent agent.go:1 > Routed message session_key=sk_v1_SECRET":  "sk_v1_SECRET",
		"INF agent agent.go:1 > Routed message chat_id=pocketclaw:abc123": "abc123",
	}
	for line, secret := range cases {
		got := normalizeUserVisibleLog(line)
		if strings.Contains(got, secret) {
			t.Errorf("normalizeUserVisibleLog(%q) leaked %q: %q", line, secret, got)
		}
	}
}

// The routes the console actually serves, checked against a real mux rather
// than against the source that registers them.
func TestRealtimeRoutesAreRegisteredUnderTheCanonicalNamespace(t *testing.T) {
	h := NewHandler(t.TempDir() + "/config.json")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	canonical := []struct {
		method string
		path   string
	}{
		{http.MethodGet, config.RealtimeWebSocketPath},
		{http.MethodGet, config.RealtimeMediaPrefix + "abc"},
		{http.MethodHead, config.RealtimeMediaPrefix + "abc"},
		{http.MethodGet, config.RealtimeAPIPrefix + "info"},
		{http.MethodPost, config.RealtimeAPIPrefix + "token"},
		{http.MethodPost, config.RealtimeAPIPrefix + "setup"},
	}
	for _, route := range canonical {
		_, pattern := mux.Handler(httptest.NewRequest(route.method, route.path, nil))
		if pattern == "" {
			t.Errorf("%s %s is not registered", route.method, route.path)
		}
	}

	// No alias. The server, the console and the app ship in one artifact, so an
	// old client is an old tab that reloads — not a supported caller.
	legacy := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/pico/ws"},
		{http.MethodGet, "/pico/media/abc"},
		{http.MethodHead, "/pico/media/abc"},
		{http.MethodGet, "/api/pico/info"},
		{http.MethodPost, "/api/pico/token"},
		{http.MethodPost, "/api/pico/setup"},
		{http.MethodGet, "/pico/events"},
		{http.MethodPost, "/pico/send"},
	}
	for _, route := range legacy {
		request := httptest.NewRequest(route.method, route.path, nil)
		_, pattern := mux.Handler(request)
		if pattern != "" {
			t.Errorf("%s %s still resolves to %q", route.method, route.path, pattern)
		}
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s %s status = %d, want %d", route.method, route.path, recorder.Code, http.StatusNotFound)
		}
	}
}
