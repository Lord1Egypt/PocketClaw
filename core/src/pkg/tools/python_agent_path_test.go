package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/pcruntime"
)

// This file tests the path a device actually uses: a real child process writing
// to real pipes, the runtime's capture, the Python tool's formatter, the tool
// registry's normalisation, and finally ContentForLLM — which is the exact
// string the agent pipeline puts into the conversation.
//
// The earlier end-to-end test stopped at PythonTool.Execute and read ForLLM off
// the result. That skipped ToolRegistry.ExecuteWithContext, which is what
// production calls and which rewrites ForLLM on the way past. A gap between the
// tool and the model could not be seen from inside the tool.

// interpreterArgv0 is the name buildArgv gives the child. The test binary
// re-executes itself under that name and behaves like an interpreter, which
// gives a real ELF of the host's own ABI without needing a Python on the box.
const interpreterArgv0 = "python"

// TestMain lets this test binary stand in for the interpreter.
//
// It reads its program from stdin exactly as `python -` does and understands
// three directives, so a test can ask for precise bytes on a precise stream and
// a precise exit status:
//
//	OUT:<text>   write text and a newline to stdout
//	ERR:<text>   write text and a newline to stderr
//	EXIT:<n>     exit with status n
//	ARGV         write the argument vector it was invoked with to stdout
func TestMain(m *testing.M) {
	if filepath.Base(os.Args[0]) == interpreterArgv0 {
		os.Exit(runStandInInterpreter())
	}
	os.Exit(m.Run())
}

func runStandInInterpreter() int {
	program, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stand-in interpreter could not read its program: %v\n", err)
		return 2
	}
	status := 0
	for _, line := range strings.Split(string(program), "\n") {
		switch {
		case strings.HasPrefix(line, "OUT:"):
			fmt.Fprintln(os.Stdout, strings.TrimPrefix(line, "OUT:"))
		case strings.HasPrefix(line, "ERR:"):
			fmt.Fprintln(os.Stderr, strings.TrimPrefix(line, "ERR:"))
		case strings.HasPrefix(line, "EXIT:"):
			fmt.Sscanf(strings.TrimPrefix(line, "EXIT:"), "%d", &status)
		case strings.HasPrefix(line, "ENVLEN:"):
			// The length, never the value: a test that printed a credential
			// would put it in the very output it is checking.
			name := strings.TrimPrefix(line, "ENVLEN:")
			fmt.Fprintf(os.Stdout, "%s_len=%d\n", name, len(os.Getenv(name)))
		case line == "ARGV":
			fmt.Fprintf(os.Stdout, "argv=%s\n", strings.Join(os.Args, " "))
		}
	}
	return status
}

