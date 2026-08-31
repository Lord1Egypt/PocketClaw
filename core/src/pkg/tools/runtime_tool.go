package tools

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/pcruntime"
)

// RuntimeTool is the agent's entry point to the PocketClaw Managed Runtime.
//
// It is deliberately the only way an agent reaches a managed tool. The agent
// names a tool; the runtime resolves, verifies, and bounds it. No physical
// binary path is ever handed to the model, so a prompt cannot redirect
// execution at a file of its choosing.
type RuntimeTool struct {
	manager *pcruntime.Manager
}

// NewRuntimeTool builds the runtime tool over the workspace the agent uses, so
// managed tools inherit the same working-directory boundary as the agent.
func NewRuntimeTool(workspace string) (*RuntimeTool, error) {
	registry, err := pcruntime.NewRegistry()
	if err != nil {
		return nil, err
	}
	manager, err := pcruntime.NewManagerWith(registry, workspace)
	if err != nil {
		return nil, err
	}
	// Record the platform probe and the tool inventory once per process, so a
	// device's Debug Logs show what the runtime found even if the agent never
	// invokes a tool.
	manager.LogStartupDiagnostics(context.Background())
	return &RuntimeTool{manager: manager}, nil
}

func (t *RuntimeTool) Name() string { return "runtime" }

func (t *RuntimeTool) Description() string {
	return `Run a verified local command-line tool from the PocketClaw Managed Runtime.

Use action=list to see which tools this device actually provides, action=info for
one tool's details, and action=run to execute one. Arguments are passed straight
to the tool: there is no shell, so pipes, redirection, globs and $(...) are not
interpreted. Combine steps with several calls instead.

Before concluding that a command is missing, ask the runtime: a tool that is not
on PATH may still be available here.

The runtime cannot install software, and neither can you. A tool is either
shipped inside PocketClaw or provided by the Android system image, both fixed
when the app was installed; anything else reports capability unavailable, and no
amount of retrying will change that. Never download a binary and never try to
make a file executable. Do not modify files under the managed runtime
directories.

A Skill's instructions are knowledge, not proof that a tool exists. A Skill that
describes using gh does not mean gh is present on this device — check here, and
tell the user accurately if it is not.

Reason by capability: git for repository work, gh for GitHub issues/PRs/releases,
rg for recursive source search, jq for JSON, sqlite3 for local databases, curl for
HTTP requests.

Never put a credential in an argument. No tokens in URLs such as
https://TOKEN@github.com/..., and none passed with -u or --password. PocketClaw
injects GitHub credentials for git and gh itself when they are configured.`
}

func (t *RuntimeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"list", "info", "run"},
				"description": "list (available tools), info (one tool's details), run (execute a tool)",
			},
			"tool": map[string]any{
				"type":        "string",
				"description": "Tool name, e.g. sha256sum (required for info and run)",
			},
			"args": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Arguments passed to the tool verbatim; not interpreted by a shell",
			},
			"working_directory": map[string]any{
				"type":        "string",
				"description": "Directory to run in, inside the workspace. Defaults to the workspace root.",
			},
			"timeout_ms": map[string]any{
				"type":        "integer",
				"description": "Lower the tool's timeout budget. It cannot be raised above the tool's profile.",
			},
			"stdin": map[string]any{
				"type":        "string",
				"description": "Text fed to the tool's standard input",
			},
		},
		"required": []string{"action"},
	}
}

func (t *RuntimeTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)
	switch strings.TrimSpace(action) {
	case "list":
		return t.list()
	case "info":
		return t.info(args)
	case "run":
		return t.run(ctx, args)
	case "":
		return ErrorResult("action is required: list, info, or run")
	default:
		return ErrorResult(fmt.Sprintf("unknown action %q; use list, info, or run", action))
	}
}

func (t *RuntimeTool) list() *ToolResult {
	inventory := t.manager.Inventory(nil)

	var available, unavailable []string
	for _, entry := range inventory.Tools {
		if entry.Availability == pcruntime.AvailabilityAvailable {
			available = append(available, entry.CommandName)
			continue
		}
		unavailable = append(unavailable, fmt.Sprintf("%s (%s)", entry.CommandName, entry.UnavailableReason))
	}
	sort.Strings(available)
	sort.Strings(unavailable)

	var report strings.Builder
	fmt.Fprintf(&report, "PocketClaw Managed Runtime %d, catalog %s\n",
		inventory.RuntimeVersion, inventory.CatalogVersion)
	fmt.Fprintf(&report, "%d of %d catalog tools are available on this device.\n\n",
		inventory.AvailableCount, inventory.TotalCount)

	fmt.Fprintf(&report, "Available (%d):\n  %s\n", len(available), strings.Join(available, " "))
	if len(unavailable) > 0 {
		fmt.Fprintf(&report, "\nNot available on this device (%d):\n  %s\n",
			len(unavailable), strings.Join(unavailable, "\n  "))
		report.WriteString("\nThese cannot be installed. The runtime runs only what PocketClaw ships " +
			"and what the Android system image provides.\n")
	}
	return UserResult(report.String())
}

