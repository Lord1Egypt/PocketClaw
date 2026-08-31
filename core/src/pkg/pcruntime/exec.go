package pcruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ExecStatus is the terminal state of a managed execution. Every operation ends
// in exactly one of these.
type ExecStatus string

const (
	StatusCompleted   ExecStatus = "completed"
	StatusFailed      ExecStatus = "failed"
	StatusTimeout     ExecStatus = "timeout"
	StatusCancelled   ExecStatus = "cancelled"
	StatusUnavailable ExecStatus = "unavailable"
)

// terminationGrace is how long a timed-out or cancelled child gets to exit on
// SIGTERM before it is killed. Tools that write files deserve the chance to
// finish the write; nothing gets to ignore the signal indefinitely.
const terminationGrace = 2 * time.Second

// envKeyPattern is the shape of a legal environment variable name.
var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// inheritedEnvKeys are the only parent variables a managed tool sees. The base
// environment is constructed rather than inherited so that provider keys and
// channel tokens held by the Core process cannot reach a child by accident.
var inheritedEnvKeys = []string{
	"PATH", "HOME", "TMPDIR", "TMP", "TEMP",
	"LANG", "LC_ALL", "TZ",
	"ANDROID_DATA", "ANDROID_ROOT", "ANDROID_STORAGE", "EXTERNAL_STORAGE",
	"SSL_CERT_DIR", "SSL_CERT_FILE",
	"NO_COLOR", "TERM",
}

// deniedEnvAdditions are variables a caller may never set. Each of these changes
// which code the loader runs, which would let a caller substitute a different
// binary for the one the registry verified.
var deniedEnvAdditions = map[string]struct{}{
	"LD_PRELOAD": {}, "LD_LIBRARY_PATH": {}, "LD_AUDIT": {},
	"DYLD_INSERT_LIBRARIES": {}, "DYLD_LIBRARY_PATH": {},
	"PATH": {},
}

// deniedEnvPrefixes are namespaces a caller may never write into. Every PYTHON*
// variable is execution control: PYTHONPATH and PYTHONHOME decide which code the
// interpreter imports, PYTHONSTARTUP and PYTHONINSPECT run code of their own,
// and PYTHONUSERBASE reintroduces a writable import location. Allowing any of
// them would let a caller replace the standard library the registry verified.
var deniedEnvPrefixes = []string{"PYTHON"}

// ExecRequest is a request to run one managed tool.
type ExecRequest struct {
	// Tool is a catalog tool id or a command name it answers to.
	Tool string
	// Args are passed to the tool verbatim. They are never concatenated into a
	// command line and never reach a shell.
	Args []string
	// WorkingDirectory must be inside the user workspace. Empty means the
	// workspace root.
	WorkingDirectory string
	// TimeoutMS may lower the tool's timeout profile but never raise it.
	TimeoutMS int64
	// Stdin is fed to the tool and closed.
	Stdin string
	// EnvironmentAdditions are added to the constructed base environment.
	EnvironmentAdditions map[string]string
}

// ExecResult is the complete record of one managed execution.
type ExecResult struct {
	OperationID     string
	Tool            string
	ResolvedVersion string
	ExitCode        int
	Stdout          string
	Stderr          string
	DurationMS      int64
	TimedOut        bool
	Cancelled       bool
	StdoutTruncated bool
	StderrTruncated bool
	Status          ExecStatus
	// Diagnostics explains a non-completed status in terms the agent can act
	// on. It is empty on success.
	Diagnostics string
}

// Manager is the Managed Runtime API. It is the only supported way to run a
// managed tool: it owns the registry, the execution policy, and the lifecycle
// logging, so no caller can obtain a tool path and run it unobserved.
type Manager struct {
	registry  *Registry
	workspace string
}

// NewManager builds the runtime from the process environment.
func NewManager() (*Manager, error) {
	registry, err := NewRegistry()
	if err != nil {
		return nil, err
	}
	return NewManagerWith(registry, registry.Paths().Workspace)
}

