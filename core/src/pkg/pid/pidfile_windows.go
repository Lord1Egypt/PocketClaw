//go:build windows

package pid

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procGetExitCodeProcess         = kernel32.NewProc("GetExitCodeProcess")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	processQueryLimitedInformation = uint32(0x1000)
	stillActive                    = uint32(259)
)

// isProcessRunning checks whether a process with the given PID is alive
// on Windows using OpenProcess + GetExitCodeProcess.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	handle, _, _ := procOpenProcess.Call(
		uintptr(processQueryLimitedInformation),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return false
	}
	defer procCloseHandle.Call(handle)

	var exitCode uint32
	ret, _, _ := procGetExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	if ret == 0 {
		return false
	}
	return exitCode == stillActive
}

// isPicoclawProcess uses QueryFullProcessImageNameW to confirm the
// process image name contains "picoclaw". Returns false when the name
// clearly does not match. Returns true if the query fails, falling
// back to trusting the liveness check alone.
func isPicoclawProcess(pid int) bool {
	handle, _, _ := procOpenProcess.Call(
		uintptr(processQueryLimitedInformation),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return true // cannot open — trust liveness check
	}
	defer procCloseHandle.Call(handle)

	var buf [260]uint16
	var size uint32 = 260
	ret, _, _ := procQueryFullProcessImageNameW.Call(
		uintptr(handle),
		0, // WIN32_NAME_FORMAT
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return true // cannot verify — trust liveness check
	}
	name := strings.ToLower(syscall.UTF16ToString(buf[:size]))
	return strings.Contains(name, "picoclaw")
}
