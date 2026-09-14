package api

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// AndroidBridgeTokenEnv is a random per-process credential shared only by the
// Android host and its bundled Core child process. It is never persisted.
const AndroidBridgeTokenEnv = "POCKETCLAW_ANDROID_BRIDGE_TOKEN"

const androidTelegramBridgePath = "/api/pocketclaw/android/telegram"
const androidNetworkModeBridgePath = "/api/pocketclaw/android/network-mode"
const androidContextMemoryBridgePath = "/api/pocketclaw/android/context-memory"
const androidGatewayStartBridgePath = "/api/pocketclaw/android/gateway/start"

// Telegram context-memory bounds. Native Settings offers presets inside this
// range and a custom value; Core is the authority, so the range is enforced
// here rather than trusted from the host.
//
// The floor keeps a turn usable: below a handful of messages the model loses
// the exchange it is answering. The ceiling keeps the prompt bounded, which is
// the whole point of the window.
const (
	minTelegramRecentContextMessages = config.MinTelegramRecentContextMessages
	maxTelegramRecentContextMessages = config.MaxTelegramRecentContextMessages
)

type androidContextMemoryRequest struct {
	RecentMessages int `json:"recent_messages"`
}

type androidContextMemoryResponse struct {
	RecentMessages int `json:"recent_messages"`
	Min            int `json:"min"`
	Max            int `json:"max"`
	Default        int `json:"default"`
}

// LauncherNetworkModeController is implemented by the Dashboard HTTP runtime.
// It deliberately has no Core lifecycle methods.
type LauncherNetworkModeController interface {
	ApplyPublicMode(public bool) error
	PublicMode() bool
	// ReconcileAfterDashboardClaimed re-applies the desired exposure once the
	// dashboard has an owner. PC-DEF-040.
	ReconcileAfterDashboardClaimed() error
}

type launcherNetworkModeState struct {
	Status string
	Public bool
	Target bool
	Error  string
}

type androidNetworkModeRequest struct {
	Public bool `json:"public"`
}

type androidNetworkModeResponse struct {
	Status string `json:"status"`
	Public bool   `json:"public"`
	Target bool   `json:"target"`
	Error  string `json:"error,omitempty"`
}

type androidTelegramCredentials struct {
	Token       string `json:"token"`
	OwnerUserID int64  `json:"owner_user_id,omitempty"`
}

// RegisterAndroidBridgeRoutes exposes the narrow loopback-only operations the
// Android host needs: managed Telegram pairing and Dashboard listener rebind.
// Native Settings does not query Telegram state through this bridge.
func (h *Handler) RegisterAndroidBridgeRoutes(mux *http.ServeMux, bridgeToken string) {
	bridgeToken = strings.TrimSpace(bridgeToken)
	if bridgeToken == "" {
		return
	}

	mux.HandleFunc("PUT "+androidTelegramBridgePath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizedAndroidBridgeRequest(r, bridgeToken) {
			http.NotFound(w, r)
			return
		}
		h.handleAndroidTelegramConfigure(w, r)
	})
	mux.HandleFunc("PUT "+androidNetworkModeBridgePath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizedAndroidBridgeRequest(r, bridgeToken) {
			http.NotFound(w, r)
			return
		}
		h.handleAndroidNetworkModeApply(w, r)
	})
	mux.HandleFunc("GET "+androidNetworkModeBridgePath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizedAndroidBridgeRequest(r, bridgeToken) {
			http.NotFound(w, r)
			return
		}
		h.handleAndroidNetworkModeStatus(w)
	})
	// Checking a candidate GitHub credential runs a bundled tool, which only the
	// Managed Runtime may do. The host holds the credential; Core only answers
	// whether GitHub accepts it and for which account.
	mux.HandleFunc("POST "+androidGitHubValidatePath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizedAndroidBridgeRequest(r, bridgeToken) {
			http.NotFound(w, r)
			return
		}
		h.handleAndroidGitHubValidate(w, r)
	})
	mux.HandleFunc("GET "+androidGitHubStatusPath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizedAndroidBridgeRequest(r, bridgeToken) {
			http.NotFound(w, r)
			return
		}
		h.handleAndroidGitHubStatus(w, r)
	})
	// Telegram context memory. Native Settings reads and writes it here so Core
	// stays the only writer of config.json, exactly as Telegram pairing does.
	// Gateway lifecycle for the Android host. The host owns the auto-start
	// preference, so when the user turns it on while the service is already
	// running the host has to be able to act on it now rather than at the next
	// service start. Dashboard credentials are deliberately not accepted for
	// this: lifecycle control belongs to the host process, not to a browser
	// session. See PC-DEF-034.
	mux.HandleFunc("POST "+androidGatewayStartBridgePath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizedAndroidBridgeRequest(r, bridgeToken) {
			http.NotFound(w, r)
			return
		}
		h.handleAndroidGatewayStart(w, r)
	})
	mux.HandleFunc("GET "+androidContextMemoryBridgePath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizedAndroidBridgeRequest(r, bridgeToken) {
			http.NotFound(w, r)
			return
		}
		h.handleAndroidContextMemoryStatus(w)
	})
	mux.HandleFunc("PUT "+androidContextMemoryBridgePath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizedAndroidBridgeRequest(r, bridgeToken) {
			http.NotFound(w, r)
			return
		}
		h.handleAndroidContextMemoryApply(w, r)
	})
}

