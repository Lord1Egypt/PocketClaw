package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// PC-DEF-034. The Android host must be able to act on "auto-start gateway" the
// moment the user turns it on, without a dashboard session -- and nothing that
// is not the host may use that power. The authorization assertions here are
// the point: this endpoint starts a process, so every way of reaching it
// without the per-process bridge token has to stay closed.

const testBridgeToken = "bridge-token-for-tests-0123456789abcdef"

func bridgeMux(t *testing.T) (*http.ServeMux, *Handler) {
	t.Helper()
	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	mux := http.NewServeMux()
	h.RegisterAndroidBridgeRoutes(mux, testBridgeToken)
	return mux, h
}

func bridgeStartRequest(token string, remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, androidGatewayStartBridgePath, nil)
	if token != "" {
		req.Header.Set("X-PocketClaw-Android-Bridge", token)
	}
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	return req
}

func TestAndroidBridgeGatewayStartRejectsMissingToken(t *testing.T) {
	mux, _ := bridgeMux(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, bridgeStartRequest("", "127.0.0.1:54321"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: an unauthenticated caller must not learn "+
			"the endpoint exists", rec.Code, http.StatusNotFound)
	}
}

func TestAndroidBridgeGatewayStartRejectsWrongToken(t *testing.T) {
	mux, _ := bridgeMux(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, bridgeStartRequest("not-the-right-token-aaaaaaaaaaaaaaaa", "127.0.0.1:54321"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// A token of a different length must be refused by length before the
// constant-time compare, and must not be treated as a prefix match.
func TestAndroidBridgeGatewayStartRejectsTruncatedToken(t *testing.T) {
	mux, _ := bridgeMux(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, bridgeStartRequest(testBridgeToken[:10], "127.0.0.1:54321"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// The correct token from off-device is still refused: the bridge is loopback
// only, so a leaked token does not become a remote start capability.
func TestAndroidBridgeGatewayStartRejectsNonLoopbackEvenWithTheToken(t *testing.T) {
	mux, _ := bridgeMux(t)
	for _, addr := range []string{"192.168.1.50:44444", "10.0.0.7:44444", "[2001:db8::1]:44444"} {
		t.Run(addr, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, bridgeStartRequest(testBridgeToken, addr))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d from %s, want %d: the bridge must stay loopback-only",
					rec.Code, addr, http.StatusNotFound)
			}
		})
	}
}

// A dashboard session is not authorization for host lifecycle control. The
// bridge must ignore cookies entirely.
func TestAndroidBridgeGatewayStartIgnoresDashboardCookies(t *testing.T) {
	mux, _ := bridgeMux(t)
	req := bridgeStartRequest("", "127.0.0.1:54321")
	req.AddCookie(&http.Cookie{Name: "launcher_dashboard", Value: "a-valid-looking-session"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: a dashboard session must never authorize "+
			"the Android bridge", rec.Code, http.StatusNotFound)
	}
}

// Registering the bridge must not also publish an unauthenticated
// /api/gateway/start on the same mux. PC-DEF-039 stays intact.
func TestAndroidBridgeDoesNotPublishAPublicGatewayStart(t *testing.T) {
	mux, _ := bridgeMux(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/gateway/start", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: the bridge must not open a public "+
			"gateway start route", rec.Code, http.StatusNotFound)
	}
}

// An empty bridge token disables the whole surface rather than accepting
// everything, matching the other bridge routes.
func TestAndroidBridgeGatewayStartUnregisteredWithoutAToken(t *testing.T) {
	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	mux := http.NewServeMux()
	h.RegisterAndroidBridgeRoutes(mux, "")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, bridgeStartRequest(testBridgeToken, "127.0.0.1:54321"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// Idempotency: an already-running gateway is reported, not started again.
func TestAndroidBridgeGatewayStartIsIdempotent(t *testing.T) {
	mux, _ := bridgeMux(t)

	cmd := startLongRunningProcess(t)
	t.Cleanup(func() {
		gateway.mu.Lock()
		if gateway.cmd == cmd {
			gateway.cmd = nil
		}
		gateway.mu.Unlock()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	gateway.mu.Lock()
	gateway.cmd = cmd
	gateway.mu.Unlock()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, bridgeStartRequest(testBridgeToken, "127.0.0.1:54321"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["status"] != "already_running" {
		t.Fatalf("status = %v, want already_running", body["status"])
	}

	gateway.mu.Lock()
	same := gateway.cmd == cmd
	gateway.mu.Unlock()
	if !same {
		t.Fatal("the running gateway was replaced by a duplicate process")
	}
}

// The response must never echo the bridge token.
func TestAndroidBridgeGatewayStartNeverEchoesTheToken(t *testing.T) {
	mux, _ := bridgeMux(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, bridgeStartRequest(testBridgeToken, "127.0.0.1:54321"))

	if bytesContains(rec.Body.Bytes(), testBridgeToken) {
		t.Fatal("the bridge token appeared in the response body")
	}
	for _, values := range rec.Header() {
		for _, v := range values {
			if v == testBridgeToken {
				t.Fatal("the bridge token appeared in a response header")
			}
		}
	}
}

func bytesContains(haystack []byte, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		stringIndex(string(haystack), needle) >= 0
}

func stringIndex(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