// newAgentPathRegistry stages the test binary as a bundled payload and returns
// the production tool registry with the Python tool registered over it.
//
// Bundled delivery is the device's delivery: the payload is checksum-verified
// and ABI-checked before it runs, and the catalog's default_args lead the
// argument vector, so the child sees exactly what a device's interpreter sees.
func newAgentPathRegistry(t *testing.T, maxOutputBytes int64) (*ToolRegistry, string) {
	t.Helper()

	self, err := os.Executable()
	if err != nil {
		t.Fatalf("cannot find the test binary to stand in as an interpreter: %v", err)
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
	if err := os.Symlink(self, payload); err != nil {
		t.Fatalf("cannot stage the payload: %v", err)
	}

	manifest := &pcruntime.Manifest{
		RuntimeVersion: pcruntime.ManifestVersion,
		CatalogVersion: "test",
		Tools: []pcruntime.Tool{{
			ToolID: "python", DisplayName: "Python", CommandName: interpreterArgv0,
			Version: "stand-in", ABI: hostABI(t),
			Delivery: pcruntime.DeliveryBundled, TrustedSource: "test",
			SHA256:         digestOf(t, payload),
			Capabilities:   []string{"script.execute"},
			TimeoutProfile: pcruntime.TimeoutQuick,
			MaxOutputBytes: maxOutputBytes,
			SecurityClass:  pcruntime.SecurityClassBundledVerified,
			LibraryName:    "libpocketclaw-python.so",
			DefaultArgs:    []string{"-P", "-s", "-S", "-B", "-u"},
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

	tools := NewToolRegistry()
	tools.Register(&PythonTool{manager: manager})
	return tools, workspace
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

func digestOf(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		t.Fatalf("cannot hash %s: %v", path, err)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

// agentSees runs one tool call the way the agent pipeline does and returns the
// string that would enter the conversation.
func agentSees(t *testing.T, tools *ToolRegistry, code string) string {
	t.Helper()
	result := tools.ExecuteWithContext(
		context.Background(), "python", map[string]any{"code": code},
		"pocketclaw", "test-chat", nil,
	)
	if result == nil {
		t.Fatal("the tool registry returned no result")
	}
	return result.ContentForLLM()
}

// stdout must survive every step between the process and the model.
func TestAgentSeesStdoutFromTheProcess(t *testing.T) {
	tools, _ := newAgentPathRegistry(t, 1<<20)

	report := agentSees(t, tools, "OUT:PYTHON-FINAL-PASS\nEXIT:0")

	if !strings.Contains(report, "PYTHON-FINAL-PASS") {
		t.Errorf("the agent-visible result lost the process's stdout:\n%s", report)
	}
	if strings.Contains(report, "stdout:\n(empty)") {
		t.Errorf("non-empty stdout was reported as empty:\n%s", report)
	}
	for _, field := range []string{"exit_code=0", "stdout_bytes=18", "stderr_bytes=0"} {
		if !strings.Contains(report, field) {
			t.Errorf("the result does not state %q:\n%s", field, report)
		}
	}
}

// stderr must survive it too, byte for byte, with the exit status beside it.
func TestAgentSeesStderrAndExitCodeFromTheProcess(t *testing.T) {
	tools, _ := newAgentPathRegistry(t, 1<<20)

	report := agentSees(t, tools,
		"ERR:Traceback (most recent call last):\n"+
			"ERR:  File \"<stdin>\", line 1, in <module>\n"+
			"ERR:ValueError: TEST-ERROR\n"+
			"EXIT:1")

	for _, want := range []string{
		"Traceback (most recent call last):",
		`File "<stdin>", line 1, in <module>`,
		"ValueError: TEST-ERROR",
		"exit_code=1",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("the agent-visible result does not contain %q:\n%s", want, report)
		}
	}
	if strings.Contains(report, "stderr:\n(empty)") {
		t.Errorf("a real traceback was reported as empty stderr:\n%s", report)
	}
}

// Unicode must arrive as the bytes the process wrote, on the first call. The
// physical failure showed a model working around this by reaching for
// sys.stdout.buffer and os.write; if plain output survives, there is nothing to
// work around.
func TestAgentSeesUnicodeStdoutExactly(t *testing.T) {
	tools, _ := newAgentPathRegistry(t, 1<<20)

	const text = "مرحبا 🐍"
	report := agentSees(t, tools, "OUT:"+text+"\nEXIT:0")

	if !strings.Contains(report, text) {
		t.Errorf("the agent-visible result does not contain %q exactly:\n%s", text, report)
	}
	// len is bytes: the count the runtime measured must be the UTF-8 length,
	// which is what proves no re-encoding happened on the way.
	if !strings.Contains(report, fmt.Sprintf("stdout_bytes=%d", len(text)+1)) {
		t.Errorf("the measured stdout size is not the UTF-8 byte length of %q:\n%s",
			text, report)
	}
}

// The counters are the boundary evidence: they come from the capture layer, not
// from the text, so a report can distinguish a silent process from a lost
// stream. This asserts they describe the process rather than the rendering.
func TestAgentResultReportsWhatTheCaptureLayerMeasured(t *testing.T) {
	tools, _ := newAgentPathRegistry(t, 1<<20)

	silent := agentSees(t, tools, "EXIT:7")
	for _, field := range []string{"stdout_bytes=0", "stderr_bytes=0", "exit_code=7"} {
		if !strings.Contains(silent, field) {
			t.Errorf("a silent run is not described as silent (%s):\n%s", field, silent)
		}
	}

	loud := agentSees(t, tools, "OUT:aaaa\nERR:bb\nEXIT:0")
	for _, field := range []string{"stdout_bytes=5", "stderr_bytes=3"} {
		if !strings.Contains(loud, field) {
			t.Errorf("the measured byte counts are wrong (%s):\n%s", field, loud)
		}
	}
}

// The Python source is the model's program and may contain anything a user
// pasted. The runtime keeps it out of its own events; the generic tool registry
// logs every tool's arguments, which is a different log and was never checked.
func TestAgentPathNeverLogsThePythonSource(t *testing.T) {
	tools, _ := newAgentPathRegistry(t, 1<<20)

	const canary = "canary-secret-inside-the-python-source"
	captured := captureToolLog(t, func() {
		agentSees(t, tools, "OUT:done\n# "+canary+"\nEXIT:0")
	})

	if strings.Contains(captured, canary) {
		t.Error("the Python source reached the ordinary tool log")
	}
}

func captureToolLog(t *testing.T, body func()) string {
	t.Helper()
	sink := filepath.Join(t.TempDir(), "tools.log")
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

	body()
	logger.DisableFileLogging()

	captured, err := os.ReadFile(sink)
	if err != nil {
		t.Fatalf("cannot read the captured log: %v", err)
	}
	return string(captured)
}

// What the interpreter is actually invoked with, observed from inside the child
// rather than asserted on the request. The entry point must lead, the caller's
// values must follow it, and the program must not be there at all.
func TestChildIsInvokedThroughTheEntryPointWithoutTheSourceInArgv(t *testing.T) {
	tools, _ := newAgentPathRegistry(t, 1<<20)

	result := tools.ExecuteWithContext(
		context.Background(), "python",
		map[string]any{
			"code": "ARGV\n# canary-source-text-must-not-appear-in-argv\nEXIT:0",
			"args": []any{"alpha", "beta gamma"},
		},
		"pocketclaw", "test-chat", nil,
	)
	report := result.ContentForLLM()

	if !strings.Contains(report, "-m "+pythonBootstrapModule) {
		t.Errorf("the child was not invoked through the entry point:\n%s", report)
	}
	// The catalog's own flags still lead the entry point.
	if !strings.Contains(report, "argv=python -P -s -S -B -u -m "+
		pythonBootstrapModule+" alpha beta gamma") {
		t.Errorf("the argument vector is not default_args, entry point, caller args:\n%s",
			report)
	}
	argvLine := ""
	for _, line := range strings.Split(report, "\n") {
		if strings.HasPrefix(line, "argv=") {
			argvLine = line
			break
		}
	}
	if argvLine == "" {
		t.Fatalf("the child did not report its argument vector:\n%s", report)
	}
	if strings.Contains(argvLine, "canary-source-text") {
		t.Errorf("the program reached the argument vector: %s", argvLine)
	}
}

// The runtime's output bound has to survive the whole path, and the report has
// to say the bound was hit. An unbounded traceback from a deep recursion would
// otherwise be pasted whole into the conversation, and a bounded one that did
// not say so would be read as the complete error.
func TestAgentSeesBoundedOutputAndIsToldItWasCut(t *testing.T) {
	const limit = 2048
	tools, _ := newAgentPathRegistry(t, limit)

	var program strings.Builder
	for i := 0; i < 2000; i++ {
		program.WriteString("ERR:0123456789012345678901234567890123456789\n")
	}
	program.WriteString("EXIT:3")

	report := agentSees(t, tools, program.String())

	if !strings.Contains(report, "stderr_truncated=true") {
		t.Errorf("truncation was not reported as a field:\n%s", firstLines(report, 6))
	}
	if !strings.Contains(report, "[stderr truncated") {
		t.Errorf("truncation was not marked where it happened:\n%s", firstLines(report, 6))
	}
	if !strings.Contains(report, "exit_code=3") {
		t.Errorf("the exit code was lost:\n%s", firstLines(report, 6))
	}
	// The measured size is the whole stream; the kept text is the bound.
	if !strings.Contains(report, "stderr_bytes=82000") {
		t.Errorf("the measured stderr size is not the full volume:\n%s", firstLines(report, 6))
	}
	if len(report) > limit*4 {
		t.Errorf("the bound did not hold: the report is %d bytes", len(report))
	}
}

func firstLines(text string, count int) string {
	lines := strings.SplitN(text, "\n", count+1)
	if len(lines) > count {
		lines = lines[:count]
	}
	return strings.Join(lines, "\n")
}
