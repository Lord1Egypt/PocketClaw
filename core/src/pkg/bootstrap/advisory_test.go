package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The record lives in the workspace, so it is untrusted input. These pin the
// one rule that matters: every way of being wrong about a document resolves to
// "leave it alone".

func writeRecord(t *testing.T, workspace, raw string) {
	t.Helper()
	dir := filepath.Join(workspace, MetadataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(workspace), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

func workspaceWithAgent(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestAmbiguousProvenanceAlwaysMeansHandsOff(t *testing.T) {
	const agentBody = "# My agent\n\nDo it my way.\n"

	cases := map[string]func(t *testing.T, workspace string){
		"no record at all": func(*testing.T, string) {},
		"record file is malformed JSON": func(t *testing.T, w string) {
			writeRecord(t, w, "{this is not json")
		},
		"record file is empty": func(t *testing.T, w string) {
			writeRecord(t, w, "")
		},
		"record file is a JSON array": func(t *testing.T, w string) {
			writeRecord(t, w, `["not", "an", "object"]`)
		},
		"schema version from the future": func(t *testing.T, w string) {
			writeRecord(t, w, `{"bootstrapVersion": 99, "templates": {"AGENT.md": {"owner": "user", "seededSha256": "`+digest(agentBody)+`"}}}`)
		},
		"schema version missing": func(t *testing.T, w string) {
			writeRecord(t, w, `{"templates": {"AGENT.md": {"owner": "user", "seededSha256": "`+digest(agentBody)+`"}}}`)
		},
		"no entry for this document": func(t *testing.T, w string) {
			writeRecord(t, w, `{"bootstrapVersion": 1, "templates": {}}`)
		},
		"entry with an empty digest": func(t *testing.T, w string) {
			writeRecord(t, w, `{"bootstrapVersion": 1, "templates": {"AGENT.md": {"owner": "user", "seededSha256": ""}}}`)
		},
		"digest does not match the file": func(t *testing.T, w string) {
			writeRecord(t, w, `{"bootstrapVersion": 1, "templates": {"AGENT.md": {"owner": "user", "seededSha256": "`+digest("something else")+`"}}}`)
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			workspace := workspaceWithAgent(t, agentBody)
			setup(t, workspace)

			// A caller that ignores the error must still be safe, so check the
			// value Load returns in both styles.
			meta, err := Load(workspace)
			if got := meta.Provenance(workspace, "AGENT.md"); got == ProvenanceMatchesSeed {
				t.Errorf("provenance = %q with err=%v; ambiguity must never read as pristine", got, err)
			}
			if !meta.UserOwns(workspace, "AGENT.md") {
				t.Error("the document must be treated as the user's")
			}
		})
	}
}

// The tampering case stated plainly: someone edits the record so their own
// AGENT.md looks like ours. That is allowed to change what we *say*, and must
// not become a way to make PocketClaw destroy the file.
func TestATamperedRecordIsNotWriteAuthority(t *testing.T) {
	const userText = "# Entirely the user's own agent file\n"
	workspace := workspaceWithAgent(t, userText)
	writeRecord(t, workspace, `{"bootstrapVersion": 1, "managedGuidanceVersion": 1, "templates":`+
		`{"AGENT.md": {"owner": "user", "seededSha256": "`+digest(userText)+`"}}}`)

	meta, err := Load(workspace)
	if err != nil {
		t.Fatal(err)
	}

	// The record can make this claim; nothing stops it.
	if got := meta.Provenance(workspace, "AGENT.md"); got != ProvenanceMatchesSeed {
		t.Fatalf("provenance = %q, want %q — the test premise is that the claim succeeds",
			got, ProvenanceMatchesSeed)
	}

	// What must hold is that the claim is not permission. The package exposes
	// no call that grants a destructive write, and the state it does expose is
	// documented as a hint requiring a separate decision. Recording again must
	// also leave the user's file untouched.
	if err := Record(workspace, 1, map[string][]byte{"AGENT.md": []byte("a bundled default")}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(workspace, "AGENT.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != userText {
		t.Errorf("the user's file was modified: %q", got)
	}
}

// A corrupt record must not be repaired by overwriting it, because a record we
// cannot read is exactly when we know least.
func TestACorruptRecordIsNotSilentlyRewritten(t *testing.T) {
	workspace := workspaceWithAgent(t, "mine\n")
	const corrupt = "{ truncated"
	writeRecord(t, workspace, corrupt)

	if err := Record(workspace, 1, map[string][]byte{"SOUL.md": []byte("soul")}); err == nil {
		t.Error("Record() silently accepted a corrupt record")
	}
	got, err := os.ReadFile(Path(workspace))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != corrupt {
		t.Errorf("the corrupt record was overwritten: %q", got)
	}
}

// Memory is never an upgrade candidate, whatever the record says about it.
func TestMemoryProvenanceIsAlwaysUnknown(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	const body = "# Memory\n\nnotes\n"
	if err := os.WriteFile(filepath.Join(workspace, "memory", "MEMORY.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// An honest record, with a digest that really does match the file.
	writeRecord(t, workspace, `{"bootstrapVersion": 1, "templates":`+
		`{"memory/MEMORY.md": {"owner": "user-private", "seededSha256": "`+digest(body)+`"}}}`)

	meta, err := Load(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got := meta.Provenance(workspace, "memory/MEMORY.md"); got != ProvenanceUnknown {
		t.Errorf("provenance = %q, want %q even when the digest matches", got, ProvenanceUnknown)
	}
	if !meta.UserOwns(workspace, "memory/MEMORY.md") {
		t.Error("memory must always read as the user's")
	}
}

// An untracked path can never become an upgrade candidate by being added to the
// record by hand.
func TestUntrackedPathsAreNotUpgradeCandidates(t *testing.T) {
	workspace := t.TempDir()
	const body = "secret\n"
	if err := os.WriteFile(filepath.Join(workspace, "notes.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	writeRecord(t, workspace, `{"bootstrapVersion": 1, "templates":`+
		`{"notes.txt": {"owner": "user", "seededSha256": "`+digest(body)+`"}}}`)

	meta, err := Load(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got := meta.Provenance(workspace, "notes.txt"); got != ProvenanceUnknown {
		t.Errorf("provenance = %q, want %q", got, ProvenanceUnknown)
	}
}

// Provenance must survive a record that round-trips through JSON with fields
// this build does not know about, rather than rejecting the whole file.
func TestUnknownFieldsDoNotInvalidateAKnownSchema(t *testing.T) {
	const body = "seeded\n"
	workspace := workspaceWithAgent(t, body)
	raw := map[string]any{
		"bootstrapVersion":       1,
		"managedGuidanceVersion": 1,
		"somethingAddedLater":    []string{"a", "b"},
		"templates": map[string]any{
			"AGENT.md": map[string]any{
				"owner":        "user",
				"seededSha256": digest(body),
				"futureField":  true,
			},
		},
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	writeRecord(t, workspace, string(encoded))

	meta, err := Load(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got := meta.Provenance(workspace, "AGENT.md"); got != ProvenanceMatchesSeed {
		t.Errorf("provenance = %q, want %q", got, ProvenanceMatchesSeed)
	}
}
