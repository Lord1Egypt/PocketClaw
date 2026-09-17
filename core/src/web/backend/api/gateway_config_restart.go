package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
)

// Applying a configuration change means restarting the gateway, and a restart
// kills whatever the gateway is doing. These bounds decide how long we are
// willing to wait for it to become idle first.
const (
	// configRestartIdleTimeout caps how long a restart waits for in-flight
	// turns to finish.
	//
	// Reaching it does NOT authorise interrupting the gateway. It only ends the
	// wait: the configuration stays saved and the automatic restart is reported
	// as unsatisfied, leaving the manual Restart Gateway action as the way to
	// apply it. Killing a running answer, a Telegram reply or a git push
	// because a timer expired would be a worse outcome than a delayed setting.
	configRestartIdleTimeout = 2 * time.Minute
	// configRestartIdlePoll is how often the gateway is asked whether it is
	// still busy.
	configRestartIdlePoll = 500 * time.Millisecond
	// configRestartHealthTimeout bounds a single /health probe.
	configRestartHealthTimeout = 2 * time.Second
	// configRestartUnknownGrace is how long a *running* gateway is given to
	// produce a trustworthy busy signal before the restart is abandoned.
	//
	// A short retry covers a health endpoint that is briefly unreachable. If the
	// signal still cannot be read, the gateway may well be mid-answer and we
	// have no way to tell, so the restart does not happen.
	configRestartUnknownGrace = 5 * time.Second
)

// configRestartIdleTimeoutForTest is the effective wait, indirected so tests can
// exercise the timeout path without waiting two minutes for it.
var configRestartIdleTimeoutForTest = configRestartIdleTimeout

// configRestartUnknownGraceForTest is the effective grace, indirected for the
// same reason.
var configRestartUnknownGraceForTest = configRestartUnknownGrace

// gatewayIdleOutcome is what the idle check concluded.
type gatewayIdleOutcome string

const (
	// gatewayIdleNotRunning: there is no gateway process, so there is nothing
	// to interrupt and the restart can proceed immediately.
	gatewayIdleNotRunning gatewayIdleOutcome = "not_running"
	// gatewayIdleReady: the gateway reported no in-flight turns.
	gatewayIdleReady gatewayIdleOutcome = "idle"
	// gatewayIdleBusyTimeout: the gateway was still busy when the wait ended.
	gatewayIdleBusyTimeout gatewayIdleOutcome = "busy_timeout"
	// gatewayIdleUnverified: the gateway is running but would not tell us
	// whether it is busy. Unknown is never treated as idle.
	gatewayIdleUnverified gatewayIdleOutcome = "unverified"
)

// canRestart reports whether this outcome permits restarting the gateway.
//
// Only two do. Neither a timeout nor an unreadable busy signal is permission to
// interrupt work that may be in progress.
func (o gatewayIdleOutcome) canRestart() bool {
	return o == gatewayIdleNotRunning || o == gatewayIdleReady
}

// ErrGatewayBusy reports that a configuration change was saved but could not be
// applied because the gateway was not safe to restart.
type ErrGatewayBusy struct {
	Outcome gatewayIdleOutcome
}

func (e *ErrGatewayBusy) Error() string {
	switch e.Outcome {
	case gatewayIdleBusyTimeout:
		return "configuration saved, but the gateway is still handling a request; " +
			"it was not restarted"
	default:
		return "configuration saved, but the gateway did not report whether it is idle; " +
			"it was not restarted"
	}
}

// configRestartState serialises automatic configuration restarts.
//
// Several restart-requiring saves in quick succession must produce one
// effective restart, not one per save, so a request that arrives while another
// is already waiting or running joins it instead of queueing behind it.
type configRestartState struct {
	mu      sync.Mutex
	running bool
	waiters []chan configRestartResult
}

type configRestartResult struct {
	pid      int
	deferred bool
	err      error
}

var configRestart configRestartState

