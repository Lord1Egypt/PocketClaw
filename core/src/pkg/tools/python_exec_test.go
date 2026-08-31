package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/pcruntime"
)

// newExecutingPythonTool builds the agent's Python tool over a real Manager and
// a real interpreter.
//
// The point is to run the actual execution seam rather than a stand-in: the
// process writes its own traceback, the runtime's bounded buffers capture it,
// and the report the model receives is produced from that capture. Nothing here
// hands the tool a hand-written ExecResult, because a hand-written one could not
// fail the way the reported gap failed.
//
// The interpreter is the host's python3, published as a bundled payload so it
// goes through the same resolution and checksum verification the device uses.
// The catalog's python environment profile is deliberately not applied: it
// points PYTHONHOME at the on-device payload, which is not where a host
// interpreter keeps its standard library.
func newExecutingPythonTool(t *testing.T, maxOutputBytes int64) *PythonTool {
	t.Helper()

	interpreter, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("no python3 on this host to execute through the runtime")
	}

	root := t.TempDir()
	libDir := filepath.Join(root, "lib")
	workspace := filepath.Join(root, "workspace")
	metadata := filepath.Join(root, "runtime")
	for _, dir := range []string{libDir, workspace, metadata} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", dir, err)
		}
	}

	payload := filepath.Join(libDir, "libpocketclaw-python.so")
	if err := os.Symlink(interpreter, payload); err != nil {
		t.Fatalf("cannot stage the interpreter payload: %v", err)
	}

	manifest := &pcruntime.Manifest{
		RuntimeVersion: pcruntime.ManifestVersion,
		CatalogVersion: "test",
		Tools: []pcruntime.Tool{{
			ToolID: "python", DisplayName: "Python", CommandName: "python",
			Version: "host", ABI: hostABI(t),
			Delivery: pcruntime.DeliveryBundled, TrustedSource: "test",
			SHA256:         fileDigest(t, payload),
			Capabilities:   []string{"script.execute"},
			TimeoutProfile: pcruntime.TimeoutQuick,
			MaxOutputBytes: maxOutputBytes,
			SecurityClass:  pcruntime.SecurityClassBundledVerified,
			LibraryName:    "libpocketclaw-python.so",
			DefaultArgs:    []string{"-P", "-s", "-B", "-u"},
		}},
	}

	registry, err := pcruntime.NewRegistryWith(manifest, &pcruntime.Paths{
		LibDir: libDir, MetadataDir: metadata, Workspace: workspace,
	})
	if err != nil {
		t.Fatalf("cannot build the runtime registry: %v", err)
	}
	manager, err := pcruntime.NewManagerWith(registry, workspace)
	if err != nil {
		t.Fatalf("cannot build the runtime manager: %v", err)
	}
	return &PythonTool{manager: manager}
}

func hostABI(t *testing.T) string {
	t.Helper()
	switch runtime.GOARCH {
	case "arm64":
		return "arm64-v8a"
	case "arm":
		return "armeabi-v7a"
	case "amd64":
		return "x86_64"
	default:
		t.Skipf("the runtime catalog has no ABI name for %s", runtime.GOARCH)
		return ""
	}
}

func fileDigest(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot read the staged payload: %v", err)
	}
	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		t.Fatalf("cannot hash the staged payload: %v", err)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

// TestUncaughtExceptionReachesTheAgentAsRealStderr is the regression test for
// the reported gap: the model asked what went wrong and could not see it.
func TestUncaughtExceptionReachesTheAgentAsRealStderr(t *testing.T) {
	tool := newExecutingPythonTool(t, 1<<20)

	result := tool.Execute(context.Background(), map[string]any{
		"code": `raise ValueError("TEST-ERROR")`,
	})

	report := result.ContentForLLM()
	for _, want := range []string{
		"stderr:",
		"Traceback (most recent call last):",
		"ValueError: TEST-ERROR",
		"exit_code=1",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("the agent-facing result does not contain %q:\n%s", want, report)
		}
	}
	// The traceback must be the interpreter's own text, complete with the frame
	// it reports for a program read from standard input.
	if !strings.Contains(report, `File "<stdin>"`) {
		t.Errorf("stderr is not the process's own traceback:\n%s", report)
	}
	if !strings.Contains(report, "timed_out=false") || !strings.Contains(report, "cancelled=false") {
		t.Errorf("the result does not report the termination flags:\n%s", report)
	}
}

