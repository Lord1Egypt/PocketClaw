package api

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
)

// PC-DEF-030, the shape the previous fix left behind.
//
// The launcher's in-memory gateway state is populated when this process started
// the gateway or when a client polled GET /api/gateway/status. Managed Telegram
// onboarding goes through the Android bridge and does neither, so the restart
// path used to look at an empty struct, conclude "no gateway is running", skip
// the busy check and start a second process without stopping the first. The PID
// file is what every other path believes; this one has to believe it too.

// writeLiveGatewayPidFile plants a record for a process that is genuinely
// alive: this test binary. A record naming a dead PID is removed as stale, so
// only a live one exercises the reconciliation.
func writeLiveGatewayPidFile(t *testing.T, port int) {
	t.Helper()
	if _, err := ppid.WritePidFile(config.GetHome(), "127.0.0.1", port); err != nil {
		t.Fatalf("cannot write pid file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(config.GetHome(), ".pocketclaw.pid"))
	})
}

func TestGatewayProcessRunningAdoptsAnUntrackedGateway(t *testing.T) {
	resetGatewayTestState(t)
	writeLiveGatewayPidFile(t, 18790)

	// The process is this test binary, which no matcher can classify as a
	// gateway; the port probe is what keeps the record from being declared
	// stale, exactly as on the device.
	gatewayProcessMatcher = func(int) (bool, bool) { return false, false }
	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return healthResponseWith(t, &health.StatusResponse{
			Status: "ok", Busy: boolPtr(false), ActiveRequests: intPtr(0),
		}), nil
	}

	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))

	if !h.gatewayProcessRunning() {
		t.Fatal("a gateway named by a live pid file read as not running; " +
			"the restart path would skip the busy check and start a second one")
	}

	gateway.mu.Lock()
	tracked := gateway.cmd != nil && gateway.cmd.Process != nil &&
		gateway.cmd.Process.Pid == os.Getpid()
	gateway.mu.Unlock()
	if !tracked {
		t.Fatal("the running gateway was not adopted, so a restart would have " +
			"nothing to stop")
	}
}

// The reconciliation must not invent a gateway. With no record on disk there is
// nothing running, and the restart is free to proceed immediately.
func TestGatewayProcessRunningStaysFalseWithoutAPidFile(t *testing.T) {
	resetGatewayTestState(t)

	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	if h.gatewayProcessRunning() {
		t.Fatal("reported a running gateway with no pid file")
	}
}

// Having adopted the process, the idle wait must go on treating an unreadable
// busy signal as unknown rather than idle. Adoption changes what is known to be
// running, never what is allowed to be interrupted.
func TestWaitForGatewayIdleDoesNotInterruptAnAdoptedGateway(t *testing.T) {
	resetGatewayTestState(t)
	writeLiveGatewayPidFile(t, 18790)

	prevGrace := configRestartUnknownGraceForTest
	t.Cleanup(func() { configRestartUnknownGraceForTest = prevGrace })
	configRestartUnknownGraceForTest = 50 * time.Millisecond

	gatewayProcessMatcher = func(int) (bool, bool) { return false, false }
	var probes atomic.Int32
	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		probes.Add(1)
		// Reachable, but says nothing about in-flight turns.
		return healthResponseWith(t, &health.StatusResponse{Status: "ok"}), nil
	}

	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	outcome, _ := h.waitForGatewayIdle()

	if outcome != gatewayIdleUnverified {
		t.Fatalf("outcome = %q, want %q: an unreadable busy signal is not idle",
			outcome, gatewayIdleUnverified)
	}
	if outcome.canRestart() {
		t.Fatal("an unverified gateway must never be restarted out from under a turn")
	}
	if probes.Load() == 0 {
		t.Fatal("the busy signal was never asked for")
	}
}

// The other half of PC-DEF-030: a change parked as "unverified" has no idle
// edge coming, because the gateway only reports the N>0 -> 0 transition and an
// idle gateway never makes it. Without a supervisor the change waits forever
// and the manual restart is the only way out -- the defect, restated.
func TestPendingApplySupervisorAppliesOnceTheGatewayIsSafeToRestart(t *testing.T) {
	resetGatewayTestState(t)
	resetPendingConfigApplyForTest(t)

	prevInterval := pendingApplySupervisorIntervalForTest
	t.Cleanup(func() { pendingApplySupervisorIntervalForTest = prevInterval })
	pendingApplySupervisorIntervalForTest = 10 * time.Millisecond

	// No pid file and nothing tracked: there is no work to interrupt, so the
	// supervisor is allowed to apply the moment it looks.
	gatewayExecCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command(os.Args[0], "-test.run=TestPendingApplyHelperProcess", "--", "sleep")
	}

	markConfigApplyPending("telegram_configured")
	if !startPendingApplySupervisor() {
		t.Fatal("a supervisor was already running")
	}

	h := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	go h.supervisePendingConfigApply()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if pending, _ := pendingConfigApplyState(); !pending {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the parked change was never applied; on the device this is " +
				"the state that needs a manual Service and Gateway restart")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Only one supervisor, however many saves park a change.
func TestPendingApplySupervisorIsSingular(t *testing.T) {
	resetPendingConfigApplyForTest(t)

	if !startPendingApplySupervisor() {
		t.Fatal("the first caller must get the supervisor")
	}
	if startPendingApplySupervisor() {
		t.Fatal("a second supervisor was started for the same pending change")
	}
	stopPendingApplySupervisor()
	if !startPendingApplySupervisor() {
		t.Fatal("the flag was not released")
	}
	stopPendingApplySupervisor()
}

// resetPendingConfigApplyForTest clears the package-level pending state, which
// several tests write, so one test cannot leave a change parked for the next.
func resetPendingConfigApplyForTest(t *testing.T) {
	t.Helper()
	clear := func() {
		pendingConfigApply.mu.Lock()
		pendingConfigApply.reason = ""
		pendingConfigApply.err = ""
		pendingConfigApply.supervised = false
		pendingConfigApply.applying = false
		pendingConfigApply.mu.Unlock()
	}
	clear()
	t.Cleanup(clear)
}

// TestPendingApplyHelperProcess stands in for a spawned gateway: it exists only
// to be a live child process, and is skipped in a normal run.
func TestPendingApplyHelperProcess(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-1] != "sleep" {
		t.Skip("helper process")
	}
	time.Sleep(2 * time.Second)
}