// NewManagerWith builds a runtime over an explicit registry and workspace root.
func NewManagerWith(registry *Registry, workspace string) (*Manager, error) {
	if registry == nil {
		return nil, fmt.Errorf("runtime manager needs a registry")
	}
	if strings.TrimSpace(workspace) == "" {
		return nil, fmt.Errorf("runtime manager needs a workspace root for its working-directory policy")
	}
	absolute, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve workspace root %q: %w", workspace, err)
	}
	return &Manager{registry: registry, workspace: filepath.Clean(absolute)}, nil
}

// Registry exposes the tool catalog for inventory and resolution queries.
func (m *Manager) Registry() *Registry { return m.registry }

// EnsureTool reports whether a catalog tool is usable on this device.
//
// It acquires nothing. On Android an executable can only arrive through the
// APK or the system image, both of which are fixed at install time, so there is
// no acquisition step for this call to perform — the honest answer for anything
// not already present is that the capability is unavailable, with the reason.
func (m *Manager) EnsureTool(name string) (*ResolvedTool, error) {
	emitDebug(EventProvisionStarted, map[string]any{"tool": name, "stage": "ensure"})

	resolved, err := m.registry.Resolve(name)
	if err != nil {
		emitWarn(EventProvisionUnsupported, map[string]any{
			"tool":   name,
			"status": string(AvailabilityUnavailable),
			"reason": string(ReasonNotInCatalog),
			"diagnostics": "PocketClaw does not support this tool; the runtime cannot install binaries, " +
				"because Android forbids executing anything outside the APK payload and the system image",
		})
		return nil, err
	}
	if !resolved.Available() {
		emitInfo(EventProvisionUnsupported, map[string]any{
			"tool":        resolved.Tool.ToolID,
			"status":      string(AvailabilityUnavailable),
			"reason":      string(resolved.UnavailableReason),
			"diagnostics": resolved.Diagnostics,
		})
	}
	return resolved, nil
}

// Execute runs one managed tool under bounded execution.
//
// It always returns a result. A nil error means the operation was carried out
// and its outcome is in Status; a non-nil error means the request itself was
// rejected before anything ran.
func (m *Manager) Execute(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	operationID := newOperationID()
	result := &ExecResult{OperationID: operationID, Tool: strings.TrimSpace(req.Tool)}

	emitInfo(EventExecQueued, map[string]any{
		"operation_id": operationID,
		"tool":         result.Tool,
		"stage":        "queued",
	})

	if !hostSupportsManagedExecution() {
		return m.rejected(result, ReasonUnsupportedRuntime,
			"the managed runtime does not run managed tools on this operating system")
	}

	resolved, err := m.registry.Resolve(result.Tool)
	if err != nil {
		if errors.Is(err, ErrNotInCatalog) {
			return m.rejected(result, ReasonNotInCatalog, fmt.Sprintf(
				"%q is not a PocketClaw runtime tool. The runtime cannot install it: "+
					"Android only executes binaries shipped in the APK or the system image.",
				result.Tool,
			))
		}
		return nil, err
	}
	if !resolved.Available() {
		return m.rejected(result, resolved.UnavailableReason, resolved.Diagnostics)
	}

	result.Tool = resolved.Tool.ToolID
	result.ResolvedVersion = resolved.ResolvedVersion

	workingDir, err := m.resolveWorkingDirectory(req.WorkingDirectory)
	if err != nil {
		return nil, err
	}
	timeout, err := resolveTimeout(&resolved.Tool, req.TimeoutMS)
	if err != nil {
		return nil, err
	}
	prepared, err := m.prepareEnvironment(resolved)
	if err != nil {
		return nil, err
	}
	environment, err := buildEnvironment(prepared, req.EnvironmentAdditions)
	if err != nil {
		return nil, err
	}

	argv := buildArgv(resolved, req.Tool, req.Args)
	return m.run(ctx, resolved, result, runPlan{
		argv:        argv,
		workingDir:  workingDir,
		timeout:     timeout,
		environment: environment,
		stdin:       req.Stdin,
		prepared:    prepared,
	}), nil
}

