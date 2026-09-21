package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// The invariant: change detection may depend on secret values, but the signature
// material the process retains must not expose them.
//
// `gateway.bootConfigSignature` is long-lived package state. Before this was
// fixed it embedded the marshalled `tools.web` subtree and every channel's
// settings verbatim, so a web-search API key, a credential-bearing proxy URL and
// a Telegram bot token were all sitting in it in the clear.

// secretBearingConfig populates one credential on every surface that reaches the
// signature, each with a distinctive marker.
func secretBearingConfig(t *testing.T) (*config.Config, map[string]string) {
	t.Helper()
	markers := map[string]string{}

	cfg := config.DefaultConfig()

	cfg.Tools.Web.Enabled = true
	cfg.Tools.Web.Brave.Enabled = true
	cfg.Tools.Web.Brave.SetAPIKeys([]string{"MARKERBRAVEKEY"})
	markers["brave web-search key"] = "MARKERBRAVEKEY"
	cfg.Tools.Web.Tavily.SetAPIKeys([]string{"MARKERTAVILYKEY"})
	markers["tavily web-search key"] = "MARKERTAVILYKEY"
	cfg.Tools.Web.Kagi.SetAPIKeys([]string{"MARKERKAGIKEY"})
	markers["kagi web-search key"] = "MARKERKAGIKEY"
	cfg.Tools.Web.Perplexity.SetAPIKey("MARKERPPLXKEY")
	markers["perplexity web-search key"] = "MARKERPPLXKEY"
	// Userinfo in a proxy URL is a credential in a plain string field, which no
	// amount of SecureString handling would have covered.
	cfg.Tools.Web.Proxy = "http://user:MARKERPROXYPASS@proxy.invalid:8080"
	markers["proxy URL password"] = "MARKERPROXYPASS"

	cfg.Channels["telegram"] = &config.Channel{
		Type:      config.ChannelTelegram,
		Enabled:   true,
		AllowFrom: config.FlexibleStringSlice{"123456"},
		Settings:  config.RawNode(json.RawMessage(`{"token":"MARKERTELEGRAMTOKEN"}`)),
	}
	markers["telegram bot token"] = "MARKERTELEGRAMTOKEN"

	cfg.Channels["slack"] = &config.Channel{
		Type:     "slack",
		Enabled:  true,
		Settings: config.RawNode(json.RawMessage(`{"bot_token":"MARKERSLACKBOT","app_token":"MARKERSLACKAPP"}`)),
	}
	markers["slack bot token"] = "MARKERSLACKBOT"
	markers["slack app token"] = "MARKERSLACKAPP"

	cfg.Channels["matrix"] = &config.Channel{
		Type:     "matrix",
		Enabled:  true,
		Settings: config.RawNode(json.RawMessage(`{"access_token":"MARKERMATRIXTOKEN","crypto_passphrase":"MARKERMATRIXPASS"}`)),
	}
	markers["matrix access token"] = "MARKERMATRIXTOKEN"
	markers["matrix crypto passphrase"] = "MARKERMATRIXPASS"

	// A settings subtree that cannot be decoded takes the raw-JSON fallback,
	// which used to be dumped verbatim.
	cfg.Channels["unknown-kind"] = &config.Channel{
		Type:     "not-a-real-channel-type",
		Enabled:  true,
		Settings: config.RawNode(json.RawMessage(`{"secret":"MARKERRAWFALLBACK"}`)),
	}
	markers["raw-fallback channel secret"] = "MARKERRAWFALLBACK"

	model := &config.ModelConfig{
		ModelName: "m", Provider: "openai", Model: "gpt-4o", Enabled: true,
		CustomHeaders: map[string]string{"authorization": "Bearer MARKERHEADERSECRET"},
	}
	model.SetAPIKey("MARKERMODELKEY")
	cfg.ModelList = []*config.ModelConfig{model}
	cfg.Agents.Defaults.ModelName = "m"
	markers["model API key"] = "MARKERMODELKEY"
	markers["model custom header credential"] = "MARKERHEADERSECRET"

	return cfg, markers
}

