package pcruntime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// WritableExecSupport is the measured answer to the question the whole delivery
// model rests on: may this app execute a file it wrote itself?
type WritableExecSupport string

const (
	WritableExecSupported    WritableExecSupport = "supported"
	WritableExecBlocked      WritableExecSupport = "blocked"
	WritableExecInconclusive WritableExecSupport = "inconclusive"
)

// ExecutionProbe records what the device actually permits.
//
// PocketClaw's runtime never relies on writable-storage execution — bundled
// payloads live in the installer-unpacked native library directory precisely so
// that they do not. This probe exists to report the platform's behaviour rather
// than assert it, so the assumption behind the delivery model is visible in the
// Debug Logs of every device instead of being taken on trust.
type ExecutionProbe struct {
	GOOS               string              `json:"goos"`
	MetadataDir        string              `json:"metadata_dir"`
	LibDir             string              `json:"lib_dir"`
	WritableExec       WritableExecSupport `json:"writable_exec"`
	WritableExecDetail string              `json:"writable_exec_detail"`
	// SymlinkExec is whether a symlink in app-private storage pointing at a
	// packaged executable can be run. Tools with helper payloads depend on it.
	SymlinkExec       WritableExecSupport `json:"symlink_exec"`
	SymlinkExecDetail string              `json:"symlink_exec_detail"`
	ProbedAt          time.Time           `json:"probed_at"`
}

// probeSourceBinaries are harmless executables to copy for the probe, in order
// of preference. `true` does nothing and exits zero, which is exactly what a
// probe wants to run.
var probeSourceBinaries = []string{
	"/system/bin/true",
	"/system/bin/toybox",
	"/bin/true",
	"/usr/bin/true",
}

// ProbeExecution measures whether an executable copied into app-private
// writable storage can be run, and cleans up after itself.
func ProbeExecution(ctx context.Context, paths *Paths) *ExecutionProbe {
	probe := &ExecutionProbe{
		GOOS:        runtime.GOOS,
		MetadataDir: paths.MetadataDir,
		LibDir:      paths.LibDir,
		ProbedAt:    time.Now(),
	}
	emitDebug(EventProbeStarted, map[string]any{"stage": "writable_exec"})

	support, detail := measureWritableExec(ctx, paths)
	probe.WritableExec = support
	probe.WritableExecDetail = detail

	symlinkSupport, symlinkDetail := measureSymlinkExec(ctx, paths)
	probe.SymlinkExec = symlinkSupport
	probe.SymlinkExecDetail = symlinkDetail

	emitInfo(EventProbeCompleted, map[string]any{
		"stage":               "execution",
		"writable_exec":       string(support),
		"diagnostics":         detail,
		"symlink_exec":        string(symlinkSupport),
		"symlink_diagnostics": symlinkDetail,
		"lib_dir":             paths.LibDir,
	})
	return probe
}

// measureSymlinkExec answers the question git depends on.
//
// git looks its transport helper up as "git-remote-https" inside GIT_EXEC_PATH,
// and Android cannot package a file under that name: the installer only unpacks
// lib/<abi>/*.so. The runtime therefore builds a directory of symlinks pointing
// at the packaged payloads. A symlink is not itself an executable — the kernel
// resolves it and executes the read-only target — so this should be permitted
// where writing a real executable is not. "Should be" is why this is measured
// and not assumed.
func measureSymlinkExec(ctx context.Context, paths *Paths) (WritableExecSupport, string) {
	source := firstExisting(probeSourceBinaries)
	if source == "" {
		return WritableExecInconclusive,
			"no harmless system executable was available to link for the probe"
	}
	if err := paths.EnsureMetadataDir(); err != nil {
		return WritableExecInconclusive, err.Error()
	}

	link := filepath.Join(paths.MetadataDir, ".symlink-probe")
	_ = os.Remove(link)
	if err := os.Symlink(source, link); err != nil {
		return WritableExecInconclusive, fmt.Sprintf("could not create the probe symlink: %v", err)
	}
	defer func() { _ = os.Remove(link) }()

	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, link)
	cmd.Args = []string{"true"}
	cmd.Env = []string{}
	isolateProcessGroup(cmd)

	err := cmd.Run()
	switch {
	case err == nil:
		return WritableExecSupported, fmt.Sprintf(
			"a symlink in %s to %s executed; helper payloads such as "+
				"git-remote-https can be presented this way",
			paths.MetadataDir, source,
		)
	case errors.Is(err, os.ErrPermission):
		return WritableExecBlocked, fmt.Sprintf(
			"the platform refused to execute through a symlink in %s (%v). "+
				"Tools with helper payloads, git most of all, cannot resolve their "+
				"helpers this way on this device.",
			paths.MetadataDir, err,
		)
	default:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return WritableExecSupported, fmt.Sprintf(
				"a symlink in %s to %s executed and exited %d",
				paths.MetadataDir, source, exitErr.ExitCode(),
			)
		}
		return WritableExecInconclusive, fmt.Sprintf(
			"the symlink probe neither ran nor was refused for a recognisable reason: %v", err,
		)
	}
}

func measureWritableExec(ctx context.Context, paths *Paths) (WritableExecSupport, string) {
	source := firstExisting(probeSourceBinaries)
	if source == "" {
		return WritableExecInconclusive,
			"no harmless system executable was available to copy for the probe"
	}
	if err := paths.EnsureMetadataDir(); err != nil {
		return WritableExecInconclusive, err.Error()
	}

	target := filepath.Join(paths.MetadataDir, ".exec-probe")
	// A stale probe copy from a previous run must never be executed instead of
	// the one this run wrote.
	_ = os.Remove(target)
	if err := copyFileMode(source, target, 0o700); err != nil {
		return WritableExecInconclusive, fmt.Sprintf("could not stage the probe binary: %v", err)
	}
	defer func() { _ = os.Remove(target) }()

	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, target)
	// toybox with no applet name prints its command list and exits non-zero;
	// either way, reaching exec at all is the answer this probe wants.
	cmd.Args = []string{"true"}
	cmd.Env = []string{}
	isolateProcessGroup(cmd)

	err := cmd.Run()
	switch {
	case err == nil:
		return WritableExecSupported, fmt.Sprintf(
			"a copy of %s executed from %s", source, paths.MetadataDir,
		)
	case errors.Is(err, os.ErrPermission):
		return WritableExecBlocked, fmt.Sprintf(
			"the platform refused to execute a file written into %s (%v). "+
				"This is the expected result on Android 10 and later: an app targeting "+
				"API 29+ may not execve a file in its own writable storage, and the file "+
				"mode does not change that. Managed executables must ship in the APK.",
			paths.MetadataDir, err,
		)
	default:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// It ran. A non-zero exit is the copied binary's own answer.
			return WritableExecSupported, fmt.Sprintf(
				"a copy of %s executed from %s and exited %d",
				source, paths.MetadataDir, exitErr.ExitCode(),
			)
		}
		return WritableExecInconclusive, fmt.Sprintf(
			"the probe neither ran nor was refused for a recognisable reason: %v", err,
		)
	}
}

func firstExisting(candidates []string) string {
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0 {
			return candidate
		}
	}
	return ""
}

func copyFileMode(source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	// O_CREATE honours the umask, so the mode is set explicitly afterwards.
	return os.Chmod(target, mode)
}

// readDirNames lists a directory, used to confirm the probe cleaned up.
func readDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}
