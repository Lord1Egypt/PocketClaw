package api

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/canonicalenv"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/status"
)

// Authoritative Telegram readiness, for the state the Dashboard is allowed to
// call "connected".
//
// PC-DEF-061. Managed pairing used to report success as soon as the gateway had
// been restarted, and the UI said Connected on that. The gateway process being
// up says nothing about whether Telegram is receiving: the channel starts
// asynchronously inside it, and the command menu is published asynchronously
// after that. So a user who was told Connected could send the first message
// into a bot that was not yet consuming updates -- which is the whole point of
// the first-message failure.
//
// Readiness is therefore read from the gateway's own status snapshot rather
// than inferred from what the launcher just did. Nothing here waits on a timer
// or sleeps; each request reports what is true at the moment it is asked, and
// the client polls.

// telegramReadinessState is what the Dashboard renders.
//
// The order is the lifecycle order, and the names are the states a user can be
// shown. `ready` is the only one that means "send your first message now".
type telegramReadinessState string

const (
	// readinessNotConfigured: no Telegram token on disk.
	readinessNotConfigured telegramReadinessState = "not_configured"
	// readinessGatewayStopped: configured, but nothing is running it.
	readinessGatewayStopped telegramReadinessState = "gateway_stopped"
	// readinessGatewayStarting: the process is up but not answering yet, or is
	// up and has not built the channel.
	readinessGatewayStarting telegramReadinessState = "gateway_starting"
	// readinessChannelStarting: the channel exists but is not consuming.
	readinessChannelStarting telegramReadinessState = "channel_starting"
	// readinessRegisteringCommands: consuming, menu not published yet.
	readinessRegisteringCommands telegramReadinessState = "registering_commands"
	// readinessReady: consuming and the menu reached Telegram.
	readinessReady telegramReadinessState = "ready"
	// readinessUnknown: the gateway would not say. Never reported as ready.
	readinessUnknown telegramReadinessState = "unknown"
)

// telegramHealthProbeTimeout bounds one readiness probe. Short on purpose: the
// client polls, so a slow answer should end this request rather than hold it.
const telegramHealthProbeTimeout = 3 * time.Second

func (h *Handler) registerTelegramReadinessRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/telegram/readiness", h.handleTelegramReadiness)
}

// handleTelegramReadiness reports how far Telegram has got.
//
//	GET /api/telegram/readiness
//
// Carries no bot identity, no owner id and no token -- only the lifecycle
// state, which is all the UI needs to decide what to say.
func (h *Handler) handleTelegramReadiness(w http.ResponseWriter, _ *http.Request) {
	state, detail := h.telegramReadiness()
	body := map[string]any{"state": string(state), "ready": state == readinessReady}
	if detail != "" {
		body["detail"] = detail
	}
	writeJSON(w, body)
}

// gatewayRunningProbe answers "is anything running the gateway".
//
// A package-level function value for the same reason gatewayHealthGet is one:
// the answer depends on a live process this package owns, which a test cannot
// produce, and every readiness stage has to be testable.
var gatewayRunningProbe = func(h *Handler) bool { return h.gatewayProcessRunning() }

// telegramReadiness resolves the state, and says why when it is not ready.
func (h *Handler) telegramReadiness() (telegramReadinessState, string) {
	configured, err := h.telegramIsConfigured()
	if err != nil {
		return readinessUnknown, "configuration_unreadable"
	}
	if !configured {
		return readinessNotConfigured, ""
	}
	if !gatewayRunningProbe(h) {
		return readinessGatewayStopped, ""
	}

	channel, err := h.gatewayTelegramChannelStatus()
	if err != nil {
		if errors.Is(err, errGatewayStatusUnavailable) {
			// Up but not answering the authenticated probe yet. Starting, not
			// broken -- and explicitly not ready.
			return readinessGatewayStarting, "status_unavailable"
		}
		return readinessUnknown, "status_unreadable"
	}
	if channel == nil {
		// The gateway is answering and does not have the channel yet.
		return readinessGatewayStarting, ""
	}
	if !channel.Running {
		return readinessChannelStarting, ""
	}
	// Absent means this channel publishes no menu, so there is nothing to wait
	// for. False means it has not landed yet.
	if channel.CommandsRegistered != nil && !*channel.CommandsRegistered {
		return readinessRegisteringCommands, ""
	}
	return readinessReady, ""
}

