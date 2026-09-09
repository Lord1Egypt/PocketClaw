//go:build !windows

package pid

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// writeRecord puts a record under an explicit filename, so the migration cases
// can be built as they actually occur on disk.
func writeRecord(t *testing.T, dir, name string, pid int) {
	t.Helper()
	raw, err := json.MarshalIndent(PidFileData{
		PID:     pid,
		Version: "v0.3.1",
		Port:    18790,
		Host:    "127.0.0.1",
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func exists(t *testing.T, dir, name string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// stubProcCommByPID answers per PID, which is what the two-live-Gateway cases
// need: one PID must read as ours and another as something else.
func stubProcCommByPID(t *testing.T, comms map[int]string) {
	t.Helper()
	prev := readProcComm
	readProcComm = func(pid int) ([]byte, error) {
		if comm, ok := comms[pid]; ok {
			return []byte(comm + "\n"), nil
		}
		return nil, os.ErrNotExist
	}
	t.Cleanup(func() { readProcComm = prev })
}

// liveGateway returns a PID that is genuinely alive and is not this process,
// reading as one of our Core executables.
func liveGateway(t *testing.T) int {
	t.Helper()
	pid := os.Getppid()
	if pid <= 1 || pid == os.Getpid() {
		t.Skip("no suitable live parent PID for this test")
	}
	return pid
}

// secondLiveProcess starts a real child so two distinct live PIDs exist.
func secondLiveProcess(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start a second live process: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	return cmd.Process.Pid
}

func TestFreshStartupWritesCanonicalOnly(t *testing.T) {
	dir := tmpDir(t)

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
		t.Fatalf("fresh startup must succeed: %v", err)
	}

	if !exists(t, dir, CanonicalPidFileName) {
		t.Errorf("startup must write %s", CanonicalPidFileName)
	}
	if exists(t, dir, legacyPidFileName) {
		t.Errorf("startup must never create %s", legacyPidFileName)
	}
}

func TestCanonicalRecordHoldsOnlyDiscoveryFields(t *testing.T) {
	dir := tmpDir(t)
	t.Setenv("POCKETCLAW_GATEWAY_TOKEN_FILE", filepath.Join(dir, "gateway_auth"))

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, CanonicalPidFileName))
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}

	// The record is deliberately discoverable, so what is absent matters as
	// much as what is present. N4G renames the file and must not widen it.
	want := map[string]bool{"host": true, "pid": true, "port": true, "version": true}
	for key := range fields {
		if !want[key] {
			t.Errorf("unexpected field %q in the discovery record", key)
		}
	}
	for key := range want {
		if _, ok := fields[key]; !ok {
			t.Errorf("missing field %q", key)
		}
	}
	if _, ok := fields["token"]; ok {
		t.Error("the discovery record must never carry a credential")
	}
}

func TestLegacyOnlyLiveGatewayBlocksDuplicateStart(t *testing.T) {
	dir := tmpDir(t)
	pid := liveGateway(t)
	writeRecord(t, dir, legacyPidFileName, pid)
	stubProcCommByPID(t, map[int]string{pid: "libpocketclaw.so"})

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err == nil {
		t.Fatal("an older build's live Gateway must still block a second start")
	}

	// Its record is the only way that process can find its own Gateway again,
	// and it is the only name that process knows. Renaming or deleting it
	// underneath a running process is the failure this ordering avoids.
	if !exists(t, dir, legacyPidFileName) {
		t.Error("a live Gateway's legacy record must not be removed")
	}
	if exists(t, dir, CanonicalPidFileName) {
		t.Error("no canonical record may be fabricated for a process that did not write one")
	}
}

func TestLegacyOnlyStaleRecordIsCleanedAndCanonicalWritten(t *testing.T) {
	dir := tmpDir(t)
	writeRecord(t, dir, legacyPidFileName, 99999999)

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
		t.Fatalf("a stale legacy record must not block startup: %v", err)
	}

	if exists(t, dir, legacyPidFileName) {
		t.Error("a stale legacy record must be cleaned up")
	}
	if !exists(t, dir, CanonicalPidFileName) {
		t.Error("startup must write the canonical record")
	}
}

