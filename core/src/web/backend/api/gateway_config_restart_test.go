package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/health"
)

// markGatewayRunningForTest makes the idle check treat the gateway as a live
// process, which is the only state in which busyness is even asked about.
func markGatewayRunningForTest(t *testing.T) {
	t.Helper()
	gateway.mu.Lock()
	prevStatus := gateway.runtimeStatus
	prevCmd := gateway.cmd
	gateway.runtimeStatus = "running"
	// A nil cmd reads as "not alive", which would degrade the status to
	// stopped, so point it at this test process.
	gateway.cmd = exec.Command("sleep", "0")
	gateway.cmd.Process = &os.Process{Pid: os.Getpid()}
	gateway.mu.Unlock()

	t.Cleanup(func() {
		gateway.mu.Lock()
		gateway.runtimeStatus = prevStatus
		gateway.cmd = prevCmd
		gateway.mu.Unlock()
	})
}

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

	markGatewayRunningForTest(t)
	h := NewHandler(t.TempDir() + "/config.json")
	outcome, waited := h.waitForGatewayIdle()

	if !waited {
		t.Fatal("a busy gateway must be waited for")
	}
	if outcome != gatewayIdleReady {
		t.Fatalf("a gateway that became idle must be restartable, got %q", outcome)
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

	markGatewayRunningForTest(t)
	h := NewHandler(t.TempDir() + "/config.json")
	outcome, waited := h.waitForGatewayIdle()
	if waited {
		t.Fatal("an idle gateway must not be waited for")
	}
	if !outcome.canRestart() {
		t.Fatalf("an idle gateway is safe to restart, got %q", outcome)
	}
}

// Unknown is never idle.
//
// A gateway that is running but will not say whether it is busy may be part way
// through an answer, a Telegram reply or a git push. Restarting on that
// uncertainty would interrupt work purely because health metadata was missing,
// so the configuration stays saved and unapplied instead.
func TestWaitForGatewayIdle_DoesNotRestartWhenBusynessIsUnreported(t *testing.T) {
	original := gatewayHealthGet
	t.Cleanup(func() { gatewayHealthGet = original })

	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return healthResponseWith(t, &health.StatusResponse{Status: "ok"}), nil
	}

	markGatewayRunningForTest(t)
	h := NewHandler(t.TempDir() + "/config.json")
	outcome, _ := h.waitForGatewayIdle()

	if outcome != gatewayIdleUnverified {
		t.Fatalf("an unreadable busy signal must be unverified, got %q", outcome)
	}
	if outcome.canRestart() {
		t.Fatal("an unverified gateway must never be force-restarted")
	}
}

// A running gateway whose health endpoint is unreachable is also unknown, not
// idle. Being unable to ask is not an answer.
func TestWaitForGatewayIdle_DoesNotRestartWhenHealthIsUnreachable(t *testing.T) {
	original := gatewayHealthGet
	t.Cleanup(func() { gatewayHealthGet = original })

	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return nil, http.ErrServerClosed
	}

	markGatewayRunningForTest(t)
	h := NewHandler(t.TempDir() + "/config.json")
	outcome, _ := h.waitForGatewayIdle()

	if outcome.canRestart() {
		t.Fatalf("an unreachable health endpoint must not authorise a restart, got %q", outcome)
	}
}