type runPlan struct {
	argv        []string
	workingDir  string
	timeout     time.Duration
	environment []string
	stdin       string
	prepared    *preparedEnvironment
}

func (m *Manager) run(
	ctx context.Context,
	resolved *ResolvedTool,
	result *ExecResult,
	plan runPlan,
) *ExecResult {
	redactedArgv := RedactArgv(resolved.Tool.ToolID, plan.argv[1:])

	// The caller's context governs cancellation; the derived one adds the
	// timeout ceiling. Both are cancelled on the way out so no watchdog leaks.
	runCtx, cancel := context.WithTimeout(ctx, plan.timeout)
	defer cancel()

	cmd := exec.Command(resolved.ExecutablePath)
	// Args[0] is set explicitly rather than left to exec.Command so a multicall
	// payload is invoked under the applet name the caller asked for.
	cmd.Args = plan.argv
	cmd.Dir = plan.workingDir
	cmd.Env = plan.environment
	cmd.Stdin = strings.NewReader(plan.stdin)
	// stdout/stderr are captured through pipes, so Wait also waits for those
	// pipes to close. A grandchild that escaped the process group would hold
	// them open and block Wait forever; WaitDelay closes them once the direct
	// child is gone.
	cmd.WaitDelay = terminationGrace
	isolateProcessGroup(cmd)

	stdout := newBoundedBuffer(resolved.Tool.MaxOutputBytes)
	stderr := newBoundedBuffer(resolved.Tool.MaxOutputBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	baseFields := map[string]any{
		"operation_id":  result.OperationID,
		"tool":          resolved.Tool.ToolID,
		"tool_version":  resolved.ResolvedVersion,
		"source":        string(resolved.Tool.Delivery),
		"argv_redacted": redactedArgv,
		"timeout_ms":    plan.timeout.Milliseconds(),
		"bytes_in":      int64(len(plan.stdin)),
	}
	emitInfo(EventExecStarted, mergeFields(baseFields, map[string]any{"stage": "starting"}))

	started := time.Now()
	if err := cmd.Start(); err != nil {
		result.Status = StatusFailed
		result.ExitCode = -1
		result.DurationMS = time.Since(started).Milliseconds()
		result.Diagnostics = describeStartFailure(resolved, err)
		emitError(EventExecFailed, mergeFields(baseFields, map[string]any{
			"status":      string(StatusFailed),
			"duration_ms": result.DurationMS,
			"diagnostics": result.Diagnostics,
		}))
		return result
	}

	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()

	var waitErr error
	select {
	case waitErr = <-waited:
	case <-runCtx.Done():
		// Terminate the group, then keep waiting on every path: the child must
		// be reaped or it stays a zombie holding its pid.
		waitErr = terminateAndReap(cmd, waited)

		if errors.Is(runCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			result.TimedOut = true
		} else {
			result.Cancelled = true
		}
	}

	result.DurationMS = time.Since(started).Milliseconds()
	result.ExitCode = exitCodeOf(cmd, waitErr)
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.StdoutTruncated = stdout.Truncated()
	result.StderrTruncated = stderr.Truncated()

	// Output volume is logged; output content is not. The bytes are useful for
	// diagnosis, and a tool's stdout is exactly where a fetched credential would
	// appear — `gh auth token` prints one.
	outcomeFields := mergeFields(baseFields, map[string]any{
		"duration_ms":      result.DurationMS,
		"exit_code":        result.ExitCode,
		"bytes_out":        stdout.TotalBytes(),
		"stdout_truncated": result.StdoutTruncated,
		"stderr_truncated": result.StderrTruncated,
	})
	emitDebug(EventExecStdout, map[string]any{
		"operation_id": result.OperationID,
		"tool":         resolved.Tool.ToolID,
		"bytes_out":    stdout.TotalBytes(),
		"truncated":    result.StdoutTruncated,
	})
	emitDebug(EventExecStderr, map[string]any{
		"operation_id": result.OperationID,
		"tool":         resolved.Tool.ToolID,
		"bytes_out":    stderr.TotalBytes(),
		"truncated":    result.StderrTruncated,
	})

	switch {
	case result.TimedOut:
		result.Status = StatusTimeout
		result.Diagnostics = fmt.Sprintf(
			"%s exceeded its %s budget and was terminated",
			resolved.Tool.ToolID, plan.timeout,
		)
		emitWarn(EventExecTimeout, mergeFields(outcomeFields, map[string]any{
			"status": string(StatusTimeout), "timed_out": true,
		}))
	case result.Cancelled:
		result.Status = StatusCancelled
		result.Diagnostics = fmt.Sprintf("%s was cancelled", resolved.Tool.ToolID)
		emitWarn(EventExecCancelled, mergeFields(outcomeFields, map[string]any{
			"status": string(StatusCancelled), "cancelled": true,
		}))
	default:
		// A non-zero exit is the tool's answer, not a runtime failure. The
		// operation completed either way; the caller reads ExitCode.
		result.Status = StatusCompleted
		emitInfo(EventExecCompleted, mergeFields(outcomeFields, map[string]any{
			"status": string(StatusCompleted),
		}))
	}

	emitDebug(EventCleanupCompleted, map[string]any{
		"operation_id": result.OperationID,
		"tool":         resolved.Tool.ToolID,
		"stage":        "cleanup",
	})
	return result
}

// terminateAndReap asks the process group to exit, kills it if it does not, and
// returns only once the child has been reaped.
func terminateAndReap(cmd *exec.Cmd, waited <-chan error) error {
	emitDebug(EventCleanupStarted, map[string]any{
		"pid":   cmd.Process.Pid,
		"stage": "terminate",
	})
	if err := terminateProcessTree(cmd, gracefulSignal()); err != nil {
		emitDebug(EventCleanupFailed, map[string]any{
			"pid": cmd.Process.Pid, "diagnostics": err.Error(),
		})
	}

	grace := time.NewTimer(terminationGrace)
	defer grace.Stop()
	select {
	case err := <-waited:
		return err
	case <-grace.C:
		_ = terminateProcessTree(cmd, forcefulSignal())
		return <-waited
	}
}

func (m *Manager) rejected(
	result *ExecResult, reason UnavailableReason, diagnostics string,
) (*ExecResult, error) {
	result.Status = StatusUnavailable
	result.ExitCode = -1
	result.Diagnostics = diagnostics
	emitInfo(EventExecFailed, map[string]any{
		"operation_id": result.OperationID,
		"tool":         result.Tool,
		"status":       string(StatusUnavailable),
		"reason":       string(reason),
		"diagnostics":  diagnostics,
	})
	return result, nil
}

// resolveWorkingDirectory enforces the working-directory policy: managed tools
// run inside the user workspace and nowhere else. Runtime metadata storage is
// deliberately not reachable, so a tool cannot rewrite the runtime's own records.
func (m *Manager) resolveWorkingDirectory(requested string) (string, error) {
	if strings.TrimSpace(requested) == "" {
		return m.workspace, nil
	}
	candidate := requested
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(m.workspace, candidate)
	}
	candidate = filepath.Clean(candidate)

	info, err := os.Stat(candidate)
	if err != nil {
		return "", fmt.Errorf("working directory %q does not exist", requested)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("working directory %q is not a directory", requested)
	}

	// Containment is checked on the resolved paths. Comparing the literal ones
	// would let a symlink inside the workspace point anywhere on the device and
	// still pass. The workspace root is resolved too, because Android's external
	// storage paths are themselves reached through symlinks.
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("cannot resolve working directory %q: %w", requested, err)
	}
	root := m.workspace
	if resolvedRoot, err := filepath.EvalSymlinks(root); err == nil {
		root = resolvedRoot
	}

	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf(
			"working directory %q is outside the workspace; managed tools run inside %s",
			requested, m.workspace,
		)
	}
	return resolved, nil
}

