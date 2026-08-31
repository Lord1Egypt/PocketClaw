package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/pcruntime"
)

// pythonBootstrapModule is PocketClaw's entry point inside the payload.
//
// Android's CPython does not connect sys.stdout and sys.stderr to descriptors 1
// and 2: it replaces both with TextLogStream, which writes to the Android system
// log. The runtime captures the descriptors, so a device ran every program with
// its output going somewhere the caller could not see — print() was a silent
// no-op that still exited 0, and an uncaught exception's traceback never
// appeared even though the exit status was 1.
//
// The bootstrap restores the two streams and then reads the program from
// standard input exactly as `python -` does. It ships inside the payload, so the
// checksum the registry verifies covers it, and it is the only code here that
// PocketClaw supplies: the program itself is still the caller's, and still
// arrives on stdin.
const pythonBootstrapModule = "pocketclaw_bootstrap"

// pythonModuleFlag runs a module as __main__. Everything after the module name
// becomes sys.argv[1:], which is what lets a caller pass data without the
// runtime ever interpolating it into the source.
const pythonModuleFlag = "-m"

// PythonTool runs Python through the Managed Runtime.
//
// It owns no execution machinery. The tool builds an ordinary ExecRequest and
// hands it to the same Manager.Execute that every other managed tool uses, so
// resolution, checksum verification, the environment profile, the timeout
// ceiling, cancellation, process-group termination, output bounds, the runtime
// event family and redaction all apply unchanged.
//
// Source travels on standard input, never in the argument vector. argv is
// capped near 128 KB, is readable from /proc/<pid>/cmdline, and appears in
// argument diagnostics; stdin is size-accounted as bytes_in and never recorded.
type PythonTool struct {
	manager *pcruntime.Manager
}

// NewPythonTool shares the Managed Runtime the agent already built, so there is
// one registry, one platform probe and one startup diagnostic for the process.
func NewPythonTool(runtime *RuntimeTool) *PythonTool {
	return &PythonTool{manager: runtime.manager}
}

func (t *PythonTool) Name() string { return "python" }

func (t *PythonTool) Description() string {
	return `Run a Python 3 program. The code is passed on standard input; there is no shell.

Python is the flexible fallback, not the first answer to every task. When one of
the runtime's direct tools clearly fits, that tool is faster and more
predictable:

  jq       simple JSON filters and transforms
  rg       text and code search
  sqlite3  a single SQL query against a database file
  curl     HTTP and API requests

Reach for Python when the work actually needs a program:

  arithmetic, statistics and numeric work
  multi-step logic and custom algorithms
  combining several transformations in one pass
  CSV, JSON and XML conversion between shapes
  generating a structured file
  custom parsing that a filter cannot express
  SQLite work where the DB-API materially helps
  anything that would otherwise take several runtime round-trips

Available: json, csv, re, math, statistics, decimal, fractions, datetime,
pathlib, hashlib, hmac, secrets, uuid, urllib.parse, html, xml.etree, zipfile,
tarfile, gzip, bz2, lzma, sqlite3, tempfile, shutil, glob, fnmatch, argparse,
subprocess, textwrap, base64, unicodedata.

Not available, and not installable: pip and third-party packages, ctypes, socket,
ssl, http, urllib.request, email and multiprocessing. Python here has no direct
network access; use curl through the runtime for HTTP, and git or gh for
repository work.

Print what you want to see. Only stdout and stderr come back, both bounded, and
a returned value is not shown.

This is not a sandbox. Python runs as the PocketClaw application itself and can
read and write anything the app can, including starting other processes. The
runtime bounds how long it runs, terminates it and its children on timeout or
cancellation, limits how much output it can return and controls its environment
— but it does not isolate the filesystem, other processes, the network or
memory. Write code you would be willing to run as the app.`
}

func (t *PythonTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "Python source to run. Print results; a returned value is not shown.",
			},
			"args": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Values passed as sys.argv[1:]. Use these instead of building data into the source.",
			},
			"timeout_ms": map[string]any{
				"type": "integer",
				"description": "Lower the runtime's timeout budget for this run. " +
					"It cannot be raised above the platform maximum.",
			},
		},
		"required": []string{"code"},
	}
}

// buildPythonRequest turns tool arguments into a runtime request, or explains
// why they cannot be. It is separate from Execute so the request itself can be
// asserted: where the source travels is the security-relevant decision here, and
// it should be provable without running an interpreter.
func buildPythonRequest(args map[string]any) (pcruntime.ExecRequest, *ToolResult) {
	code, ok := args["code"].(string)
	if !ok {
		if args["code"] == nil {
			return pcruntime.ExecRequest{}, ErrorResult(
				"code is required: pass the Python source to run.")
		}
		return pcruntime.ExecRequest{}, ErrorResult(fmt.Sprintf(
			"code must be a string containing Python source, not %T.", args["code"]))
	}
	if strings.TrimSpace(code) == "" {
		return pcruntime.ExecRequest{}, ErrorResult(
			"code is empty: pass the Python source to run.")
	}

	// The bootstrap leads, so the interpreter runs PocketClaw's entry point and
	// every later argument lands in sys.argv rather than being read as source.
	// The bootstrap restores sys.argv[0] to "-", which is what `python -` sets.
	argv := append([]string{pythonModuleFlag, pythonBootstrapModule},
		stringSliceArg(args["args"])...)

	return pcruntime.ExecRequest{
		Tool:      "python",
		Args:      argv,
		Stdin:     code,
		TimeoutMS: intArg(args["timeout_ms"]),
	}, nil
}

