package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A config in the shape the console actually writes: the managed channel among
// others, with settings the migration knows nothing about and a top-level key
// the Config struct does not model at all.
const legacyConfigJSON = `{
  "version": 3,
  "unknown_future_top_level": {
    "kept": true
  },
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "settings": {
        "owner_user_id": 4242
      }
    },
    "pico": {
      "enabled": true,
      "type": "pico",
      "allow_from": [
        "pico-user"
      ],
      "settings": {
        "ping_interval": 30,
        "unknown_future_setting": "kept"
      }
    },
    "pico_client": {
      "enabled": false,
      "type": "pico_client",
      "settings": {
        "url": "wss://example.invalid/ws"
      }
    }
  },
  "agents": {
    "defaults": {
      "workspace": "~/workspace"
    }
  }
}
`

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readObject(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("migrated config is not valid JSON: %v", err)
	}
	return out
}

func channelsOf(t *testing.T, path string) map[string]any {
	t.Helper()
	channels, ok := readObject(t, path)["channel_list"].(map[string]any)
	if !ok {
		t.Fatal("no channels object")
	}
	return channels
}

func TestLegacyChannelsMigrateToCanonicalNames(t *testing.T) {
	path := writeConfig(t, t.TempDir(), legacyConfigJSON)

	changed, err := migrateChannelIdentities(path)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if !changed {
		t.Fatal("a legacy config must report as migrated")
	}

	channels := channelsOf(t, path)
	for _, legacy := range []string{"pico", "pico_client"} {
		if _, present := channels[legacy]; present {
			t.Errorf("the legacy channel key %q survived the migration", legacy)
		}
	}

	managed, ok := channels["pocketclaw"].(map[string]any)
	if !ok {
		t.Fatal("the managed channel is not under its canonical key")
	}
	if managed["type"] != "pocketclaw" {
		t.Errorf("type = %v, want pocketclaw", managed["type"])
	}
	client, ok := channels["pocketclaw_client"].(map[string]any)
	if !ok {
		t.Fatal("the client channel is not under its canonical key")
	}
	if client["type"] != "pocketclaw_client" {
		t.Errorf("client type = %v, want pocketclaw_client", client["type"])
	}
}

func TestOwnerPrincipalMigratesInsideTheChannel(t *testing.T) {
	path := writeConfig(t, t.TempDir(), legacyConfigJSON)

	if _, err := migrateChannelIdentities(path); err != nil {
		t.Fatal(err)
	}

	managed := channelsOf(t, path)["pocketclaw"].(map[string]any)
	allow, ok := managed["allow_from"].([]any)
	if !ok || len(allow) != 1 {
		t.Fatalf("allow_from = %#v, want exactly one owner", managed["allow_from"])
	}
	if allow[0] != PocketClawOwnerPrincipal {
		t.Errorf("owner = %v, want %s", allow[0], PocketClawOwnerPrincipal)
	}
}

// Everything the migration was not asked to touch has to come out the other
// side unchanged — including fields the Config struct does not model, which a
// load-and-save migration would have silently dropped.
func TestMigrationPreservesUnrelatedConfiguration(t *testing.T) {
	path := writeConfig(t, t.TempDir(), legacyConfigJSON)

	if _, err := migrateChannelIdentities(path); err != nil {
		t.Fatal(err)
	}

	root := readObject(t, path)
	unknown, ok := root["unknown_future_top_level"].(map[string]any)
	if !ok || unknown["kept"] != true {
		t.Errorf("an unmodelled top-level key was lost: %#v", root["unknown_future_top_level"])
	}
	if root["version"] != json.Number("3") && root["version"] != float64(3) {
		t.Errorf("version = %#v, want 3", root["version"])
	}

	channels := channelsOf(t, path)
	telegram, ok := channels["telegram"].(map[string]any)
	if !ok {
		t.Fatal("an unrelated channel was lost")
	}
	settings := telegram["settings"].(map[string]any)
	if settings["owner_user_id"] != float64(4242) {
		t.Errorf("unrelated channel settings changed: %#v", settings)
	}

	managed := channels["pocketclaw"].(map[string]any)
	managedSettings := managed["settings"].(map[string]any)
	if managedSettings["unknown_future_setting"] != "kept" {
		t.Errorf("an unknown setting inside the migrated channel was lost: %#v", managedSettings)
	}
	if managedSettings["ping_interval"] != float64(30) {
		t.Errorf("a known setting inside the migrated channel changed: %#v", managedSettings)
	}
}