func (h *Handler) SetLauncherNetworkModeController(controller LauncherNetworkModeController) {
	h.launcherNetworkModeMu.Lock()
	defer h.launcherNetworkModeMu.Unlock()
	h.launcherNetworkMode = controller
	if controller != nil {
		h.launcherNetworkModeState.Public = controller.PublicMode()
		h.launcherNetworkModeState.Target = h.launcherNetworkModeState.Public
	}
	if h.launcherNetworkModeState.Status == "" {
		h.launcherNetworkModeState.Status = "idle"
	}
}

func (h *Handler) handleAndroidNetworkModeApply(w http.ResponseWriter, r *http.Request) {
	var request androidNetworkModeRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	h.launcherNetworkModeMu.Lock()
	controller := h.launcherNetworkMode
	if controller == nil {
		h.launcherNetworkModeMu.Unlock()
		http.Error(w, "Network mode control unavailable", http.StatusServiceUnavailable)
		return
	}
	if h.launcherNetworkModeState.Status == "applying" {
		h.launcherNetworkModeMu.Unlock()
		http.Error(w, "Network mode change already in progress", http.StatusConflict)
		return
	}
	h.launcherNetworkModeState = launcherNetworkModeState{
		Status: "applying",
		Public: controller.PublicMode(),
		Target: request.Public,
	}
	response := h.launcherNetworkModeState
	h.launcherNetworkModeMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(androidNetworkModeResponse{
		Status: response.Status,
		Public: response.Public,
		Target: response.Target,
	})

	// Let the accepted response reach Android before replacing the listener
	// which carried this request.
	time.AfterFunc(100*time.Millisecond, func() {
		err := controller.ApplyPublicMode(request.Public)
		actualPublic := controller.PublicMode()
		h.launcherNetworkModeMu.Lock()
		defer h.launcherNetworkModeMu.Unlock()
		h.launcherNetworkModeState.Public = actualPublic
		h.launcherNetworkModeState.Target = request.Public
		if err != nil {
			h.launcherNetworkModeState.Status = "failed"
			if request.Public {
				h.launcherNetworkModeState.Error = "Could not enable LAN access. PocketClaw remains available locally."
			} else {
				h.launcherNetworkModeState.Error = "Could not disable LAN access. PocketClaw remains in its previous network mode."
			}
			return
		}
		h.launcherNetworkModeState.Status = "succeeded"
		h.launcherNetworkModeState.Error = ""
	})
}

func (h *Handler) handleAndroidNetworkModeStatus(w http.ResponseWriter) {
	h.launcherNetworkModeMu.Lock()
	state := h.launcherNetworkModeState
	if h.launcherNetworkMode != nil && state.Status != "applying" {
		state.Public = h.launcherNetworkMode.PublicMode()
	}
	h.launcherNetworkModeMu.Unlock()
	if state.Status == "" {
		state.Status = "idle"
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(androidNetworkModeResponse{
		Status: state.Status,
		Public: state.Public,
		Target: state.Target,
		Error:  state.Error,
	})
}

func authorizedAndroidBridgeRequest(r *http.Request, expectedToken string) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return false
	}

	provided := r.Header.Get("X-PocketClaw-Android-Bridge")
	if len(provided) != len(expectedToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expectedToken)) == 1
}

func (h *Handler) handleAndroidTelegramConfigure(w http.ResponseWriter, r *http.Request) {
	var request androidTelegramCredentials
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	request.Token = strings.TrimSpace(request.Token)
	if request.Token == "" {
		http.Error(w, "Telegram token is required", http.StatusBadRequest)
		return
	}
	if request.OwnerUserID <= 0 {
		http.Error(w, "Telegram owner user ID is required", http.StatusBadRequest)
		return
	}

	cfg, channel, settings, err := h.loadTelegramConfigForUpdate()
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}
	settings.Token.Set(request.Token)
	channel.Enabled = true
	channel.Type = config.ChannelTelegram
	channel.AllowFrom = config.FlexibleStringSlice{strconv.FormatInt(request.OwnerUserID, 10)}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	// PC-DEF-030. Saving Telegram used to end here, so the running channel
	// never learned about the change and the user was told to restart the
	// Gateway by hand. Applying is part of saving now: immediately when the
	// gateway is idle, and otherwise marked pending for its own idle
	// notification to pick up. Either way nothing is asked of the user.
	applied, pending := h.applyTelegramConfigChange("telegram_configured")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"applied": applied,
		"pending": pending,
	})
}

