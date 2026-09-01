package selfchat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func writeConfigWithSelfNumber(t *testing.T, number string) string {
	t.Helper()
	raw := map[string]any{
		"channel_list": map[string]any{
			ConfigKey: map[string]any{
				"type":     ConfigKey,
				"enabled":  false,
				"settings": map[string]any{"self_number": number},
			},
		},
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConfiguredNumberReadsCanonicalNumber(t *testing.T) {
	t.Setenv(config.EnvConfig, writeConfigWithSelfNumber(t, "+20 101 234 5678"))
	if got := ConfiguredNumber(); got != "+201012345678" {
		t.Errorf("ConfiguredNumber() = %q, want %q", got, "+201012345678")
	}
}

// A number hand-edited into config.json never passed the console's validation,
// so an unusable one must read as "not configured" rather than reach Android.
func TestConfiguredNumberRejectsInvalidStoredNumber(t *testing.T) {
	t.Setenv(config.EnvConfig, writeConfigWithSelfNumber(t, "01012345678"))
	if got := ConfiguredNumber(); got != "" {
		t.Errorf("ConfiguredNumber() = %q, want empty", got)
	}
}

func TestConfiguredNumberIsEmptyWhenDisconnected(t *testing.T) {
	t.Setenv(config.EnvConfig, writeConfigWithSelfNumber(t, ""))
	if got := ConfiguredNumber(); got != "" {
		t.Errorf("ConfiguredNumber() = %q, want empty", got)
	}
}

func TestConfiguredNumberIsEmptyWhenNeverConfigured(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"channel_list":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvConfig, path)
	if got := ConfiguredNumber(); got != "" {
		t.Errorf("ConfiguredNumber() = %q, want empty", got)
	}
}
