// Package modelaccess decides whether a model_list entry is configured enough
// to be offered to a user.
//
// The rule lived in web/backend/api, where the Dashboard's model list needs it.
// The Telegram model picker needs exactly the same answer, and pkg/commands
// cannot import the web backend — so rather than have two interpretations of
// "configured" drift apart, the rule lives here and both callers use it.
//
// This package answers a question about configuration only. Whether a locally
// hosted model is actually reachable is a network probe, and probing belongs to
// the caller that can afford to wait; a chat command cannot.
package modelaccess

import (
	"net/url"
	"strings"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// OAuth provider identities, matching the names the credential store uses.
const (
	oauthProviderOpenAI            = "openai"
	oauthProviderAnthropic         = "anthropic"
	oauthProviderGoogleAntigravity = "google-antigravity"
)

// getCredential is a variable so tests can supply a credential store without
// touching the user's real one.
var getCredential = auth.GetCredential

// IsConfigured reports whether the entry carries enough configuration to be
// selectable.
//
// It is deliberately permissive in the two cases where being strict would hide
// a model that works: a provider reading ambient credentials from its own SDK
// chain, and a local runtime whose availability is a probe rather than a
// stored secret. Concrete credential failures surface at runtime with a clear
// error, which is better than silently omitting a model the user configured.
func IsConfigured(m *config.ModelConfig) bool {
	if m == nil {
		return false
	}

	protocol := Protocol(m)
	authMethod := strings.ToLower(strings.TrimSpace(m.AuthMethod))

	if authMethod == "oauth" || authMethod == "token" {
		if configured, checked := hasStoredOAuthCredential(m); checked {
			return configured
		}
	}

	if authMethod == "" && providerUsesImplicitOAuth(protocol) {
		if configured, checked := hasStoredOAuthCredential(m); checked {
			return configured
		}
	}

	if providerUsesAmbientCredentials(protocol) {
		return true
	}

	if RequiresRuntimeProbe(m) {
		return true
	}

	return strings.TrimSpace(m.APIKey()) != ""
}

// Protocol returns the entry's normalized provider protocol.
func Protocol(m *config.ModelConfig) string {
	if m == nil {
		return ""
	}
	protocol, _ := providers.ExtractProtocol(m)
	return strings.ToLower(strings.TrimSpace(protocol))
}

// RequiresRuntimeProbe reports whether availability can only be settled by
// contacting the model, rather than by inspecting configuration.
func RequiresRuntimeProbe(m *config.ModelConfig) bool {
	if m == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(m.AuthMethod), "local") {
		return true
	}

	protocol := Protocol(m)
	switch protocol {
	case "claude-cli", "codex-cli", "github-copilot":
		return true
	}

	// A protocol that permits an empty API key is a local runtime — Ollama,
	// LM Studio and similar. With no api_base it uses its own default local
	// endpoint, so an absent base is as much a local runtime as an explicit one.
	if providers.IsHTTPAPIProtocol(protocol) && providers.IsEmptyAPIKeyAllowedForProtocol(protocol) {
		apiBase := strings.TrimSpace(m.APIBase)
		return apiBase == "" || hasLocalAPIBase(apiBase)
	}

	return hasLocalAPIBase(m.APIBase)
}

func hasStoredOAuthCredential(m *config.ModelConfig) (configured, checked bool) {
	provider, ok := oauthProviderFor(m)
	if !ok {
		return false, false
	}
	cred, err := getCredential(provider)
	if err != nil || cred == nil {
		return false, true
	}
	return strings.TrimSpace(cred.AccessToken) != "" ||
		strings.TrimSpace(cred.RefreshToken) != "", true
}

func oauthProviderFor(m *config.ModelConfig) (string, bool) {
	switch Protocol(m) {
	case "openai":
		return oauthProviderOpenAI, true
	case "anthropic":
		return oauthProviderAnthropic, true
	case "antigravity":
		return oauthProviderGoogleAntigravity, true
	default:
		return "", false
	}
}

func providerUsesImplicitOAuth(protocol string) bool {
	return protocol == "antigravity"
}

// providerUsesAmbientCredentials marks providers that read credentials from
// their own environment chain. Bedrock uses the AWS SDK's, which cannot be
// preflighted here without misclassifying working setups as unconfigured.
func providerUsesAmbientCredentials(protocol string) bool {
	return protocol == "bedrock"
}

// hasLocalAPIBase parses the host rather than matching substrings, so an
// api_base merely containing "localhost" somewhere in a path or query is not
// mistaken for a local endpoint.
func hasLocalAPIBase(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		u, err = url.Parse("//" + raw)
		if err != nil {
			return false
		}
	}

	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	default:
		return false
	}
}
