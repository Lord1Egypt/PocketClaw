package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Who owns what, and who wins when they disagree.
//
// The workspace owns persona, preferences and the user's own operating
// instructions. PocketClaw-managed guidance owns facts about the build actually
// running: what the runtime provides, what the bundled tools are, what this
// device can and cannot do. When a user's file carries a stale capability claim
// we do not edit it — we place the current facts after it and say plainly which
// is authoritative, and only for capability questions.

// The exact assembled order. It is asserted rather than described because the
// ordering is the mechanism: the managed facts are read after the workspace
// text, and before the skill catalog that may reference those tools.
func TestSystemPromptPartOrder(t *testing.T) {
	dir := fullWorkspace(t)
	cb := NewContextBuilder(dir)
	cb.splitOnMarker = true

	type place struct{ layer, slot, source, id string }
	want := []place{
		{"kernel", "identity", "runtime.kernel", "kernel.identity"},
		{"instruction", "workspace", "workspace.definition", "instruction.workspace"},
		{"capability", "tooling", "runtime.managed_guidance", "capability.managed_runtime"},
		{"capability", "skill_catalog", "skill:index", "capability.skill_catalog"},
		{"context", "memory", "memory:workspace", "context.memory"},
		{"context", "output", "runtime.output", "context.output_policy.split_on_marker"},
	}

	parts := sortPromptParts(cb.BuildSystemPromptParts())
	if len(parts) != len(want) {
		for i, p := range parts {
			t.Logf("%d %s/%s %s %s", i+1, p.Layer, p.Slot, p.Source.ID, p.ID)
		}
		t.Fatalf("got %d parts, want %d", len(parts), len(want))
	}
	for i, w := range want {
		got := place{string(parts[i].Layer), string(parts[i].Slot), string(parts[i].Source.ID), parts[i].ID}
		if got != (place{w.layer, w.slot, w.source, w.id}) {
			t.Errorf("part %d = %+v, want %+v", i+1, got, w)
		}
	}
}

// A user file that states an out-of-date capability fact. Their words survive
// untouched; the current facts follow them and are marked authoritative.
func TestStaleUserCapabilityClaimIsOutrankedWithoutBeingEdited(t *testing.T) {
	const userClaim = "PocketClaw has no `jq`, so never attempt JSON work with it.\n" +
		"Install anything you need with apt before starting.\n"
	body := "---\nname: PocketClaw\n---\n\nYou are terse and you always answer in Arabic.\n\n" +
		"## House rules\n\n" + userClaim
	dir := fullWorkspace(t)
	writeWorkspaceFile(t, dir, "AGENT.md", body)

	prompt := NewContextBuilder(dir).BuildSystemPrompt()

	// 1. The user's text is preserved byte for byte.
	if !strings.Contains(prompt, userClaim) {
		t.Fatalf("the user's own text was altered or dropped:\n%s", prompt)
	}
	// And on disk, untouched.
	onDisk, err := os.ReadFile(filepath.Join(dir, "AGENT.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != body {
		t.Errorf("AGENT.md on disk was modified:\n%s", onDisk)
	}

	// 2. The current facts are present and contradict the stale claim.
	if !strings.Contains(prompt, "`sqlite3` and `curl`") {
		t.Error("the managed capability facts are missing")
	}
	if !strings.Contains(prompt, "Nothing can be installed.") {
		t.Error("the managed guidance no longer states that nothing can be installed")
	}

	// 3. The precedence rule is stated, and scoped to capability facts only.
	authority := "authoritative for what\nthis device can do"
	if !strings.Contains(prompt, authority) {
		t.Error("the managed guidance does not claim authority over capability facts")
	}
	if !strings.Contains(prompt, "Persona, tone and the user's preferences\nremain the workspace's.") {
		t.Error("the managed guidance does not disclaim authority over persona and preferences")
	}

	// 4. Order: the user's text first, the current facts after it. Recency and
	//    the explicit rule point the same way; either alone would be weaker.
	userAt := strings.Index(prompt, userClaim)
	managedAt := strings.Index(prompt, "# Managed Runtime")
	if userAt < 0 || managedAt < 0 {
		t.Fatalf("missing sections: user=%d managed=%d", userAt, managedAt)
	}
	if managedAt < userAt {
		t.Error("the managed facts are placed before the workspace text they must outrank")
	}

	// 5. The persona the workspace owns is untouched by any of this.
	if !strings.Contains(prompt, "you always answer in Arabic") {
		t.Error("the user's persona instruction was lost")
	}
}

// Managed guidance must not be phrased as a general override. It speaks only
// about capabilities.
func TestManagedGuidanceDoesNotClaimBroadOverride(t *testing.T) {
	content := managedGuidancePart().Content
	for _, overreach := range []string{
		"ignore the workspace",
		"ignore AGENT.md",
		"override",
		"takes precedence over all",
		"disregard",
	} {
		if strings.Contains(strings.ToLower(content), overreach) {
			t.Errorf("managed guidance claims broad authority via %q", overreach)
		}
	}
}

func fullWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeWorkspaceFile(t, dir, "AGENT.md", "---\nname: PocketClaw\n---\n\nAssist.\n")
	writeWorkspaceFile(t, dir, "SOUL.md", "# Soul\n\nCalm.\n")
	writeWorkspaceFile(t, dir, "USER.md", "# User\n\nAda.\n")
	if err := os.MkdirAll(filepath.Join(dir, "memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "memory", "MEMORY.md"),
		[]byte("# Memory\n\nremembered\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "skills", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skills", "demo", "SKILL.md"),
		[]byte("---\nname: demo\ndescription: a demo\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