// Key order and the untouched bytes stay put, so the migration shows up in the
// user's file as a rename and not as a reformat.
func TestMigrationRewritesOnlyTheChannelIdentity(t *testing.T) {
	path := writeConfig(t, t.TempDir(), legacyConfigJSON)

	if _, err := migrateChannelIdentities(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	before := strings.Split(legacyConfigJSON, "\n")
	after := strings.Split(string(raw), "\n")
	if len(before) != len(after) {
		t.Fatalf("line count changed: %d -> %d\n%s", len(before), len(after), raw)
	}
	var differing []int
	for i := range before {
		if before[i] != after[i] {
			differing = append(differing, i+1)
		}
	}
	// Exactly the four identity lines: two channel keys and two type values.
	if len(differing) != 5 {
		t.Errorf("expected only the identity lines to change, got lines %v:\n%s", differing, raw)
	}
}

func TestCanonicalConfigIsLeftAlone(t *testing.T) {
	canonical := strings.NewReplacer(
		`"pico"`, `"pocketclaw"`,
		`"pico_client"`, `"pocketclaw_client"`,
		`"pico-user"`, `"pocketclaw-user"`,
	).Replace(legacyConfigJSON)
	path := writeConfig(t, t.TempDir(), canonical)

	changed, err := migrateChannelIdentities(path)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("an already-canonical config must not be rewritten")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != canonical {
		t.Error("an already-canonical config was modified")
	}
}

func TestEquivalentLegacyAndCanonicalCollapseToCanonical(t *testing.T) {
	both := `{
  "channel_list": {
    "pico": {
      "enabled": true,
      "type": "pico",
      "settings": {"ping_interval": 30}
    },
    "pocketclaw": {
      "enabled": true,
      "type": "pocketclaw",
      "settings": {"ping_interval": 30}
    }
  }
}
`
	path := writeConfig(t, t.TempDir(), both)

	changed, err := migrateChannelIdentities(path)
	if err != nil {
		t.Fatalf("two identical definitions are not a conflict: %v", err)
	}
	if !changed {
		t.Fatal("the redundant legacy definition should have been removed")
	}

	channels := channelsOf(t, path)
	if _, present := channels["pico"]; present {
		t.Error("the redundant legacy definition survived")
	}
	if _, present := channels["pocketclaw"]; !present {
		t.Error("the canonical definition was lost")
	}
}

func TestConflictingDefinitionsFailClosed(t *testing.T) {
	conflicting := `{
  "channel_list": {
    "pico": {
      "enabled": true,
      "type": "pico",
      "settings": {"ping_interval": 30}
    },
    "pocketclaw": {
      "enabled": false,
      "type": "pocketclaw",
      "settings": {"ping_interval": 90}
    }
  }
}
`
	dir := t.TempDir()
	path := writeConfig(t, dir, conflicting)

	_, err := migrateChannelIdentities(path)
	if err == nil {
		t.Fatal("two different channel definitions must not be resolved by guessing")
	}
	if !errors.Is(err, ErrChannelMigrationConflict) {
		t.Errorf("error = %v, want a migration conflict", err)
	}

	// Neither definition is touched. They may carry different tokens, and
	// picking one silently is how a user loses the working one.
	raw, _ := os.ReadFile(path)
	if string(raw) != conflicting {
		t.Error("a conflicting config must be left exactly as found")
	}

	// And the conflict is fatal to the load, rather than starting on half of it.
	if _, err := LoadConfig(path); !errors.Is(err, ErrChannelMigrationConflict) {
		t.Errorf("LoadConfig error = %v, want the migration conflict", err)
	}
}

func TestSecurityFileChannelKeyMigratesWithTheConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, legacyConfigJSON)
	security := `channel_list:
  pico:
    settings:
      token: SECRET-TOKEN-VALUE
  telegram:
    settings:
      token: OTHER-TOKEN
`
	if err := os.WriteFile(securityPath(path), []byte(security), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := migrateChannelIdentities(path); err != nil {
		t.Fatal(err)
	}

	migrated, err := os.ReadFile(securityPath(path))
	if err != nil {
		t.Fatal(err)
	}
	body := string(migrated)
	if !strings.Contains(body, "pocketclaw:") {
		t.Errorf("the credential file did not follow the channel rename:\n%s", body)
	}
	if strings.Contains(body, "\n  pico:") {
		t.Errorf("the legacy channel key survived in the credential file:\n%s", body)
	}
	// The token itself must arrive, or the channel silently loses its
	// credential while looking migrated.
	if !strings.Contains(body, "SECRET-TOKEN-VALUE") {
		t.Errorf("the channel token was lost:\n%s", body)
	}
	if !strings.Contains(body, "OTHER-TOKEN") {
		t.Errorf("an unrelated channel's token was lost:\n%s", body)
	}
}

func TestMigrationIsIdempotent(t *testing.T) {
	path := writeConfig(t, t.TempDir(), legacyConfigJSON)

	if _, err := migrateChannelIdentities(path); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)

	changed, err := migrateChannelIdentities(path)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("a second run must find nothing to do")
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Error("a second run rewrote the file")
	}
}

// A legacy installation has to come up on its first start after the upgrade,
// with its channel, its owner and its settings intact.
func TestLoadConfigNormalizesALegacyInstallation(t *testing.T) {
	// Without the unmodelled top-level key: LoadConfig rejects unknown fields,
	// which is the pre-existing config contract and precisely why the migration
	// edits the serialized document rather than round-tripping it through the
	// typed struct.
	loadable := strings.Replace(legacyConfigJSON, `  "unknown_future_top_level": {
    "kept": true
  },
`, "", 1)
	path := writeConfig(t, t.TempDir(), loadable)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("a legacy installation must still load: %v", err)
	}

	channel := cfg.Channels.GetByType(ChannelPocketClaw)
	if channel == nil {
		t.Fatal("the managed channel is not reachable under its canonical type")
	}
	if len(channel.AllowFrom) != 1 || channel.AllowFrom[0] != PocketClawOwnerPrincipal {
		t.Errorf("allow_from = %v, want [%s]", channel.AllowFrom, PocketClawOwnerPrincipal)
	}
	if cfg.Channels.GetByType(LegacyChannelPocketClaw) != nil {
		t.Error("the legacy channel type is still reachable after migration")
	}
}
