package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// PC-DEF-080. The launch-at-login API wrote a macOS LaunchAgent, an XDG
// autostart file or a Windows Run key for the launcher binary. The Core only
// runs on Android, where runtime.GOOS is "android" and every branch answered
// "unsupported" — so its only consumer, a Config-page toggle that said so
// itself, went, and the route went with it.

func TestAutoStartRouteIsNotRegistered(t *testing.T) {
	mux := newRoutedHandler(t)
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		_, pattern := mux.Handler(httptest.NewRequest(method, "/api/system/autostart", nil))
		if pattern != "" {
			t.Fatalf("%s /api/system/autostart resolved to %q; the desktop "+
				"launch-at-login surface is back", method, pattern)
		}
	}
}

func TestAutoStartHandlerSourceIsRemoved(t *testing.T) {
	if _, err := os.Stat("startup.go"); !os.IsNotExist(err) {
		t.Fatalf("api/startup.go exists again (stat err = %v); PC-DEF-080 removed it", err)
	}
	router, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	if strings.Contains(string(router), "registerStartupRoutes") {
		t.Fatal("router.go still calls registerStartupRoutes")
	}
}
