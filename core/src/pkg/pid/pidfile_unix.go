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
		if strings.Contains(strings.TrimSpace(string(data)), "picoclaw") {
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
