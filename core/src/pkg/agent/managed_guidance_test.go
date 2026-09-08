package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The bug this milestone exists to fix: a workspace seeded before the Managed
// Runtime existed never learns about it, because AGENT.md is written once and
// never refreshed. Guidance that ships in the binary reaches that install on the
// next upgrade without touching its files.
func TestManagedGuidanceIsIndependentOfTheWorkspace(t *testing.T) {
	part := managedGuidancePart()

	if strings.TrimSpace(part.Content) == "" {
		t.Fatal("managed guidance is empty")
	}
	if !strings.Contains(part.Content, "Managed Runtime") {
		t.Error("managed guidance no longer describes the Managed Runtime")
	}
	if part.Layer != PromptLayerCapability || part.Slot != PromptSlotTooling {
		t.Errorf("placement = %s/%s, want capability/tooling", part.Layer, part.Slot)
	}
	if part.Source.ID != PromptSourceManagedGuidance {
		t.Errorf("source = %q, want %q", part.Source.ID, PromptSourceManagedGuidance)
	}
	if !part.Stable {
		t.Error("managed guidance is the same every turn and should be cacheable")
	}

	// It must be a placement the registry actually permits, or it would be
	// dropped at assembly time with only a warning.
	if err := NewPromptRegistry().ValidatePart(part); err != nil {
		t.Errorf("registry rejects the managed guidance part: %v", err)
	}
}

func TestSupersededSectionIsDroppedOnlyWhenItIsVerbatimOurs(t *testing.T) {
	const preamble = "You are the default assistant for this workspace.\n\n"
	const tail = "## Working Principles\n\n- Be clear, direct, and accurate\n"

	seededSection := legacyManagedRuntimeSection(t)

	t.Run("verbatim seeded copy is dropped", func(t *testing.T) {
		body := preamble + seededSection + "\n\n" + tail
		got, dropped := stripSupersededManagedGuidance(body)
		if len(dropped) != 1 || dropped[0] != 1 {
			t.Fatalf("dropped = %v, want [1]", dropped)
		}
		if strings.Contains(got, "## Managed Runtime") {
			t.Error("the superseded section is still in the prompt body")
		}
		if !strings.Contains(got, "default assistant") || !strings.Contains(got, "Working Principles") {
			t.Errorf("surrounding content was damaged:\n%s", got)
		}
	})

	t.Run("an edited copy is kept", func(t *testing.T) {
		edited := seededSection + "\n- Also: always ask me before using git.\n"
		body := preamble + edited + "\n\n" + tail
		got, dropped := stripSupersededManagedGuidance(body)
		if len(dropped) != 0 {
			t.Fatalf("dropped %v; an edited section belongs to the user", dropped)
		}
		if !strings.Contains(got, "always ask me before using git") {
			t.Error("the user's own instruction was removed")
		}
	})

	t.Run("a workspace without the section is untouched", func(t *testing.T) {
		body := preamble + tail
		got, dropped := stripSupersededManagedGuidance(body)
		if len(dropped) != 0 {
			t.Fatalf("dropped = %v, want none", dropped)
		}
		if got != body {
			t.Errorf("body changed:\n%q", got)
		}
	})

	t.Run("a same-named section the user wrote themselves is kept", func(t *testing.T) {
		body := preamble + "## Managed Runtime\n\nMy own notes about the runtime.\n\n" + tail
		got, dropped := stripSupersededManagedGuidance(body)
		if len(dropped) != 0 {
			t.Fatalf("dropped %v; matching by heading alone would delete user content", dropped)
		}
		if !strings.Contains(got, "My own notes about the runtime.") {
			t.Error("the user's own section was removed")
		}
	})
}