func TestLegacyOnlyMalformedRecordDoesNotWedgeStartup(t *testing.T) {
	dir := tmpDir(t)
	if err := os.WriteFile(filepath.Join(dir, legacyPidFileName), []byte("{ not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Malformed discovery metadata holds no credential and nothing to preserve.
	// Refusing to start on it would be a permanent wedge fixable only by hand.
	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
		t.Fatalf("a malformed legacy record must not wedge startup: %v", err)
	}
	if exists(t, dir, legacyPidFileName) {
		t.Error("a malformed legacy record must be cleaned up")
	}
	if !exists(t, dir, CanonicalPidFileName) {
		t.Error("startup must write the canonical record")
	}
}

func TestBothRecordsNameTheSameLiveGateway(t *testing.T) {
	dir := tmpDir(t)
	pid := liveGateway(t)
	writeRecord(t, dir, CanonicalPidFileName, pid)
	writeRecord(t, dir, legacyPidFileName, pid)
	stubProcCommByPID(t, map[int]string{pid: "libpocketclaw.so"})

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err == nil {
		t.Fatal("a live Gateway must block a second start")
	}

	// Two names for one Gateway. The canonical one is this build's, and the
	// legacy one is redundant rather than authoritative.
	if !exists(t, dir, CanonicalPidFileName) {
		t.Error("the canonical record must survive")
	}
	if exists(t, dir, legacyPidFileName) {
		t.Error("a legacy record naming the same live Gateway is redundant and should be cleaned")
	}
}

func TestCanonicalLiveWithStaleLegacy(t *testing.T) {
	dir := tmpDir(t)
	pid := liveGateway(t)
	writeRecord(t, dir, CanonicalPidFileName, pid)
	writeRecord(t, dir, legacyPidFileName, 99999999)
	stubProcCommByPID(t, map[int]string{pid: "libpocketclaw.so"})

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err == nil {
		t.Fatal("a live Gateway must block a second start")
	}
	if !exists(t, dir, CanonicalPidFileName) {
		t.Error("the live canonical record must survive")
	}
	if exists(t, dir, legacyPidFileName) {
		t.Error("the stale legacy record should be cleaned")
	}
}

func TestCanonicalStaleWithLiveLegacy(t *testing.T) {
	dir := tmpDir(t)
	pid := liveGateway(t)
	writeRecord(t, dir, CanonicalPidFileName, 99999999)
	writeRecord(t, dir, legacyPidFileName, pid)
	stubProcCommByPID(t, map[int]string{pid: "libpocketclaw.so"})

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err == nil {
		t.Fatal("a live Gateway named only by the legacy record must still block a start")
	}

	if !exists(t, dir, legacyPidFileName) {
		t.Error("the live legacy record must not be removed")
	}
	if exists(t, dir, CanonicalPidFileName) {
		t.Error("the stale canonical record should be cleaned")
	}
}

func TestTwoLiveGatewaysFailClosed(t *testing.T) {
	dir := tmpDir(t)
	first := liveGateway(t)
	second := secondLiveProcess(t)
	if first == second {
		t.Skip("could not obtain two distinct live PIDs")
	}
	writeRecord(t, dir, CanonicalPidFileName, first)
	writeRecord(t, dir, legacyPidFileName, second)
	stubProcCommByPID(t, map[int]string{
		first:  "libpocketclaw.so",
		second: "libpocketclaw.so",
	})

	_, err := WritePidFile(dir, "127.0.0.1", 18790)
	if err == nil {
		t.Fatal("two live Gateways must not resolve to a start")
	}
	if !strings.Contains(err.Error(), "conflicting") {
		t.Errorf("the conflict must be reported as one, got: %v", err)
	}

	// Neither record is overwritten, neither process is touched, and no third
	// Gateway is started. Nothing on disk can say which of the two is meant.
	if !exists(t, dir, CanonicalPidFileName) || !exists(t, dir, legacyPidFileName) {
		t.Error("a split-brain state must be left exactly as found")
	}
	if got := ReadPidFileWithCheck(dir); got != nil {
		t.Errorf("the status read must not pick one of two live Gateways, got %+v", got)
	}
}

func TestBothRecordsStaleAreCleanedThenCanonicalWritten(t *testing.T) {
	dir := tmpDir(t)
	writeRecord(t, dir, CanonicalPidFileName, 99999998)
	writeRecord(t, dir, legacyPidFileName, 99999999)

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
		t.Fatalf("two stale records must not block startup: %v", err)
	}

	if exists(t, dir, legacyPidFileName) {
		t.Error("the stale legacy record must be cleaned")
	}
	raw, err := os.ReadFile(filepath.Join(dir, CanonicalPidFileName))
	if err != nil {
		t.Fatal(err)
	}
	var data PidFileData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	if data.PID != os.Getpid() {
		t.Errorf("canonical record PID = %d, want this process %d", data.PID, os.Getpid())
	}
}

