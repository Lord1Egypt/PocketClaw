//go:build !mipsle && !netbsd && !(freebsd && arm)

package dashboardauth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Every password here is fabricated and exists only in this file.
const (
	fakePassword      = "fabricated-dashboard-password-01"
	fakeOtherPassword = "fabricated-different-password-02"
)

func seedLegacy(t *testing.T, dir, password string) {
	t.Helper()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("seed legacy store: %v", err)
	}
	if err := store.SetPassword(context.Background(), password); err != nil {
		t.Fatalf("seed password: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close seeded store: %v", err)
	}
}

func verifies(t *testing.T, dir, password string) bool {
	t.Helper()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	ok, err := store.VerifyPassword(context.Background(), password)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	return ok
}

// The migration exists to move the verifier off shared storage without costing
// the user the password they already set.
func TestMigrationPreservesTheExistingPassword(t *testing.T) {
	legacy, private := t.TempDir(), filepath.Join(t.TempDir(), "auth")
	seedLegacy(t, legacy, fakePassword)

	legacyBytes, err := os.ReadFile(filepath.Join(legacy, DBFilename))
	if err != nil {
		t.Fatal(err)
	}

	result, err := MigrateLegacyDatabase(context.Background(), legacy, private)
	if err != nil {
		t.Fatalf("MigrateLegacyDatabase() error = %v", err)
	}
	if !result.Migrated {
		t.Fatalf("nothing was migrated: %s", result.Reason)
	}
	if !result.LegacyRemoved {
		t.Error("the legacy database was left on shared storage")
	}

	if !verifies(t, private, fakePassword) {
		t.Fatal("the existing password no longer verifies after migration")
	}
	if verifies(t, private, fakeOtherPassword) {
		t.Error("an unrelated password verifies against the migrated store")
	}

	// The stored verifier is carried across unchanged; migration is a move,
	// not a re-hash, so it cannot silently reset anything.
	migratedBytes, err := os.ReadFile(filepath.Join(private, DBFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(migratedBytes) != string(legacyBytes) {
		t.Error("the migrated database is not a faithful copy of the legacy one")
	}

	// And the shared copy is gone, so there is nothing left to roll back to.
	if _, err := os.Stat(filepath.Join(legacy, DBFilename)); !os.IsNotExist(err) {
		t.Error("the legacy database still exists on shared storage")
	}
}

// Private state is authoritative. Letting a shared copy overwrite it would put
// the rollback-to-attacker-controlled-state path back.
func TestAnExistingPrivateDatabaseIsNeverOverwritten(t *testing.T) {
	legacy, private := t.TempDir(), t.TempDir()
	seedLegacy(t, legacy, fakeOtherPassword)
	seedLegacy(t, private, fakePassword)

	result, err := MigrateLegacyDatabase(context.Background(), legacy, private)
	if err != nil {
		t.Fatalf("MigrateLegacyDatabase() error = %v", err)
	}
	if result.Migrated {
		t.Fatal("a legacy database overwrote the private one")
	}

	if !verifies(t, private, fakePassword) {
		t.Error("the private password was replaced by the legacy one")
	}
	if verifies(t, private, fakeOtherPassword) {
		t.Error("the legacy credential reached the private store")
	}
}

// Running it twice must do nothing the second time.
func TestMigrationIsIdempotent(t *testing.T) {
	legacy, private := t.TempDir(), filepath.Join(t.TempDir(), "auth")
	seedLegacy(t, legacy, fakePassword)

	first, err := MigrateLegacyDatabase(context.Background(), legacy, private)
	if err != nil || !first.Migrated {
		t.Fatalf("first migration failed: %v (%+v)", err, first)
	}
	second, err := MigrateLegacyDatabase(context.Background(), legacy, private)
	if err != nil {
		t.Fatalf("second migration errored: %v", err)
	}
	if second.Migrated {
		t.Error("the second run migrated again")
	}
	if !verifies(t, private, fakePassword) {
		t.Error("the password stopped verifying after a second run")
	}
}

// A failure must never cost the user the only verifier that works.
func TestAFailedMigrationKeepsTheLegacyDatabase(t *testing.T) {
	legacy := t.TempDir()
	seedLegacy(t, legacy, fakePassword)

	// A file where the private directory should be, so MkdirAll cannot succeed.
	blocked := filepath.Join(t.TempDir(), "auth")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := MigrateLegacyDatabase(context.Background(), legacy, blocked)
	if err == nil {
		t.Fatal("a migration into an unusable destination reported success")
	}
	if result.Migrated || result.LegacyRemoved {
		t.Fatalf("a failed migration claimed progress: %+v", result)
	}

	if !verifies(t, legacy, fakePassword) {
		t.Fatal("a failed migration destroyed the only working credential verifier")
	}
}

// Nothing to migrate is not an error, and it must not create anything.
func TestNoLegacyDatabaseIsANoOp(t *testing.T) {
	legacy, private := t.TempDir(), filepath.Join(t.TempDir(), "auth")

	result, err := MigrateLegacyDatabase(context.Background(), legacy, private)
	if err != nil {
		t.Fatalf("MigrateLegacyDatabase() error = %v", err)
	}
	if result.Migrated {
		t.Error("something was migrated from an empty directory")
	}
	if _, err := os.Stat(filepath.Join(private, DBFilename)); !os.IsNotExist(err) {
		t.Error("a database was created where there was nothing to migrate")
	}
}

// Same directory means no private storage was configured; migrating a file
// onto itself must not be attempted.
func TestSameDirectoryIsANoOp(t *testing.T) {
	dir := t.TempDir()
	seedLegacy(t, dir, fakePassword)

	result, err := MigrateLegacyDatabase(context.Background(), dir, dir)
	if err != nil {
		t.Fatalf("MigrateLegacyDatabase() error = %v", err)
	}
	if result.Migrated {
		t.Error("a database was migrated onto itself")
	}
	if !verifies(t, dir, fakePassword) {
		t.Fatal("the credential was damaged by a no-op migration")
	}
}

// The migration copies the main database file, which is only safe because the
// store uses the default rollback journal rather than WAL. Measured, not
// assumed: a WAL database keeps committed data in a -wal sidecar, and copying
// the main file alone would silently lose it.
func TestStoreUsesRollbackJournalAndLeavesNoSidecars(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetPassword(context.Background(), fakePassword); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != DBFilename {
			t.Errorf("closed store left a sidecar: %s", entry.Name())
		}
	}

	// SQLite header offsets 18 and 19 are the write and read file-format
	// versions: 1 is the rollback journal, 2 is WAL.
	header, err := os.ReadFile(filepath.Join(dir, DBFilename))
	if err != nil {
		t.Fatal(err)
	}
	if len(header) < 20 {
		t.Fatalf("database is too small to carry a header: %d bytes", len(header))
	}
	if header[18] != 1 || header[19] != 1 {
		t.Fatalf("journal mode is not the rollback journal (write=%d read=%d); "+
			"a plain file copy is no longer a safe migration",
			header[18], header[19])
	}
}

// Nothing but the verifier is ever persisted — no session token, no plaintext.
func TestTheDatabaseHoldsOnlyTheVerifier(t *testing.T) {
	dir := t.TempDir()
	seedLegacy(t, dir, fakePassword)

	raw, err := os.ReadFile(filepath.Join(dir, DBFilename))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)

	if contains(body, fakePassword) {
		t.Fatal("the plaintext password was written to disk")
	}
	// One table, one column of interest. A session table appearing here would
	// mean bearer material had started being persisted.
	for _, unexpected := range []string{"session", "token", "cookie", "bearer"} {
		if contains(body, unexpected) {
			t.Errorf("the database schema mentions %q; only the verifier belongs here", unexpected)
		}
	}
	if !contains(body, "dashboard_credentials") || !contains(body, "bcrypt_hash") {
		t.Error("the expected credential schema is missing")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
