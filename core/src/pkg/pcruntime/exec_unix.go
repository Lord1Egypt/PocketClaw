//go:build !windows

package pcruntime

import (
	"os/exec"
	"syscall"
)

// isolateProcessGroup puts the child in its own process group so that a timeout
// or a cancellation can kill the whole tree. Killing only the direct child would
// leave a grandchild — a pipeline started by tar or xargs — running unsupervised
// with nothing left to reap it.
func isolateProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminateProcessTree signals the child's whole process group.
func terminateProcessTree(cmd *exec.Cmd, signal syscall.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	// The negative pid addresses the group created by Setpgid. If the group is
	// already gone, fall back to the process itself rather than reporting a
	// failure the caller cannot act on.
	if err := syscall.Kill(-cmd.Process.Pid, signal); err == nil {
		return nil
	}
	return syscall.Kill(cmd.Process.Pid, signal)
}

func gracefulSignal() syscall.Signal { return syscall.SIGTERM }
func forcefulSignal() syscall.Signal { return syscall.SIGKILL }