// telegramIsConfigured reports whether a Telegram token is on disk.
func (h *Handler) telegramIsConfigured() (bool, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return false, err
	}
	channel := cfg.Channels.Get(config.ChannelTelegram)
	if channel == nil || !channel.Enabled {
		return false, nil
	}
	decoded, err := channel.GetDecoded()
	if err != nil {
		return false, err
	}
	settings, ok := decoded.(*config.TelegramSettings)
	if !ok || settings == nil {
		return false, nil
	}
	return strings.TrimSpace(settings.Token.String()) != "", nil
}

// errGatewayStatusUnavailable means the gateway did not serve its detail
// snapshot -- unreachable, or refusing the credential. Distinguished from a
// malformed answer so a starting gateway is not reported as broken.
var errGatewayStatusUnavailable = errors.New("gateway status unavailable")

// gatewayTelegramChannelStatus returns Telegram's entry from the gateway's
// status snapshot, or nil when the gateway does not have the channel.
func (h *Handler) gatewayTelegramChannelStatus() (*status.Channel, error) {
	snapshot, err := h.gatewayStatusSnapshot(telegramHealthProbeTimeout)
	if err != nil {
		return nil, err
	}
	for i := range snapshot.Channels {
		if strings.EqualFold(snapshot.Channels[i].Name, config.ChannelTelegram) {
			return &snapshot.Channels[i], nil
		}
	}
	return nil, nil
}

// gatewayStatusSnapshot reads the gateway's authenticated detail snapshot.
//
// Detail mode is credentialed, and the credential is the gateway's own bearer
// token -- the same one the launcher already holds to talk to it. Nothing new
// is minted here and nothing is stored.
func (h *Handler) gatewayStatusSnapshot(timeout time.Duration) (*status.Snapshot, error) {
	token := gatewayBearerToken()
	if token == "" {
		return nil, errGatewayStatusUnavailable
	}

	url, ok := h.gatewayHealthURL()
	if !ok {
		return nil, errGatewayStatusUnavailable
	}

	req, err := http.NewRequest(http.MethodGet, url+"?detail=1", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errGatewayStatusUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 401 here means the credential moved, which is a launcher problem
		// rather than a Telegram one. Either way it is not readiness.
		logger.DebugCF("web", "Gateway status detail refused", map[string]any{
			"status": resp.StatusCode,
		})
		return nil, errGatewayStatusUnavailable
	}

	var body health.StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Detail == nil {
		return nil, errGatewayStatusUnavailable
	}
	return body.Detail, nil
}

// gatewayHealthURL builds the gateway's health address from the pid record,
// falling back to the configured port exactly as the existing health probe does.
func (h *Handler) gatewayHealthURL() (string, bool) {
	var port int
	var host string
	gateway.mu.Lock()
	if d := gateway.pidData; d != nil && d.Port > 0 {
		port = d.Port
		host = gatewayProbeHost(d.Host)
	}
	gateway.mu.Unlock()

	if port == 0 {
		cfg, err := config.LoadConfig(h.configPath)
		if err != nil {
			return "", false
		}
		port = cfg.Gateway.Port
		if port == 0 {
			port = 18790
		}
		host = gatewayProbeHost(h.effectiveGatewayBindHost(cfg))
	}
	if host == "" {
		return "", false
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/health", true
}

// gatewayBearerToken returns the gateway's bearer credential.
//
// Two places hold it, for one reason. On desktop the pid record is already in a
// private directory and the token stays in it. On Android PICOCLAW_HOME is
// user-visible shared storage where the record's 0600 is synthesised by the
// filesystem, so the credential is written to a separate app-private file
// instead and the record carries none. Both are read here, private file first.
func gatewayBearerToken() string {
	if path := strings.TrimSpace(canonicalenv.Getenv(config.EnvGatewayTokenFile)); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			if token := strings.TrimSpace(string(data)); token != "" {
				return token
			}
		}
	}
	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	if gateway.pidData == nil {
		return ""
	}
	return strings.TrimSpace(gateway.pidData.Token)
}