// pendingConfigApply is a configuration change that is persisted but not yet
// live, because the gateway was still busy when the bounded wait ran out.
//
// PC-DEF-030. It is applied by the gateway's own idle notification, not by a
// timer and not by anything a client does: GET /api/gateway/status reports
// this state and must never act on it.
var pendingConfigApply = struct {
	mu         sync.Mutex
	reason     string
	err        string
	supervised bool
	applying   bool
}{}

// The gateway's idle notification is the fast path, and for a gateway that was
// genuinely busy it is the only one needed: it finishes the answer, crosses
// N>0 -> 0 and the parked change goes live.
//
// PC-DEF-030. It cannot cover the other way a change gets parked. An
// "unverified" outcome means the gateway is running but would not say whether
// it is busy, and a gateway that is in fact idle never crosses that edge -- so
// the change waited for a notification that could not arrive, and the manual
// restart was the only way out. That is the defect, in the shape the previous
// fix left it in.
//
// So a parked change gets a supervisor: it re-asks the same idle question the
// wait asks, on a slow cadence, and applies the change the first time the
// answer is trustworthy. It never interrupts anything -- an unknown answer is
// still not idle -- it stops as soon as nothing is pending, and it is bounded,
// because a gateway that has not become readable in this long is a problem a
// retry will not solve.
const (
	pendingApplySupervisorInterval = 10 * time.Second
	pendingApplySupervisorAttempts = 30
)

// pendingApplySupervisorIntervalForTest is the effective cadence, indirected so
// tests do not wait minutes to exercise the supervisor.
var pendingApplySupervisorIntervalForTest = pendingApplySupervisorInterval

// markConfigApplyPending records that a saved change is waiting for idle.
//
// Later saves collapse into the one pending entry rather than queueing. The
// configuration file is written before any of this runs, so a single apply
// always boots the latest persisted configuration -- A, B and C become one
// restart that loads C.
func markConfigApplyPending(reason string) {
	pendingConfigApply.mu.Lock()
	pendingConfigApply.reason = reason
	pendingConfigApply.err = ""
	pendingConfigApply.mu.Unlock()
}

// startPendingApplySupervisor ensures exactly one supervisor is watching.
//
// Returns whether it started one, so a caller that has already parked several
// changes does not spawn a goroutine per change.
func startPendingApplySupervisor() bool {
	pendingConfigApply.mu.Lock()
	defer pendingConfigApply.mu.Unlock()
	if pendingConfigApply.supervised {
		return false
	}
	pendingConfigApply.supervised = true
	return true
}

func stopPendingApplySupervisor() {
	pendingConfigApply.mu.Lock()
	pendingConfigApply.supervised = false
	pendingConfigApply.mu.Unlock()
}

// supervisePendingConfigApply applies a parked change once the gateway can be
// verified idle, or gives up and says so.
func (h *Handler) supervisePendingConfigApply() {
	defer stopPendingApplySupervisor()

	for attempt := 0; attempt < pendingApplySupervisorAttempts; attempt++ {
		time.Sleep(pendingApplySupervisorIntervalForTest)

		if pending, _ := pendingConfigApplyState(); !pending {
			// The idle notification got there first, which is the intended path.
			return
		}
		if !h.gatewayIdleNow() {
			continue
		}

		reason := claimPendingConfigApply()
		if reason == "" {
			return
		}
		// Not a return: a restart that parks the change again (the gateway
		// became busy between the check and the attempt) would otherwise be
		// left with no supervisor, since this one still holds the flag. The
		// next pass sees nothing pending and exits.
		h.applyPendingConfigRestart(reason)
	}

	if pending, _ := pendingConfigApplyState(); pending {
		setPendingConfigApplyError("the gateway never reported an idle state")
		logger.WarnCF("gateway",
			"Gave up applying a saved configuration change automatically; "+
				"the gateway did not become verifiably idle",
			map[string]any{"attempts": pendingApplySupervisorAttempts})
	}
}

