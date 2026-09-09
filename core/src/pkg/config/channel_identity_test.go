package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/canonicalenv"
)

func TestCanonicalChannelIdentities(t *testing.T) {
	cases := []struct{ name, got, want string }{
		{"managed channel", ChannelPocketClaw, "pocketclaw"},
		{"client channel", ChannelPocketClawClient, "pocketclaw_client"},
		{"owner principal", PocketClawOwnerPrincipal, "pocketclaw-user"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
		if strings.Contains(strings.ToLower(tc.got), "pico") {
			t.Errorf("%s still carries the legacy namespace: %q", tc.name, tc.got)
		}
	}
}

func TestDefaultConfigUsesTheCanonicalChannel(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Channels.GetByType(ChannelPocketClaw) == nil {
		t.Error("a fresh install has no managed channel under its canonical type")
	}
	if cfg.Channels.GetByType(LegacyChannelPocketClaw) != nil {
		t.Error("a fresh install created the legacy channel")
	}
	if _, present := cfg.Channels[LegacyChannelPocketClaw]; present {
		t.Error("a fresh install created the legacy channel key")
	}
	if _, present := cfg.Channels[ChannelPocketClaw]; !present {
		t.Error("a fresh install has no channel under the canonical key")
	}
}

// No writer may put a legacy identity on disk or on the wire. The legacy names
// exist to be read, once, and the places allowed to name them are listed here
// rather than left to a reviewer to recognise.
func TestNoProductionWriterEmitsALegacyChannelIdentity(t *testing.T) {
	// Matched as quoted literals, so the comments that explain why these names
	// were retired are not themselves violations.
	legacy := regexp.MustCompile(`"(pico|pico_client|pico-user)"`)

	// The log component is deliberately still the legacy word. It travels
	// through the same redaction rules as the /pico/ route and the caller
	// filename, and those move as one piece in the route/sanitizer phase;
	// renaming the component on its own would leave a build whose log lines
	// the sanitizer no longer recognises. Only this call shape is exempt, so a
	// legacy literal anywhere else in the same file is still a failure.
	loggerComponent := regexp.MustCompile(
		`logger\.[A-Za-z]+\("(pico|pico_client)",`)

	// Each of these is a reader or a route, both documented as such.
	allowed := map[string]string{
		// The one table of legacy identities, and the migration that consumes it.
		"pkg/config/channel_legacy.go":    "the legacy identity table",
		"pkg/config/channel_migration.go": "the migration that reads them",
		// Redaction has to recognise what older builds wrote.
		"pkg/logger/logger.go": "log sanitizer compatibility",
		// Session discovery over pre-migration scope metadata.
		"web/backend/api/session.go": "legacy session key discovery",
	}

	root := moduleRootForConfig(t)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case "vendor", "build", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		relative = filepath.ToSlash(relative)
		if _, ok := allowed[relative]; ok {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		scanned := loggerComponent.ReplaceAll(raw, []byte("logger.C(<deferred>,"))
		if match := legacy.Find(scanned); match != nil {
			t.Errorf("%s contains the legacy channel identity %s; it belongs in "+
				"the migration owner, not in a writer", relative, match)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

// The channel package moved with the identity, and nothing imports the old path.
func TestTheChannelPackageLivesUnderItsCanonicalPath(t *testing.T) {
	root := moduleRootForConfig(t)
	if _, err := os.Stat(filepath.Join(root, "pkg/channels/pocketclaw")); err != nil {
		t.Fatalf("pkg/channels/pocketclaw is not the channel's home: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pkg/channels/pico")); !os.IsNotExist(err) {
		t.Error("the legacy channel package still exists")
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case "vendor", "build", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(raw), `picoclaw/pkg/channels/pico"`) {
			relative, _ := filepath.Rel(root, path)
			t.Errorf("%s imports the legacy channel package path", relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The HTTP surface derives from the channel identity, so a rename cannot leave
// the route behind. This guard used to pin the opposite — that the route had
// deliberately not moved yet — and flipping it is how that deferral was closed.
func TestTheRouteSurfaceFollowsTheChannelIdentity(t *testing.T) {
	if RealtimeRoutePrefix != "/"+ChannelPocketClaw+"/" {
		t.Errorf("RealtimeRoutePrefix = %q, want it derived from the channel name", RealtimeRoutePrefix)
	}
	for name, got := range map[string]string{
		"websocket": RealtimeWebSocketPath,
		"media":     RealtimeMediaPrefix,
		"api":       RealtimeAPIPrefix,
	} {
		if strings.Contains(got, "pico/") {
			t.Errorf("%s route %q is still in the legacy namespace", name, got)
		}
	}

	root := moduleRootForConfig(t)
	raw, err := os.ReadFile(filepath.Join(root, "pkg/channels/pocketclaw/pocketclaw.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "RoutePrefix      = config.RealtimeRoutePrefix") {
		t.Error("the channel no longer derives its route from the shared constant")
	}
}

func moduleRootForConfig(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// The canonical environment variable still reaches the managed channel's token.
//
// The struct tag it lands on is deliberately untouched: retagging it would put
// a single canonical name among ~175 legacy ones for no behavioural gain, so
// the canonical-env adapter continues to carry it.
func TestCanonicalChannelTokenEnvReachesTheChannel(t *testing.T) {
	const canonical = "POCKETCLAW_CHANNELS_POCKETCLAW_TOKEN"
	const legacy = "PICOCLAW_CHANNELS_PICO_TOKEN"

	cases := []struct {
		name           string
		canonicalValue *string
		legacyValue    *string
		want           string
	}{
		{"canonical only", ptr("canonical-token"), nil, "canonical-token"},
		{"legacy only remains supported", nil, ptr("legacy-token"), "legacy-token"},
		{"canonical wins over legacy", ptr("canonical-token"), ptr("legacy-token"), "canonical-token"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setOrUnset(t, canonical, tc.canonicalValue)
			setOrUnset(t, legacy, tc.legacyValue)

			var settings PocketClawSettings
			if err := parseEnv(&settings); err != nil {
				t.Fatalf("parseEnv: %v", err)
			}
			if got := settings.Token.String(); got != tc.want {
				t.Errorf("token = %q, want %q", got, tc.want)
			}
		})
	}
}

// The host emits only the canonical name. The two spellings that still contain
// the legacy namespace must never be produced, whatever the adapter accepts.
func TestTheHostNeverEmitsALegacyChannelTokenName(t *testing.T) {
	aliases := canonicalEnvAliasesForTest()
	if aliases["POCKETCLAW_CHANNELS_POCKETCLAW_TOKEN"] != "PICOCLAW_CHANNELS_PICO_TOKEN" {
		t.Errorf("the channel token alias changed: %v", aliases)
	}
	for canonical := range aliases {
		if strings.Contains(strings.ToUpper(canonical), "PICO") {
			t.Errorf("canonical env name %q carries the legacy namespace", canonical)
		}
	}
}

// canonicalEnvAliasesForTest reaches the one mapping table through the same
// accessor production code uses.
func canonicalEnvAliasesForTest() map[string]string {
	return canonicalenv.Aliases()
}
