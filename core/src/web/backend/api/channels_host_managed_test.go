package api

import (
	"encoding/json"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// picoConfigWithTokenQuery returns the realtime channel's config response with
// allow_token_query explicitly enabled in the stored settings.
func picoConfigWithTokenQuery(t *testing.T) map[string]any {
	t.Helper()

	cfg := config.DefaultConfig()
	bc := cfg.Channels.Get(config.ChannelPico)
	if bc == nil {
		t.Fatal("the realtime channel is missing from the default config")
	}
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	settings, ok := decoded.(*config.PicoSettings)
	if !ok {
		t.Fatalf("unexpected settings type %T", decoded)
	}
	settings.AllowTokenQuery = true
	encoded, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	bc.Settings = encoded

	item, found := findChannelCatalogItem("pico")
	if !found {
		t.Fatal("pico is not in the channel catalog")
	}

	resp := buildChannelConfigResponse(cfg, item)
	raw, err := json.Marshal(resp.Config)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	return out
}

// Both halves of the contract, because getting one right and the other wrong is
// the actual risk here.
//
// The runtime refuses query-string authentication for a host-supplied
// credential no matter what the config says, so offering the toggle in that
// case would present a choice that does not exist. Hiding it in the frontend
// instead would have removed the capability from every deployment, including
// self-managed ones where a browser client genuinely cannot set a header.
func TestAllowTokenQueryIsHiddenOnlyWhenTheCredentialIsHostManaged(t *testing.T) {
	t.Run("hidden when the host supplies the credential", func(t *testing.T) {
		t.Setenv(config.EnvChannelsPicoToken, "host-managed-credential-0123456789")

		settings := picoConfigWithTokenQuery(t)
		if _, present := settings["allow_token_query"]; present {
			t.Error("the toggle is offered for a credential the runtime will never accept in a URL")
		}
		// Hiding one field must not empty the form.
		if len(settings) == 0 {
			t.Error("the realtime channel config came back empty")
		}
	})

	t.Run("offered to a self-managed deployment", func(t *testing.T) {
		t.Setenv(config.EnvChannelsPicoToken, "")

		settings := picoConfigWithTokenQuery(t)
		value, present := settings["allow_token_query"]
		if !present {
			t.Fatal("the capability was removed from deployments that legitimately have it")
		}
		if enabled, _ := value.(bool); !enabled {
			t.Errorf("allow_token_query = %v, want the stored value to survive", value)
		}
	})

	t.Run("other channels are untouched", func(t *testing.T) {
		t.Setenv(config.EnvChannelsPicoToken, "host-managed-credential-0123456789")

		settings := map[string]any{"allow_token_query": true}
		hideHostManagedChannelFields(settings, "telegram")
		if _, present := settings["allow_token_query"]; !present {
			t.Error("a non-realtime channel lost a field to the realtime rule")
		}
	})
}
