package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The log had no rotation at all and was pure append, so a long-lived install
// accumulated an unbounded file — one was observed at 18 MB, still carrying
// lines a much older build had written.
func TestLogRotationIsBounded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.log")

	oversized := strings.Repeat("x", maxLogFileBytes+1024)
	for generation := range 5 {
		if err := os.WriteFile(path, []byte(oversized), 0o600); err != nil {
			t.Fatalf("seed generation %d: %v", generation, err)
		}
		rotateLogFileIfLarge(path)
	}

	// gateway.log itself was rotated away each time, so what remains is the
	// retained generations and whatever is written next.
	for i := 1; i <= maxLogRotations; i++ {
		rotated := fmt.Sprintf("%s.%d", path, i)
		if _, err := os.Stat(rotated); err != nil {
			t.Errorf("expected retained generation %s: %v", rotated, err)
		}
	}
	beyond := fmt.Sprintf("%s.%d", path, maxLogRotations+1)
	if _, err := os.Stat(beyond); !os.IsNotExist(err) {
		t.Errorf("%s should have been deleted, not retained", beyond)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}
	var total int64
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("stat %s: %v", entry.Name(), err)
		}
		total += info.Size()
	}
	if limit := int64(maxLogRotations+1) * maxLogFileBytes * 2; total > limit {
		t.Errorf("log directory grew to %d bytes, beyond the intended bound", total)
	}
}

// A log below the bound must be left exactly as it is: rotating on every open
// would throw away the tail that makes a log worth reading.
func TestASmallLogIsNotRotated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.log")
	if err := os.WriteFile(path, []byte("one line\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	rotateLogFileIfLarge(path)

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the active log was rotated away: %v", err)
	}
	if string(body) != "one line\n" {
		t.Errorf("log contents changed: %q", body)
	}
	if _, err := os.Stat(path + ".1"); !os.IsNotExist(err) {
		t.Error("a rotation was created for a log below the bound")
	}
}

// Logs are private now, and the file mode says so on any filesystem that
// honours it.
func TestFileLoggingCreatesOwnerOnlyFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	path := filepath.Join(dir, "gateway.log")

	// EnableFileLogging refuses to run when a writer is already attached, and
	// the logger's writer list is package-global — so another test in this
	// package can leave one behind. Start from a known state rather than from
	// whatever ran first.
	DisableFileLogging()
	if err := EnableFileLogging(path); err != nil {
		t.Fatalf("EnableFileLogging() error = %v", err)
	}
	defer DisableFileLogging()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat log: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("log file mode = %04o, want 0600", perm)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat log dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("log directory mode = %04o, want 0700", perm)
	}
}
