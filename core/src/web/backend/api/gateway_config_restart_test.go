package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/health"
)

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }

func healthResponseWith(t *testing.T, status *health.StatusResponse) *http.Response {
	t.Helper()
	body, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("cannot encode health response: %v", err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

// A gateway that reports itself busy must not be restarted out from under a
// running answer. Waiting is why apply-config exists separately from the manual
// restart action.
func TestWaitForGatewayIdle_WaitsWhileBusyThenProceeds(t *testing.T) {
	original := gatewayHealthGet
	t.Cleanup(func() { gatewayHealthGet = original })

	var mu sync.Mutex
	probes := 0
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		mu.Lock()
		probes++
		busy := probes < 3
		mu.Unlock()
		active := 0
		if busy {
			active = 1
		}
		return healthResponseWith(t, &health.StatusResponse{
			Status: "ok", Busy: boolPtr(busy), ActiveRequests: intPtr(active),
		}), nil
	}

	h := NewHandler(t.TempDir() + "/config.json")
	waited := h.waitForGatewayIdle()

	if !waited {
		t.Fatal("a busy gateway must be waited for")
	}
	mu.Lock()
	defer mu.Unlock()
	if probes < 3 {
		t.Fatalf("expected polling until idle, got %d probes", probes)
	}
}

func TestWaitForGatewayIdle_DoesNotWaitWhenIdle(t *testing.T) {
	original := gatewayHealthGet
	t.Cleanup(func() { gatewayHealthGet = original })

	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return healthResponseWith(t, &health.StatusResponse{
			Status: "ok", Busy: boolPtr(false), ActiveRequests: intPtr(0),
		}), nil
	}

	h := NewHandler(t.TempDir() + "/config.json")
	if h.waitForGatewayIdle() {
		t.Fatal("an idle gateway must not be waited for")
	}
}

// An older gateway build cannot report busyness. Blocking forever on a signal
// that will never arrive would make configuration changes impossible to apply.
func TestWaitForGatewayIdle_ProceedsWhenBusynessIsUnreported(t *testing.T) {
	original := gatewayHealthGet
	t.Cleanup(func() { gatewayHealthGet = original })

	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return healthResponseWith(t, &health.StatusResponse{Status: "ok"}), nil
	}

	h := NewHandler(t.TempDir() + "/config.json")
	if h.waitForGatewayIdle() {
		t.Fatal("an unreported busy field must not block the restart")
	}
}

func TestWaitForGatewayIdle_ProceedsWhenHealthIsUnreachable(t *testing.T) {
	original := gatewayHealthGet
	t.Cleanup(func() { gatewayHealthGet = original })

	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return nil, http.ErrServerClosed
	}

	h := NewHandler(t.TempDir() + "/config.json")
	if h.waitForGatewayIdle() {
		t.Fatal("an unreachable gateway cannot be busy; do not block on it")
	}
}

// The health server distinguishes "zero in flight" from "not reported", so a
// restart decision cannot misread a missing field as an idle gateway.
func TestHealthReportsActiveRequestsWhenProbeIsSet(t *testing.T) {
	server := health.NewServer("127.0.0.1", 0, "")
	server.SetActiveRequestsProbe(func() int { return 2 })

	mux := http.NewServeMux()
	server.RegisterOnMux(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	var body health.StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.ActiveRequests == nil || *body.ActiveRequests != 2 {
		t.Fatalf("expected 2 active requests, got %v", body.ActiveRequests)
	}
	if body.Busy == nil || !*body.Busy {
		t.Fatalf("two in-flight turns is busy, got %v", body.Busy)
	}
}

func TestHealthOmitsBusyWhenNoProbeIsSet(t *testing.T) {
	server := health.NewServer("127.0.0.1", 0, "")

	mux := http.NewServeMux()
	server.RegisterOnMux(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	var body health.StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Absent, not false: a gateway with no probe has not claimed to be idle.
	if body.Busy != nil || body.ActiveRequests != nil {
		t.Fatalf("expected the fields to be omitted, got busy=%v active=%v",
			body.Busy, body.ActiveRequests)
	}
}

func TestHealthReportsIdleAsNotBusy(t *testing.T) {
	server := health.NewServer("127.0.0.1", 0, "")
	server.SetActiveRequestsProbe(func() int { return 0 })

	mux := http.NewServeMux()
	server.RegisterOnMux(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	var body health.StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Busy == nil || *body.Busy {
		t.Fatalf("zero in-flight turns is idle, got %v", body.Busy)
	}
	if body.ActiveRequests == nil || *body.ActiveRequests != 0 {
		t.Fatalf("expected an explicit zero, got %v", body.ActiveRequests)
	}
}
