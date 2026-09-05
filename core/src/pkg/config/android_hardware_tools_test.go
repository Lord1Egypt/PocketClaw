package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// androidHardwareToolEnv is the override PocketClaw Android's managed Gateway is
// launched with. It is asserted here as data so the Kotlin side and the Go side
// cannot drift apart silently — the Android guard checks the same names.
var androidHardwareToolEnv = map[string]string{
	"PICOCLAW_TOOLS_I2C_ENABLED":    "false",
	"PICOCLAW_TOOLS_SPI_ENABLED":    "false",
	"PICOCLAW_TOOLS_SERIAL_ENABLED": "false",
}

// D. A config that claims the host-bus tools are enabled must not be able to
//
//	turn them on under PocketClaw Android. Registration in agent_init.go is
//	gated on IsToolEnabled, so a config that loads as disabled is what keeps
//	them out of ToolRegistry and therefore out of ToProviderDefs.
//
// An old install, a config imported from a Linux machine, or a hand-edited file
// are all the same case: the environment is applied after the file.
func TestAndroidEnvForcesHardwareToolsOffOverAnEnablingConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tools.I2C.Enabled = true
	cfg.Tools.SPI.Enabled = true
	cfg.Tools.Serial.Enabled = true

	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Without the override the file wins, which is the behaviour every other
	// platform keeps.
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	for name, enabled := range map[string]bool{
		"i2c":    loaded.Tools.IsToolEnabled("i2c"),
		"spi":    loaded.Tools.IsToolEnabled("spi"),
		"serial": loaded.Tools.IsToolEnabled("serial"),
	} {
		if !enabled {
			t.Fatalf("%s should be enabled from the file when no override is set; "+
				"this test would otherwise prove nothing", name)
		}
	}

	for key, value := range androidHardwareToolEnv {
		t.Setenv(key, value)
	}

	androidLoaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig with the Android override: %v", err)
	}
	for _, name := range []string{"i2c", "spi", "serial"} {
		if androidLoaded.Tools.IsToolEnabled(name) {
			t.Errorf("%s is still enabled under the Android override; it would be "+
				"registered into ToolRegistry and reach the model tool definitions", name)
		}
	}
}

// The override must not disturb anything else the user configured.
func TestAndroidHardwareOverrideLeavesOtherToolsAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	encoded, err := json.Marshal(DefaultConfig())
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	before, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	for key, value := range androidHardwareToolEnv {
		t.Setenv(key, value)
	}
	after, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig with the Android override: %v", err)
	}

	for _, name := range []string{
		"web", "web_fetch", "exec", "message", "read_file", "list_dir", "python",
		"runtime", "skills", "find_skills", "install_skill", "subagent", "spawn",
		"cron", "load_image", "edit_file", "append_file", "media_cleanup",
	} {
		if before.Tools.IsToolEnabled(name) != after.Tools.IsToolEnabled(name) {
			t.Errorf("the Android hardware override changed %q", name)
		}
	}
}
