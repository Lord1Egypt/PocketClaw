//go:build !windows

package pid

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// isProcessRunning checks whether a process with the given PID is alive
// on Unix-like systems using signal(0).
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal(0) does not kill the process but checks existence on Unix.
	err = p.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	var errno syscall.Errno
	// EPERM means the process exists but we are not allowed to signal it.
	return errors.As(err, &errno) && errno == syscall.EPERM
}

// procVerdict is what /proc/<pid>/comm says about a PID that isProcessRunning
// has already reported as alive.
type procVerdict int

const (
	// procMatch: comm was readable and names the PocketClaw runtime.
	procMatch procVerdict = iota
	// procForeign: comm was readable and names something else, so the PID was
	// reused by an unrelated process.
	procForeign
	// procNotVisible: the kernel will not show this PID to us at all.
	procNotVisible
	// procUnknown: the read failed for a reason that says nothing about who
	// owns the PID.
	procUnknown
)

// readProcComm is a seam so the classification can be tested without a real
// process to point at.
var readProcComm = func(pid int) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
}

// commVisibleBytes is how much of a process name /proc/<pid>/comm shows.
//
// The kernel's buffer is TASK_COMM_LEN, 16 bytes including the terminator, so
// 15 characters are visible. This is not a rounding detail here: the packaged
// gateway is libpocketclaw.so, 16 characters, so on a real device it is *always*
// truncated. Measured on the validated Samsung, not taken from a header —
// libpocketclaw.so reports libpocketclaw.s and libpocketclaw-web.so reports
// libpocketclaw-w.
const commVisibleBytes = 15

// ownedProcessNames are the two packaged Core executables whose live process a
// pid file may legitimately be referring to: the gateway and the launcher.
//
// COMPATIBILITY: these are the canonical staged names installed by
// core/build-android-arm64.sh. RECHECK AFTER FULL NAMESPACE MIGRATION.
var ownedProcessNames = []string{"libpocketclaw.so", "libpocketclaw-web.so"}

// commForm is what the kernel will show for an executable of this name.
func commForm(execName string) string {
	if len(execName) > commVisibleBytes {
		return execName[:commVisibleBytes]
	}
	return execName
}

// isOwnedComm reports whether a /proc/<pid>/comm value names one of our two
// Core executables.
//
// It compares against the whole name rather than searching for the product
// inside it, and that distinction is the fix for a real regression. A substring
// rule accepted anything containing "pocketclaw", and Android truncates an app
// process name from the *left*: com.lord1egypt.pocketclaw reports
// gypt.pocketclaw, so PocketClaw's own UI process was classified as a live Core
// runtime. A stale pid file whose PID the kernel had recycled onto the app
// would then have been honoured, and the gateway would have refused to start —
// losing exactly the self-healing this code exists to provide.
//
// A "libpocketclaw" prefix would have fixed that case and kept a narrower
// version of the same bug: every Managed Runtime payload is libpocketclaw-*,
// so gh, git, python and the rest would still have counted as the runtime a pid
// file refers to. They are ours, but they are not the gateway.
//
// Both the truncated and the full form are accepted, because the same binary is
// visible under its full name wherever the name is short enough.
func isOwnedComm(comm string) bool {
	for _, name := range ownedProcessNames {
		if comm == name || comm == commForm(name) {
			return true
		}
	}
	return false
}

// classifyProcComm turns one /proc/<pid>/comm read into an ownership verdict.
//
// A "not visible" error is evidence of foreignness, not an inconclusive
// result: a gateway whose pid file we are asked to respect was spawned by this
// launcher and therefore shares our UID, and a same-UID process is always
// readable under every /proc mode. Android mounts /proc with
// hidepid=invisible, so a recycled PID owned by any other UID lands here on
// every device — reading that as "cannot verify, assume it is ours" is what
// wedged startup behind a dead gateway's pid file.
func classifyProcComm(data []byte, err error) procVerdict {
	if err == nil {
		if isOwnedComm(strings.TrimSpace(string(data))) {
			return procMatch
		}
		return procForeign
	}
	if errors.Is(err, os.ErrNotExist) ||
		errors.Is(err, os.ErrPermission) ||
		errors.Is(err, syscall.ESRCH) {
		return procNotVisible
	}
	return procUnknown
}

// isPicoclawProcess reports whether the live process holding pid is a
// PocketClaw runtime, and so whether its pid file must be honoured.
//
// It stays conservative for genuinely ambiguous read failures. That cannot
// cause a double start: the gateway opens its listeners before it commits the
// pid file, so a second instance that gets past this check still fails to bind
// the gateway port.
func isPicoclawProcess(pid int) bool {
	switch classifyProcComm(readProcComm(pid)) {
	case procMatch:
		return true
	case procForeign, procNotVisible:
		return false
	default:
		return true
	}
}