// resolveTimeout applies the tool's profile as a ceiling. A caller may ask for
// less time; nothing may ask for more, so no managed process can outlive its
// declared budget.
func resolveTimeout(tool *Tool, requestedMS int64) (time.Duration, error) {
	ceiling, ok := tool.TimeoutProfile.Duration()
	if !ok {
		return 0, fmt.Errorf("tool %q has no usable timeout profile", tool.ToolID)
	}
	if requestedMS <= 0 {
		return ceiling, nil
	}
	requested := time.Duration(requestedMS) * time.Millisecond
	if requested > ceiling {
		return ceiling, nil
	}
	return requested, nil
}

// buildEnvironment constructs the child environment from an allowlist, then
// applies the tool's prepared profile, then the caller's additions.
//
// The order matters. Profile values come from the runtime and may legitimately
// set variables a caller may not, such as PATH for gh; caller additions are
// applied last but are still checked against the denied list, so a caller
// cannot undo the loader protections by overwriting a profile value.
func buildEnvironment(prepared *preparedEnvironment, additions map[string]string) ([]string, error) {
	resolved := make(map[string]string, len(inheritedEnvKeys)+len(additions))
	ordered := make([]string, 0, len(inheritedEnvKeys)+len(additions))
	put := func(key, value string) {
		if _, seen := resolved[key]; !seen {
			ordered = append(ordered, key)
		}
		resolved[key] = value
	}

	for _, key := range inheritedEnvKeys {
		if value, present := os.LookupEnv(key); present {
			put(key, value)
		}
	}
	if prepared != nil {
		for key, value := range prepared.variables {
			put(key, value)
		}
	}

	for key, value := range additions {
		if !envKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("%q is not a valid environment variable name", key)
		}
		if _, denied := deniedEnvAdditions[key]; denied {
			return nil, fmt.Errorf(
				"%s may not be set for a managed tool: it would change which binary the loader runs",
				key,
			)
		}
		for _, prefix := range deniedEnvPrefixes {
			if strings.HasPrefix(key, prefix) {
				return nil, fmt.Errorf(
					"%s may not be set for a managed tool: the %s* namespace controls which code the interpreter runs",
					key, prefix,
				)
			}
		}
		put(key, value)
	}

	environment := make([]string, 0, len(ordered))
	for _, key := range ordered {
		environment = append(environment, key+"="+resolved[key])
	}
	return environment, nil
}

