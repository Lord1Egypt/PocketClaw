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

	"github.com/sipeed/picoclaw/pkg/logger"
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
		logger.WarnCF("gateway",
			"Configuration saved but the gateway was not safe to restart",
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
	unknownDeadline := time.Now().Add(configRestartUnknownGrace)
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
			unknownDeadline = time.Now().Add(configRestartUnknownGrace)
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
						"grace": configRestartUnknownGrace.String(),
					})
				return gatewayIdleUnverified, waited
			}
		}

		waited = true
		time.Sleep(configRestartIdlePoll)
	}
}

// gatewayProcessRunning reports whether a gateway process is currently tracked
// and alive.
func (h *Handler) gatewayProcessRunning() bool {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	return gatewayStatusWithoutHealthLocked() == "running"
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
