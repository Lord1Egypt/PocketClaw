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

// isPicoclawProcess reads /proc/<pid>/comm to confirm the process name
// contains "picoclaw". Returns false when the comm file can be read and
// the name does not match (e.g., PID was reused by an unrelated process).
// Returns true if /proc/<pid>/comm is unreadable so the call site falls
// back to trusting the liveness check alone.
func isPicoclawProcess(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return true // cannot verify — trust liveness check
	}
	return strings.Contains(strings.TrimSpace(string(data)), "picoclaw")
}
