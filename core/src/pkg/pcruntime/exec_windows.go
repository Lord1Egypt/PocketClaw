//go:build windows

package pcruntime

import (
	"os/exec"
	"syscall"
)

// isolateProcessGroup is a no-op on Windows: PocketClaw's managed runtime ships
// on Android, and the desktop Windows build has no bundled or system catalog
// entries to run.
func isolateProcessGroup(cmd *exec.Cmd) {}

func terminateProcessTree(cmd *exec.Cmd, _ syscall.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

func gracefulSignal() syscall.Signal { return syscall.Signal(0) }
func forcefulSignal() syscall.Signal { return syscall.Signal(0) }
