package tools

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/pcruntime"
)

func TestPythonToolIsNamedAndSchemaed(t *testing.T) {
	tool := &PythonTool{}
	if tool.Name() != "python" {
		t.Errorf("tool name = %q, want python", tool.Name())
	}

	params := tool.Parameters()
	props, _ := params["properties"].(map[string]any)
	for _, field := range []string{"code", "args", "timeout_ms"} {
		if _, ok := props[field]; !ok {
			t.Errorf("schema is missing %q", field)
		}
	}
	// v1 is deliberately small: no second input channel competing with code.
	if _, present := props["stdin"]; present {
		t.Error("schema exposes a separate stdin field; v1 keeps code as the only input")
	}
	required, _ := params["required"].([]string)
	if len(required) != 1 || required[0] != "code" {
		t.Errorf("required = %v, want exactly [code]", required)
	}
}

func TestPythonRequestPutsSourceOnStdinAndNotInArgv(t *testing.T) {
	const code = "import sys\nprint('hello', sys.argv)"
	request, refusal := buildPythonRequest(map[string]any{"code": code})
	if refusal != nil {
		t.Fatalf("valid code was refused: %s", refusal.ForLLM)
	}

	if request.Stdin != code {
		t.Errorf("source did not travel on stdin: %q", request.Stdin)
	}
	if request.Tool != "python" {
		t.Errorf("request targets %q, want the catalog's python tool", request.Tool)
	}
	for _, arg := range request.Args {
		if strings.Contains(arg, "print(") {
			t.Fatalf("source leaked into the argument vector: %q", arg)
		}
	}
	if len(request.Args) == 0 || request.Args[0] != "-" {
		t.Fatalf("args = %v, want the stdin marker first", request.Args)
	}
}

// A regression guard. -c would put the program in the argument vector, where it
// is size-capped near 128 KB, readable from /proc/<pid>/cmdline, and recorded in
// argument diagnostics. stdin is size-accounted only.
func TestPythonRequestNeverUsesDashC(t *testing.T) {
	if pythonStdinMarker != "-" {
		t.Fatalf("the stdin marker is %q; it must stay \"-\"", pythonStdinMarker)
	}
	request, _ := buildPythonRequest(map[string]any{
		"code": "print(1)",
		"args": []any{"-c", "print('smuggled')"},
	})
	if request.Args[0] != "-" {
		t.Fatalf("a caller-supplied argument reached the front of argv: %v", request.Args)
	}
	// Caller arguments are still passed through, but only after "-", where the
	// interpreter treats them as sys.argv rather than as source.
	if len(request.Args) != 3 || request.Args[1] != "-c" {
		t.Fatalf("args = %v, want the marker followed by the caller's values", request.Args)
	}
	if request.Stdin != "print(1)" {
		t.Errorf("stdin = %q, want the code field only", request.Stdin)
	}
}

func TestPythonRequestArgsBecomeScriptArguments(t *testing.T) {
	request, _ := buildPythonRequest(map[string]any{
		"code": "import sys; print(sys.argv[1:])",
		"args": []any{"alpha", "beta gamma", "🐍"},
	})
	want := []string{"-", "alpha", "beta gamma", "🐍"}
	if len(request.Args) != len(want) {
		t.Fatalf("args = %v, want %v", request.Args, want)
	}
	for i := range want {
		if request.Args[i] != want[i] {
			t.Fatalf("args = %v, want %v", request.Args, want)
		}
	}
}

func TestPythonRequestPassesTimeoutThroughForTheRuntimeToClamp(t *testing.T) {
	request, _ := buildPythonRequest(map[string]any{"code": "print(1)", "timeout_ms": 500})
	if request.TimeoutMS != 500 {
		t.Errorf("timeout_ms = %d, want 500 handed to the runtime", request.TimeoutMS)
	}
	// Absent means "use the tool's profile", not "no limit".
	absent, _ := buildPythonRequest(map[string]any{"code": "print(1)"})
	if absent.TimeoutMS != 0 {
		t.Errorf("an absent timeout became %d; it must defer to the tool profile", absent.TimeoutMS)
	}
}

func TestPythonRequestRejectsMissingOrNonStringCode(t *testing.T) {
	cases := map[string]map[string]any{
		"absent":     {},
		"whitespace": {"code": "   \n\t"},
		"number":     {"code": 42},
		"object":     {"code": map[string]any{"src": "print(1)"}},
	}
	for name, args := range cases {
		request, refusal := buildPythonRequest(args)
		if refusal == nil {
			t.Errorf("%s code was accepted; it should be refused", name)
			continue
		}
		if !refusal.IsError || strings.TrimSpace(refusal.ForLLM) == "" {
			t.Errorf("%s code produced an unusable error: %+v", name, refusal)
		}
		if request.Stdin != "" {
			t.Errorf("%s code still produced a request", name)
		}
	}
}

func TestPythonResultKeepsTheTracebackAndExplainsExitCode(t *testing.T) {
	report := formatPythonResult(&pcruntime.ExecResult{
		Tool: "python", ExitCode: 1, DurationMS: 12, Status: pcruntime.StatusCompleted,
		Stderr: "Traceback (most recent call last):\n  File \"<stdin>\", line 1\nValueError: TEST-ERROR\n",
	})
	for _, want := range []string{
		"Traceback (most recent call last):", "ValueError: TEST-ERROR", "exited 1",
		"read the last line for the exception type",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("the report does not contain %q:\n%s", want, report)
		}
	}
}