// takePendingConfigApply claims the pending change, if any, exactly once.
func takePendingConfigApply() string {
	pendingConfigApply.mu.Lock()
	defer pendingConfigApply.mu.Unlock()
	reason := pendingConfigApply.reason
	pendingConfigApply.reason = ""
	return reason
}

// claimPendingConfigApply transfers a parked change to the apply path without
// creating a readiness gap. The persisted configuration is not live merely
// because its pending entry has been removed; it remains applying until the
// gateway restart has completed.
func claimPendingConfigApply() string {
	pendingConfigApply.mu.Lock()
	defer pendingConfigApply.mu.Unlock()
	reason := pendingConfigApply.reason
	pendingConfigApply.reason = ""
	if reason != "" {
		pendingConfigApply.applying = true
	}
	return reason
}

func finishPendingConfigApply() {
	pendingConfigApply.mu.Lock()
	pendingConfigApply.applying = false
	pendingConfigApply.mu.Unlock()
}

// setPendingConfigApplyError records why the last pending apply failed.
func setPendingConfigApplyError(message string) {
	pendingConfigApply.mu.Lock()
	pendingConfigApply.err = message
	pendingConfigApply.mu.Unlock()
}

// pendingConfigApplyState reports what status should show. Read-only.
func pendingConfigApplyState() (pending bool, applyErr string) {
	pendingConfigApply.mu.Lock()
	defer pendingConfigApply.mu.Unlock()
	return pendingConfigApply.reason != "" || pendingConfigApply.applying, pendingConfigApply.err
}

// configApplyInProgress is the configuration-generation boundary used by
// readiness checks. It covers both an immediate coalesced restart and a parked
// change that has been claimed by the idle path but is not live yet.
func configApplyInProgress() bool {
	if pending, _ := pendingConfigApplyState(); pending {
		return true
	}
	configRestart.mu.Lock()
	defer configRestart.mu.Unlock()
	return configRestart.running
}

// sanitizeConfigApplyError keeps a user-visible reason free of anything the
// configuration might carry. Only the shape of the failure is reported.
func sanitizeConfigApplyError(err error) string {
	if err == nil {
		return ""
	}
	var busy *ErrGatewayBusy
	if errors.As(err, &busy) {
		return "the gateway was still busy"
	}
	return "the gateway could not be restarted"
}

// RestartGatewayForConfigChange restarts the gateway once the gateway is idle.
//
// It is the automatic counterpart to the manual restart action, and it
// deliberately reuses RestartGateway rather than reimplementing the lifecycle:
// PID ownership validation, start preconditions and the runtime status
// transitions all stay in one place.
//
// The configuration has already been persisted by the time this runs, so a
// failure here never loses the user's change; it only means the change is not
// live yet and the manual Restart Gateway action is still needed.
func (h *Handler) RestartGatewayForConfigChange(reason string) (int, bool, error) {
	configRestart.mu.Lock()
	if configRestart.running {
		// Join the in-flight restart. Whatever it boots will already include
		// this save, because the config file was written before we got here.
		waiter := make(chan configRestartResult, 1)
		configRestart.waiters = append(configRestart.waiters, waiter)
		configRestart.mu.Unlock()

		logger.InfoCF("gateway", "Coalescing configuration restart into the one already running",
			map[string]any{"reason": reason})
		result := <-waiter
		return result.pid, result.deferred, result.err
	}
	configRestart.running = true
	configRestart.mu.Unlock()

	pid, deferred, err := h.runConfigRestart(reason)

	configRestart.mu.Lock()
	waiters := configRestart.waiters
	configRestart.waiters = nil
	configRestart.running = false
	configRestart.mu.Unlock()

	for _, waiter := range waiters {
		waiter <- configRestartResult{pid: pid, deferred: deferred, err: err}
		close(waiter)
	}
	return pid, deferred, err
}