// The digest in supersededManagedSections has to correspond to text that was
// really shipped. This reconstructs the 0.2.0+58 section and fails if the
// recorded digest does not match it, so the entry cannot rot into a value that
// matches nothing.
func TestSupersededDigestMatchesTheTextThatWasShipped(t *testing.T) {
	section := legacyManagedRuntimeSection(t)
	version, ok := supersededManagedSections[digestOf(section)]
	if !ok {
		t.Fatalf("the recorded superseded digest does not match the shipped section;\n"+
			"got %s", digestOf(section))
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
}

// legacyManagedRuntimeSection is the "## Managed Runtime" section exactly as
// PocketClaw seeded it into AGENT.md up to and including 0.2.0+58. It is
// reproduced here rather than read from the template because the template no
// longer contains it — that is the change this file is testing — and because an
// existing install still has these bytes on disk.
func legacyManagedRuntimeSection(t *testing.T) string {
	t.Helper()
	return "## Managed Runtime\n" + strings.TrimPrefix(managedRuntimeGuidance, "# Managed Runtime\n")
}

// The whole point, assembled: an install whose AGENT.md predates the Managed
// Runtime gets the guidance, and an install that already has PocketClaw's copy
// of it gets it exactly once.
func TestAssembledPromptDeliversManagedGuidanceExactlyOnce(t *testing.T) {
	const marker = "Do not assume \"command not found\". Ask the runtime first."

	t.Run("workspace predating the guidance", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkspaceFile(t, dir, "AGENT.md",
			"---\nname: PocketClaw\n---\n\nYou are the default assistant.\n\n"+
				"## Working Principles\n\n- Be clear\n")

		prompt := NewContextBuilder(dir).BuildSystemPrompt()
		if got := strings.Count(prompt, marker); got != 1 {
			t.Errorf("managed guidance appears %d times, want 1", got)
		}
	})

	t.Run("workspace carrying PocketClaw's own seeded copy", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkspaceFile(t, dir, "AGENT.md",
			"---\nname: PocketClaw\n---\n\nYou are the default assistant.\n\n"+
				legacyManagedRuntimeSection(t)+"\n\n## Working Principles\n\n- Be clear\n")

		prompt := NewContextBuilder(dir).BuildSystemPrompt()
		if got := strings.Count(prompt, marker); got != 1 {
			t.Errorf("managed guidance appears %d times, want 1 "+
				"(the seeded duplicate should have been dropped)", got)
		}
		if !strings.Contains(prompt, "You are the default assistant.") {
			t.Error("the user's own instructions were lost")
		}
	})

	t.Run("workspace where the user edited that copy", func(t *testing.T) {
		dir := t.TempDir()
		edited := legacyManagedRuntimeSection(t) + "\n- Also: always ask me before using git.\n"
		writeWorkspaceFile(t, dir, "AGENT.md",
			"---\nname: PocketClaw\n---\n\nYou are the default assistant.\n\n"+
				edited+"\n\n## Working Principles\n\n- Be clear\n")

		prompt := NewContextBuilder(dir).BuildSystemPrompt()
		if !strings.Contains(prompt, "always ask me before using git") {
			t.Error("the user's edit was dropped from the prompt")
		}
		if got := strings.Count(prompt, marker); got != 2 {
			t.Errorf("marker appears %d times, want 2: their edited section is "+
				"theirs to keep, and the managed guidance still applies", got)
		}
	})
}

func writeWorkspaceFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A sub-turn restricted to a tool set without `runtime` should not be told to
// ask the runtime first. An unrestricted caller always gets the guidance.
func TestManagedGuidanceFollowsTheRuntimeTool(t *testing.T) {
	dir := t.TempDir()
	writeWorkspaceFile(t, dir, "AGENT.md", "---\nname: PocketClaw\n---\n\nAssist.\n")
	cb := NewContextBuilder(dir)

	cases := map[string]struct {
		allowedTools []string
		want         bool
	}{
		"unrestricted":        {nil, true},
		"runtime allowed":     {[]string{"runtime", "read_file"}, true},
		"runtime not allowed": {[]string{"read_file"}, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			parts := cb.buildSystemPromptParts(systemPromptBuildOptions{
				IncludeSkillCatalog: true,
				IncludeToolUseRule:  true,
				AllowedTools:        tc.allowedTools,
			})
			present := false
			for _, part := range parts {
				if part.Source.ID == PromptSourceManagedGuidance {
					present = true
				}
			}
			if present != tc.want {
				t.Errorf("managed guidance present = %v, want %v", present, tc.want)
			}
		})
	}
}
