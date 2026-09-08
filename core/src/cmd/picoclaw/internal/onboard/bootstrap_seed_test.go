package onboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bootstrap"
)

// Seeding a fresh workspace must leave a record of what it wrote, and seeding
// again must change nothing at all — not the files, not the record.
func TestSeedingIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := copyMissingEmbeddedToTarget(dir); err != nil {
		t.Fatalf("first seed: %v", err)
	}

	meta, err := bootstrap.Load(dir)
	if err != nil {
		t.Fatalf("Load() after seeding: %v", err)
	}
	for rel := range bootstrap.TrackedTemplates {
		if _, recorded := meta.Templates[rel]; !recorded {
			t.Errorf("fresh seed did not record %s", rel)
		}
		if meta.UserOwns(dir, rel) && rel != "memory/MEMORY.md" {
			t.Errorf("%s should read as pristine immediately after seeding", rel)
		}
	}

	before := snapshot(t, dir)
	if err := copyMissingEmbeddedToTarget(dir); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	after := snapshot(t, dir)

	for path, content := range before {
		if after[path] != content {
			t.Errorf("reseeding changed %s", path)
		}
	}
	if len(before) != len(after) {
		t.Errorf("reseeding changed the file set: %d -> %d", len(before), len(after))
	}
}

// TestSeedingNeverOverwritesUserFiles already proves the files survive. The new
// property is that the record does not claim otherwise: a file PocketClaw found
// rather than wrote must be marked as the user's, because that is what a future
// upgrade will consult before deciding whether it may touch anything.
func TestSeedingClaimsNoProvenanceForPreexistingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	preexisting := []string{"AGENT.md", "USER.md", filepath.Join("memory", "MEMORY.md")}
	for _, rel := range preexisting {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte("mine: "+rel+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := copyMissingEmbeddedToTarget(dir); err != nil {
		t.Fatalf("seed: %v", err)
	}

	meta, err := bootstrap.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range preexisting {
		key := filepath.ToSlash(rel)
		if _, claimed := meta.Templates[key]; claimed {
			t.Errorf("recorded provenance for %s, which PocketClaw did not write", key)
		}
		if !meta.UserOwns(dir, key) {
			t.Errorf("%s must read as the user's", key)
		}
	}
	// SOUL.md was missing, so this run did write it and may say so.
	if _, recorded := meta.Templates["SOUL.md"]; !recorded {
		t.Error("SOUL.md was seeded by this run and should be recorded")
	}
}

// The seeded AGENT.md no longer carries the Managed Runtime section: that
// guidance ships in the binary now, so a fresh workspace must not contain a
// second copy that would age independently of it.
func TestSeededAgentFileNoLongerCarriesManagedGuidance(t *testing.T) {
	dir := t.TempDir()
	if err := copyMissingEmbeddedToTarget(dir); err != nil {
		t.Fatal(err)
	}
	agent, err := os.ReadFile(filepath.Join(dir, "AGENT.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(agent), "## Managed Runtime") {
		t.Error("the seeded AGENT.md still carries the managed runtime section")
	}
	// The capability line stays: it describes what the assistant can do, which
	// is the user's file to edit. Only the operating instructions moved.
	if !strings.Contains(string(agent), "Managed Runtime") {
		t.Error("the seeded AGENT.md no longer mentions the Managed Runtime at all")
	}
}

func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