// The legacy record is discovered through the same ownership rule as the
// canonical one. A recycled PID reached through the old filename must not be
// honoured any more than one reached through the new.
func TestLegacyRecordUsesTheSameOwnershipRule(t *testing.T) {
	foreign := []struct {
		name string
		comm string
	}{
		{"the Android app process", "gypt.pocketclaw"},
		{"a Managed Runtime tool", "libpocketclaw-py"},
		{"an unrelated process", "system_server"},
		{"the pre-N3 Core executable", "libpicoclaw.so"},
		{"the pre-N3 launcher", "libpicoclaw-web.so"},
	}

	for _, tc := range foreign {
		t.Run(tc.name, func(t *testing.T) {
			dir := tmpDir(t)
			pid := liveGateway(t)
			writeRecord(t, dir, legacyPidFileName, pid)
			stubProcCommByPID(t, map[int]string{pid: tc.comm})

			if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
				t.Fatalf("%s must not be honoured as a live Gateway: %v", tc.name, err)
			}
			if !exists(t, dir, CanonicalPidFileName) {
				t.Error("startup must write the canonical record")
			}
		})
	}
}

// PID 1 recorded in either file is a container leftover on a shared volume:
// the host's init, not a Gateway.
func TestRecordedPid1IsStaleUnderBothNames(t *testing.T) {
	if os.Getpid() == 1 {
		t.Skip("running as PID 1")
	}
	for _, name := range []string{CanonicalPidFileName, legacyPidFileName} {
		t.Run(name, func(t *testing.T) {
			dir := tmpDir(t)
			writeRecord(t, dir, name, 1)

			if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
				t.Fatalf("a recorded PID 1 must not block startup: %v", err)
			}
			if got := ReadPidFileWithCheck(dir); got == nil || got.PID != os.Getpid() {
				t.Errorf("the canonical record should now name this process, got %+v", got)
			}
		})
	}
}

func TestShutdownRemovesTheCanonicalRecord(t *testing.T) {
	dir := tmpDir(t)
	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
		t.Fatal(err)
	}

	RemovePidFile(dir)

	if exists(t, dir, CanonicalPidFileName) {
		t.Error("shutdown must remove the canonical record")
	}
	if exists(t, dir, legacyPidFileName) {
		t.Error("shutdown must never leave a legacy record behind")
	}
}

// The console stops a Gateway and then clears its record. One started by an
// older build left that record under the legacy name; the process is gone
// either way.
func TestRemovePidFileIfPIDClearsEitherName(t *testing.T) {
	for _, name := range []string{CanonicalPidFileName, legacyPidFileName} {
		t.Run(name, func(t *testing.T) {
			dir := tmpDir(t)
			writeRecord(t, dir, name, 4242)

			if !RemovePidFileIfPID(dir, 4242) {
				t.Fatalf("the record under %s should have been removed", name)
			}
			if exists(t, dir, name) {
				t.Error("the record was not removed")
			}
		})
	}
}

// The legacy filename must exist in exactly one production place: the migration
// owner's constant. Anywhere else is a path that could recreate it.
func TestOnlyTheMigrationOwnerNamesTheLegacyRecord(t *testing.T) {
	root := moduleRootForPid(t)
	// Matched as a filename literal, so the prose that explains why the name
	// was retired is not itself a violation.
	legacyLiteral := regexp.MustCompile(`"\.picoclaw\.pid"`)

	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case "vendor", "build", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.HasSuffix(path, "pidfile_migration.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if legacyLiteral.Match(raw) {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(offenders) != 0 {
		t.Errorf("the legacy record name appears outside the migration owner: %v", offenders)
	}
}

// Every consumer reaches the records through the shared resolver, so the
// migration, precedence and stale policies cannot drift apart per caller.
func TestConsumersGoThroughTheSharedResolver(t *testing.T) {
	raw, err := os.ReadFile("pidfile.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, fn := range []string{"WritePidFile", "ReadPidFileWithCheck"} {
		if !strings.Contains(source, fn) {
			t.Fatalf("%s is gone", fn)
		}
	}
	if strings.Count(source, "resolvePidRecords(homePath") != 2 {
		t.Error("both WritePidFile and ReadPidFileWithCheck must resolve through resolvePidRecords")
	}
	if strings.Contains(source, "activeRecord(canonical, legacy)") == false {
		t.Error("record precedence must come from activeRecord, not from a caller")
	}
}

func moduleRootForPid(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the pid package")
		}
		dir = parent
	}
}
