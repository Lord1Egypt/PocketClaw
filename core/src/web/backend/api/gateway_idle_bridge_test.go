package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// PC-DEF-030 / C1. This endpoint can cause a gateway restart, so every way of
// reaching it without the current generation's credential has to stay closed.

func idleMux(t *testing.T) (*http.ServeMux, string) {
	t.Helper()
	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	mux := http.NewServeMux()
	h.RegisterGatewayIdleRoute(mux)
	token := newGatewayIdleToken()
	t.Cleanup(clearGatewayIdleToken)
	t.Cleanup(func() { takePendingConfigApply(); setPendingConfigApplyError("") })
	return mux, token
}

func idleRequest(token, remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, androidGatewayIdlePath, nil)
	if token != "" {
		req.Header.Set("X-PocketClaw-Gateway-Idle", token)
	}
	req.RemoteAddr = remoteAddr
	return req
}

func TestGatewayIdleAcceptsTheCurrentTokenFromLoopback(t *testing.T) {
	mux, token := idleMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, idleRequest(token, "127.0.0.1:51000"))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("code = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

func TestGatewayIdleRejectsMissingAndWrongTokens(t *testing.T) {
	mux, token := idleMux(t)

	for name, given := range map[string]string{
		"missing":   "",
		"wrong":     "0000000000000000000000000000000000000000000000000000000000000000",
		"truncated": token[:16],
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, idleRequest(given, "127.0.0.1:51000"))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("code = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

// A superseded gateway must not drive the lifecycle of the one that replaced
// it: spawning rotates the credential.
func TestGatewayIdleRejectsAPreviousGenerationToken(t *testing.T) {
	mux, previous := idleMux(t)

	current := newGatewayIdleToken()
	if current == previous {
		t.Fatal("the idle token did not rotate on a new gateway generation")
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, idleRequest(previous, "127.0.0.1:51000"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("a stale generation's token was accepted (code = %d)", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, idleRequest(current, "127.0.0.1:51000"))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("the current generation was refused (code = %d)", rec.Code)
	}
}

// No current gateway means no valid credential at all.
func TestGatewayIdleRejectsEverythingWhenNoGatewayIsCurrent(t *testing.T) {
	mux, token := idleMux(t)
	clearGatewayIdleToken()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, idleRequest(token, "127.0.0.1:51000"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want %d with no current gateway", rec.Code, http.StatusNotFound)
	}
}

func TestGatewayIdleRejectsNonLoopbackEvenWithTheToken(t *testing.T) {
	mux, token := idleMux(t)

	for _, addr := range []string{"192.168.1.50:44444", "10.0.0.7:44444", "[2001:db8::1]:44444"} {
		t.Run(addr, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, idleRequest(token, addr))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("code = %d from %s, want %d", rec.Code, addr, http.StatusNotFound)
			}
		})
	}
}

func TestGatewayIdleIgnoresSpoofableHeaders(t *testing.T) {
	mux, token := idleMux(t)

	for _, h := range []struct{ key, value string }{
		{"X-Forwarded-For", "127.0.0.1"},
		{"X-Real-IP", "127.0.0.1"},
		{"Forwarded", "for=127.0.0.1"},
		{"Origin", "http://localhost:18800"},
	} {
		t.Run(h.key, func(t *testing.T) {
			req := idleRequest(token, "192.168.1.50:44444")
			req.Header.Set(h.key, h.value)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("spoofed %s was accepted (code = %d)", h.key, rec.Code)
			}
		})
	}
}

// The gateway holds a purpose-scoped credential. Neither the Android bridge
// token nor a dashboard session authorizes this.
func TestGatewayIdleRefusesOtherCredentials(t *testing.T) {
	mux, _ := idleMux(t)

	req := idleRequest("", "127.0.0.1:51000")
	req.Header.Set("X-PocketClaw-Android-Bridge", testBridgeToken)
	req.AddCookie(&http.Cookie{Name: "launcher_dashboard", Value: "a-valid-looking-session"})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("an unrelated credential authorized the idle endpoint (code = %d)", rec.Code)
	}
}

// An idle gateway with nothing pending is the common case and must be inert.
func TestGatewayIdleWithNoPendingApplyIsHarmless(t *testing.T) {
	mux, token := idleMux(t)
	takePendingConfigApply()

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, idleRequest(token, "127.0.0.1:51000"))
		if rec.Code != http.StatusAccepted {
			t.Fatalf("code = %d, want %d", rec.Code, http.StatusAccepted)
		}
	}
	if pending, _ := pendingConfigApplyState(); pending {
		t.Fatal("an idle notification invented pending work")
	}
}

// Repeated notifications must not each claim the same pending change: the
// first claims it, the rest find nothing. That is what stops a restart storm.
func TestPendingApplyIsClaimedExactlyOnce(t *testing.T) {
	t.Cleanup(func() { takePendingConfigApply(); setPendingConfigApplyError("") })

	markConfigApplyPending("telegram_configured")
	if pending, _ := pendingConfigApplyState(); !pending {
		t.Fatal("pending was not recorded")
	}

	if first := takePendingConfigApply(); first != "telegram_configured" {
		t.Fatalf("first claim = %q, want the pending reason", first)
	}
	if second := takePendingConfigApply(); second != "" {
		t.Fatalf("second claim = %q, want empty: the change was already claimed", second)
	}
}

// A, B, C while busy collapse to one pending entry, and the config file
// already holds C, so one restart converges on the latest configuration.
func TestRepeatedSavesCollapseIntoOnePendingApply(t *testing.T) {
	t.Cleanup(func() { takePendingConfigApply(); setPendingConfigApplyError("") })

	markConfigApplyPending("telegram_a")
	markConfigApplyPending("telegram_b")
	markConfigApplyPending("telegram_c")

	if got := takePendingConfigApply(); got != "telegram_c" {
		t.Fatalf("pending reason = %q, want the latest", got)
	}
	if got := takePendingConfigApply(); got != "" {
		t.Fatalf("a second apply was queued: %q", got)
	}
}

// Status is observational. Reading it must never apply, clear or retry.
func TestGatewayStatusNeverActsOnPendingApply(t *testing.T) {
	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	t.Cleanup(func() { takePendingConfigApply(); setPendingConfigApplyError("") })
	markConfigApplyPending("telegram_configured")

	for i := 0; i < 3; i++ {
		data := h.gatewayStatusData()
		if data["config_apply_pending"] != true {
			t.Fatalf("status did not report the pending apply: %#v", data["config_apply_pending"])
		}
	}

	if pending, _ := pendingConfigApplyState(); !pending {
		t.Fatal("reading status consumed the pending apply; monitoring must be side-effect free")
	}
}
