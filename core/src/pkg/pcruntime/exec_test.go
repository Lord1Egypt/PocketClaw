package pcruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecuteCapturesStdoutStderrAndExitCode(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("report", TimeoutQuick, 65536)))
	writeScript(t, binDir, "report", `
echo "on stdout"
echo "on stderr" >&2
exit 3
`)

	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "report"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (%s)", result.Status, result.Diagnostics)
	}
	if result.ExitCode != 3 {
		t.Fatalf("a non-zero exit must be preserved, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "on stdout") {
		t.Fatalf("stdout not captured: %q", result.Stdout)
	}
	if !strings.Contains(result.Stderr, "on stderr") {
		t.Fatalf("stderr not captured: %q", result.Stderr)
	}
}

// The runtime never builds a command line, so arguments reach the tool exactly
// as given — including ones a shell would have split, expanded, or executed.
func TestExecutePreservesArgvExactly(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("argdump", TimeoutQuick, 65536)))
	writeScript(t, binDir, "argdump", `
for arg in "$@"; do echo "[$arg]"; done
`)

	args := []string{"a b c", "$(touch /tmp/pwned)", "*", "--flag=va lue", "'quoted'", "back\\slash"}
	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "argdump", Args: args})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}

	var got []string
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		got = append(got, strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
	}
	if len(got) != len(args) {
		t.Fatalf("expected %d arguments, tool saw %d: %q", len(args), len(got), result.Stdout)
	}
	for i := range args {
		if got[i] != args[i] {
			t.Fatalf("argument %d changed in transit: sent %q, tool saw %q", i, args[i], got[i])
		}
	}
}

func TestExecutePassesStdin(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("upper", TimeoutQuick, 65536)))
	writeScript(t, binDir, "upper", "cat\n")

	result, err := manager.Execute(context.Background(), ExecRequest{
		Tool: "upper", Stdin: "hello from stdin",
	})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !strings.Contains(result.Stdout, "hello from stdin") {
		t.Fatalf("stdin was not delivered: %q", result.Stdout)
	}
}

func TestExecuteTimesOutAndReapsTheChild(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("slow", TimeoutQuick, 4096)))
	writeScript(t, binDir, "slow", "sleep 30\n")

	started := time.Now()
	result, err := manager.Execute(context.Background(), ExecRequest{
		Tool: "slow", TimeoutMS: 300,
	})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !result.TimedOut || result.Status != StatusTimeout {
		t.Fatalf("expected a timeout, got status %s timed_out=%v", result.Status, result.TimedOut)
	}
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Fatalf("timeout did not terminate the child promptly: took %s", elapsed)
	}
	if result.DurationMS <= 0 {
		t.Fatal("a timed-out operation must still record its duration")
	}
}

func TestExecuteCancels(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("slow", TimeoutExtended, 4096)))
	writeScript(t, binDir, "slow", "sleep 30\n")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	result, err := manager.Execute(ctx, ExecRequest{Tool: "slow"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !result.Cancelled || result.Status != StatusCancelled {
		t.Fatalf("expected a cancellation, got status %s cancelled=%v", result.Status, result.Cancelled)
	}
	if result.TimedOut {
		t.Fatal("a cancellation must not be reported as a timeout")
	}
}

// A cancelled or timed-out tool must not leave a grandchild running. The script
// backgrounds a sleeper whose parent exits immediately; only killing the whole
// process group stops it.
func TestExecuteTerminatesTheWholeProcessGroup(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("spawner", TimeoutQuick, 4096)))
	marker := filepath.Join(t.TempDir(), "grandchild-survived")
	writeScript(t, binDir, "spawner", "sh -c 'sleep 2; touch "+marker+"' & sleep 30\n")

	result, err := manager.Execute(context.Background(), ExecRequest{
		Tool: "spawner", TimeoutMS: 300,
	})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if result.Status != StatusTimeout {
		t.Fatalf("expected a timeout, got %s", result.Status)
	}

	time.Sleep(3 * time.Second)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("a grandchild outlived the timeout: the process group was not terminated")
	}
}

func TestExecuteBoundsOutput(t *testing.T) {
	const limit = 2048
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("flood", TimeoutStandard, limit)))
	writeScript(t, binDir, "flood", `
i=0
while [ $i -lt 2000 ]; do echo "0123456789012345678901234567890123456789"; i=$((i+1)); done
`)

	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "flood"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !result.StdoutTruncated {
		t.Fatal("output far above the limit must be reported as truncated")
	}
	if !strings.Contains(result.Stdout, "output truncated") {
		t.Fatal("truncated output must carry a truncation marker")
	}
	// The marker itself is appended to the retained bytes, so allow for it.
	if len(result.Stdout) > limit+256 {
		t.Fatalf("retained %d bytes for a %d-byte limit", len(result.Stdout), limit)
	}
}

func TestExecuteRefusesWorkingDirectoryOutsideWorkspace(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("pwd", TimeoutQuick, 4096)))
	writeScript(t, binDir, "pwd", "pwd\n")

	_, err := manager.Execute(context.Background(), ExecRequest{
		Tool: "pwd", WorkingDirectory: "../..",
	})
	if err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Fatalf("expected the working-directory policy to refuse, got %v", err)
	}
}

