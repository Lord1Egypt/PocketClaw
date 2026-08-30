package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestRuntimeTool(t *testing.T) *RuntimeTool {
	t.Helper()
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("cannot create workspace: %v", err)
	}
	t.Setenv("PICOCLAW_HOME", workspace)
	t.Setenv("POCKETCLAW_RUNTIME_DIR", filepath.Join(root, "runtime"))
	t.Setenv("POCKETCLAW_RUNTIME_LIB_DIR", filepath.Join(root, "lib"))

	tool, err := NewRuntimeTool(workspace)
	if err != nil {
		t.Fatalf("cannot build runtime tool: %v", err)
	}
	return tool
}

func TestRuntimeToolIsNamedAndDescribed(t *testing.T) {
	tool := newTestRuntimeTool(t)
	if tool.Name() != "runtime" {
		t.Fatalf("unexpected tool name %q", tool.Name())
	}
	// The agent must not be told to retry an install; there is no install.
	if !strings.Contains(tool.Description(), "cannot install") {
		t.Fatal("the description must state that the runtime cannot install software")
	}
}

func TestRuntimeToolListsTheCatalog(t *testing.T) {
	tool := newTestRuntimeTool(t)

	result := tool.Execute(context.Background(), map[string]any{"action": "list"})
	if result.IsError {
		t.Fatalf("list failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "PocketClaw Managed Runtime") {
		t.Fatalf("unexpected list output: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "catalog tools are available") {
		t.Fatalf("list must report availability against the catalog: %s", result.ForLLM)
	}
}

// An unsupported tool must produce an answer the agent can act on, not an
// invitation to keep trying.
func TestRuntimeToolExplainsWhyAnUnsupportedToolCannotBeInstalled(t *testing.T) {
	tool := newTestRuntimeTool(t)

	info := tool.Execute(context.Background(), map[string]any{
		"action": "info", "tool": "nmap",
	})
	if !strings.Contains(info.ForLLM, "cannot install it") {
		t.Fatalf("info must explain the constraint: %s", info.ForLLM)
	}

	run := tool.Execute(context.Background(), map[string]any{
		"action": "run", "tool": "nmap",
	})
	if !run.IsError {
		t.Fatal("running an unsupported tool must be an error result")
	}
}

func TestRuntimeToolRejectsUnknownAction(t *testing.T) {
	tool := newTestRuntimeTool(t)

	result := tool.Execute(context.Background(), map[string]any{"action": "install"})
	if !result.IsError || !strings.Contains(result.ForLLM, "unknown action") {
		t.Fatalf("expected an unknown-action error, got %q", result.ForLLM)
	}
}

func TestRuntimeToolRequiresAToolNameForRun(t *testing.T) {
	tool := newTestRuntimeTool(t)

	result := tool.Execute(context.Background(), map[string]any{"action": "run"})
	if !result.IsError {
		t.Fatal("run without a tool name must be refused")
	}
}
