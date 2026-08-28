package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolCallLogMessageOmitsArguments(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		hookRespond bool
		want        string
	}{
		{name: "normal", want: "Tool call: sendMessage"},
		{name: "hook response", hookRespond: true, want: "Tool call (hook respond): sendMessage"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := toolCallLogMessage("sendMessage", test.hookRespond)
			if got != test.want {
				t.Fatalf("toolCallLogMessage() = %q, want %q", got, test.want)
			}
			for _, forbidden := range []string{"private user content", `{"text":`, "SUPERSECRETVALUE"} {
				if strings.Contains(got, forbidden) {
					t.Fatalf("tool log exposed argument content %q in %q", forbidden, got)
				}
			}
		})
	}
}

func TestInferSkillNamesFromToolCall_ReadFileSkillMarkdown(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "three-one")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: three-one\ndescription: test\n---\n# Three One\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cb := NewContextBuilder(workspace)
	ts := &turnState{
		workspace: workspace,
		agent: &AgentInstance{
			Workspace:      workspace,
			ContextBuilder: cb,
		},
	}

	got := inferSkillNamesFromToolCall(ts, "read_file", map[string]any{
		"path": filepath.Join(workspace, "skills", "three-one", "SKILL.md"),
	})
	if len(got) != 1 || got[0] != "three-one" {
		t.Fatalf("inferSkillNamesFromToolCall = %v, want [three-one]", got)
	}
}

func TestInferSkillNamesFromToolCall_NonSkillFileIgnored(t *testing.T) {
	workspace := t.TempDir()
	ts := &turnState{workspace: workspace}

	got := inferSkillNamesFromToolCall(ts, "read_file", map[string]any{
		"path": filepath.Join(workspace, "README.md"),
	})
	if len(got) != 0 {
		t.Fatalf("inferSkillNamesFromToolCall = %v, want empty", got)
	}
}
