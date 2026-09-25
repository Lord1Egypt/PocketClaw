package api

import (
	"github.com/sipeed/picoclaw/pkg/telegramonboarding"
	"net/http"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
)

// Handler serves HTTP API requests.
type Handler struct {
	configPath                 string
	serverPort                 int
	serverPublic               bool
	serverPublicExplicit       bool
	serverHostInput            string
	serverHostExplicit         bool
	serverCIDRs                []string
	serverAllowLocalhostBypass bool
	serverTrustedProxyCIDRs    []string
	debug                      bool
	oauthMu                    sync.Mutex
	oauthFlows                 map[string]*oauthFlow
	oauthState                 map[string]string
	weixinMu                   sync.Mutex
	weixinFlows                map[string]*weixinFlow
	wecomMu                    sync.Mutex
	wecomFlows                 map[string]*wecomFlow
	launcherNetworkMode        LauncherNetworkModeController
	// PC-DEF-060. Managed Telegram onboarding for a client with no Android host.
	// Resolved once from the environment; the store keeps poll tokens server-side.
	telegramOnboardingOnce   sync.Once
	telegramOnboarding       *telegramonboarding.Client
	telegramOnboardingStore  *telegramOnboardingStore
	launcherNetworkModeMu    sync.Mutex
	launcherNetworkModeState launcherNetworkModeState
	// githubValidator overrides how a candidate GitHub credential is checked.
	// Production leaves it nil and goes through the Managed Runtime.
	githubValidator GitHubTokenValidator
	// telegramCredentialValidator overrides the pre-commit Telegram getMe
	// check. Production leaves it nil; tests use it to avoid external traffic.
	telegramCredentialValidator TelegramCredentialValidator
}

// SetGitHubTokenValidator replaces credential validation. It exists for tests,
// which have no gh binary and must not depend on network access.
func (h *Handler) SetGitHubTokenValidator(validator GitHubTokenValidator) {
	h.githubValidator = validator
}

// SetTelegramCredentialValidator replaces candidate validation for tests.
func (h *Handler) SetTelegramCredentialValidator(validator TelegramCredentialValidator) {
	h.telegramCredentialValidator = validator
}

// NewHandler creates an instance of the API handler.
func NewHandler(configPath string) *Handler {
	return &Handler{
		configPath:                 configPath,
		serverPort:                 launcherconfig.DefaultPort,
		serverAllowLocalhostBypass: launcherconfig.Default().AllowLocalhostBypass,
		oauthFlows:                 make(map[string]*oauthFlow),
		oauthState:                 make(map[string]string),
		weixinFlows:                make(map[string]*weixinFlow),
		wecomFlows:                 make(map[string]*wecomFlow),
	}
}

// SetServerOptions stores current backend listen options for fallback behavior.
func (h *Handler) SetServerOptions(port int, public bool, publicExplicit bool, allowedCIDRs []string) {
	h.serverPort = port
	h.serverPublic = public
	h.serverPublicExplicit = publicExplicit
	h.serverHostInput = ""
	h.serverHostExplicit = false
	h.serverCIDRs = append([]string(nil), allowedCIDRs...)
}

func (h *Handler) SetServerAccessOptions(allowLocalhostBypass bool, trustedProxyCIDRs []string) {
	h.serverAllowLocalhostBypass = allowLocalhostBypass
	h.serverTrustedProxyCIDRs = append([]string(nil), trustedProxyCIDRs...)
}

// SetServerBindHost stores the launcher's effective bind host.
// When explicit is true, hostInput is the normalized -host / PICOCLAW_LAUNCHER_HOST value.
func (h *Handler) SetServerBindHost(hostInput string, explicit bool) {
	h.serverHostInput = strings.TrimSpace(hostInput)
	if !explicit {
		h.serverHostInput = ""
	}
	h.serverHostExplicit = explicit
}

func (h *Handler) SetDebug(debug bool) {
	h.debug = debug
}

// RegisterRoutes binds all API endpoint handlers to the ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Config CRUD
	h.registerConfigRoutes(mux)

	// PocketClaw channel (WebSocket chat)
	h.registerPocketClawRoutes(mux)

	// Gateway process lifecycle
	h.registerGatewayRoutes(mux)

	// Session history
	h.registerSessionRoutes(mux)

	// OAuth login and credential management
	h.registerOAuthRoutes(mux)

	// Model list management
	h.registerModelRoutes(mux)
	h.registerProviderRoutes(mux)
	h.registerTelegramOnboardingRoutes(mux)
	h.registerTelegramReadinessRoutes(mux)
	h.registerTelegramLifecycleRoutes(mux)
	h.registerTelegramIdentityRoutes(mux)

	// Channel catalog (for frontend navigation/config pages)
	h.registerChannelRoutes(mux)

	// Skills and tools support/actions
	h.registerSkillRoutes(mux)
	h.registerToolRoutes(mux)

	// Launcher service parameters (port/public)
	h.registerLauncherConfigRoutes(mux)

	// Runtime build/version metadata
	h.registerVersionRoutes(mux)

	// WeChat QR login flow
	h.registerWeixinRoutes(mux)

	// WeCom QR login flow
	h.registerWecomRoutes(mux)
}

// Shutdown gracefully shuts down the handler, stopping the gateway if it was started by this handler.
func (h *Handler) Shutdown() {
	h.StopGateway()
}