func (t *PythonTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	request, refusal := buildPythonRequest(args)
	if refusal != nil {
		return refusal
	}

	result, err := t.manager.Execute(ctx, request)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if result.Status == pcruntime.StatusUnavailable {
		return ErrorResult(pythonUnavailableMessage(result.Diagnostics))
	}
	return UserResult(formatPythonResult(result))
}

// formatPythonResult renders the complete outcome of one run.
//
// Every field the runtime measured is named and always present, an empty stream
// included. A model that cannot tell "the interpreter printed nothing" from
// "the result dropped it" has to guess, and guessing about a traceback is the
// one thing this tool must never require. The streams are the runtime's own
// bounded, redacted capture: nothing here reconstructs, summarises or replaces
// what the process actually wrote.
func formatPythonResult(result *pcruntime.ExecResult) string {
	var report strings.Builder
	fmt.Fprintf(&report, "%s exited %d after %dms (%s)\n",
		result.Tool, result.ExitCode, result.DurationMS, result.Status)
	fmt.Fprintf(&report,
		"exit_code=%d timed_out=%t cancelled=%t stdout_truncated=%t stderr_truncated=%t\n",
		result.ExitCode, result.TimedOut, result.Cancelled,
		result.StdoutTruncated, result.StderrTruncated)
	// What the runtime measured coming out of the process, next to what it is
	// showing. If these disagree — bytes counted but no text below — the loss is
	// downstream of the capture, and the report says so itself instead of
	// needing a device log to find out.
	fmt.Fprintf(&report, "stdout_bytes=%d stderr_bytes=%d\n",
		result.StdoutBytes, result.StderrBytes)

	writeStream(&report, "stdout", result.Stdout, result.StdoutTruncated,
		"[stdout truncated at the tool's output limit; "+
			"narrow the output rather than assuming this is all of it]")
	// stderr is printed on every path, including a zero exit: a traceback, a
	// warning and a silent success are three different answers.
	writeStream(&report, "stderr", result.Stderr, result.StderrTruncated,
		"[stderr truncated at the tool's output limit; the traceback above may be "+
			"cut off, and its last line may not be the exception line]")

	if result.Diagnostics != "" {
		fmt.Fprintf(&report, "\n%s\n", result.Diagnostics)
	}

	switch {
	case result.TimedOut:
		fmt.Fprintf(&report,
			"\nPython was stopped after %dms because it ran past its time budget, and its "+
				"child processes were terminated with it. Any output above is what it had "+
				"produced by then. Make the work smaller or bounded rather than retrying "+
				"the same program.\n", result.DurationMS)
	case result.Cancelled:
		report.WriteString("\nThe run was cancelled before it finished.\n")
	case result.ExitCode != 0 && result.StdoutBytes == 0 && result.StderrBytes == 0:
		// A failing program produces a traceback. Silence with a non-zero status
		// means the interpreter never reached the program: PocketClaw's entry
		// point could not be imported, and CPython reported that on Android's
		// system log rather than on a descriptor anyone here can read. Saying so
		// is the difference between a diagnosable install fault and a repeat of
		// the bug this entry point exists to fix.
		fmt.Fprintf(&report,
			"\nPython exited %d and wrote nothing to either stream. A program that fails "+
				"produces a traceback, so this is not the program failing: the interpreter "+
				"could not start PocketClaw's %s entry point, which means the packaged "+
				"payload is not the one the catalog pins. Report it; retrying will not "+
				"change it.\n", result.ExitCode, pythonBootstrapModule)
	case result.ExitCode != 0:
		report.WriteString("\nPython exited non-zero. If a traceback is shown above, it is the " +
			"real error: read the last line for the exception type and message. " +
			"Line numbers refer to the code you supplied.\n")
	}
	return report.String()
}

// writeStream renders one captured stream under its own name. An empty stream
// is stated rather than omitted, so silence is distinguishable from a field
// that never made it into the report.
func writeStream(report *strings.Builder, label, text string, truncated bool, note string) {
	fmt.Fprintf(report, "\n%s:\n", label)
	if text == "" {
		report.WriteString("(empty)\n")
		return
	}
	report.WriteString(text)
	if !strings.HasSuffix(text, "\n") {
		report.WriteString("\n")
	}
	if truncated {
		report.WriteString(note + "\n")
	}
}

func pythonUnavailableMessage(diagnostics string) string {
	message := "Python is not available on this device.\n\n" +
		"It ships inside PocketClaw as a managed runtime payload, so it cannot be " +
		"installed or downloaded. Use the runtime tool to see what this device does " +
		"provide, and solve the task with those tools."
	if strings.TrimSpace(diagnostics) != "" {
		message += "\n\n" + diagnostics
	}
	return message
}
