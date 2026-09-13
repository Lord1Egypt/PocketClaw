package providers

import "strings"

// OpenCode Zen and OpenCode Go are mixed-protocol gateways: a single base URL
// and a single API key serve several request protocols, and which one applies
// is a property of the selected model, not of the provider. Every routing
// decision for those two providers lives in this file so that model-name
// conditionals never spread into the factory or into unrelated packages.

// OpenCodeProtocol names the request protocol a model is served over.
type OpenCodeProtocol string

const (
	// OpenCodeResponses is the OpenAI Responses API: POST {base}/responses.
	OpenCodeResponses OpenCodeProtocol = "responses"
	// OpenCodeChatCompletions is OpenAI-compatible chat: POST {base}/chat/completions.
	OpenCodeChatCompletions OpenCodeProtocol = "chat_completions"
	// OpenCodeMessages is the Anthropic Messages API: POST {base}/messages.
	OpenCodeMessages OpenCodeProtocol = "messages"
)

const (
	openCodeZenProvider = "opencode_zen"
	openCodeGoProvider  = "opencode_go"
)

// OpenCodeSessionHeader is the conversation identity these gateways route on.
//
// PC-DEF-032. Owner-verified against the live service on 2026-09-13: the same
// endpoint, key and model that answers HTTP 200 with this header present
// answers HTTP 400 without it:
//
//	{"type":"error","error":{"type":"MissingSessionID","message":
//	 "Error from provider (Console Go): Request is missing x-opencode-session
//	 and cannot be routed efficiently..."}}
//
// The value is derived, never the session key itself — see
// common.StableSessionID for why a PocketClaw session key must not leave the
// device.
//
// Applied to both gateways rather than to Go alone. They are one service behind
// one account key, sharing this file's routing; the requirement was proven on
// Go, and an additive routing header is not something the Zen surface can be
// harmed by. It is scoped to this provider family and reaches nothing else.
const OpenCodeSessionHeader = "x-opencode-session"

// OpenCodeUserAgent identifies PocketClaw to the OpenCode gateways.
//
// The factory's default is the upstream "PicoClaw/<core version>", which says
// nothing true about the client actually making the request. Bump this with the
// product version in pubspec.yaml; Core does not carry the app's version.
const OpenCodeUserAgent = "PocketClaw/0.2.0"

// IsOpenCodeProvider reports whether a normalized provider ID is one of the
// mixed-protocol OpenCode gateways.
func IsOpenCodeProvider(provider string) bool {
	switch NormalizeProvider(provider) {
	case openCodeZenProvider, openCodeGoProvider:
		return true
	}
	return false
}

// openCodeModelFamilies maps a model-ID prefix to the protocol that family is
// served over. Longest match wins, so a more specific prefix can override a
// broader one. Prefixes are matched against the normalized (namespace-stripped,
// lowercased) model ID.
//
// This is deliberately family-based rather than an exhaustive model list:
// OpenCode adds and retires individual models frequently, and Fetch Models
// already returns the authoritative current list from the provider. Pinning
// exact model IDs here would make the list wrong within weeks.
var openCodeModelFamilies = map[string]OpenCodeProtocol{
	// OpenAI Responses family.
	"gpt-":   OpenCodeResponses,
	"gpt5":   OpenCodeResponses,
	"o1":     OpenCodeResponses,
	"o3":     OpenCodeResponses,
	"o4":     OpenCodeResponses,
	"codex":  OpenCodeResponses,
	"openai": OpenCodeResponses,

	// Anthropic Messages family.
	"claude":    OpenCodeMessages,
	"anthropic": OpenCodeMessages,
	"sonnet":    OpenCodeMessages,
	"opus":      OpenCodeMessages,
	"haiku":     OpenCodeMessages,

	// OpenAI-compatible chat completions family.
	"kimi":     OpenCodeChatCompletions,
	"moonshot": OpenCodeChatCompletions,
	"deepseek": OpenCodeChatCompletions,
	"glm":      OpenCodeChatCompletions,
	"zai":      OpenCodeChatCompletions,
	"qwen":     OpenCodeChatCompletions,
	"grok":     OpenCodeChatCompletions,
	"llama":    OpenCodeChatCompletions,
	"mistral":  OpenCodeChatCompletions,
	"minimax":  OpenCodeChatCompletions,
	"gemini":   OpenCodeChatCompletions,
	"google":   OpenCodeChatCompletions,
}

// openCodeFallbackProtocol is used for a model this build does not recognize.
// Chat completions is the broadest OpenAI-style surface both gateways expose,
// so it is the least-bad guess — but the caller is told the classification was
// a fallback so it can say so rather than failing opaquely.
const openCodeFallbackProtocol = OpenCodeChatCompletions

// NormalizeOpenCodeModelID strips the OpenCode TUI namespace prefix from a
// model identifier. The direct HTTP endpoints expect the bare model ID
// ("kimi-k3"), not the namespaced form the OpenCode CLI displays
// ("opencode-go/kimi-k3"), so the prefix is removed before the request is
// built and is never persisted into the outgoing model field.
func NormalizeOpenCodeModelID(model string) string {
	trimmed := strings.TrimSpace(model)
	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{
		"opencode-zen/", "opencode_zen/", "opencode-go/", "opencode_go/", "opencode/",
	} {
		if strings.HasPrefix(lower, prefix) {
			return trimmed[len(prefix):]
		}
	}
	return trimmed
}

// ClassifyOpenCodeModel resolves the request protocol for an OpenCode model.
//
// The second return value reports whether the model matched a known family.
// When it is false the returned protocol is the fallback, and the caller must
// surface that rather than treating the routing as authoritative — an unknown
// model is allowed to be configured and attempted, but never silently
// presented as a confirmed route.
func ClassifyOpenCodeModel(model string) (protocol OpenCodeProtocol, known bool) {
	normalized := strings.ToLower(NormalizeOpenCodeModelID(model))
	if normalized == "" {
		return openCodeFallbackProtocol, false
	}

	// Longest matching prefix wins so that a specific family beats a general one.
	best := ""
	var bestProtocol OpenCodeProtocol
	for prefix, candidate := range openCodeModelFamilies {
		if strings.HasPrefix(normalized, prefix) && len(prefix) > len(best) {
			best = prefix
			bestProtocol = candidate
		}
	}
	if best == "" {
		return openCodeFallbackProtocol, false
	}
	return bestProtocol, true
}
