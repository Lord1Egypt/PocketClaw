package modelaccess

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

func withoutCredentials(t *testing.T) {
	t.Helper()
	previous := getCredential
	getCredential = func(string) (*auth.AuthCredential, error) { return nil, nil }
	t.Cleanup(func() { getCredential = previous })
}

func modelEntry(name, provider, model, apiKey string) *config.ModelConfig {
	entry := &config.ModelConfig{
		ModelName: name,
		Provider:  provider,
		Model:     model,
		Enabled:   true,
	}
	if apiKey != "" {
		entry.APIKeys = config.SecureStrings{config.NewSecureString(apiKey)}
	}
	return entry
}

// B. The keyless provider templates DefaultConfig seeds into model_list are
//
//	configured by nobody and must never be offered.
//
//	The exception is deliberate and documented: an entry whose availability is
//	a probe rather than a stored secret — a local runtime, or a CLI-backed
//	provider — has no key to check, so filtering it on one would hide a model
//	that works. Those are identified by RequiresRuntimeProbe, not by name.
func TestSeededTemplatesRequiringCredentialsAreNotConfigured(t *testing.T) {
	withoutCredentials(t)

	checked, exempt := 0, 0
	for _, entry := range config.DefaultConfig().ModelList {
		if entry == nil {
			continue
		}
		if RequiresRuntimeProbe(entry) {
			exempt++
			continue
		}
		checked++
		if IsConfigured(entry) {
			t.Errorf("seeded template %q reports configured with no credentials", entry.ModelName)
		}
	}
	if checked == 0 {
		t.Fatal("no credential-requiring templates were checked; this test would prove nothing")
	}
	if exempt == 0 {
		t.Fatal("expected some probe-based templates; the exemption is untested otherwise")
	}
	t.Logf("checked %d credential-requiring templates, %d probe-based exemptions", checked, exempt)
}

// A. An entry with a key is configured.
func TestEntryWithAPIKeyIsConfigured(t *testing.T) {
	withoutCredentials(t)

	if !IsConfigured(modelEntry("Gemini", "gemini", "gemini-2.5-flash", "k-live")) {
		t.Fatal("an entry with an API key must be configured")
	}
	if IsConfigured(modelEntry("Gemini", "gemini", "gemini-2.5-flash", "")) {
		t.Fatal("the same entry without a key must not be")
	}
}

// A local runtime is configured by being reachable, not by holding a secret,
// so it must not be filtered out for lacking one.
func TestLocalRuntimeIsConfiguredWithoutAKey(t *testing.T) {
	withoutCredentials(t)

	local := modelEntry("Local", "ollama", "llama3", "")
	if !IsConfigured(local) {
		t.Fatal("a local runtime with a default base must be offered")
	}
	if !RequiresRuntimeProbe(local) {
		t.Fatal("a local runtime's availability is a probe, not a stored secret")
	}
}

// An api_base that merely mentions localhost in a path is not a local endpoint.
func TestLocalDetectionParsesTheHostRatherThanMatchingText(t *testing.T) {
	remote := modelEntry("Remote", "openai_compat", "some-model", "")
	remote.APIBase = "https://api.example.com/proxy/localhost/v1"
	if hasLocalAPIBase(remote.APIBase) {
		t.Fatal("a remote base containing the word localhost was treated as local")
	}
	if !hasLocalAPIBase("http://127.0.0.1:11434/v1") {
		t.Fatal("a genuinely local base was not recognised")
	}
}

// A stored OAuth credential configures an entry that carries no API key.
func TestStoredOAuthCredentialConfiguresTheEntry(t *testing.T) {
	previous := getCredential
	getCredential = func(string) (*auth.AuthCredential, error) {
		return &auth.AuthCredential{AccessToken: "token-value"}, nil
	}
	t.Cleanup(func() { getCredential = previous })

	entry := modelEntry("Claude", "anthropic", "claude-4", "")
	entry.AuthMethod = "oauth"
	if !IsConfigured(entry) {
		t.Fatal("an entry with a stored OAuth credential must be configured")
	}
}

// A picker must only offer models that can be switched to now. The seeded
// probe-based templates ship disabled, so nothing has established that the
// local runtime behind them exists.
func TestSeededProbeTemplatesAreNotSelectable(t *testing.T) {
	withoutCredentials(t)

	probeSeeds := map[string]bool{
		"llama3": true, "local-model": true, "lmstudio-local": true, "copilot-gpt-5.4": true,
	}
	found := 0
	for _, entry := range config.DefaultConfig().ModelList {
		if entry == nil || !probeSeeds[entry.ModelName] {
			continue
		}
		found++
		if !RequiresRuntimeProbe(entry) {
			t.Errorf("%q is expected to be probe-based", entry.ModelName)
		}
		// The Dashboard still gets to consider it a probe candidate...
		if !IsConfigured(entry) {
			t.Errorf("%q should remain configured enough for the Dashboard to probe", entry.ModelName)
		}
		// ...but a picker must not offer it.
		if IsSelectable(entry) {
			t.Errorf("%q is offered for switching though nothing proved it available", entry.ModelName)
		}
	}
	if found != len(probeSeeds) {
		t.Fatalf("found %d of the %d probe-based seeds", found, len(probeSeeds))
	}
}

// Every seeded template is unselectable on a fresh install: the picker starts
// empty rather than full of things that cannot be chosen.
func TestNoSeededTemplateIsSelectable(t *testing.T) {
	withoutCredentials(t)

	for _, entry := range config.DefaultConfig().ModelList {
		if entry != nil && IsSelectable(entry) {
			t.Errorf("seeded template %q is selectable on a fresh install", entry.ModelName)
		}
	}
}

// A model the user materialized is enabled, and that is the evidence a picker
// acts on — including for a local runtime.
func TestMaterializedModelsAreSelectable(t *testing.T) {
	withoutCredentials(t)

	keyed := modelEntry("Gemini", "gemini", "gemini-2.5-flash", "k-live")
	keyed.Enabled = true
	if !IsSelectable(keyed) {
		t.Fatal("a configured, enabled model must be selectable")
	}

	local := modelEntry("My Ollama", "ollama", "llama3", "")
	local.Enabled = true
	if !RequiresRuntimeProbe(local) {
		t.Fatal("this entry should be probe-based")
	}
	if !IsSelectable(local) {
		t.Fatal("a deliberately materialized local model must remain selectable")
	}
}

// Enabling is not enough on its own: an entry with no way to authenticate stays
// out regardless.
func TestEnabledButUnconfiguredIsNotSelectable(t *testing.T) {
	withoutCredentials(t)

	entry := modelEntry("Groq", "groq", "llama-3.3", "")
	entry.Enabled = true
	if IsSelectable(entry) {
		t.Fatal("an enabled entry with no credentials must not be selectable")
	}
}

// The Dashboard's question is unchanged by the picker's stricter one.
func TestSelectableDoesNotChangeConfiguredSemantics(t *testing.T) {
	withoutCredentials(t)

	for _, entry := range config.DefaultConfig().ModelList {
		if entry == nil {
			continue
		}
		if IsSelectable(entry) && !IsConfigured(entry) {
			t.Errorf("%q is selectable but not configured; selectable must be the stricter rule",
				entry.ModelName)
		}
	}
}
