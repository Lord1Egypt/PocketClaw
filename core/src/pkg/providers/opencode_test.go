package providers

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	anthropicmessages "github.com/sipeed/picoclaw/pkg/providers/anthropic_messages"
	"github.com/sipeed/picoclaw/pkg/providers/common"
	openairesponses "github.com/sipeed/picoclaw/pkg/providers/openai_responses"
)

// providercommonNormalizeBaseURL is the exact rule the Messages provider applies.
var providercommonNormalizeBaseURL = common.NormalizeBaseURL

const testAPIKey = "oc-test-key-not-a-real-secret"

func openCodeConfig(provider, model string) *config.ModelConfig {
	return &config.ModelConfig{
		ModelName: "t",
		Provider:  provider,
		Model:     model,
		APIKeys:   config.SimpleSecureStrings(testAPIKey),
	}
}

func TestOpenCodePresetsExistWithOfficialEndpoints(t *testing.T) {
	cases := map[string]struct {
		displayName string
		apiBase     string
	}{
		"opencode_zen": {"OpenCode Zen", "https://opencode.ai/zen/v1"},
		"opencode_go":  {"OpenCode Go", "https://opencode.ai/zen/go/v1"},
	}

	for id, want := range cases {
		option, ok := modelProviderOptionForName(id)
		if !ok {
			t.Fatalf("provider %q is missing from the catalog", id)
		}
		if option.DisplayName != want.displayName {
			t.Errorf("%s DisplayName = %q, want %q", id, option.DisplayName, want.displayName)
		}
		if option.DefaultAPIBase != want.apiBase {
			t.Errorf("%s DefaultAPIBase = %q, want %q", id, option.DefaultAPIBase, want.apiBase)
		}
		if !option.SupportsFetch {
			t.Errorf("%s must support model discovery", id)
		}
		if option.EmptyAPIKeyAllowed {
			t.Errorf("%s must require an API key", id)
		}
		if !option.CreateAllowed || !option.DefaultModelAllowed {
			t.Errorf("%s must be creatable and usable as the default model", id)
		}
		if option.Category != "cloud" {
			t.Errorf("%s Category = %q, want cloud so the base URL stays hidden", id, option.Category)
		}
	}
}

// The Fetch Models URL is the catalog base plus "/models". These are the exact
// official endpoints, asserted literally so a base-URL edit cannot pass silently.
func TestOpenCodeModelsEndpoints(t *testing.T) {
	cases := map[string]string{
		"opencode_zen": "https://opencode.ai/zen/v1/models",
		"opencode_go":  "https://opencode.ai/zen/go/v1/models",
	}
	for id, want := range cases {
		got := strings.TrimRight(DefaultAPIBaseForProtocol(id), "/") + "/models"
		if got != want {
			t.Errorf("%s models endpoint = %q, want %q", id, got, want)
		}
	}
}

func TestOpenCodeAliasesResolve(t *testing.T) {
	cases := map[string]string{
		"opencode-zen": "opencode_zen",
		"OpenCodeZen":  "opencode_zen",
		"zen":          "opencode_zen",
		"opencode-go":  "opencode_go",
		"OPENCODEGO":   "opencode_go",
	}
	for alias, want := range cases {
		if got := NormalizeProvider(alias); got != want {
			t.Errorf("NormalizeProvider(%q) = %q, want %q", alias, got, want)
		}
	}
}

func TestIsOpenCodeProvider(t *testing.T) {
	for _, id := range []string{"opencode_zen", "opencode_go", "opencode-zen", "zen"} {
		if !IsOpenCodeProvider(id) {
			t.Errorf("IsOpenCodeProvider(%q) = false, want true", id)
		}
	}
	for _, id := range []string{"openai", "anthropic", "custom-openai", ""} {
		if IsOpenCodeProvider(id) {
			t.Errorf("IsOpenCodeProvider(%q) = true, want false", id)
		}
	}
}