func (h *Handler) runConfigRestart(reason string) (int, bool, error) {
	started := time.Now()
	logger.InfoCF("gateway", "Configuration restart requested", map[string]any{
		"reason": reason,
	})

	outcome, deferred := h.waitForGatewayIdle()
	if deferred {
		logger.InfoCF("gateway", "Configuration restart waited for the gateway",
			map[string]any{"reason": reason, "outcome": string(outcome)})
	}

	if !outcome.canRestart() {
		// The configuration is already persisted. Not restarting leaves it
		// pending, which the restart-required indicator continues to show, and
		// the manual Restart Gateway action still applies it.
		// PC-DEF-030. Persisted but not live is not a resting state: the
		// gateway's own idle notification applies this later, with no timer,
		// no polling and nothing for the user to do.
		markConfigApplyPending(reason)
		if startPendingApplySupervisor() {
			go h.supervisePendingConfigApply()
		}
		logger.WarnCF("gateway",
			"Configuration saved but the gateway was not safe to restart; "+
				"it will be applied when the gateway becomes idle",
			map[string]any{
				"reason":      reason,
				"outcome":     string(outcome),
				"duration_ms": time.Since(started).Milliseconds(),
			})
		return 0, deferred, &ErrGatewayBusy{Outcome: outcome}
	}

	pid, err := h.RestartGateway()
	if err != nil {
		logger.ErrorCF("gateway", "Configuration restart failed", map[string]any{
			"reason":      reason,
			"deferred":    deferred,
			"duration_ms": time.Since(started).Milliseconds(),
			"error":       err.Error(),
		})
		return 0, deferred, fmt.Errorf("gateway restart after configuration change failed: %w", err)
	}

	logger.InfoCF("gateway", "Configuration restart completed", map[string]any{
		"reason":      reason,
		"deferred":    deferred,
		"duration_ms": time.Since(started).Milliseconds(),
	})
	if reason == "telegram_configured" {
		logger.DebugCF("telegram", "Telegram configuration applied", map[string]any{
			"event":               "config_applied",
			"observed_at_unix_ms": time.Now().UnixMilli(),
		})
	}
	return pid, deferred, nil
}

// waitForGatewayIdle waits for the gateway to become safe to restart.
//
// The rule it enforces: a restart caused by saving settings may never interrupt
// work in progress. Unknown is not idle, and a timeout is not permission — both
// end the wait without restarting, leaving the saved configuration to be applied
// by the manual Restart Gateway action.
func (h *Handler) waitForGatewayIdle() (gatewayIdleOutcome, bool) {
	deadline := time.Now().Add(configRestartIdleTimeoutForTest)
	unknownDeadline := time.Now().Add(configRestartUnknownGraceForTest)
	waited := false

	for {
		if !h.gatewayProcessRunning() {
			// Nothing is running, so nothing can be interrupted.
			return gatewayIdleNotRunning, waited
		}

		busy, known := h.gatewayBusyState()
		switch {
		case known && !busy:
			return gatewayIdleReady, waited
		case known && busy:
			unknownDeadline = time.Now().Add(configRestartUnknownGraceForTest)
			if time.Now().After(deadline) {
				logger.WarnCF("gateway",
					"Gateway still busy after the idle wait; leaving the configuration unapplied",
					map[string]any{
						"timeout": configRestartIdleTimeoutForTest.String(),
					})
				return gatewayIdleBusyTimeout, waited
			}
		default:
			// Running, but the busy signal is unreadable. Retry briefly in case
			// health is momentarily unavailable, then give up rather than
			// gamble that it is idle.
			if time.Now().After(unknownDeadline) {
				logger.WarnCF("gateway",
					"Gateway is running but did not report an idle state; "+
						"leaving the configuration unapplied",
					map[string]any{
						"grace": configRestartUnknownGraceForTest.String(),
					})
				return gatewayIdleUnverified, waited
			}
		}

		waited = true
		time.Sleep(configRestartIdlePoll)
	}
}

