package providers

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// pocketClawPresets are the provider IDs PocketClaw's simplified "Add Provider"
// flow depends on. Each must be present in the catalog, creatable, and — this
// is the part a catalog-only change would silently break — dispatchable by
// CreateProviderFromConfig.
var pocketClawPresets = []string{
	"openai", "anthropic", "gemini", "deepseek", "openrouter",
	"qwen-portal", "moonshot", "groq", "mistral", "nvidia", "cerebras",
	"xai", "together", "fireworks",
	"ollama", "lmstudio", "custom-openai",
}

func TestPocketClawPresetsAreSupportedAndCreatable(t *testing.T) {
	for _, id := range pocketClawPresets {
		if !IsSupportedModelProvider(id) {
			t.Errorf("provider %q is not in the catalog", id)
		}
		if !IsCreatableModelProvider(id) {
			t.Errorf("provider %q is not creatable", id)
		}
	}
}

// A preset registered in the catalog but missing from the protocol switch in
// CreateProviderFromConfig fails at request time with "unknown protocol". This
// test is the guard against shipping that combination.
func TestPocketClawPresetsResolveToAProvider(t *testing.T) {
	for _, id := range pocketClawPresets {
		option, ok := modelProviderOptionForName(id)
		if !ok {
			t.Fatalf("provider %q missing from catalog", id)
		}

		apiBase := option.DefaultAPIBase
		if apiBase == "" {
			apiBase = "https://example.invalid/v1"
		}
		cfg := &config.ModelConfig{
			ModelName: "t",
			Provider:  id,
			Model:     "test-model",
			APIBase:   apiBase,
			APIKeys:   config.SimpleSecureStrings("test-key-not-a-real-secret"),
		}

		provider, modelID, err := CreateProviderFromConfig(cfg)
		if err != nil {
			t.Errorf("CreateProviderFromConfig(%q) error = %v", id, err)
			continue
		}
		if provider == nil {
			t.Errorf("CreateProviderFromConfig(%q) returned a nil provider", id)
		}
		if modelID != "test-model" {
			t.Errorf("CreateProviderFromConfig(%q) modelID = %q, want %q", id, modelID, "test-model")
		}
	}
}

func TestNewPresetsHaveAccurateMetadata(t *testing.T) {
	want := map[string]struct {
		apiBase  string
		fetch    bool
		category string
	}{
		"xai":           {"https://api.x.ai/v1", true, "cloud"},
		"together":      {"https://api.together.xyz/v1", true, "cloud"},
		"fireworks":     {"https://api.fireworks.ai/inference/v1", true, "cloud"},
		"custom-openai": {"", true, "custom"},
	}

	for id, expected := range want {
		option, ok := modelProviderOptionForName(id)
		if !ok {
			t.Fatalf("provider %q missing from catalog", id)
		}
		if option.DefaultAPIBase != expected.apiBase {
			t.Errorf("%s DefaultAPIBase = %q, want %q", id, option.DefaultAPIBase, expected.apiBase)
		}
		if option.SupportsFetch != expected.fetch {
			t.Errorf("%s SupportsFetch = %v, want %v", id, option.SupportsFetch, expected.fetch)
		}
		if option.Category != expected.category {
			t.Errorf("%s Category = %q, want %q", id, option.Category, expected.category)
		}
	}
}

// Custom OpenAI-compatible endpoints must never inherit OpenAI's base: an empty
// api_base has to surface as a configuration error the user can act on.
func TestCustomOpenAIHasNoDefaultAPIBase(t *testing.T) {
	if got := DefaultAPIBaseForProtocol("custom-openai"); got != "" {
		t.Fatalf("DefaultAPIBaseForProtocol(custom-openai) = %q, want empty", got)
	}
	if base := ResolveAPIBase(&config.ModelConfig{Provider: "custom-openai", Model: "m"}); base != "" {
		t.Fatalf("ResolveAPIBase = %q, want empty", base)
	}
	custom := "https://inference.example.com/v1"
	if base := ResolveAPIBase(&config.ModelConfig{
		Provider: "custom-openai",
		Model:    "m",
		APIBase:  custom,
	}); base != custom {
		t.Fatalf("ResolveAPIBase = %q, want %q", base, custom)
	}
}

func TestGeminiSupportsModelDiscovery(t *testing.T) {
	if !IsModelProviderFetchable("gemini") {
		t.Fatal("gemini must support model discovery")
	}
}

// Keyless local providers must not be forced to carry an API key.
func TestLocalProvidersAllowEmptyAPIKey(t *testing.T) {
	for _, id := range []string{"ollama", "lmstudio", "custom-openai"} {
		if !IsEmptyAPIKeyAllowedForProtocol(id) {
			t.Errorf("provider %q must allow an empty API key", id)
		}
	}
	for _, id := range []string{"openai", "anthropic", "gemini", "xai"} {
		if IsEmptyAPIKeyAllowedForProtocol(id) {
			t.Errorf("cloud provider %q must require an API key", id)
		}
	}
}

func TestUnknownProviderIsRejected(t *testing.T) {
	if IsSupportedModelProvider("definitely-not-a-provider") {
		t.Fatal("unknown provider must not be reported as supported")
	}
	if _, _, err := CreateProviderFromConfig(&config.ModelConfig{
		ModelName: "t",
		Provider:  "definitely-not-a-provider",
		Model:     "m",
		APIKeys:   config.SimpleSecureStrings("k"),
	}); err == nil || !strings.Contains(err.Error(), "unknown protocol") {
		t.Fatalf("error = %v, want an unknown protocol error", err)
	}
}

// Every catalog entry must carry a category so the picker can group it.
func TestEveryCatalogEntryHasACategory(t *testing.T) {
	allowed := map[string]bool{
		"cloud": true, "local": true, "managed": true, "custom": true, "speech": true,
	}
	for _, option := range ModelProviderOptions() {
		if !allowed[option.Category] {
			t.Errorf("provider %q has category %q, which is not a known group", option.ID, option.Category)
		}
	}
}

// Aliases must stay unambiguous, otherwise NormalizeProvider silently routes a
// user's configuration to the wrong provider.
func TestNewAliasesResolveToTheirProvider(t *testing.T) {
	cases := map[string]string{
		"grok":              "xai",
		"x-ai":              "xai",
		"togetherai":        "together",
		"fireworks-ai":      "fireworks",
		"custom":            "custom-openai",
		"openai-compatible": "custom-openai",
	}
	for alias, want := range cases {
		if got := NormalizeProvider(alias); got != want {
			t.Errorf("NormalizeProvider(%q) = %q, want %q", alias, got, want)
		}
	}
}