func TestExecuteDefaultsToWorkspaceAndAcceptsSubdirectories(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("here", TimeoutQuick, 4096)))
	writeScript(t, binDir, "here", "pwd\n")

	nested := filepath.Join(manager.workspace, "project")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("cannot create nested dir: %v", err)
	}

	defaulted, err := manager.Execute(context.Background(), ExecRequest{Tool: "here"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !strings.Contains(defaulted.Stdout, filepath.Base(manager.workspace)) {
		t.Fatalf("default working directory should be the workspace, got %q", defaulted.Stdout)
	}

	scoped, err := manager.Execute(context.Background(), ExecRequest{
		Tool: "here", WorkingDirectory: "project",
	})
	if err != nil {
		t.Fatalf("a workspace-relative directory must be accepted: %v", err)
	}
	if !strings.Contains(scoped.Stdout, "project") {
		t.Fatalf("expected to run in the nested directory, got %q", scoped.Stdout)
	}
}

// The child environment is constructed, not inherited, so a provider key held by
// the Core process cannot leak into a managed tool.
func TestExecuteDoesNotInheritArbitraryEnvironment(t *testing.T) {
	t.Setenv("POCKETCLAW_TEST_PROVIDER_KEY", "sk-must-not-leak-to-a-child")

	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("dumpenv", TimeoutQuick, 65536)))
	writeScript(t, binDir, "dumpenv", "env\n")

	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if strings.Contains(result.Stdout, "must-not-leak-to-a-child") {
		t.Fatal("the managed tool inherited an environment variable outside the allowlist")
	}
}

func TestExecuteAppliesEnvironmentAdditions(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("dumpenv", TimeoutQuick, 65536)))
	writeScript(t, binDir, "dumpenv", "env\n")

	result, err := manager.Execute(context.Background(), ExecRequest{
		Tool:                 "dumpenv",
		EnvironmentAdditions: map[string]string{"POCKETCLAW_RUNTIME_TEST": "visible"},
	})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !strings.Contains(result.Stdout, "POCKETCLAW_RUNTIME_TEST=visible") {
		t.Fatalf("declared addition did not reach the tool: %q", result.Stdout)
	}
}

// LD_PRELOAD would let a caller substitute code for the binary the registry
// verified, which defeats the entire integrity model.
func TestExecuteRefusesLoaderEnvironmentOverrides(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("dumpenv", TimeoutQuick, 4096)))
	writeScript(t, binDir, "dumpenv", "env\n")

	for _, key := range []string{"LD_PRELOAD", "LD_LIBRARY_PATH", "PATH"} {
		_, err := manager.Execute(context.Background(), ExecRequest{
			Tool:                 "dumpenv",
			EnvironmentAdditions: map[string]string{key: "/tmp/evil"},
		})
		if err == nil || !strings.Contains(err.Error(), "may not be set") {
			t.Fatalf("%s must be refused, got %v", key, err)
		}
	}
}

// A caller may lower a tool's budget but never raise it, so no managed process
// can outlive the profile the catalog declares.
func TestTimeoutProfileIsACeiling(t *testing.T) {
	tool := systemTool("quick", TimeoutQuick, 4096)
	ceiling, _ := TimeoutQuick.Duration()

	raised, err := resolveTimeout(&tool, ceiling.Milliseconds()*10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raised != ceiling {
		t.Fatalf("a request above the profile must be clamped to %s, got %s", ceiling, raised)
	}

	lowered, err := resolveTimeout(&tool, 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lowered != 500*time.Millisecond {
		t.Fatalf("a request below the profile must be honoured, got %s", lowered)
	}
}

func TestExecuteUnsupportedToolFailsSafely(t *testing.T) {
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("cat", TimeoutQuick, 4096)))

	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "nmap"})
	if err != nil {
		t.Fatalf("an unsupported tool must produce a result, not an error: %v", err)
	}
	if result.Status != StatusUnavailable {
		t.Fatalf("expected unavailable, got %s", result.Status)
	}
	if result.ExitCode != -1 {
		t.Fatalf("an operation that never ran must not report a real exit code, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Diagnostics, "cannot install") {
		t.Fatalf("diagnostics must explain that installation is not possible: %q", result.Diagnostics)
	}
}

// EnsureTool acquires nothing; it reports. The runtime has no download path at
// all, which is what makes arbitrary binary installation structurally impossible.
func TestEnsureToolNeverAcquires(t *testing.T) {
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("dig", TimeoutQuick, 4096)))

	resolved, err := manager.EnsureTool("dig")
	if err != nil {
		t.Fatalf("EnsureTool on a catalog tool must not error: %v", err)
	}
	if resolved.Available() {
		t.Fatal("EnsureTool must not make an absent tool available")
	}

	if _, err := manager.EnsureTool("https://example.invalid/evil"); err == nil {
		t.Fatal("EnsureTool must refuse anything outside the catalog")
	}
}

// Containment is checked on resolved paths. A symlink inside the workspace that
// points at the wider filesystem would otherwise pass a literal path check and
// let a managed tool run anywhere on the device.
func TestExecuteRefusesASymlinkOutOfTheWorkspace(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("here", TimeoutQuick, 4096)))
	writeScript(t, binDir, "here", "pwd\n")

	outside := t.TempDir()
	escape := filepath.Join(manager.workspace, "escape")
	if err := os.Symlink(outside, escape); err != nil {
		t.Skipf("cannot create a symlink on this host: %v", err)
	}

	_, err := manager.Execute(context.Background(), ExecRequest{
		Tool: "here", WorkingDirectory: "escape",
	})
	if err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Fatalf("a symlink out of the workspace must be refused, got %v", err)
	}
}
