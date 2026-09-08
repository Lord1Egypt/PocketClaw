package coresource

import (
	"os"
	"path/filepath"
	"testing"
)

func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "pubspec.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// TestStagedCoreWasBuiltFromTheCurrentSource is the second half of the staleness
// gate.
//
// TestStagedCoreEmbedsTheCurrentCatalog already catches a Core built before the
// current manifest.json. It cannot catch anything else: edit python_tool.go,
// leave the catalog alone, and a Core built last week still carries this week's
// catalog version and every pinned checksum, so that guard passes while the
// binary Gradle packages does not contain the tool change at all. That is not
// hypothetical — it is what Phase C shipped past, and it is invisible on a
// device until someone calls the tool and gets the old behaviour.
//
// The fix is structural rather than procedural: the canonical build stamps the
// source fingerprint into the binary, and this recomputes it from the working
// tree. Remembering to run the build script is not a control.
func TestStagedCoreWasBuiltFromTheCurrentSource(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout; there is no staged Core to compare")
	}
	corePath := filepath.Join(root, "android", "app", "src", "main", "jniLibs",
		"arm64-v8a", "libpocketclaw.so")
	core, err := os.ReadFile(corePath)
	if err != nil {
		t.Skipf("no staged Core binary to check: %v", err)
	}

	fingerprint, err := Fingerprint(filepath.Join(root, "core", "src"))
	if err != nil {
		t.Fatalf("cannot fingerprint the Core source: %v", err)
	}

	if !StagedCoreMatches(core, fingerprint) {
		t.Errorf(
			"the staged Core does not match the current core/src source fingerprint.\n"+
				"  expected: %s\n"+
				"It was built from different source, or built outside the canonical\n"+
				"script, and would ship code that is not in this working tree.\n"+
				"Rebuild and re-stage it:\n"+
				"  %s",
			fingerprint, RebuildInstruction,
		)
	}
}
