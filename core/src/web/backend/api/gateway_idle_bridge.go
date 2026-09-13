package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// The gateway tells the launcher, once, that it has finished the last answer
// it had in flight.
//
// PC-DEF-030. A Telegram configuration change is applied by restarting the
// gateway, and a gateway that is mid-answer must not be interrupted.
// RestartGatewayForConfigChange already waits out a bounded window, but a
// gateway busy past that window used to leave the save persisted and not live,
// with a manual restart as the only way out -- which is the defect in another
// form.
//
// So the gateway reports the one transition that matters, in-flight count
// N>0 -> 0, and the launcher applies whatever is pending. No timer, no poll,
// and no client has to be watching.
const androidGatewayIdlePath = "/api/pocketclaw/internal/gateway-idle"

// gatewayIdleAuth holds the credential for the CURRENT gateway generation.
//
// The token is deliberately not the Android bridge token. That one authorizes
// Telegram writes, network-mode rebinds and GitHub credential checks; the
// gateway needs none of those, and a process should not hold a credential
// broader than its job. It is regenerated on every spawn, so a superseded
// gateway cannot drive the lifecycle of the one that replaced it.
var gatewayIdleAuth = struct {
	mu    sync.Mutex
	token string
}{}

// newGatewayIdleToken rotates the credential and returns it for the child's
// environment. Never persisted, never logged, never sent to Flutter.
func newGatewayIdleToken() string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		// Without a token the endpoint cannot be authorized at all, which is
		// the safe direction: the feature degrades, nothing opens up.
		gatewayIdleAuth.mu.Lock()
		gatewayIdleAuth.token = ""
		gatewayIdleAuth.mu.Unlock()
		return ""
	}
	token := hex.EncodeToString(raw)
	gatewayIdleAuth.mu.Lock()
	gatewayIdleAuth.token = token
	gatewayIdleAuth.mu.Unlock()
	return token
}

// clearGatewayIdleToken invalidates the credential when no gateway is current.
func clearGatewayIdleToken() {
	gatewayIdleAuth.mu.Lock()
	gatewayIdleAuth.token = ""
	gatewayIdleAuth.mu.Unlock()
}

// authorizedGatewayIdleRequest: loopback, and the current generation's token.
//
// RemoteAddr only. Host, Origin, Forwarded and X-Forwarded-For are request
// content. A dashboard session authorizes nothing here, and neither does the
// Android bridge token -- this compares against one specific secret.
func authorizedGatewayIdleRequest(r *http.Request) bool {
	gatewayIdleAuth.mu.Lock()
	expected := gatewayIdleAuth.token
	gatewayIdleAuth.mu.Unlock()
	if expected == "" {
		return false
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return false
	}

	provided := r.Header.Get("X-PocketClaw-Gateway-Idle")
	if len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// RegisterGatewayIdleRoute exposes the one internal notification.
func (h *Handler) RegisterGatewayIdleRoute(mux *http.ServeMux) {
	mux.HandleFunc("POST "+androidGatewayIdlePath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizedGatewayIdleRequest(r) {
			http.NotFound(w, r)
			return
		}
		h.handleGatewayIdleNotification(w)
	})
}

// handleGatewayIdleNotification answers immediately and applies afterwards.
//
// The caller is the gateway process itself. Restarting it synchronously here
// would have the launcher waiting for the gateway to exit while the gateway
// waits for this response, so the apply is deliberately scheduled outside the
// handler and the response is released first.
func (h *Handler) handleGatewayIdleNotification(w http.ResponseWriter) {
	pending := takePendingConfigApply()
	w.WriteHeader(http.StatusAccepted)

	if pending == "" {
		return
	}
	go h.applyPendingConfigRestart(pending)
}

// applyPendingConfigRestart performs exactly one apply for a pending change.
//
// One attempt, no automatic retry. A failure leaves the configuration
// persisted and records a sanitized reason; the next explicit configuration
// change or a normal gateway start is what tries again. Spinning here would
// turn a broken configuration into a restart loop.
func (h *Handler) applyPendingConfigRestart(reason string) {
	if _, _, err := h.RestartGatewayForConfigChange(reason); err != nil {
		setPendingConfigApplyError(sanitizeConfigApplyError(err))
		logger.WarnCF("gateway", "Pending configuration apply failed after the gateway became idle",
			map[string]any{"reason": reason})
		return
	}
	setPendingConfigApplyError("")
	logger.InfoCF("gateway", "Pending configuration applied after the gateway became idle",
		map[string]any{"reason": reason})
}

// launcherIdleNotifyURL is where the gateway reports that it became idle.
//
// Loopback only. The launcher may be bound to a LAN address as well, but the
// child is on this machine and this endpoint is never offered off it.
func (h *Handler) launcherIdleNotifyURL() string {
	port := h.serverPort
	if port == 0 {
		port = 18800
	}
	return "http://127.0.0.1:" + strconv.Itoa(port) + androidGatewayIdlePath
}
