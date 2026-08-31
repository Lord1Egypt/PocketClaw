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
)

// AndroidBridgeTokenEnv is a random per-process credential shared only by the
// Android host and its bundled Core child process. It is never persisted.
const AndroidBridgeTokenEnv = "POCKETCLAW_ANDROID_BRIDGE_TOKEN"

const androidTelegramBridgePath = "/api/pocketclaw/android/telegram"
const androidNetworkModeBridgePath = "/api/pocketclaw/android/network-mode"

// LauncherNetworkModeController is implemented by the Dashboard HTTP runtime.
// It deliberately has no Core lifecycle methods.
type LauncherNetworkModeController interface {
	ApplyPublicMode(public bool) error
	PublicMode() bool
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
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
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