// A successful run must still name both streams. "stderr was empty" and "stderr
// was dropped" have to be distinguishable, or the previous failure mode is only
// hidden rather than fixed.
func TestSuccessfulRunStillNamesBothStreams(t *testing.T) {
	tool := newExecutingPythonTool(t, 1<<20)

	result := tool.Execute(context.Background(), map[string]any{
		"code": `print("PYTHON-FINAL-PASS")`,
	})

	report := result.ContentForLLM()
	if !strings.Contains(report, "PYTHON-FINAL-PASS") {
		t.Errorf("stdout did not reach the agent:\n%s", report)
	}
	if !strings.Contains(report, "stderr:\n(empty)") {
		t.Errorf("an empty stderr was omitted instead of stated:\n%s", report)
	}
	if !strings.Contains(report, "exit_code=0") {
		t.Errorf("the exit code was not reported:\n%s", report)
	}
}

// The bound is the runtime's, and it must survive being routed to the model:
// an unbounded traceback from a deep recursion would otherwise be pasted whole
// into the conversation.
func TestStderrIsBoundedAndItsTruncationIsReported(t *testing.T) {
	const limit = 4096
	tool := newExecutingPythonTool(t, limit)

	result := tool.Execute(context.Background(), map[string]any{
		"code": `import sys
sys.stderr.write("E" * 200000)
sys.exit(3)`,
	})

	report := result.ContentForLLM()
	if strings.Count(report, "E") > limit+512 {
		t.Errorf("stderr was not bounded: the report carries %d bytes", len(report))
	}
	if !strings.Contains(report, "stderr_truncated=true") {
		t.Errorf("stderr truncation was not reported as a field:\n%s",
			firstLines(report, 6))
	}
	if !strings.Contains(report, "[stderr truncated") {
		t.Errorf("stderr truncation was not marked where it happened:\n%s",
			firstLines(report, 6))
	}
	if !strings.Contains(report, "exit_code=3") {
		t.Errorf("the process's own exit code was not reported:\n%s", firstLines(report, 6))
	}
}

// The source is the one thing that must never be written down. It is the
// model's program, it can contain anything the user pasted, and it travels on
// stdin precisely so no log, event or process listing ever sees it.
func TestExecutingPythonNeverLogsTheSource(t *testing.T) {
	tool := newExecutingPythonTool(t, 1<<20)

	const source = `SECRET = "canary-value-inside-the-source"
raise ValueError("TEST-ERROR")`

	sink := filepath.Join(t.TempDir(), "runtime.log")
	previous := logger.GetLevel()
	logger.SetLevel(logger.DEBUG)
	logger.DisableConsole()
	if err := logger.EnableFileLogging(sink); err != nil {
		t.Fatalf("cannot capture logs: %v", err)
	}
	t.Cleanup(func() {
		logger.DisableFileLogging()
		logger.EnableConsole()
		logger.SetLevel(previous)
	})

	result := tool.Execute(context.Background(), map[string]any{"code": source})
	logger.DisableFileLogging()

	if !strings.Contains(result.ContentForLLM(), "ValueError: TEST-ERROR") {
		t.Fatalf("the program did not run; the test would prove nothing:\n%s",
			result.ContentForLLM())
	}

	captured, err := os.ReadFile(sink)
	if err != nil {
		t.Fatalf("cannot read the captured log: %v", err)
	}
	for _, forbidden := range []string{"canary-value-inside-the-source", "SECRET ="} {
		if strings.Contains(string(captured), forbidden) {
			t.Errorf("the runtime log recorded the Python source (%q)", forbidden)
		}
	}
	if !strings.Contains(string(captured), "bytes_in") {
		t.Error("the log did not record the size of stdin; that much must stay observable")
	}
}

// Nothing may invent a traceback. A process that fails silently must be
// reported as having failed silently, so the model retries or investigates
// rather than reading a plausible-looking error the interpreter never produced.
func TestNoTracebackIsSynthesisedWhenThereWasNone(t *testing.T) {
	tool := newExecutingPythonTool(t, 1<<20)

	result := tool.Execute(context.Background(), map[string]any{
		"code": `import sys
sys.exit(7)`,
	})

	report := result.ContentForLLM()
	if strings.Contains(report, "Traceback") {
		t.Errorf("a traceback appeared for a run that produced none:\n%s", report)
	}
	if !strings.Contains(report, "stderr:\n(empty)") {
		t.Errorf("a silent failure was not reported as silent:\n%s", report)
	}
	if !strings.Contains(report, "exit_code=7") {
		t.Errorf("the process's exit code was not reported:\n%s", report)
	}
}

func firstLines(text string, count int) string {
	lines := strings.SplitN(text, "\n", count+1)
	if len(lines) > count {
		lines = lines[:count]
	}
	return strings.Join(lines, "\n")
}
