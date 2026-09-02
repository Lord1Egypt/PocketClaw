package api

import (
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	ppid "github.com/sipeed/picoclaw/pkg/pid"
)

// timeoutError is a net.Error that reports a timeout, matching what an HTTP
// client surfaces when a probe hangs rather than being refused.
type timeoutError struct{}

func (timeoutError) Error() string   { return "context deadline exceeded" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

// refusedError reproduces the error chain net/http builds for a closed port.
func refusedError() error {
	return &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &net.OpError{Op: "connect", Err: syscall.ECONNREFUSED},
	}
}

func TestGatewayHealthProbeRefused(t *testing.T) {
	var netErr net.Error = timeoutError{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error is not a refusal", err: nil, want: false},
		{name: "connection refused is decisive", err: refusedError(), want: true},
		{name: "timeout is ambiguous, never decisive", err: netErr, want: false},
		{name: "unrelated error is ambiguous", err: errors.New("boom"), want: false},
		{name: "host unreachable is ambiguous", err: &net.OpError{Op: "dial", Err: syscall.EHOSTUNREACH}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gatewayHealthProbeRefused(tt.err); got != tt.want {
				t.Errorf("gatewayHealthProbeRefused() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestValidateGatewayPidDataRefusedIsDecisive is the launcher half of the
// device incident: the process could not be classified and nothing was
// listening on the gateway port, so the pid file must be declared stale.
func TestValidateGatewayPidDataRefusedIsDecisive(t *testing.T) {
	resetGatewayTestState(t)

	h := NewHandler(t.TempDir() + "/config.json")
	pidData := &ppid.PidFileData{PID: 14605, Host: "127.0.0.1", Port: 18790}

	gatewayProcessMatcher = func(int) (bool, bool) { return false, false }
	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return nil, refusedError()
	}

	ok, decisive, reason := h.validateGatewayPidData(pidData, nil, "test")
	if ok {
		t.Fatal("validateGatewayPidData() ok = true, want false")
	}
	if !decisive {
		t.Fatal("a refused health probe must be decisive so the stale pid file is removed")
	}
	if !strings.Contains(reason, "refused") {
		t.Errorf("reason = %q, want it to mention the refusal", reason)
	}
}

// TestValidateGatewayPidDataTimeoutStaysAmbiguous protects the opposite case: a
// slow or wedged gateway must not have its pid file deleted underneath it.
func TestValidateGatewayPidDataTimeoutStaysAmbiguous(t *testing.T) {
	resetGatewayTestState(t)

	h := NewHandler(t.TempDir() + "/config.json")
	pidData := &ppid.PidFileData{PID: 14605, Host: "127.0.0.1", Port: 18790}

	gatewayProcessMatcher = func(int) (bool, bool) { return false, false }
	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return nil, timeoutError{}
	}

	ok, decisive, _ := h.validateGatewayPidData(pidData, nil, "test")
	if ok {
		t.Fatal("validateGatewayPidData() ok = true, want false")
	}
	if decisive {
		t.Fatal("a timed-out health probe must stay non-decisive; deleting the pid file would be the bug")
	}
}

// TestValidateGatewayPidDataHealthyWins confirms a reachable gateway still
// keeps its pid file regardless of how the probe is classified.
func TestValidateGatewayPidDataHealthyWins(t *testing.T) {
	resetGatewayTestState(t)

	h := NewHandler(t.TempDir() + "/config.json")
	const testPID = 14605
	pidData := &ppid.PidFileData{PID: testPID, Host: "127.0.0.1", Port: 18790}

	gatewayProcessMatcher = func(int) (bool, bool) { return false, false }
	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return mockGatewayHealthResponse(http.StatusOK, testPID), nil
	}

	ok, decisive, reason := h.validateGatewayPidData(pidData, nil, "test")
	if !ok || !decisive {
		t.Fatalf("a healthy gateway must be honoured: ok=%v decisive=%v reason=%q", ok, decisive, reason)
	}
}

func TestGatewayStartupFailureReason(t *testing.T) {
	exitErr := errors.New("exit status 1")

	tests := []struct {
		name  string
		lines []string
		err   error
		want  string
	}{
		{
			name:  "singleton rejection is named plainly",
			lines: []string{"Error: singleton check failed: gateway is already running (PID: 14605, version: v0.3.1)"},
			err:   exitErr,
			want:  "stale previous process record",
		},
		{
			name:  "port conflict",
			lines: []string{"Error: error opening gateway listeners: failed to open adaptive localhost listener on port 18790"},
			err:   exitErr,
			want:  "network port is already in use",
		},
		{
			name:  "config rejection",
			lines: []string{`{"message":"config pre-check failed: bad channel"}`},
			err:   exitErr,
			want:  "configuration was rejected",
		},
		{
			name:  "unrecognised failure falls back to the exit status only",
			lines: []string{"PICOCLAW_CHANNELS_PICO_TOKEN=supersecret", "panic: runtime error"},
			err:   exitErr,
			want:  "Gateway exited during startup (exit status 1)",
		},
		{
			name:  "no output at all",
			lines: nil,
			err:   exitErr,
			want:  "Gateway exited during startup (exit status 1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gatewayStartupFailureReason(tt.lines, tt.err)
			if !strings.Contains(got, tt.want) {
				t.Errorf("gatewayStartupFailureReason() = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

// TestGatewayStartupFailureReasonNeverLeaksChildOutput pins the whitelist: no
// captured line is ever echoed back to the console verbatim.
func TestGatewayStartupFailureReasonNeverLeaksChildOutput(t *testing.T) {
	secrets := []string{
		"PICOCLAW_CHANNELS_PICO_TOKEN=tok_live_abcdef",
		"POCKETCLAW_GITHUB_TOKEN=ghp_zzzzzzzz",
		`{"config_path":"/data/user/0/com.lord1egypt.pocketclaw/files/picoclaw/config.json"}`,
		"Authorization: Bearer sk-secret",
	}

	got := gatewayStartupFailureReason(secrets, errors.New("exit status 1"))
	for _, line := range secrets {
		if strings.Contains(got, line) {
			t.Fatalf("reason %q leaked child output %q", got, line)
		}
	}
	for _, needle := range []string{"tok_live", "ghp_", "sk-secret", "config.json"} {
		if strings.Contains(got, needle) {
			t.Fatalf("reason %q leaked %q", got, needle)
		}
	}
}

// waitForGatewayStatus polls the runtime status until it matches or the
// deadline passes.
func waitForGatewayStatus(t *testing.T, want string) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		gateway.mu.Lock()
		got := gateway.runtimeStatus
		gateway.mu.Unlock()
		if got == want || time.Now().After(deadline) {
			return got
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestStartGatewayLocked_EarlyExitLeavesStarting is the UI half of the
// incident: the child died immediately with status 1 and the console sat on
// "starting" with nothing to show the user.
func TestStartGatewayLocked_EarlyExitLeavesStarting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based child is not portable to Windows")
	}
	h := newGatewayStartTestHandler(t)

	gatewayExecCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c",
			`echo 'Error: singleton check failed: gateway is already running (PID: 14605, version: v0.3.1)' >&2; exit 1`)
	}

	if _, err := h.startGatewayLocked("starting", 0); err != nil {
		t.Fatalf("startGatewayLocked() error = %v", err)
	}

	if got := waitForGatewayStatus(t, "error"); got != "error" {
		t.Fatalf("runtime status = %q, want %q (the gateway must not stay in starting)", got, "error")
	}

	gateway.mu.Lock()
	reason := gateway.lastStartupError
	gateway.mu.Unlock()

	if !strings.Contains(reason, "stale previous process record") {
		t.Errorf("lastStartupError = %q, want the stale-pid explanation", reason)
	}

	data := h.gatewayStatusData()
	if data["gateway_status"] != "error" {
		t.Errorf("gateway_status = %v, want error", data["gateway_status"])
	}
	if surfaced, _ := data["gateway_last_error"].(string); surfaced != reason {
		t.Errorf("gateway_last_error = %q, want %q", surfaced, reason)
	}
}

// TestStartGatewayLocked_RetryAfterFailureClearsError proves the user can retry
// once the blocking condition is gone, and that the stale message does not
// linger into the next attempt.
func TestStartGatewayLocked_RetryAfterFailureClearsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based child is not portable to Windows")
	}
	h := newGatewayStartTestHandler(t)

	gatewayExecCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", `echo 'Error: singleton check failed' >&2; exit 1`)
	}
	if _, err := h.startGatewayLocked("starting", 0); err != nil {
		t.Fatalf("first startGatewayLocked() error = %v", err)
	}
	if got := waitForGatewayStatus(t, "error"); got != "error" {
		t.Fatalf("runtime status = %q, want error", got)
	}

	// The stale record is gone; the retry now behaves like a healthy start.
	gatewayExecCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sleep", "30")
	}
	if _, err := h.startGatewayLocked("starting", 0); err != nil {
		t.Fatalf("retry startGatewayLocked() error = %v", err)
	}

	gateway.mu.Lock()
	status := gateway.runtimeStatus
	reason := gateway.lastStartupError
	cmd := gateway.cmd
	gateway.mu.Unlock()

	t.Cleanup(func() {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})

	if status != "starting" {
		t.Errorf("retry runtime status = %q, want starting", status)
	}
	if reason != "" {
		t.Errorf("lastStartupError = %q, want it cleared on retry", reason)
	}
}

