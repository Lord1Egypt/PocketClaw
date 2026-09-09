package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
)

// registerPocketClawRoutes binds the managed realtime channel's management
// endpoints to the ServeMux.
func (h *Handler) registerPocketClawRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+config.RealtimeAPIPrefix+"info", h.handleGetPocketClawInfo)
	mux.HandleFunc("POST "+config.RealtimeAPIPrefix+"token", h.handleRegenPocketClawToken)
	mux.HandleFunc("POST "+config.RealtimeAPIPrefix+"setup", h.handlePocketClawSetup)

	// WebSocket proxy: forward the realtime socket to the gateway. This lets
	// the frontend connect on the same port as the web UI, so no extra port has
	// to be exposed for WebSocket traffic.
	mux.HandleFunc("GET "+config.RealtimeWebSocketPath, h.handleWebSocketProxy())
	mux.HandleFunc("GET "+realtimeMediaPattern, h.handlePocketClawMediaProxy())
	mux.HandleFunc("HEAD "+realtimeMediaPattern, h.handlePocketClawMediaProxy())
}

// realtimeMediaPattern is the ServeMux pattern for the media proxy. The paths
// themselves come from pkg/config, which is also where the channel and the
// middleware read them.
const realtimeMediaPattern = config.RealtimeMediaPrefix + "{id}"

// createWsProxy creates a reverse proxy to the current gateway WebSocket endpoint.
// The gateway bind host and port are resolved from the latest configuration.
func (h *Handler) createWsProxy(origProtocol string, upstreamProtocol string) *httputil.ReverseProxy {
	wsProxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target := h.gatewayProxyURL()
			r.SetURL(target)
			r.Out.Header.Del(protocolKey)
			if upstreamProtocol != "" {
				r.Out.Header.Set(protocolKey, upstreamProtocol)
			}
		},
		ModifyResponse: func(r *http.Response) error {
			if prot := r.Header.Values(protocolKey); len(prot) > 0 {
				r.Header.Del(protocolKey)
				if origProtocol != "" {
					r.Header.Set(protocolKey, origProtocol)
				}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Errorf("Failed to proxy WebSocket: %v", err)
			http.Error(w, "Gateway unavailable: "+err.Error(), http.StatusBadGateway)
		},
	}
	return wsProxy
}

func (h *Handler) createPicoHTTPProxy(token string) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target := h.gatewayProxyURL()
			r.SetURL(target)
			r.Out.Header.Set("Authorization", "Bearer "+token)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Errorf("Failed to proxy Pico HTTP request: %v", err)
			http.Error(w, "Gateway unavailable: "+err.Error(), http.StatusBadGateway)
		},
	}
}

func (h *Handler) gatewayAvailableForProxy() bool {
	gateway.mu.Lock()
	ensurePicoTokenCachedLocked(h.configPath)
	cachedPID := gateway.pidData
	trackedCmd := gateway.cmd
	gateway.mu.Unlock()

	if pidData := h.sanitizeGatewayPidData(ppid.ReadPidFileWithCheck(globalConfigDir()), nil, "realtime"); pidData != nil {
		gateway.mu.Lock()
		gateway.pidData = pidData
		setGatewayRuntimeStatusLocked("running")
		gateway.mu.Unlock()
		return true
	}

	if cachedPID == nil {
		return false
	}

	if isCmdProcessAliveLocked(trackedCmd) {
		return true
	}

	gateway.mu.Lock()
	if gateway.cmd == trackedCmd {
		gateway.pidData = nil
		setGatewayRuntimeStatusLocked("stopped")
	}
	available := gateway.pidData != nil
	gateway.mu.Unlock()
	return available
}

func decodePocketClawSettings(cfg *config.Config) (config.PocketClawSettings, bool) {
	if cfg == nil {
		return config.PocketClawSettings{}, false
	}

	bc := cfg.Channels.GetByType(config.ChannelPocketClaw)
	if bc == nil {
		return config.PocketClawSettings{}, false
	}

	var picoCfg config.PocketClawSettings
	if err := bc.Decode(&picoCfg); err != nil {
		return config.PocketClawSettings{}, false
	}

	return picoCfg, bc.Enabled
}

