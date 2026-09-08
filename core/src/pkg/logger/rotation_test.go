package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// logDirBytes is the total size of every file in a log directory.
func logDirBytes(t *testing.T, dir string) int64 {
	t.Helper()
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
	return total
}

func logDirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// The bound has to hold inside one process, not merely across restarts.
//
// Rotating only when the file is opened was the original mistake: the gateway
// is long-lived, so it would have appended past the threshold for as long as it
// ran — which is exactly how the unbounded file this bound exists to prevent
// came about. Nothing here closes or reopens the writer.
func TestRotationHappensDuringOneLongLivedSession(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.log")

	writer, err := openRotatingFile(path)
	if err != nil {
		t.Fatalf("openRotatingFile() error = %v", err)
	}
	defer writer.Close()

	// Comfortably more than one threshold, written through the same open
	// writer throughout.
	record := []byte(strings.Repeat("d", 4096) + "\n")
	total := 0
	for total < 3*maxLogFileBytes {
		n, err := writer.Write(record)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		total += n
	}

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("no rotation occurred while the writer stayed open: %v", err)
	}

	active, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat active log: %v", err)
	}
	// The record that crosses the threshold lands in the old file, so the
	// active file is bounded by the threshold plus at most one record.
	if limit := int64(maxLogFileBytes) + int64(len(record)); active.Size() > limit {
		t.Errorf("active log grew to %d bytes, beyond %d", active.Size(), limit)
	}
}

// Repeated writes must not accumulate generations without limit.
func TestRotationKeepsAtMostThreeFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.log")

	writer, err := openRotatingFile(path)
	if err != nil {
		t.Fatalf("openRotatingFile() error = %v", err)
	}
	defer writer.Close()

	record := []byte(strings.Repeat("d", 8192) + "\n")
	// Enough to rotate many times over, so a leak would be obvious.
	for written := 0; written < 12*maxLogFileBytes; {
		n, err := writer.Write(record)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		written += n
	}

	names := logDirNames(t, dir)
	if len(names) > maxLogRotations+1 {
		t.Errorf("log directory holds %d files (%v), want at most %d",
			len(names), names, maxLogRotations+1)
	}
	beyond := fmt.Sprintf("%s.%d", path, maxLogRotations+1)
	if _, err := os.Stat(beyond); !os.IsNotExist(err) {
		t.Errorf("%s should have been deleted, not retained", beyond)
	}

	// Steady state stays near the intended footprint rather than growing with
	// the number of bytes ever written.
	if limit := int64(maxLogRotations+1)*maxLogFileBytes + int64(len(record))*8; logDirBytes(t, dir) > limit {
		t.Errorf("log directory is %d bytes, beyond the intended bound %d",
			logDirBytes(t, dir), limit)
	}
}

// Two goroutines rotating at once must not leave one of them writing to a
// descriptor whose file has been renamed away.
func TestConcurrentWritesDoNotCorruptRotationState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.log")

	writer, err := openRotatingFile(path)
	if err != nil {
		t.Fatalf("openRotatingFile() error = %v", err)
	}
	defer writer.Close()

	record := []byte(strings.Repeat("c", 2048) + "\n")
	const writers, perWriter = 8, 400

	var wg sync.WaitGroup
	errs := make(chan error, writers*perWriter)
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perWriter {
				if _, err := writer.Write(record); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent Write() error = %v", err)
	}

	if names := logDirNames(t, dir); len(names) > maxLogRotations+1 {
		t.Errorf("concurrent writes produced %d files: %v", len(names), names)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the active log is missing after concurrent rotation: %v", err)
	}
}

// A file inherited from a previous run that is already over the threshold
// rotates at once, rather than being appended to for another full threshold.
func TestAnInheritedOversizedLogRotatesOnOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.log")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxLogFileBytes+1024)), 0o600); err != nil {
		t.Fatal(err)
	}

	writer, err := openRotatingFile(path)
	if err != nil {
		t.Fatalf("openRotatingFile() error = %v", err)
	}
	defer writer.Close()

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("an oversized inherited log was not rotated: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat active log: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("active log is %d bytes after rotation, want 0", info.Size())
	}
}

// A log below the bound must be left exactly as it is: rotating on every open
// would throw away the tail that makes a log worth reading.
func TestASmallLogIsNotRotated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.log")
	if err := os.WriteFile(path, []byte("one line\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	writer, err := openRotatingFile(path)
	if err != nil {
		t.Fatalf("openRotatingFile() error = %v", err)
	}
	defer writer.Close()

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

// Rotation failing must never take the gateway with it. This runs underneath
// the logger, so it cannot even report a problem by logging one.
func TestRotationFailureIsNotFatal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.log")

	writer, err := openRotatingFile(path)
	if err != nil {
		t.Fatalf("openRotatingFile() error = %v", err)
	}
	defer writer.Close()

	// A directory where the rotated name should go: the rename cannot succeed,
	// and neither can the fallback truncate of a name that is not a file.
	if err := os.Mkdir(path+".1", 0o755); err != nil {
		t.Fatal(err)
	}

	record := []byte(strings.Repeat("f", 4096) + "\n")
	for written := 0; written < 2*maxLogFileBytes; {
		n, err := writer.Write(record)
		if err != nil {
			t.Fatalf("Write() returned an error when rotation could not proceed: %v", err)
		}
		written += n
	}

	// Still writing, and the counter was reset rather than retrying a doomed
	// rotation on every subsequent write.
	if _, err := writer.Write([]byte("still alive\n")); err != nil {
		t.Fatalf("logging stopped after a rotation failure: %v", err)
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

// Redaction runs before logMessage hands anything to a writer, so a credential
// never reaches the file even though the file is now private.
func TestRedactionHappensBeforeBytesArePersisted(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	path := filepath.Join(dir, "gateway.log")

	DisableFileLogging()
	if err := EnableFileLogging(path); err != nil {
		t.Fatalf("EnableFileLogging() error = %v", err)
	}
	defer DisableFileLogging()

	const fakeKey = "abcdefghijklmnopqrstuvwxyz012345"
	WarnCF("test", "upstream rejected the request", map[string]any{
		"detail": "x-api-key: " + fakeKey,
	})
	DisableFileLogging()

	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if strings.Contains(string(persisted), fakeKey) {
		t.Fatalf("a credential reached the log file:\n%s", persisted)
	}
	if !strings.Contains(string(persisted), "<redacted>") {
		t.Fatalf("the field was not redacted on its way to the file:\n%s", persisted)
	}
}