// applyTelegramConfigChange makes a saved Telegram change live.
//
// Reports what actually happened rather than assuming success: a gateway that
// is busy leaves the change pending, and a gateway that is stopped leaves it
// for the next start. Neither is an error, and neither may be presented as a
// working Telegram channel -- runtime status decides that, not this.
//
// The idle question is asked once here rather than waited out. The wait inside
// RestartGatewayForConfigChange runs for up to two minutes, and this handler is
// answering an Android host whose bridge call has its own read timeout: a save
// that blocks past it is reported to the user as a failed configuration even
// though the token is on disk. Parking the change instead loses nothing --
// the gateway's idle notification and the pending supervisor both apply it with
// no user action -- and it keeps "do not interrupt a running answer" intact,
// because a gateway that will not say it is idle is never restarted.
func (h *Handler) applyTelegramConfigChange(reason string) (applied bool, pending bool) {
	if !h.gatewayIdleNow() {
		markConfigApplyPending(reason)
		if startPendingApplySupervisor() {
			go h.supervisePendingConfigApply()
		}
		logger.InfoCF("gateway",
			"Telegram configuration saved while the gateway was busy; "+
				"it will be applied automatically once the gateway is idle",
			map[string]any{"reason": reason})
		return false, true
	}

	if _, _, err := h.RestartGatewayForConfigChange(reason); err != nil {
		isPending, _ := pendingConfigApplyState()
		return false, isPending
	}
	return true, false
}

func (h *Handler) loadTelegramConfigForUpdate() (*config.Config, *config.Channel, *config.TelegramSettings, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return nil, nil, nil, err
	}
	if cfg.Channels == nil {
		cfg.Channels = make(config.ChannelsConfig)
	}
	channel := cfg.Channels.Get(config.ChannelTelegram)
	if channel == nil {
		channel = &config.Channel{Type: config.ChannelTelegram}
		channel.SetName(config.ChannelTelegram)
		cfg.Channels[config.ChannelTelegram] = channel
	}
	decoded, err := channel.GetDecoded()
	if err != nil {
		return nil, nil, nil, err
	}
	settings, ok := decoded.(*config.TelegramSettings)
	if !ok || settings == nil {
		settings = &config.TelegramSettings{}
		if err := channel.Decode(settings); err != nil {
			return nil, nil, nil, err
		}
	}
	return cfg, channel, settings, nil
}

// handleAndroidContextMemoryStatus reports the effective Telegram context
// memory, so Settings shows what Core will actually use rather than what the
// file happens to contain. An unset or out-of-range stored value reads as the
// default, which is the same resolution the agent applies.
func (h *Handler) handleAndroidContextMemoryStatus(w http.ResponseWriter) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(androidContextMemoryResponse{
		RecentMessages: effectiveTelegramRecentContextMessages(cfg),
		Min:            minTelegramRecentContextMessages,
		Max:            maxTelegramRecentContextMessages,
		Default:        config.DefaultTelegramRecentContextMessages,
	})
}

// handleAndroidContextMemoryApply stores a new limit through Core's own
// SaveConfig.
//
// It changes one number and nothing else: the loaded config is saved back with
// only this field altered, so Telegram credentials, channels, models and every
// other setting are carried through untouched. It deletes no message, clears no
// summary and touches no session history — the next turn simply projects a
// different number of recent messages.
func (h *Handler) handleAndroidContextMemoryApply(w http.ResponseWriter, r *http.Request) {
	var request androidContextMemoryRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if request.RecentMessages < minTelegramRecentContextMessages ||
		request.RecentMessages > maxTelegramRecentContextMessages {
		http.Error(w, "Recent message count is out of range", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}
	cfg.Agents.Defaults.TelegramRecentContextMessages = request.RecentMessages
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(androidContextMemoryResponse{
		RecentMessages: request.RecentMessages,
		Min:            minTelegramRecentContextMessages,
		Max:            maxTelegramRecentContextMessages,
		Default:        config.DefaultTelegramRecentContextMessages,
	})
}

// effectiveTelegramRecentContextMessages resolves what the agent will use.
func effectiveTelegramRecentContextMessages(cfg *config.Config) int {
	if cfg == nil {
		return config.DefaultTelegramRecentContextMessages
	}
	return config.ResolveTelegramRecentContextMessages(
		cfg.Agents.Defaults.TelegramRecentContextMessages,
	)
}

// handleAndroidGatewayStart starts the gateway on behalf of the Android host.
//
// Idempotent: a gateway that is already running is reported as such rather
// than started twice. Everything else delegates to handleGatewayStart, so the
// lifecycle, the precondition check and the PID-file attach behaviour are the
// same code the manual start uses -- this endpoint is an authorization
// boundary, not a second implementation.
//
// After PC-DEF-038 this succeeds with zero providers and zero models
// configured, which is the entire point: the host must be able to bring
// infrastructure up before the user has configured any AI provider.
func (h *Handler) handleAndroidGatewayStart(w http.ResponseWriter, r *http.Request) {
	gateway.mu.Lock()
	if gateway.cmd != nil && isCmdProcessAliveLocked(gateway.cmd) {
		pid := 0
		if gateway.cmd.Process != nil {
			pid = gateway.cmd.Process.Pid
		}
		gateway.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "already_running",
			"pid":    pid,
		})
		return
	}
	gateway.mu.Unlock()

	h.handleGatewayStart(w, r)
}