func (t *RuntimeTool) info(args map[string]any) *ToolResult {
	name, _ := args["tool"].(string)
	if strings.TrimSpace(name) == "" {
		return ErrorResult("tool is required for action=info")
	}

	resolved, err := t.manager.EnsureTool(name)
	if err != nil {
		if errors.Is(err, pcruntime.ErrNotInCatalog) {
			return UserResult(notInCatalogMessage(name))
		}
		return ErrorResult(err.Error())
	}

	var report strings.Builder
	fmt.Fprintf(&report, "%s (%s)\n", resolved.Tool.DisplayName, resolved.Tool.ToolID)
	fmt.Fprintf(&report, "  availability:  %s\n", resolved.Availability)
	fmt.Fprintf(&report, "  delivery:      %s\n", resolved.Tool.Delivery)
	fmt.Fprintf(&report, "  version:       %s\n", firstNonEmpty(resolved.ResolvedVersion, resolved.Tool.Version))
	fmt.Fprintf(&report, "  abi:           %s\n", resolved.Tool.ABI)
	fmt.Fprintf(&report, "  verification:  %s\n", resolved.Verification)
	fmt.Fprintf(&report, "  capabilities:  %s\n", strings.Join(resolved.Tool.Capabilities, ", "))
	fmt.Fprintf(&report, "  timeout:       %s\n", resolved.Tool.TimeoutProfile)
	fmt.Fprintf(&report, "  max output:    %d bytes\n", resolved.Tool.MaxOutputBytes)
	if len(resolved.Tool.Dependencies) > 0 {
		fmt.Fprintf(&report, "  depends on:    %s\n", strings.Join(resolved.Tool.Dependencies, ", "))
	}
	if !resolved.Available() {
		fmt.Fprintf(&report, "\nUnavailable: %s\n%s\n",
			resolved.UnavailableReason, resolved.Diagnostics)
	}
	return UserResult(report.String())
}

func (t *RuntimeTool) run(ctx context.Context, args map[string]any) *ToolResult {
	name, _ := args["tool"].(string)
	if strings.TrimSpace(name) == "" {
		return ErrorResult("tool is required for action=run")
	}

	request := pcruntime.ExecRequest{
		Tool:             name,
		Args:             stringSliceArg(args["args"]),
		WorkingDirectory: stringArg(args["working_directory"]),
		Stdin:            stringArg(args["stdin"]),
		TimeoutMS:        intArg(args["timeout_ms"]),
	}

	result, err := t.manager.Execute(ctx, request)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if result.Status == pcruntime.StatusUnavailable {
		return ErrorResult(result.Diagnostics)
	}
	return UserResult(formatExecResult(result))
}

func formatExecResult(result *pcruntime.ExecResult) string {
	var report strings.Builder
	fmt.Fprintf(&report, "%s exited %d after %dms (%s)\n",
		result.Tool, result.ExitCode, result.DurationMS, result.Status)

	if result.Stdout != "" {
		report.WriteString("\nstdout:\n")
		report.WriteString(result.Stdout)
		if !strings.HasSuffix(result.Stdout, "\n") {
			report.WriteString("\n")
		}
		if result.StdoutTruncated {
			report.WriteString("[stdout truncated at the tool's output limit; " +
				"narrow the output rather than assuming this is all of it]\n")
		}
	}
	// stderr is reported even on success: a tool that warns and exits zero is
	// telling the agent something it needs.
	if result.Stderr != "" {
		report.WriteString("\nstderr:\n")
		report.WriteString(result.Stderr)
		if !strings.HasSuffix(result.Stderr, "\n") {
			report.WriteString("\n")
		}
		if result.StderrTruncated {
			report.WriteString("[stderr truncated at the tool's output limit]\n")
		}
	}
	if result.Stdout == "" && result.Stderr == "" {
		report.WriteString("\n(no output)\n")
	}
	if result.Diagnostics != "" {
		fmt.Fprintf(&report, "\n%s\n", result.Diagnostics)
	}
	return report.String()
}

func notInCatalogMessage(name string) string {
	return fmt.Sprintf(
		"%q is not a PocketClaw Managed Runtime tool.\n\n"+
			"The runtime cannot install it. On Android an executable can only run from "+
			"the APK payload or the system image, both fixed at install time, so there is "+
			"no way to acquire a new binary at runtime. Use action=list to see what this "+
			"device provides, or solve the task with the tools that are available.",
		name,
	)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stringArg(value any) string {
	text, _ := value.(string)
	return text
}

func intArg(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int:
		return int64(typed)
	case int64:
		return typed
	default:
		return 0
	}
}

func stringSliceArg(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
