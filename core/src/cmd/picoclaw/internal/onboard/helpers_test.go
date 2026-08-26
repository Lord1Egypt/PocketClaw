package onboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyEmbeddedToTargetUsesStructuredAgentFiles(t *testing.T) {
	targetDir := t.TempDir()

	if err := copyEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyEmbeddedToTarget() error = %v", err)
	}

	agentPath := filepath.Join(targetDir, "AGENT.md")
	if _, err := os.Stat(agentPath); err != nil {
		t.Fatalf("expected %s to exist: %v", agentPath, err)
	}

	soulPath := filepath.Join(targetDir, "SOUL.md")
	if _, err := os.Stat(soulPath); err != nil {
		t.Fatalf("expected %s to exist: %v", soulPath, err)
	}

	userPath := filepath.Join(targetDir, "USER.md")
	if _, err := os.Stat(userPath); err != nil {
		t.Fatalf("expected %s to exist: %v", userPath, err)
	}

	for _, legacyName := range []string{"AGENTS.md", "IDENTITY.md"} {
		legacyPath := filepath.Join(targetDir, legacyName)
		if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
			t.Fatalf("expected legacy file %s to be absent, got err=%v", legacyPath, err)
		}
	}
}

// PocketClaw does not seed the upstream-specific agent skill, but must still
// seed the rest of the bundled skills.
func TestCopyEmbeddedToTargetDoesNotSeedUpstreamAgentSkill(t *testing.T) {
	targetDir := t.TempDir()

	if err := copyEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyEmbeddedToTarget() error = %v", err)
	}

	upstreamSkill := filepath.Join(targetDir, "skills", "picoclaw-agent")
	if _, err := os.Stat(upstreamSkill); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, got err=%v", upstreamSkill, err)
	}

	for _, seeded := range []string{"github", "summarize", "skill-creator", "hardware"} {
		skillPath := filepath.Join(targetDir, "skills", seeded, "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			t.Fatalf("expected bundled skill %s to be seeded: %v", skillPath, err)
		}
	}
}

// Seeding must never remove a skill the user already has, including one
// PocketClaw chose not to seed.
func TestCopyEmbeddedToTargetKeepsExistingUserSkill(t *testing.T) {
	targetDir := t.TempDir()

	existing := filepath.Join(targetDir, "skills", "picoclaw-agent")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatalf("failed to stage existing skill: %v", err)
	}
	existingFile := filepath.Join(existing, "SKILL.md")
	if err := os.WriteFile(existingFile, []byte("user copy"), 0o644); err != nil {
		t.Fatalf("failed to stage existing skill file: %v", err)
	}

	if err := copyEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyEmbeddedToTarget() error = %v", err)
	}

	data, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("expected the user's existing skill to survive seeding: %v", err)
	}
	if string(data) != "user copy" {
		t.Fatalf("expected the user's existing skill to be untouched, got %q", string(data))
	}
}

func TestCopyMissingEmbeddedToTargetAddsSkillsWithoutReplacingUserFiles(t *testing.T) {
	targetDir := t.TempDir()
	existing := filepath.Join(targetDir, "skills", "github", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(existing), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("user github skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyMissingEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyMissingEmbeddedToTarget() error = %v", err)
	}

	data, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "user github skill" {
		t.Fatalf("existing user skill was replaced: %q", data)
	}
	for _, seeded := range []string{"agent-browser", "hardware", "skill-creator", "summarize", "tmux", "weather"} {
		if _, err := os.Stat(filepath.Join(targetDir, "skills", seeded, "SKILL.md")); err != nil {
			t.Fatalf("missing repaired bundled skill %s: %v", seeded, err)
		}
	}
	if _, err := os.Stat(filepath.Join(targetDir, "skills", "picoclaw-agent")); !os.IsNotExist(err) {
		t.Fatalf("upstream-specific skill must remain unseeded, got %v", err)
	}
}

func TestIsUnseeded(t *testing.T) {
	unseeded := []string{
		"AGENTS.md",
		"IDENTITY.md",
		"skills/picoclaw-agent",
		"skills/picoclaw-agent/SKILL.md",
		"skills/picoclaw-agent/references/notes.md",
	}
	for _, path := range unseeded {
		if !isUnseeded(path) {
			t.Errorf("isUnseeded(%q) = false, want true", path)
		}
	}

	seeded := []string{
		"AGENT.md",
		"SOUL.md",
		"skills/github/SKILL.md",
		"skills/picoclaw-agent-extra/SKILL.md",
	}
	for _, path := range seeded {
		if isUnseeded(path) {
			t.Errorf("isUnseeded(%q) = true, want false", path)
		}
	}
}