// TestStartGatewayLocked_CleanExitIsStoppedNotError keeps the new error state
// narrow: a gateway that exits zero is stopped, not failed.
func TestStartGatewayLocked_CleanExitIsStoppedNotError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based child is not portable to Windows")
	}
	h := newGatewayStartTestHandler(t)

	gatewayExecCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}

	if _, err := h.startGatewayLocked("starting", 0); err != nil {
		t.Fatalf("startGatewayLocked() error = %v", err)
	}

	if got := waitForGatewayStatus(t, "stopped"); got != "stopped" {
		t.Fatalf("runtime status = %q, want stopped", got)
	}

	gateway.mu.Lock()
	reason := gateway.lastStartupError
	gateway.mu.Unlock()
	if reason != "" {
		t.Errorf("lastStartupError = %q, want empty for a clean exit", reason)
	}
}

// TestSanitizeGatewayPidDataRemovesStaleFile closes the loop end to end: an
// unclassifiable PID whose port refuses connections must have its pid file
// deleted, which is the step that unblocks the next start. On the device this
// never happened, so every retry spawned a child that killed itself.
func TestSanitizeGatewayPidDataRemovesStaleFile(t *testing.T) {
	resetGatewayTestState(t)

	h := NewHandler(t.TempDir() + "/config.json")
	const stalePID = 14605
	path := writeTestPidFile(t, ppid.PidFileData{
		PID: stalePID, Host: "127.0.0.1", Port: 18790, Version: "v0.3.1",
	})

	gatewayProcessMatcher = func(int) (bool, bool) { return false, false }
	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return nil, refusedError()
	}

	got := h.sanitizeGatewayPidData(&ppid.PidFileData{
		PID: stalePID, Host: "127.0.0.1", Port: 18790,
	}, nil, "manual_start")

	if got != nil {
		t.Fatalf("sanitizeGatewayPidData() = %+v, want nil for a stale pid file", got)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("stale pid file was not removed (stat err = %v)", err)
	}
}

// TestSanitizeGatewayPidDataKeepsFileOnTimeout is the guard rail: an ambiguous
// probe must leave the pid file alone.
func TestSanitizeGatewayPidDataKeepsFileOnTimeout(t *testing.T) {
	resetGatewayTestState(t)

	h := NewHandler(t.TempDir() + "/config.json")
	const pid = 14605
	path := writeTestPidFile(t, ppid.PidFileData{
		PID: pid, Host: "127.0.0.1", Port: 18790, Version: "v0.3.1",
	})

	gatewayProcessMatcher = func(int) (bool, bool) { return false, false }
	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return nil, timeoutError{}
	}

	if got := h.sanitizeGatewayPidData(&ppid.PidFileData{
		PID: pid, Host: "127.0.0.1", Port: 18790,
	}, nil, "status"); got != nil {
		t.Fatalf("sanitizeGatewayPidData() = %+v, want nil", got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("pid file must survive an ambiguous probe: %v", err)
	}
}
