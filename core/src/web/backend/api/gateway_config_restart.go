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
	// turns to finish. A turn that runs longer than this is most likely stuck,
	// and holding the user's configuration change hostage to it indefinitely is
	// worse than interrupting it.
	configRestartIdleTimeout = 2 * time.Minute
	// configRestartIdlePoll is how often the gateway is asked whether it is
	// still busy.
	configRestartIdlePoll = 500 * time.Millisecond
	// configRestartHealthTimeout bounds a single /health probe.
	configRestartHealthTimeout = 2 * time.Second
)

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

	deferred := h.waitForGatewayIdle()
	if deferred {
		logger.InfoCF("gateway", "Configuration restart deferred until the gateway went idle",
			map[string]any{"reason": reason})
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

// waitForGatewayIdle blocks until the gateway reports no in-flight turns, and
// reports whether it actually had to wait.
//
// A gateway that does not report the field at all is treated as unknown and
// restarted immediately: an older gateway build has no way to tell us, and
// blocking forever on a signal that will never arrive would make configuration
// changes impossible to apply.
func (h *Handler) waitForGatewayIdle() bool {
	deadline := time.Now().Add(configRestartIdleTimeout)
	waited := false

	for {
		healthResponse, _, err := h.getGatewayHealth(nil, configRestartHealthTimeout)
		if err != nil || healthResponse == nil || healthResponse.Busy == nil {
			return waited
		}
		if !*healthResponse.Busy {
			return waited
		}
		if time.Now().After(deadline) {
			logger.WarnCF("gateway",
				"Gateway still busy after the idle timeout; restarting anyway",
				map[string]any{
					"timeout": configRestartIdleTimeout.String(),
				})
			return waited
		}
		waited = true
		time.Sleep(configRestartIdlePoll)
	}
}

// handleGatewayApplyConfig restarts the gateway so a saved configuration change
// takes effect, waiting for in-flight turns to finish first.
//
//	POST /api/gateway/apply-config
//
// It is separate from POST /api/gateway/restart on purpose. The manual restart
// is an immediate recovery action the user asked for explicitly; this one is a
// consequence of saving settings, and must not interrupt an answer in progress.
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
