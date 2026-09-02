package onboard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The workspace a brand-new user receives is seeded from the embedded
// `workspace/` tree by copyMissingEmbeddedToTarget. These pin what that
// workspace says about its own identity, and that seeding never touches a file
// the user already has.

func seedFresh(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := copyMissingEmbeddedToTarget(dir); err != nil {
		t.Fatalf("copyMissingEmbeddedToTarget() error = %v", err)
	}
	return dir
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}

func TestFreshWorkspaceSeedsRequiredDefaults(t *testing.T) {
	dir := seedFresh(t)

	for _, rel := range []string{
		"AGENT.md",
		"SOUL.md",
		"USER.md",
		filepath.Join("memory", "MEMORY.md"),
		filepath.Join("skills", "skill-creator", "SKILL.md"),
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("fresh workspace is missing %s: %v", rel, err)
		}
	}

	// HEARTBEAT.md is deliberately not seeded here: the heartbeat service
	// creates it on first use, and only when it does not already exist.
	if _, err := os.Stat(filepath.Join(dir, "HEARTBEAT.md")); !os.IsNotExist(err) {
		t.Errorf("HEARTBEAT.md should not be seeded by onboard, got err=%v", err)
	}
}

// TestFreshWorkspaceIdentityIsPocketClaw is the branding regression: the
// frontmatter name used to say "pico" while the body two lines below said
// "Your name is PocketClaw."
func TestFreshWorkspaceIdentityIsPocketClaw(t *testing.T) {
	agent := readFile(t, filepath.Join(seedFresh(t), "AGENT.md"))

	name := regexp.MustCompile(`(?m)^name:\s*(.*)$`).FindStringSubmatch(agent)
	if name == nil {
		t.Fatal("AGENT.md has no name field in its frontmatter")
	}
	if got := strings.TrimSpace(name[1]); got != "PocketClaw" {
		t.Errorf("AGENT.md frontmatter name = %q, want %q", got, "PocketClaw")
	}
	if !strings.Contains(agent, "Your name is PocketClaw.") {
		t.Error("AGENT.md body no longer states the PocketClaw identity")
	}
}

// TestFreshWorkspaceHasNoUpstreamBranding sweeps every seeded file. The one
// permitted occurrence is the real config directory path, which names a
// location on disk rather than the product.
func TestFreshWorkspaceHasNoUpstreamBranding(t *testing.T) {
	dir := seedFresh(t)
	const allowedPathLiteral = "$HOME/.picoclaw/workspace"

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		for i, line := range strings.Split(readFile(t, path), "\n") {
			if !strings.Contains(strings.ToLower(line), "pico") {
				continue
			}
			if strings.Contains(line, allowedPathLiteral) {
				continue
			}
			rel, _ := filepath.Rel(dir, path)
			t.Errorf("%s:%d carries upstream branding: %s", rel, i+1, strings.TrimSpace(line))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
}

// TestFreshWorkspaceKeepsRealConfigPath guards the other direction: the
// permitted occurrence is a genuine path and must not be "cleaned up" into a
// directory that does not exist.
func TestFreshWorkspaceKeepsRealConfigPath(t *testing.T) {
	skill := readFile(t, filepath.Join(seedFresh(t), "skills", "skill-creator", "SKILL.md"))
	if !strings.Contains(skill, "$HOME/.picoclaw/workspace") {
		t.Error("skill-creator no longer documents the real workspace path")
	}
}

// TestSeedingNeverOverwritesUserFiles is the migration safety property. No
// branding migration is implemented, and this is why one is not needed to be
// careful about: seeding only ever fills gaps.
func TestSeedingNeverOverwritesUserFiles(t *testing.T) {
	dir := t.TempDir()

	customized := map[string]string{
		"AGENT.md":     "---\nname: MyOwnAgent\n---\n\nMy own instructions.\n",
		"SOUL.md":      "My own soul, hand written.\n",
		"USER.md":      "My name is Ada. I prefer terse answers.\n",
		"HEARTBEAT.md": "- check my calendar\n",
	}
	for name, content := range customized {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A legacy file whose provenance cannot be established either way.
	ambiguous := filepath.Join(dir, "memory")
	if err := os.MkdirAll(ambiguous, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(ambiguous, "MEMORY.md")
	if err := os.WriteFile(legacy, []byte("half-edited legacy memory\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyMissingEmbeddedToTarget(dir); err != nil {
		t.Fatalf("copyMissingEmbeddedToTarget() error = %v", err)
	}

	for name, want := range customized {
		if got := readFile(t, filepath.Join(dir, name)); got != want {
			t.Errorf("%s was modified by seeding:\n got: %q\nwant: %q", name, got, want)
		}
	}
	if got := readFile(t, legacy); got != "half-edited legacy memory\n" {
		t.Errorf("ambiguous legacy memory/MEMORY.md was modified: %q", got)
	}
}

// TestSeedingStillFillsGapsAlongsideUserFiles keeps the previous test honest:
// preserving user files must not turn seeding into a no-op.
func TestSeedingStillFillsGapsAlongsideUserFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyMissingEmbeddedToTarget(dir); err != nil {
		t.Fatalf("copyMissingEmbeddedToTarget() error = %v", err)
	}

	if got := readFile(t, filepath.Join(dir, "AGENT.md")); got != "mine\n" {
		t.Errorf("AGENT.md was overwritten: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "SOUL.md")); err != nil {
		t.Errorf("SOUL.md should still have been seeded: %v", err)
	}
}