// buildArgv assembles the argument vector. argv[0] is the name the caller asked
// for, which is what lets one multicall payload answer to several commands.
func buildArgv(resolved *ResolvedTool, requestedName string, args []string) []string {
	argv0 := resolved.Tool.CommandName
	for _, applet := range resolved.Tool.Applets {
		if applet == requestedName {
			argv0 = applet
			break
		}
	}
	argv := make([]string, 0, len(args)+len(resolved.Tool.DefaultArgs)+1)
	argv = append(argv, argv0)
	// Catalog arguments come first so a caller cannot get in front of them.
	argv = append(argv, resolved.Tool.DefaultArgs...)
	argv = append(argv, args...)
	return argv
}

func describeStartFailure(resolved *ResolvedTool, err error) string {
	if errors.Is(err, os.ErrPermission) && resolved.Tool.Delivery == DeliveryBundled {
		return fmt.Sprintf(
			"%s could not be executed from %s. On Android an executable must live in the "+
				"installer-unpacked native library directory; a copy in writable app storage "+
				"is refused by the platform regardless of its mode bits. Original error: %v",
			resolved.Tool.ToolID, resolved.ExecutablePath, err,
		)
	}
	return fmt.Sprintf("%s could not be started: %v", resolved.Tool.ToolID, err)
}

func exitCodeOf(cmd *exec.Cmd, waitErr error) int {
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	if waitErr != nil {
		return -1
	}
	return 0
}

func mergeFields(base, extra map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func newOperationID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		// A collision here would only confuse a log reader, and refusing to run
		// a tool because the CSPRNG hiccuped would be worse.
		return fmt.Sprintf("rt-%d", time.Now().UnixNano())
	}
	return "rt-" + hex.EncodeToString(buf)
}