// The direct HTTP endpoints take the bare model ID. The namespaced form the
// OpenCode CLI displays must never reach the wire.
func TestNormalizeOpenCodeModelID(t *testing.T) {
	cases := map[string]string{
		"kimi-k3":                 "kimi-k3",
		"opencode-go/kimi-k3":     "kimi-k3",
		"opencode_go/kimi-k3":     "kimi-k3",
		"opencode-zen/gpt-5.4":    "gpt-5.4",
		"OpenCode-Go/deepseek-v4": "deepseek-v4",
		"opencode/glm-5.3":        "glm-5.3",
		"  glm-5.3  ":             "glm-5.3",
		"anthropic/claude-opus-4": "anthropic/claude-opus-4",
		"":                        "",
	}
	for in, want := range cases {
		if got := NormalizeOpenCodeModelID(in); got != want {
			t.Errorf("NormalizeOpenCodeModelID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClassifyOpenCodeModel(t *testing.T) {
	cases := []struct {
		model    string
		protocol OpenCodeProtocol
		known    bool
	}{
		// OpenAI Responses family.
		{"gpt-5.4", OpenCodeResponses, true},
		{"gpt-5.4-codex", OpenCodeResponses, true},
		{"o3-mini", OpenCodeResponses, true},
		{"codex-mini", OpenCodeResponses, true},
		{"opencode-zen/gpt-5.4", OpenCodeResponses, true},
		// Anthropic Messages family.
		{"claude-opus-4-7", OpenCodeMessages, true},
		{"claude-sonnet-4-6", OpenCodeMessages, true},
		{"anthropic/claude-haiku-4-5", OpenCodeMessages, true},
		// OpenAI-compatible chat completions family.
		{"kimi-k3", OpenCodeChatCompletions, true},
		{"deepseek-v4-flash", OpenCodeChatCompletions, true},
		{"glm-5.3", OpenCodeChatCompletions, true},
		{"qwen3-coder-next", OpenCodeChatCompletions, true},
		{"grok-4", OpenCodeChatCompletions, true},
		// Unknown models fall back but are reported as unknown.
		{"totally-new-model-2099", OpenCodeChatCompletions, false},
		{"", OpenCodeChatCompletions, false},
	}
	for _, tc := range cases {
		protocol, known := ClassifyOpenCodeModel(tc.model)
		if protocol != tc.protocol || known != tc.known {
			t.Errorf("ClassifyOpenCodeModel(%q) = (%q, %v), want (%q, %v)",
				tc.model, protocol, known, tc.protocol, tc.known)
		}
	}
}

// Longest-prefix matching means a specific family beats a broader one.
func TestClassifyOpenCodePrefersLongestPrefix(t *testing.T) {
	if protocol, _ := ClassifyOpenCodeModel("openai/gpt-oss-120b"); protocol != OpenCodeResponses {
		t.Errorf("openai/... routed to %q, want %q", protocol, OpenCodeResponses)
	}
}

// Each representative model must produce the concrete provider implementation
// for its protocol — the point of the whole routing table.
func TestOpenCodeRoutingBuildsTheRightProvider(t *testing.T) {
	cases := []struct {
		provider string
		model    string
		want     string
	}{
		{"opencode_zen", "gpt-5.4", "*openairesponses.Provider"},
		{"opencode_zen", "claude-opus-4-7", "*anthropicmessages.Provider"},
		{"opencode_zen", "kimi-k3", "openai_compat"},
		{"opencode_go", "gpt-5.4-codex", "*openairesponses.Provider"},
		{"opencode_go", "deepseek-v4-flash", "openai_compat"},
		{"opencode_go", "glm-5.3", "openai_compat"},
		{"opencode_go", "claude-sonnet-4-6", "*anthropicmessages.Provider"},
	}

	for _, tc := range cases {
		provider, modelID, err := CreateProviderFromConfig(openCodeConfig(tc.provider, tc.model))
		if err != nil {
			t.Errorf("%s/%s: CreateProviderFromConfig error = %v", tc.provider, tc.model, err)
			continue
		}
		if modelID != NormalizeOpenCodeModelID(tc.model) {
			t.Errorf("%s/%s: modelID = %q, want the bare ID %q",
				tc.provider, tc.model, modelID, NormalizeOpenCodeModelID(tc.model))
		}

		switch tc.want {
		case "*openairesponses.Provider":
			if _, ok := provider.(*openairesponses.Provider); !ok {
				t.Errorf("%s/%s: provider is %T, want *openairesponses.Provider",
					tc.provider, tc.model, provider)
			}
		case "*anthropicmessages.Provider":
			if _, ok := provider.(*anthropicmessages.Provider); !ok {
				t.Errorf("%s/%s: provider is %T, want *anthropicmessages.Provider",
					tc.provider, tc.model, provider)
			}
		case "openai_compat":
			if _, ok := provider.(*openairesponses.Provider); ok {
				t.Errorf("%s/%s: routed to Responses, want chat completions", tc.provider, tc.model)
			}
			if _, ok := provider.(*anthropicmessages.Provider); ok {
				t.Errorf("%s/%s: routed to Messages, want chat completions", tc.provider, tc.model)
			}
		}
	}
}

// A namespaced model ID must route correctly and still be sent bare.
func TestOpenCodeNamespacedModelIsStrippedBeforeUse(t *testing.T) {
	_, modelID, err := CreateProviderFromConfig(openCodeConfig("opencode_go", "opencode-go/kimi-k3"))
	if err != nil {
		t.Fatalf("CreateProviderFromConfig error = %v", err)
	}
	if modelID != "kimi-k3" {
		t.Fatalf("modelID = %q, want %q", modelID, "kimi-k3")
	}
}

// An unknown model is still usable rather than rejected, but it must not be
// presented as a confirmed route.
func TestOpenCodeUnknownModelIsUsableAndFlagged(t *testing.T) {
	protocol, known := ClassifyOpenCodeModel("brand-new-model-x")
	if known {
		t.Fatal("an unrecognized model must not be reported as a known route")
	}
	if protocol != OpenCodeChatCompletions {
		t.Fatalf("fallback protocol = %q, want %q", protocol, OpenCodeChatCompletions)
	}

	provider, _, err := CreateProviderFromConfig(openCodeConfig("opencode_zen", "brand-new-model-x"))
	if err != nil {
		t.Fatalf("an unknown model must remain configurable and attemptable: %v", err)
	}
	if provider == nil {
		t.Fatal("provider is nil")
	}
}

func TestOpenCodeRequiresAPIKey(t *testing.T) {
	for _, id := range []string{"opencode_zen", "opencode_go"} {
		if IsEmptyAPIKeyAllowedForProtocol(id) {
			t.Errorf("%s must require an API key", id)
		}
		_, _, err := CreateProviderFromConfig(&config.ModelConfig{
			ModelName: "t", Provider: id, Model: "gpt-5.4",
		})
		if err == nil || !strings.Contains(err.Error(), "api_key is required") {
			t.Errorf("%s without a key: error = %v, want an api_key requirement", id, err)
		}
	}
}

// Nothing on the OpenCode construction path may echo the key, including the
// error raised when the key is missing or the base is unresolvable.
func TestOpenCodeErrorsDoNotLeakAPIKey(t *testing.T) {
	cases := []*config.ModelConfig{
		openCodeConfig("opencode_zen", ""),
		{ModelName: "t", Provider: "opencode_go", Model: "kimi-k3",
			APIKeys: config.SimpleSecureStrings(testAPIKey), APIBase: ""},
		{ModelName: "t", Provider: "opencode_zen", Model: "gpt-5.4",
			APIKeys: config.SimpleSecureStrings(testAPIKey)},
	}
	for _, cfg := range cases {
		_, _, err := CreateProviderFromConfig(cfg)
		if err != nil && strings.Contains(err.Error(), testAPIKey) {
			t.Errorf("error message contains the API key: %v", err)
		}
	}
}

// The Messages provider normalizes its base URL by stripping then re-appending
// "/v1". That round-trip must leave the official OpenCode bases intact, or the
// Anthropic route would post to the wrong host path.
func TestOpenCodeMessagesBaseURLRoundTrip(t *testing.T) {
	cases := map[string]string{
		"https://opencode.ai/zen/v1":     "https://opencode.ai/zen/v1",
		"https://opencode.ai/zen/go/v1":  "https://opencode.ai/zen/go/v1",
		"https://opencode.ai/zen/v1/":    "https://opencode.ai/zen/v1",
		"https://opencode.ai/zen/go/v1/": "https://opencode.ai/zen/go/v1",
	}
	for in, want := range cases {
		if got := commonNormalizeBaseURLForTest(in); got != want {
			t.Errorf("normalized base for %q = %q, want %q", in, got, want)
		}
	}
}

// Guards the invariant the whole milestone rests on: a provider must never be
// offered in the catalog while being unusable at inference time. Every HTTP-API
// provider that can drive a chat model has to construct from a plain
// key-plus-base configuration.
func TestEveryHTTPChatProviderInCatalogIsConstructible(t *testing.T) {
	checked := 0
	for _, option := range ModelProviderOptions() {
		if !option.DefaultModelAllowed || !IsHTTPAPIProtocol(option.ID) {
			continue
		}

		apiBase := option.DefaultAPIBase
		if apiBase == "" {
			apiBase = "https://endpoint.example/v1"
		}
		cfg := &config.ModelConfig{
			ModelName: "t",
			Provider:  option.ID,
			Model:     "test-model",
			APIBase:   apiBase,
			APIKeys:   config.SimpleSecureStrings(testAPIKey),
		}
		if _, _, err := CreateProviderFromConfig(cfg); err != nil {
			t.Errorf("catalog provider %q cannot be constructed: %v", option.ID, err)
		}
		checked++
	}

	// Guards against the assertion silently becoming vacuous if the catalog
	// shape or the filter above ever changes.
	if checked < 30 {
		t.Fatalf("only %d providers were checked; the catalog filter is too narrow to be meaningful", checked)
	}
}

// commonNormalizeBaseURLForTest mirrors what anthropicmessages.NewProviderWithTimeout
// does to the configured base, so the round-trip assertion above exercises the
// real normalization rule rather than a restatement of it.
func commonNormalizeBaseURLForTest(apiBase string) string {
	return providercommonNormalizeBaseURL(apiBase, "https://api.anthropic.com/v1", true)
}
