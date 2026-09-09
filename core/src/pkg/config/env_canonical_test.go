package config

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// logDirTarget mirrors how upstream reads a migrated variable: a plain struct
// tag naming the legacy key. The adapter's whole job is to make the canonical
// name reach a tag like this one without the tag changing.
type logDirTarget struct {
	LogDir string `env:"PICOCLAW_LOG_DIR"`
}

// unrelatedTarget stands in for the env-tagged configuration that has nothing
// to do with this migration and must keep working untouched.
type unrelatedTarget struct {
	Level string `env:"PICOCLAW_LOG_LEVEL"`
}

func TestParseEnvCanonicalPrecedence(t *testing.T) {
	cases := []struct {
		name      string
		canonical *string
		legacy    *string
		want      string
	}{
		{name: "canonical only", canonical: ptr("/canonical"), want: "/canonical"},
		{name: "legacy only remains supported", legacy: ptr("/legacy"), want: "/legacy"},
		{name: "both present, canonical wins", canonical: ptr("/canonical"), legacy: ptr("/legacy"), want: "/canonical"},
		// Presence beats emptiness: a blanked canonical variable must not fall
		// through to a stale legacy value.
		{name: "empty canonical beats non-empty legacy", canonical: ptr(""), legacy: ptr("/legacy"), want: ""},
		{name: "neither set", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setOrUnset(t, "POCKETCLAW_LOG_DIR", tc.canonical)
			setOrUnset(t, "PICOCLAW_LOG_DIR", tc.legacy)

			var target logDirTarget
			if err := parseEnv(&target); err != nil {
				t.Fatalf("parseEnv: %v", err)
			}
			if target.LogDir != tc.want {
				t.Fatalf("LogDir = %q, want %q", target.LogDir, tc.want)
			}
		})
	}
}

// The compatibility translation is parser-local. If it reached the live
// environment it would be inherited by every child process the Core spawns,
// which is the namespace this migration exists to remove.
func TestParseEnvDoesNotMutateProcessEnvironment(t *testing.T) {
	setOrUnset(t, "POCKETCLAW_LOG_DIR", ptr("/canonical"))
	setOrUnset(t, "PICOCLAW_LOG_DIR", nil)

	before := sortedEnviron()

	var target logDirTarget
	if err := parseEnv(&target); err != nil {
		t.Fatalf("parseEnv: %v", err)
	}
	if target.LogDir != "/canonical" {
		t.Fatalf("adapter did not apply the canonical value: %q", target.LogDir)
	}

	if _, present := os.LookupEnv("PICOCLAW_LOG_DIR"); present {
		t.Error("parseEnv wrote the legacy name into the process environment")
	}
	after := sortedEnviron()
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Error("parseEnv changed the process environment")
	}
}

// The parser sees the real environment plus the overlay, not a small map of
// PocketClaw keys — otherwise every other env-tagged setting would silently
// stop resolving.
func TestParserEnvironmentPreservesUnrelatedEntries(t *testing.T) {
	t.Setenv("PICOCLAW_LOG_LEVEL", "debug")

	var target unrelatedTarget
	if err := parseEnv(&target); err != nil {
		t.Fatalf("parseEnv: %v", err)
	}
	if target.Level != "debug" {
		t.Fatalf("unrelated variable did not survive into parser input: %q", target.Level)
	}
}

// Both decode points must go through the adapter. One that works in the main
// config but not in channel settings would be worse than one that works in
// neither, because it would look supported.
func TestBothDecodePointsUseTheSharedAdapter(t *testing.T) {
	rawParse := regexp.MustCompile(`\benv\.Parse\(`)
	withOptions := regexp.MustCompile(`\benv\.ParseWithOptions\(`)

	var adapters []string
	for _, path := range packageSources(t, ".") {
		source := readSource(t, path)
		name := filepath.Base(path)
		if rawParse.MatchString(source) {
			t.Errorf("%s calls env.Parse directly; it must use parseEnv so canonical names apply", name)
		}
		if withOptions.MatchString(source) {
			adapters = append(adapters, name)
		}
	}
	if len(adapters) != 1 || adapters[0] != "env_canonical.go" {
		t.Errorf("env.ParseWithOptions should exist only in env_canonical.go, found in %v", adapters)
	}

	for _, name := range []string{"config.go", "config_channel.go"} {
		if !strings.Contains(readSource(t, name), "parseEnv(") {
			t.Errorf("%s no longer decodes through the shared adapter", name)
		}
	}
}

// The adapter exists so upstream's env tags do not have to move. If a rename
// starts leaking in, the divergence it was built to avoid is happening anyway.
func TestUpstreamStructTagsUnchanged(t *testing.T) {
	const wantLegacyTags = 175

	legacy := regexp.MustCompile(`env:"PICOCLAW_[A-Z0-9_]*"`)
	canonical := regexp.MustCompile(`env:"POCKETCLAW_[A-Z0-9_]*"`)

	legacyCount, canonicalCount := 0, 0
	root := moduleRoot(t)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == "build" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source := readSource(t, path)
		legacyCount += len(legacy.FindAllString(source, -1))
		canonicalCount += len(canonical.FindAllString(source, -1))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	if legacyCount != wantLegacyTags {
		t.Errorf("legacy env struct tags = %d, want %d; the adapter is meant to leave every one of them alone", legacyCount, wantLegacyTags)
	}
	if canonicalCount != 0 {
		t.Errorf("canonical env struct tags = %d, want 0; canonical names reach the parser through the adapter, not through renamed tags", canonicalCount)
	}
}

func ptr(s string) *string { return &s }

// setOrUnset applies a case's intent exactly: nil means absent, which is a
// different input from present-but-empty.
func setOrUnset(t *testing.T, key string, value *string) {
	t.Helper()
	if value == nil {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		return
	}
	t.Setenv(key, *value)
}

func sortedEnviron() []string {
	environ := append([]string(nil), os.Environ()...)
	sort.Strings(environ)
	return environ
}

func packageSources(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var sources []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		sources = append(sources, filepath.Join(dir, name))
	}
	return sources
}

func readSource(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the config package")
		}
		dir = parent
	}
}