// A gateway that is not running has nothing to interrupt, so the restart
// proceeds at once. This is the one case where "cannot read busyness" is safe.
func TestWaitForGatewayIdle_RestartsImmediatelyWhenNotRunning(t *testing.T) {
	original := gatewayHealthGet
	t.Cleanup(func() { gatewayHealthGet = original })
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		t.Fatal("health must not be probed when no gateway process is running")
		return nil, nil
	}

	gateway.mu.Lock()
	prevStatus := gateway.runtimeStatus
	prevCmd := gateway.cmd
	gateway.runtimeStatus = "stopped"
	gateway.cmd = nil
	gateway.mu.Unlock()
	t.Cleanup(func() {
		gateway.mu.Lock()
		gateway.runtimeStatus = prevStatus
		gateway.cmd = prevCmd
		gateway.mu.Unlock()
	})

	h := NewHandler(t.TempDir() + "/config.json")
	outcome, waited := h.waitForGatewayIdle()

	if outcome != gatewayIdleNotRunning {
		t.Fatalf("expected not_running, got %q", outcome)
	}
	if !outcome.canRestart() {
		t.Fatal("there is nothing to interrupt, so the restart may proceed")
	}
	if waited {
		t.Fatal("there is no reason to wait for a stopped gateway")
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

// The two-minute wait is a limit on waiting, not a licence to interrupt.
//
// A gateway that is still busy when it expires keeps its work: the outcome is
// busy_timeout, which does not permit a restart. Nothing an LLM turn, a Telegram
// reply, a git push or any runtime tool is doing may be killed because a timer
// ran out.
func TestBusyBeyondTheIdleTimeoutNeverForcesARestart(t *testing.T) {
	if gatewayIdleBusyTimeout.canRestart() {
		t.Fatal(
			"a busy timeout must never authorise a restart: no LLM turn, Telegram " +
				"reply, git push or runtime tool may be killed because a timer expired",
		)
	}
}

// Applying a configuration never writes configuration. The save happened before
// this endpoint was called, so a refusal to restart cannot lose it.
func TestApplyConfigDoesNotModifyTheSavedConfiguration(t *testing.T) {
	original := gatewayHealthGet
	t.Cleanup(func() { gatewayHealthGet = original })
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return healthResponseWith(t, &health.StatusResponse{
			Status: "ok", Busy: boolPtr(true), ActiveRequests: intPtr(1),
		}), nil
	}
	markGatewayRunningForTest(t)

	prevTimeout := configRestartIdleTimeoutForTest
	configRestartIdleTimeoutForTest = 50 * time.Millisecond
	t.Cleanup(func() { configRestartIdleTimeoutForTest = prevTimeout })

	configPath, mux := fallbackTestEnv(t)
	if rec := postFallbacks(t, mux, `{"fallbacks": ["Gemini"]}`); rec.Code != http.StatusOK {
		t.Fatalf("save failed: %s", rec.Body.String())
	}
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("cannot read config: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/gateway/apply-config",
		bytes.NewBufferString(`{"reason":"fallbacks_changed"}`))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 saved_not_applied, got %d: %s", rec.Code, rec.Body.String())
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("cannot re-read config: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("a refused restart must leave the saved configuration untouched")
	}
}

func TestOnlyIdleOrStoppedGatewaysMayBeRestarted(t *testing.T) {
	permitted := map[gatewayIdleOutcome]bool{
		gatewayIdleNotRunning:  true,
		gatewayIdleReady:       true,
		gatewayIdleBusyTimeout: false,
		gatewayIdleUnverified:  false,
	}
	for outcome, want := range permitted {
		if got := outcome.canRestart(); got != want {
			t.Fatalf("outcome %q: canRestart = %v, want %v", outcome, got, want)
		}
	}
}

// Reaching the busy timeout produces a typed result the endpoint reports as
// "saved but not applied", never as a failed save.
func TestBusyTimeoutIsReportedAsSavedNotApplied(t *testing.T) {
	err := &ErrGatewayBusy{Outcome: gatewayIdleBusyTimeout}
	if !strings.Contains(err.Error(), "configuration saved") {
		t.Fatalf("the message must make clear the save succeeded: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "not restarted") {
		t.Fatalf("the message must say the gateway was left alone: %q", err.Error())
	}

	unverified := &ErrGatewayBusy{Outcome: gatewayIdleUnverified}
	if !strings.Contains(unverified.Error(), "did not report whether it is idle") {
		t.Fatalf("the unverified message must name the cause: %q", unverified.Error())
	}
}

// The endpoint answers 202 with saved_not_applied rather than an error status:
// the user's configuration is safely persisted, it simply is not live yet.
func TestApplyConfigReportsSavedNotAppliedWhenBusy(t *testing.T) {
	original := gatewayHealthGet
	t.Cleanup(func() { gatewayHealthGet = original })
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return healthResponseWith(t, &health.StatusResponse{
			Status: "ok", Busy: boolPtr(true), ActiveRequests: intPtr(1),
		}), nil
	}
	markGatewayRunningForTest(t)

	// A short wait keeps the test fast; the policy under test is what happens
	// when the wait ends, not how long it is.
	prevTimeout := configRestartIdleTimeoutForTest
	configRestartIdleTimeoutForTest = 50 * time.Millisecond
	t.Cleanup(func() { configRestartIdleTimeoutForTest = prevTimeout })

	h := NewHandler(t.TempDir() + "/config.json")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/gateway/apply-config",
		bytes.NewBufferString(`{"reason":"fallbacks_changed"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "saved_not_applied" {
		t.Fatalf("status = %v, want saved_not_applied", body["status"])
	}
	if body["outcome"] != string(gatewayIdleBusyTimeout) {
		t.Fatalf("outcome = %v, want busy_timeout", body["outcome"])
	}
}
