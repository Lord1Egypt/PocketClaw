package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
)

func TestImportedSkillAddsToExistingWorkspaceSet(t *testing.T) {
	workspace := t.TempDir()
	baseline := []string{
		"agent-browser",
		"github",
		"hardware",
		"skill-creator",
		"summarize",
		"tmux",
		"weather",
	}
	for _, name := range baseline {
		dir := filepath.Join(workspace, "skills", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := fmt.Sprintf("---\nname: %s\ndescription: baseline %s\n---\n", name, name)
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	_, status, err := importUploadedMarkdownSkill(
		cfg,
		"gh.md",
		[]byte("---\nname: gh\ndescription: GitHub CLI helper\n---\n\n# gh\n"),
	)
	if err != nil || status != http.StatusOK {
		t.Fatalf("import status=%d error=%v", status, err)
	}

	loaded := skills.NewSkillsLoader(workspace, "", "").ListSkills()
	if len(loaded) != len(baseline)+1 {
		t.Fatalf("skills after import = %d, want %d", len(loaded), len(baseline)+1)
	}
	names := make(map[string]bool, len(loaded))
	for _, skill := range loaded {
		names[skill.Name] = true
	}
	for _, name := range append(baseline, "gh") {
		if !names[name] {
			t.Fatalf("skill %q disappeared after gh import", name)
		}
	}
}
