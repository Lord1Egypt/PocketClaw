package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// PocketClaw Android enables hot reload by exporting
// PICOCLAW_GATEWAY_HOT_RELOAD to the managed Gateway rather than by writing
// gateway.hot_reload into config.json. That only works if the environment is
// applied when the config is loaded, and applied after the file — otherwise the
// host boundary would be silently ineffective.
func TestGatewayHotReloadEnvOverridesTheLoadedConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		inFile   bool
		envValue string
		want     bool
	}{
		{name: "off in file, no env", inFile: false, envValue: "", want: false},
		{name: "off in file, env on", inFile: false, envValue: "true", want: true},
		{name: "on in file, no env", inFile: true, envValue: "", want: true},
		{name: "on in file, env off", inFile: true, envValue: "false", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Gateway.HotReload = test.inFile

			encoded, err := json.Marshal(cfg)
			if err != nil {
				t.Fatalf("marshal config: %v", err)
			}
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, encoded, 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}

			if test.envValue == "" {
				t.Setenv("PICOCLAW_GATEWAY_HOT_RELOAD", "")
				os.Unsetenv("PICOCLAW_GATEWAY_HOT_RELOAD")
			} else {
				t.Setenv("PICOCLAW_GATEWAY_HOT_RELOAD", test.envValue)
			}

			loaded, err := LoadConfig(path)
			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}
			if loaded.Gateway.HotReload != test.want {
				t.Fatalf("hot_reload = %v, want %v (file=%v env=%q)",
					loaded.Gateway.HotReload, test.want, test.inFile, test.envValue)
			}
		})
	}
}