func TestPythonResultExplainsTimeoutAndCancellation(t *testing.T) {
	timedOut := formatPythonResult(&pcruntime.ExecResult{
		Tool: "python", ExitCode: -1, DurationMS: 300,
		Status: pcruntime.StatusTimeout, TimedOut: true, Stdout: "partial\n",
	})
	if !strings.Contains(timedOut, "time budget") {
		t.Errorf("a timeout was not explained:\n%s", timedOut)
	}
	if !strings.Contains(timedOut, "partial") {
		t.Error("output produced before the timeout was dropped")
	}
	if !strings.Contains(timedOut, "child processes were terminated") {
		t.Error("the timeout report does not mention that children were reaped")
	}

	cancelled := formatPythonResult(&pcruntime.ExecResult{
		Tool: "python", Status: pcruntime.StatusCancelled, Cancelled: true,
	})
	if !strings.Contains(cancelled, "cancelled") {
		t.Errorf("a cancelled run was not explained:\n%s", cancelled)
	}
}

func TestPythonResultReportsTruncatedOutput(t *testing.T) {
	report := formatPythonResult(&pcruntime.ExecResult{
		Tool: "python", Status: pcruntime.StatusCompleted,
		Stdout: "lots of output\n", StdoutTruncated: true,
		Stderr: "lots of noise\n", StderrTruncated: true,
	})
	if strings.Count(report, "truncated") < 2 {
		t.Errorf("truncation of stdout and stderr was not both reported:\n%s", report)
	}
	if !strings.Contains(report, "rather than assuming this is all of it") {
		t.Errorf("truncation was reported without telling the model what it means:\n%s", report)
	}
}

func TestPythonUnavailableMessageIsActionable(t *testing.T) {
	message := pythonUnavailableMessage("libpocketclaw-python.so was not unpacked")
	lower := strings.ToLower(message)
	for _, want := range []string{"not available", "cannot be installed", "runtime tool"} {
		if !strings.Contains(lower, want) {
			t.Errorf("the unavailable message does not mention %q:\n%s", want, message)
		}
	}
	if !strings.Contains(message, "libpocketclaw-python.so was not unpacked") {
		t.Error("the unavailable message dropped the runtime's own diagnostics")
	}
}

// The description is what stops the model reaching for a 90ms interpreter to do
// jq's job, and what stops it believing Python is contained.
func TestPythonDescriptionSteersAndDoesNotClaimASandbox(t *testing.T) {
	description := (&PythonTool{}).Description()
	lower := strings.ToLower(description)

	for _, preferred := range []string{"jq", "rg", "sqlite3", "curl"} {
		if !strings.Contains(description, preferred) {
			t.Errorf("the description does not point at %s for the work it suits", preferred)
		}
	}
	for _, when := range []string{"statistics", "algorithm", "round-trip"} {
		if !strings.Contains(lower, when) {
			t.Errorf("the description does not say when Python is the right choice (%q)", when)
		}
	}
	if !strings.Contains(lower, "not a sandbox") {
		t.Error("the description does not state that Python is not a sandbox")
	}
	for _, overclaim := range []string{
		"sandboxed", "safely isolated", "isolated environment",
		"cannot access the filesystem", "network sandbox", "memory limit",
	} {
		if strings.Contains(lower, overclaim) {
			t.Errorf("the description claims containment Python does not have: %q", overclaim)
		}
	}
	if !strings.Contains(lower, "not available, and not installable") {
		t.Error("the description does not say pip and third-party packages are unavailable")
	}
	for _, absent := range []string{"pip", "ctypes", "socket"} {
		if !strings.Contains(lower, absent) {
			t.Errorf("the description does not name %q as unavailable", absent)
		}
	}
}

func TestPythonToolSharesTheRuntimeManager(t *testing.T) {
	runtimeTool := &RuntimeTool{}
	if NewPythonTool(runtimeTool).manager != runtimeTool.manager {
		t.Error("the Python tool built its own manager; it must share the runtime's")
	}
}

// The reported gap was a presentation gap: the run was fine and the model could
// not see why it failed. The contract is therefore the field list itself, and
// it is asserted by name so it cannot quietly shrink again.
func TestPythonResultNamesEveryMeasuredField(t *testing.T) {
	report := formatPythonResult(&pcruntime.ExecResult{
		Tool: "python", ExitCode: 2, DurationMS: 5, Status: pcruntime.StatusCompleted,
		Stdout: "out\n", Stderr: "err\n",
	})
	for _, field := range []string{
		"exit_code=2", "timed_out=false", "cancelled=false",
		"stdout_truncated=false", "stderr_truncated=false",
		"stdout:", "stderr:",
	} {
		if !strings.Contains(report, field) {
			t.Errorf("the report does not state %q:\n%s", field, report)
		}
	}
}

// Diagnostics explain a non-completed status. They belong in the report, but
// never in place of what the process actually wrote.
func TestPythonResultKeepsStderrAlongsideDiagnostics(t *testing.T) {
	report := formatPythonResult(&pcruntime.ExecResult{
		Tool: "python", ExitCode: -1, DurationMS: 2000,
		Status: pcruntime.StatusTimeout, TimedOut: true,
		Stderr:      "Traceback (most recent call last):\nKeyboardInterrupt\n",
		Diagnostics: "python exceeded its 2s budget and was terminated",
	})
	if !strings.Contains(report, "KeyboardInterrupt") {
		t.Errorf("stderr was replaced by the runtime's diagnostics:\n%s", report)
	}
	if !strings.Contains(report, "exceeded its 2s budget") {
		t.Errorf("the runtime's diagnostics were dropped:\n%s", report)
	}
	if !strings.Contains(report, "timed_out=true") {
		t.Errorf("the timeout flag was not reported:\n%s", report)
	}
}
