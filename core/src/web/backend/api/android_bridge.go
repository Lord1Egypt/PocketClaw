package api

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// AndroidBridgeTokenEnv is a random per-process credential shared only by the
// Android host and its bundled Core child process. It is never persisted.
const AndroidBridgeTokenEnv = "POCKETCLAW_ANDROID_BRIDGE_TOKEN"

const androidTelegramBridgePath = "/api/pocketclaw/android/telegram"

type androidTelegramCredentials struct {
	Token       string `json:"token"`
	OwnerUserID int64  `json:"owner_user_id,omitempty"`
}

// RegisterAndroidBridgeRoutes exposes only the credential write needed after
// an explicit managed-pairing action in the Core console. Native Settings does
// not query Telegram state through this bridge.
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

	cfg, channel, settings, err := h.loadTelegramConfigForUpdate()
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}
	settings.Token.Set(request.Token)
	channel.Enabled = true
	channel.Type = config.ChannelTelegram
	if request.OwnerUserID > 0 {
		channel.AllowFrom = config.FlexibleStringSlice{strconv.FormatInt(request.OwnerUserID, 10)}
	} else {
		channel.AllowFrom = config.FlexibleStringSlice{}
	}

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
