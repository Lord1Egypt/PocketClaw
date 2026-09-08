package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func seed(t *testing.T, workspace, rel, content string) []byte {
	t.Helper()
	full := filepath.Join(workspace, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return []byte(content)
}

func TestLoadTreatsAMissingRecordAsUnknownRatherThanAnError(t *testing.T) {
	// Every install that exists today has no record. That is a normal state,
	// not a failure, and it must mean "assume nothing" rather than "crash".
	meta, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load() on a workspace with no record: %v", err)
	}
	if len(meta.Templates) != 0 || meta.BootstrapVersion != 0 {
		t.Errorf("expected a zero record, got %+v", meta)
	}
}

func TestRecordStoresOnlyWhatWasActuallyWritten(t *testing.T) {
	workspace := t.TempDir()
	written := map[string][]byte{
		"AGENT.md":         seed(t, workspace, "AGENT.md", "agent\n"),
		"memory/MEMORY.md": seed(t, workspace, "memory/MEMORY.md", "memory\n"),
		// Seeded, but not a tracked bootstrap document.
		"skills/weather/SKILL.md": seed(t, workspace, "skills/weather/SKILL.md", "skill\n"),
	}
	if err := Record(workspace, 1, written); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	meta, err := Load(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if meta.BootstrapVersion != Version {
		t.Errorf("bootstrapVersion = %d, want %d", meta.BootstrapVersion, Version)
	}
	if meta.ManagedGuidanceVersion != 1 {
		t.Errorf("managedGuidanceVersion = %d, want 1", meta.ManagedGuidanceVersion)
	}
	if len(meta.Templates) != 2 {
		t.Fatalf("tracked %d templates, want 2 (skills are not tracked): %+v",
			len(meta.Templates), meta.Templates)
	}
	if got := meta.Templates["memory/MEMORY.md"].Owner; got != OwnerUserPrivate {
		t.Errorf("MEMORY.md owner = %q, want %q", got, OwnerUserPrivate)
	}
	sum := sha256.Sum256([]byte("agent\n"))
	if got := meta.Templates["AGENT.md"].SeededSHA256; got != hex.EncodeToString(sum[:]) {
		t.Errorf("AGENT.md digest = %q, want the digest of what was written", got)
	}
}

// A workspace that already had AGENT.md is the case that matters: PocketClaw did
// not write that file, so it must not claim to know what is in it.
func TestRecordClaimsNoProvenanceForFilesItDidNotWrite(t *testing.T) {
	workspace := t.TempDir()
	seed(t, workspace, "AGENT.md", "the user's own agent file\n")

	// Seeding skipped AGENT.md because it existed; only SOUL.md was written.
	written := map[string][]byte{"SOUL.md": seed(t, workspace, "SOUL.md", "soul\n")}
	if err := Record(workspace, 1, written); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	meta, err := Load(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, claimed := meta.Templates["AGENT.md"]; claimed {
		t.Error("recorded provenance for a file PocketClaw never wrote")
	}
	if !meta.UserOwns(workspace, "AGENT.md") {
		t.Error("an unrecorded file must be treated as the user's")
	}
}

func TestRerunIsIdempotent(t *testing.T) {
	workspace := t.TempDir()
	written := map[string][]byte{"AGENT.md": seed(t, workspace, "AGENT.md", "agent\n")}
	if err := Record(workspace, 1, written); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(Path(workspace))
	if err != nil {
		t.Fatal(err)
	}

	// A second seeding run writes nothing, because everything already exists.
	if err := Record(workspace, 1, map[string][]byte{}); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(Path(workspace))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("rerun rewrote the record:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// The upgrade-safety property, stated as the question a future upgrade will ask.
func TestUserOwnsDistinguishesPristineFromEdited(t *testing.T) {
	workspace := t.TempDir()
	written := map[string][]byte{
		"AGENT.md": seed(t, workspace, "AGENT.md", "seeded default\n"),
		"SOUL.md":  seed(t, workspace, "SOUL.md", "seeded soul\n"),
	}
	if err := Record(workspace, 1, written); err != nil {
		t.Fatal(err)
	}
	meta, err := Load(workspace)
	if err != nil {
		t.Fatal(err)
	}

	if meta.UserOwns(workspace, "AGENT.md") {
		t.Error("an untouched seeded file should not read as user-owned")
	}

	seed(t, workspace, "AGENT.md", "seeded default\nand my own note\n")
	if !meta.UserOwns(workspace, "AGENT.md") {
		t.Error("an edited file must read as user-owned")
	}
	if meta.UserOwns(workspace, "SOUL.md") {
		t.Error("editing one file must not change the answer for another")
	}

	// Re-recording after the edit must not relabel the user's text as ours.
	if err := Record(workspace, 1, map[string][]byte{
		"AGENT.md": []byte("seeded default\nand my own note\n"),
	}); err != nil {
		t.Fatal(err)
	}
	meta, err = Load(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !meta.UserOwns(workspace, "AGENT.md") {
		t.Error("re-recording overwrote the original provenance and lost the user's edit")
	}
}

// Memory is recorded so the seed is accounted for, and is never a candidate for
// an automatic refresh no matter what its digest says.
func TestMemoryIsNeverEligibleForRefresh(t *testing.T) {
	workspace := t.TempDir()
	written := map[string][]byte{
		"memory/MEMORY.md": seed(t, workspace, "memory/MEMORY.md", "memory\n"),
	}
	if err := Record(workspace, 1, written); err != nil {
		t.Fatal(err)
	}
	meta, err := Load(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !meta.UserOwns(workspace, "memory/MEMORY.md") {
		t.Error("MEMORY.md must always read as the user's, even untouched")
	}
}

func TestRecordDoesNotModifyWorkspaceFiles(t *testing.T) {
	workspace := t.TempDir()
	const original = "the user's own agent file\n"
	seed(t, workspace, "AGENT.md", original)
	seed(t, workspace, "memory/MEMORY.md", "private notes\n")

	if err := Record(workspace, 1, map[string][]byte{
		"AGENT.md":         []byte("a different default"),
		"memory/MEMORY.md": []byte("a different memory"),
	}); err != nil {
		t.Fatal(err)
	}

	for rel, want := range map[string]string{
		"AGENT.md":         original,
		"memory/MEMORY.md": "private notes\n",
	} {
		got, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s was modified: %q", rel, got)
		}
	}
}
