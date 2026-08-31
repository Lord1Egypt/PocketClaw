package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/pcruntime"
)

// pythonStdinMarker makes the interpreter read the program from standard input.
//
// It must stay the first argument. Everything after it becomes sys.argv[1:],
// which is what lets a caller pass data without the runtime ever interpolating
// it into the source.
const pythonStdinMarker = "-"

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

	// "-" first, so the interpreter reads the program from stdin and every
	// later argument lands in sys.argv rather than being read as source.
	argv := append([]string{pythonStdinMarker}, stringSliceArg(args["args"])...)

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

// formatPythonResult reuses the runtime's own result rendering and adds only
// what a Python caller needs on top: why a run ended badly, in terms the model
// can act on. A traceback is left intact in stderr rather than being collapsed
// into a generic failure.
func formatPythonResult(result *pcruntime.ExecResult) string {
	var report strings.Builder
	report.WriteString(formatExecResult(result))

	switch {
	case result.TimedOut:
		fmt.Fprintf(&report,
			"\nPython was stopped after %dms because it ran past its time budget, and its "+
				"child processes were terminated with it. Any output above is what it had "+
				"produced by then. Make the work smaller or bounded rather than retrying "+
				"the same program.\n", result.DurationMS)
	case result.Cancelled:
		report.WriteString("\nThe run was cancelled before it finished.\n")
	case result.ExitCode != 0:
		report.WriteString("\nPython exited non-zero. If a traceback is shown above, it is the " +
			"real error: read the last line for the exception type and message. " +
			"Line numbers refer to the code you supplied.\n")
	}
	return report.String()
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
