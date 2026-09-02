//go:build !windows

package pid

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// pathErr wraps errno the way os.ReadFile does, so the tests exercise the same
// unwrapping that classifyProcComm performs in production.
func pathErr(errno syscall.Errno) error {
	return &os.PathError{Op: "open", Path: "/proc/1234/comm", Err: errno}
}

// stubProcComm replaces the /proc reader for one test.
func stubProcComm(t *testing.T, data []byte, err error) {
	t.Helper()
	prev := readProcComm
	readProcComm = func(int) ([]byte, error) { return data, err }
	t.Cleanup(func() { readProcComm = prev })
}

func TestClassifyProcComm(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		err  error
		want procVerdict
	}{
		{
			name: "same-UID live PocketClaw gateway",
			data: []byte("libpicoclaw.so\n"),
			want: procMatch,
		},
		{
			name: "same-UID launcher also matches",
			data: []byte("libpicoclaw-web.so\n"),
			want: procMatch,
		},
		{
			name: "PID reused by an unrelated process",
			data: []byte("system_server\n"),
			want: procForeign,
		},
		{
			name: "missing PID (process gone)",
			err:  pathErr(syscall.ENOENT),
			want: procNotVisible,
		},
		{
			name: "Android hidepid hides a foreign UID's PID",
			err:  pathErr(syscall.EACCES),
			want: procNotVisible,
		},
		{
			name: "permission denied on the comm entry",
			err:  pathErr(syscall.EPERM),
			want: procNotVisible,
		},
		{
			name: "no such process",
			err:  pathErr(syscall.ESRCH),
			want: procNotVisible,
		},
		{
			name: "ambiguous I/O error stays conservative",
			err:  pathErr(syscall.EIO),
			want: procUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyProcComm(tt.data, tt.err); got != tt.want {
				t.Errorf("classifyProcComm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPicoclawProcess(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		err  error
		want bool
	}{
		{name: "live PocketClaw process is honoured", data: []byte("libpicoclaw.so\n"), want: true},
		{name: "reused PID is not honoured", data: []byte("system_server\n"), want: false},
		{name: "missing PID is not honoured", err: pathErr(syscall.ENOENT), want: false},
		{name: "hidepid-invisible PID is not honoured", err: pathErr(syscall.EACCES), want: false},
		{name: "ambiguous error remains conservative", err: pathErr(syscall.EIO), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubProcComm(t, tt.data, tt.err)
			if got := isPicoclawProcess(1234); got != tt.want {
				t.Errorf("isPicoclawProcess() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsProcessRunningEPERM pins the documented EPERM behaviour: a process we
// are not allowed to signal still exists. This is what makes the /proc identity
// check the only thing standing between a recycled PID and a blocked startup.
func TestIsProcessRunningEPERM(t *testing.T) {
	if !isProcessRunning(os.Getpid()) {
		t.Error("current process should be reported as running")
	}
	if isProcessRunning(99999999) {
		t.Error("a PID that does not exist should not be reported as running")
	}
	if isProcessRunning(0) || isProcessRunning(-1) {
		t.Error("non-positive PIDs should never be reported as running")
	}

	var errno syscall.Errno
	err := error(syscall.EPERM)
	if !errors.As(err, &errno) || errno != syscall.EPERM {
		t.Fatal("EPERM should unwrap to syscall.Errno")
	}
}

// writeStalePidFile drops a pid file naming pid into dir.
func writeStalePidFile(t *testing.T, dir string, pid int) {
	t.Helper()
	raw, err := json.MarshalIndent(PidFileData{
		PID:     pid,
		Token:   "deadbeef12345678deadbeef12345678",
		Version: "v0.3.1",
		Port:    18790,
		Host:    "127.0.0.1",
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, pidFileName), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestWritePidFileRecoversFromInvisibleReusedPID is the regression test for the
// device incident: a killed gateway left a pid file behind, the PID was reused
// by a process belonging to another UID, and Android's hidepid=invisible /proc
// made that process unreadable. Startup must recover instead of refusing
// forever.
func TestWritePidFileRecoversFromInvisibleReusedPID(t *testing.T) {
	dir := tmpDir(t)

	// A PID that is genuinely alive but is not this process, standing in for
	// the recycled PID 14605 on the device.
	alivePID := os.Getppid()
	if alivePID <= 1 || alivePID == os.Getpid() {
		t.Skip("no suitable live parent PID for this test")
	}
	writeStalePidFile(t, dir, alivePID)

	// hidepid: the kernel will not show us this PID at all.
	stubProcComm(t, nil, pathErr(syscall.ENOENT))

	data, err := WritePidFile(dir, "127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("startup must recover from an invisible reused PID, got: %v", err)
	}
	if data.PID != os.Getpid() {
		t.Errorf("pid file PID = %d, want %d", data.PID, os.Getpid())
	}
}

// TestWritePidFilePreservesLiveGateway is the other half of the guarantee: a
// pid file whose process really is a visible, live PocketClaw runtime must
// still block a second start.
func TestWritePidFilePreservesLiveGateway(t *testing.T) {
	dir := tmpDir(t)

	alivePID := os.Getppid()
	if alivePID <= 1 || alivePID == os.Getpid() {
		t.Skip("no suitable live parent PID for this test")
	}
	writeStalePidFile(t, dir, alivePID)

	stubProcComm(t, []byte("libpicoclaw.so\n"), nil)

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err == nil {
		t.Fatal("a live PocketClaw gateway's pid file must block a second start")
	}

	// The pid file must survive the rejection.
	if _, err := os.Stat(filepath.Join(dir, pidFileName)); err != nil {
		t.Errorf("pid file of a live gateway must not be removed: %v", err)
	}
}

// TestReadPidFileWithCheckInvisibleReusedPID covers the launcher-side read path
// for the same condition.
func TestReadPidFileWithCheckInvisibleReusedPID(t *testing.T) {
	dir := tmpDir(t)
	writeStalePidFile(t, dir, 99999999)

	if got := ReadPidFileWithCheck(dir); got != nil {
		t.Errorf("a dead PID must read as no pid file, got %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, pidFileName)); !os.IsNotExist(err) {
		t.Error("a dead PID's pid file should be removed")
	}
}
