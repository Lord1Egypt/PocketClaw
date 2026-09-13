package api

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// opencodeGoConfig is the shape the owner reported the credential defect
// against: one configured provider preset with one model and one key.
func opencodeGoConfig(t *testing.T, key string) *config.Config {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "opencode-go",
		Provider:  "opencode",
		Model:     "deepseek-v4.1-flash",
		APIBase:   "https://opencode.ai/zen/v1",
		Enabled:   true,
	}}
	cfg.ModelList[0].SetAPIKey(key)
	cfg.Agents.Defaults.ModelName = "opencode-go"
	return cfg
}

func TestConfigSignatureChangesWhenAPIKeyIsRotated(t *testing.T) {
	cfg := opencodeGoConfig(t, "key-before")
	before := computeConfigSignature(cfg)

	cfg.ModelList[0].SetAPIKey("key-after")
	after := computeConfigSignature(cfg)

	if before == after {
		t.Fatal("config signature must change when a model API key is rotated, " +
			"or gateway_restart_required stays false and the running gateway keeps the old credential")
	}
}

func TestConfigSignatureChangesWhenAPIBaseIsEdited(t *testing.T) {
	cfg := opencodeGoConfig(t, "key")
	before := computeConfigSignature(cfg)

	cfg.ModelList[0].APIBase = "https://opencode.ai/zen/v2"
	if computeConfigSignature(cfg) == before {
		t.Fatal("config signature must change when a model api_base is edited")
	}
}

func TestConfigSignatureChangesWhenCustomHeadersChange(t *testing.T) {
	cfg := opencodeGoConfig(t, "key")
	before := computeConfigSignature(cfg)

	cfg.ModelList[0].CustomHeaders = map[string]string{"x-opencode-session": "s1"}
	added := computeConfigSignature(cfg)
	if added == before {
		t.Fatal("config signature must change when custom headers are added")
	}

	cfg.ModelList[0].CustomHeaders = map[string]string{"x-opencode-session": "s2"}
	if computeConfigSignature(cfg) == added {
		t.Fatal("config signature must change when a custom header value changes")
	}
}

// A non-default model's credential is material the booted gateway holds too.
// Narrowing coverage to referenced entries would leave the same silent
// staleness for every sibling model of a provider.
func TestConfigSignatureChangesWhenNonDefaultSiblingKeyIsRotated(t *testing.T) {
	cfg := opencodeGoConfig(t, "shared-key")
	sibling := &config.ModelConfig{
		ModelName: "opencode-go-coder",
		Provider:  "opencode",
		Model:     "qwen3-coder",
		APIBase:   "https://opencode.ai/zen/v1",
		Enabled:   true,
	}
	sibling.SetAPIKey("shared-key")
	cfg.ModelList = append(cfg.ModelList, sibling)

	before := computeConfigSignature(cfg)
	cfg.ModelList[1].SetAPIKey("rotated-key")

	if computeConfigSignature(cfg) == before {
		t.Fatal("config signature must change when a non-default sibling model's key is rotated")
	}
}

func TestConfigSignatureChangesWhenModelIsDisabled(t *testing.T) {
	cfg := opencodeGoConfig(t, "key")
	before := computeConfigSignature(cfg)

	cfg.ModelList[0].Enabled = false
	if computeConfigSignature(cfg) == before {
		t.Fatal("config signature must change when a model is disabled")
	}
}

// An unchanged configuration must produce an unchanged signature, or every
// status poll would report a restart as required and the gateway would be
// restarted on a schedule.
func TestConfigSignatureIsStableForUnchangedCredentials(t *testing.T) {
	cfg := opencodeGoConfig(t, "key")
	cfg.ModelList[0].CustomHeaders = map[string]string{
		"x-opencode-session": "s1",
		"x-another":          "v",
	}

	first := computeConfigSignature(cfg)
	for i := 0; i < 20; i++ {
		if computeConfigSignature(cfg) != first {
			t.Fatal("config signature must be stable across repeated computation of one config")
		}
	}
}

// The signature is compared in process and never returned to a client, but it
// is the kind of value that reaches a debug log. A secret does not need to be
// in a comparison value.
func TestConfigSignatureDoesNotCarryTheRawAPIKey(t *testing.T) {
	const secret = "sk-live-do-not-leak-me"
	cfg := opencodeGoConfig(t, secret)
	cfg.ModelList[0].CustomHeaders = map[string]string{"authorization": "Bearer " + secret}

	signature := computeConfigSignature(cfg)
	if strings.Contains(signature, secret) {
		t.Fatal("config signature must not contain a raw API key or header credential")
	}
}

func TestAPIKeysDigestCoversEveryKeyAndTheirOrder(t *testing.T) {
	first := &config.ModelConfig{}
	first.APIKeys = config.SimpleSecureStrings("a", "b")
	second := &config.ModelConfig{}
	second.APIKeys = config.SimpleSecureStrings("a", "c")
	reordered := &config.ModelConfig{}
	reordered.APIKeys = config.SimpleSecureStrings("b", "a")

	if apiKeysDigest(first) == apiKeysDigest(second) {
		t.Fatal("digest must cover every key, not only the first")
	}
	if apiKeysDigest(first) == apiKeysDigest(reordered) {
		t.Fatal("digest must cover key order, which decides which key is tried first")
	}
	if apiKeysDigest(&config.ModelConfig{}) != "nokey" {
		t.Fatal("a model with no key must produce the stable no-key marker")
	}
}