func (h *Handler) writePocketClawInfoResponse(
	w http.ResponseWriter,
	r *http.Request,
	cfg *config.Config,
	changed *bool,
) {
	picoCfg, enabled := decodePocketClawSettings(cfg)

	resp := map[string]any{
		"ws_url":  h.buildWsURL(r),
		"enabled": enabled,
	}
	if changed != nil {
		resp["changed"] = *changed
	}
	if picoCfg.Token.String() != "" {
		resp["configured"] = true
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleWebSocketProxy wraps a reverse proxy to handle WebSocket connections.
// It relies on launcher dashboard auth, then injects the raw pico token only
// on the upstream gateway request.
func (h *Handler) handleWebSocketProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.gatewayAvailableForProxy() {
			logger.Warnf("Gateway not available for WebSocket proxy")
			http.Error(w, "Gateway not available", http.StatusServiceUnavailable)
			return
		}

		upstreamProtocol := picoGatewayProtocol()
		if upstreamProtocol == "" {
			logger.Warn("Pico token unavailable for WebSocket proxy")
			http.Error(w, "Pico channel not configured", http.StatusServiceUnavailable)
			return
		}

		var origProtocol string
		if prot := r.Header.Values(protocolKey); len(prot) > 0 {
			origProtocol = prot[0]
		}

		h.createWsProxy(origProtocol, upstreamProtocol).ServeHTTP(w, r)
	}
}

func (h *Handler) handlePocketClawMediaProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.gatewayAvailableForProxy() {
			logger.Warnf("Gateway not available for Pico media proxy")
			http.Error(w, "Gateway not available", http.StatusServiceUnavailable)
			return
		}

		gateway.mu.Lock()
		picoToken := gateway.picoToken
		gateway.mu.Unlock()

		if picoToken == "" {
			logger.Warnf("Missing Pico token for media proxy")
			http.Error(w, "Invalid Pico token", http.StatusForbidden)
			return
		}

		h.createPicoHTTPProxy(picoToken).ServeHTTP(w, r)
	}
}

// handleGetPocketClawInfo returns non-secret Pico connection info for the launcher UI.
//
//	GET /api/pocketclaw/info
func (h *Handler) handleGetPocketClawInfo(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	h.writePocketClawInfoResponse(w, r, cfg, nil)
}

// handleRegenPocketClawToken rotates the raw Pico WebSocket token and returns
// non-secret connection info for the launcher UI.
//
//	POST /api/pocketclaw/token
func (h *Handler) handleRegenPocketClawToken(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	token, err := generateSecureToken()
	if err != nil {
		http.Error(w, "Failed to generate Pico credential", http.StatusInternalServerError)
		return
	}
	if bc := cfg.Channels.GetByType(config.ChannelPocketClaw); bc != nil {
		decoded, err := bc.GetDecoded()
		if err == nil && decoded != nil {
			if settings, ok := decoded.(*config.PocketClawSettings); ok {
				settings.Token = *config.NewSecureString(token)
			}
		}
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	gateway.mu.Lock()
	gateway.picoToken = token
	gateway.mu.Unlock()

	h.writePocketClawInfoResponse(w, r, cfg, nil)
}

// EnsurePocketClawChannel enables the Pico channel with sane defaults if it isn't
// already configured. Returns true when the config was modified.
func (h *Handler) EnsurePocketClawChannel() (bool, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return false, fmt.Errorf("failed to load config: %w", err)
	}

	changed := false

	bc := cfg.Channels.GetByType(config.ChannelPocketClaw)
	if bc == nil {
		bc = &config.Channel{Type: config.ChannelPocketClaw}
		cfg.Channels[config.ChannelPocketClaw] = bc
	}

	if !bc.Enabled {
		bc.Enabled = true
		changed = true
	}
	ownerAllowFrom := config.FlexibleStringSlice{config.PocketClawOwnerPrincipal}
	if len(bc.AllowFrom) != 1 || bc.AllowFrom[0] != config.PocketClawOwnerPrincipal {
		bc.AllowFrom = ownerAllowFrom
		changed = true
	}

	if decoded, err := bc.GetDecoded(); err == nil && decoded != nil {
		if picoCfg, ok := decoded.(*config.PocketClawSettings); ok {
			if picoCfg.Token.String() == "" {
				token, tokenErr := generateSecureToken()
				if tokenErr != nil {
					return false, fmt.Errorf("failed to generate pico credential: %w", tokenErr)
				}
				picoCfg.Token = *config.NewSecureString(token)
				changed = true
			}
		}
	}

	if changed {
		if err := config.SaveConfig(h.configPath, cfg); err != nil {
			return false, fmt.Errorf("failed to save config: %w", err)
		}
	}

	return changed, nil
}

// handlePocketClawSetup automatically configures everything needed for the Pico Channel to work.
//
//	POST /api/pocketclaw/setup
func (h *Handler) handlePocketClawSetup(w http.ResponseWriter, r *http.Request) {
	changed, err := h.EnsurePocketClawChannel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload config (EnsurePocketClawChannel may have modified it).
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	h.writePocketClawInfoResponse(w, r, cfg, &changed)
}

// generateSecureToken creates a random 32-character hex string. Credential
// creation fails closed if the platform CSPRNG is unavailable.
func generateSecureToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