// gatewayProcessRunning reports whether a gateway process is currently alive.
//
// PC-DEF-030. This read only the launcher's in-memory state, which is populated
// when this process started the gateway or when a client polled
// GET /api/gateway/status. A gateway this launcher generation had never
// attached to therefore read as "not running", and both halves of the physical
// defect followed from that one answer: the busy check was skipped, because
// there was believed to be nothing to interrupt, and the restart stopped
// nothing and started a second gateway that could not bind the port the first
// one still held. The saved Telegram configuration sat on disk while the
// original process went on serving the previous one, which is exactly the
// "restart the Service and the Gateway by hand" the fix was meant to remove.
//
// The PID file is what every other path treats as authoritative. This one
// reconciles against it before answering.
func (h *Handler) gatewayProcessRunning() bool {
	h.reconcileGatewayWithPidFile()

	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	return gatewayStatusWithoutHealthLocked() == "running"
}

// reconcileGatewayWithPidFile adopts a live gateway this process is not tracking.
//
// Adoption is what makes the running process stoppable: stopGatewayProcessForRestart
// signals gateway.cmd, and a nil cmd is silently "nothing to stop". Reading the
// file happens outside the lock because it touches the filesystem and, on a
// stale entry, removes it.
func (h *Handler) reconcileGatewayWithPidFile() {
	cfg, cfgErr := config.LoadConfig(h.configPath)
	if cfgErr != nil {
		cfg = nil
	}
	pidData := h.sanitizeGatewayPidData(
		ppid.ReadPidFileWithCheck(globalConfigDir()), cfg, "config_restart")
	if pidData == nil {
		return
	}

	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	gateway.pidData = pidData
	if gateway.cmd != nil && gateway.cmd.Process != nil &&
		gateway.cmd.Process.Pid == pidData.PID {
		return
	}
	_ = attachToGatewayProcessLocked(pidData.PID, cfg)
}

// gatewayIdleNow answers the idle question once, without waiting.
//
// It is the same rule waitForGatewayIdle enforces, with the same refusal to
// read an unknown answer as idle -- only asked as a single question, for the
// supervisor that retries a parked change.
func (h *Handler) gatewayIdleNow() bool {
	if !h.gatewayProcessRunning() {
		return true
	}
	busy, known := h.gatewayBusyState()
	return known && !busy
}

// gatewayBusyState reads the gateway's in-flight turn count.
//
// The second return value is whether the answer is trustworthy. A probe error,
// a missing response or an absent field all yield "unknown", never "idle": a
// gateway too old to report, or one whose health endpoint is briefly
// unreachable, may still be part way through an answer.
func (h *Handler) gatewayBusyState() (busy bool, known bool) {
	healthResponse, _, err := h.getGatewayHealth(nil, configRestartHealthTimeout)
	if err != nil || healthResponse == nil || healthResponse.Busy == nil {
		return false, false
	}
	return *healthResponse.Busy, true
}

// handleGatewayApplyConfig restarts the gateway so a saved configuration change
// takes effect, but only when doing so cannot interrupt work in progress.
//
//	POST /api/gateway/apply-config
//
// It is separate from POST /api/gateway/restart on purpose. The manual restart
// is an immediate recovery action the user asked for explicitly; this one is a
// consequence of saving settings, so a busy gateway — or one that will not say
// whether it is busy — results in the change staying saved but unapplied rather
// than an answer being cut off.
func (h *Handler) handleGatewayApplyConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err == nil && len(body) > 0 {
			_ = json.Unmarshal(body, &req)
		}
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "config_changed"
	}

	pid, deferred, err := h.RestartGatewayForConfigChange(reason)
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		// A gateway that was not safe to restart is not an error the user did
		// anything wrong. The configuration is saved; it is simply not live yet.
		var busyErr *ErrGatewayBusy
		if errors.As(err, &busyErr) {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":   "saved_not_applied",
				"outcome":  string(busyErr.Outcome),
				"reason":   reason,
				"message":  busyErr.Error(),
				"deferred": deferred,
			})
			return
		}
		var precondErr *preconditionFailedError
		if errors.As(err, &precondErr) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  "precondition_failed",
				"message": precondErr.reason,
			})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"pid":      pid,
		"deferred": deferred,
		"reason":   reason,
	})
}
