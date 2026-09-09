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
			data: []byte("libpocketclaw.so\n"),
			want: procMatch,
		},
		{
			name: "same-UID launcher also matches",
			data: []byte("libpocketclaw-web.so\n"),
			want: procMatch,
		},
		{
			// The pre-N3 packaged name. A stale .picoclaw.pid can name a PID
			// the kernel has since reused; if the old name still matched, that
			// reused process would be honoured as our own gateway and wedge
			// startup behind it. Android replaces nativeLibraryDir wholesale on
			// update, so nothing can legitimately still be running under it.
			name: "the pre-N3 executable name is foreign, not ours",
			data: []byte("libpicoclaw.so\n"),
			want: procForeign,
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
		{name: "live PocketClaw process is honoured", data: []byte("libpocketclaw.so\n"), want: true},
		{name: "stale pre-N3 name is not honoured", data: []byte("libpicoclaw.so\n"), want: false},
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
	if err := os.WriteFile(filepath.Join(dir, CanonicalPidFileName), raw, 0o600); err != nil {
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

	stubProcComm(t, []byte("libpocketclaw.so\n"), nil)

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err == nil {
		t.Fatal("a live PocketClaw gateway's pid file must block a second start")
	}

	// The pid file must survive the rejection.
	if _, err := os.Stat(filepath.Join(dir, CanonicalPidFileName)); err != nil {
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
	if _, err := os.Stat(filepath.Join(dir, CanonicalPidFileName)); !os.IsNotExist(err) {
		t.Error("a dead PID's pid file should be removed")
	}
}

func TestCommTruncationKeepsOwnershipDecidable(t *testing.T) {
	// The truncation itself, so a future rename cannot silently move the cut
	// into the part of the name ownership depends on.
	for _, tc := range []struct {
		execName string
		wantComm string
	}{
		{"libpocketclaw.so", "libpocketclaw.s"},
		{"libpocketclaw-web.so", "libpocketclaw-w"},
		{"libpicoclaw.so", "libpicoclaw.so"},
	} {
		if got := commForm(tc.execName); got != tc.wantComm {
			t.Errorf("commForm(%q) = %q, want %q", tc.execName, got, tc.wantComm)
		}
	}

	// Both Core executables must remain distinguishable after the cut. If a
	// future rename made their first 15 bytes equal, this rule would stop
	// telling the gateway from the launcher.
	if commForm(ownedProcessNames[0]) == commForm(ownedProcessNames[1]) {
		t.Fatalf("both Core executables truncate to %q", commForm(ownedProcessNames[0]))
	}
}

// The ownership verdict against the comm strings a device actually reports.
func TestOwnershipAgainstRealTruncatedComm(t *testing.T) {
	tests := []struct {
		name     string
		execName string
		want     bool
		why      string
	}{
		{
			name:     "truncated gateway is ours",
			execName: "libpocketclaw.so",
			want:     true,
			why:      "comm is libpocketclaw.s on every device; the match must survive it",
		},
		{
			name:     "truncated launcher is ours",
			execName: "libpocketclaw-web.so",
			want:     true,
			why:      "comm is libpocketclaw-w; the launcher is a PocketClaw runtime too",
		},
		{
			name:     "pre-N3 gateway name is foreign",
			execName: "libpicoclaw.so",
			want:     false,
			why:      "short enough not to truncate, and deliberately not an alias",
		},
		{
			name:     "unrelated long process is foreign",
			execName: "system_server_and_more.so",
			want:     false,
			why:      "truncation must not turn an unrelated name into a match",
		},
		{
			name:     "unrelated short process is foreign",
			execName: "system_server",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubProcComm(t, []byte(commForm(tt.execName)+"\n"), nil)
			if got := isPicoclawProcess(1234); got != tt.want {
				t.Errorf("isPicoclawProcess() for comm %q = %v, want %v; %s",
					commForm(tt.execName), got, tt.want, tt.why)
			}
		})
	}
}

// comm alone does not grant ownership, and truncation does not lose it.
//
// WritePidFile honours an existing pid file only when the recorded PID is not
// 1, the process is alive, and comm names our runtime. A test that exercised
// only classifyProcComm would pass just as happily against an implementation
// that had dropped the other two gates, so these drive the real path.
func TestRecordedPidOwnershipThroughTheRealPath(t *testing.T) {
	alivePID := os.Getppid()
	if alivePID <= 1 || alivePID == os.Getpid() {
		t.Skip("no suitable live parent PID for this test")
	}

	t.Run("a live gateway with truncated comm still blocks a second start", func(t *testing.T) {
		dir := tmpDir(t)
		writeStalePidFile(t, dir, alivePID)
		stubProcComm(t, []byte(commForm("libpocketclaw.so")+"\n"), nil)

		if _, err := WritePidFile(dir, "127.0.0.1", 18790); err == nil {
			t.Error("a live PocketClaw gateway must still be honoured when its " +
				"comm is truncated to libpocketclaw.s")
		}
	})

	t.Run("PID 1 is stale however well its comm matches", func(t *testing.T) {
		dir := tmpDir(t)
		writeStalePidFile(t, dir, 1)
		stubProcComm(t, []byte(commForm("libpocketclaw.so")+"\n"), nil)

		if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
			t.Errorf("PID 1 must be treated as stale even with a matching comm: %v", err)
		}
	})

	t.Run("a live process with the pre-N3 comm does not block", func(t *testing.T) {
		dir := tmpDir(t)
		writeStalePidFile(t, dir, alivePID)
		stubProcComm(t, []byte(commForm("libpicoclaw.so")+"\n"), nil)

		if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
			t.Errorf("a stale .picoclaw.pid naming a reused PID must not wedge "+
				"startup: %v", err)
		}
	})
}