func TestConfigSignatureExposesNoSecretValue(t *testing.T) {
	cfg, markers := secretBearingConfig(t)

	signature := computeConfigSignature(cfg)

	for description, marker := range markers {
		if strings.Contains(signature, marker) {
			t.Errorf("signature exposes the %s in the clear", description)
		}
	}
}

// Digesting must not cost the sensitivity the signature exists for. Each of
// these is the reason the plaintext was there in the first place.
func TestConfigSignatureStillChangesWhenASecretChanges(t *testing.T) {
	cases := []struct {
		name   string
		change func(*config.Config)
	}{
		{"web-search key", func(c *config.Config) {
			c.Tools.Web.Brave.SetAPIKeys([]string{"ROTATED"})
		}},
		{"proxy URL", func(c *config.Config) {
			c.Tools.Web.Proxy = "http://user:ROTATED@proxy.invalid:8080"
		}},
		{"telegram token", func(c *config.Config) {
			c.Channels["telegram"].Settings = config.RawNode(
				json.RawMessage(`{"token":"ROTATED"}`))
		}},
		{"slack bot token", func(c *config.Config) {
			c.Channels["slack"].Settings = config.RawNode(
				json.RawMessage(`{"bot_token":"ROTATED","app_token":"MARKERSLACKAPP"}`))
		}},
		{"raw-fallback channel secret", func(c *config.Config) {
			c.Channels["unknown-kind"].Settings = config.RawNode(
				json.RawMessage(`{"secret":"ROTATED"}`))
		}},
		{"model API key", func(c *config.Config) {
			c.ModelList[0].APIKeys = nil
			c.ModelList[0].SetAPIKey("ROTATED")
		}},
		{"model custom header", func(c *config.Config) {
			c.ModelList[0].CustomHeaders = map[string]string{"authorization": "Bearer ROTATED"}
		}},
		{"channel enabled flag", func(c *config.Config) {
			c.Channels["slack"].Enabled = false
		}},
		{"channel owner allowlist", func(c *config.Config) {
			c.Channels["telegram"].AllowFrom = config.FlexibleStringSlice{"999"}
		}},
	}

	// The change is applied to a separately built config, before anything has
	// computed a signature from it. `Channel.GetDecoded` decodes lazily and
	// caches into `extend`, so mutating raw `Settings` after a decode reads back
	// the stale typed value -- a documented property of Channel, not something
	// the signature controls. Production is unaffected because
	// handleGatewayStatus calls config.LoadConfig on every poll, so each
	// comparison is against a freshly decoded config. Comparing two fresh
	// configs is what that actually looks like.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseline, _ := secretBearingConfig(t)
			before := computeConfigSignature(baseline)

			changed, _ := secretBearingConfig(t)
			tc.change(changed)

			if computeConfigSignature(changed) == before {
				t.Fatalf("signature must change when the %s changes, or the "+
					"gateway keeps running on the old value", tc.name)
			}
		})
	}
}

func TestConfigSignatureIsStableForAnUnchangedSecretBearingConfig(t *testing.T) {
	cfg, _ := secretBearingConfig(t)

	first := computeConfigSignature(cfg)
	for i := 0; i < 20; i++ {
		if computeConfigSignature(cfg) != first {
			t.Fatal("signature must be stable across repeated computation, or every " +
				"status poll would report a restart as required")
		}
	}
}

// Length-prefixing each part, so ("ab","c") and ("a","bc") cannot collide.
func TestSignatureDigestIsUnambiguousAcrossPartBoundaries(t *testing.T) {
	if signatureDigest("ab", "c") == signatureDigest("a", "bc") {
		t.Fatal("digest parts must be length-prefixed: a shifted boundary must not collide")
	}
	if signatureDigest("x") == signatureDigest("y") {
		t.Fatal("different material must produce different digests")
	}
	if signatureDigest("x") != signatureDigest("x") {
		t.Fatal("the digest must be deterministic")
	}
	if got := len(signatureDigest("x")); got != signatureDigestHexLength {
		t.Fatalf("digest length = %d, want %d", got, signatureDigestHexLength)
	}
}