// The regression found on the device, and the rule that fixes it.
//
// Android truncates an application process name from the *left*, so
// com.lord1egypt.pocketclaw reports gypt.pocketclaw. A substring rule looking
// for "pocketclaw" therefore classified PocketClaw's own UI process as a live
// Core runtime — and a stale pid file whose PID the kernel had recycled onto it
// would have blocked gateway startup instead of self-healing.
//
// Every comm value here was either read from the validated Samsung (SM-A165F,
// vc60) or derived from a packaged executable name by the same truncation the
// kernel applies.
func TestOwnershipRejectsEverythingThatIsNotACoreExecutable(t *testing.T) {
	cases := []struct {
		name string
		comm string
		want bool
		why  string
	}{
		{"gateway, as the device reports it", "libpocketclaw.s", true,
			"observed on SM-A165F, pid 1261"},
		{"launcher, as the device reports it", "libpocketclaw-w", true,
			"observed on SM-A165F, pid 1188"},
		{"gateway under its full name", "libpocketclaw.so", true,
			"the same binary where the name is not truncated"},
		{"launcher under its full name", "libpocketclaw-web.so", true, ""},

		{"the Android app process", "gypt.pocketclaw", false,
			"observed on SM-A165F, pid 23674; this is the regression"},
		{"the app process untruncated", "com.lord1egypt.pocketclaw", false, ""},
		{"a Managed Runtime tool", "libpocketclaw-g", false,
			"gh is ours, but a pid file never refers to it"},
		{"another Managed Runtime tool", "libpocketclaw-p", false, "python"},
		{"the pre-N3 gateway name", "libpicoclaw.so", false,
			"deliberately not an alias"},
		{"the upstream desktop binary", "picoclaw", false, ""},
		{"an unrelated process", "system_server", false, ""},
		{"an unrelated truncated process", "system_server_a", false, ""},
		{"something merely starting the same way", "libpocket", false, ""},
		{"empty comm", "", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubProcComm(t, []byte(tc.comm+"\n"), nil)
			if got := isPicoclawProcess(1234); got != tc.want {
				t.Errorf("comm %q -> owned=%v, want %v; %s", tc.comm, got, tc.want, tc.why)
			}
		})
	}
}

// The stale-PID recovery this protects, driven through WritePidFile.
func TestStalePidRecyclingSelfHeals(t *testing.T) {
	alivePID := os.Getppid()
	if alivePID <= 1 || alivePID == os.Getpid() {
		t.Skip("no suitable live parent PID for this test")
	}

	recycled := map[string]string{
		"onto the Android app process": "gypt.pocketclaw",
		"onto a Managed Runtime tool":  "libpocketclaw-g",
		"onto an unrelated process":    "system_server",
		"onto a pre-N3 named process":  "libpicoclaw.so",
	}
	for name, comm := range recycled {
		t.Run(name, func(t *testing.T) {
			dir := tmpDir(t)
			writeStalePidFile(t, dir, alivePID)
			stubProcComm(t, []byte(comm+"\n"), nil)

			if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
				t.Errorf("a stale pid file recycled %s must not block startup: %v", name, err)
			}
		})
	}

	t.Run("a live gateway still blocks a duplicate start", func(t *testing.T) {
		dir := tmpDir(t)
		writeStalePidFile(t, dir, alivePID)
		stubProcComm(t, []byte(commForm("libpocketclaw.so")+"\n"), nil)

		if _, err := WritePidFile(dir, "127.0.0.1", 18790); err == nil {
			t.Error("the real gateway must still be honoured")
		}
	})
}
