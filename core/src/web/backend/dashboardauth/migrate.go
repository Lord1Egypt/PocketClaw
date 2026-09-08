//go:build !mipsle && !netbsd && !(freebsd && arm)

package dashboardauth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// legacySidecarSuffixes are the auxiliary files SQLite can leave beside a
// database. The store uses the default rollback journal rather than WAL —
// verified by TestStoreUsesRollbackJournalAndLeavesNoSidecars — so `-journal`
// is the only one that can exist here, and only transiently during a write.
// The others are listed so a cleanup after migration cannot leave a stale
// sidecar behind if the mode ever changes.
var legacySidecarSuffixes = []string{"-journal", "-wal", "-shm"}

// MigrationResult describes what MigrateLegacyDatabase did.
type MigrationResult struct {
	// Migrated is true when a legacy database was copied into the destination.
	Migrated bool
	// LegacyRemoved is true when the legacy database was deleted afterwards.
	LegacyRemoved bool
	// Reason explains a no-op, for the log line.
	Reason string
}

// MigrateLegacyDatabase moves launcher-auth.db from legacyDir into privateDir
// before either is opened for normal use.
//
// The database holds the Dashboard's bcrypt verifier. On Android legacyDir is
// PICOCLAW_HOME — shared external storage — where another app with storage
// write access can overwrite the verifier with one for a password it chose and
// then authenticate normally. Moving the file to app-private storage removes
// that vector; this function is what keeps the user's existing password when it
// does.
//
// The order is chosen so the only working credential verifier is never the
// thing at risk:
//
//  1. an existing private database always wins and is never overwritten —
//     otherwise a rollback to attacker-controlled shared state would be one
//     file copy away
//  2. the legacy database is opened through the real store first, which lets
//     SQLite recover any journal a crashed writer left behind, so what gets
//     copied is a consistent file rather than a torn one
//  3. the copy lands on a temp file inside the destination directory, is
//     fsynced, and is renamed within that same filesystem — os.Rename across
//     /sdcard and app-private storage would be a cross-device link error, so
//     the copy is not optional
//  4. the destination is opened and validated through the same store contract
//     before anything is deleted
//  5. only then is the legacy file removed
//
// Every failure path leaves the legacy database intact and returns without
// migrating. A caller that sees an error should keep using the legacy store
// rather than locking the user out of their own Dashboard.
func MigrateLegacyDatabase(ctx context.Context, legacyDir, privateDir string) (MigrationResult, error) {
	if legacyDir == "" || privateDir == "" || filepath.Clean(legacyDir) == filepath.Clean(privateDir) {
		return MigrationResult{Reason: "no separate private directory configured"}, nil
	}

	destination := filepath.Join(privateDir, DBFilename)
	if _, err := os.Stat(destination); err == nil {
		// Private state is authoritative. Idempotent on every later start.
		return MigrationResult{Reason: "private database already present"}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return MigrationResult{}, fmt.Errorf("stat destination: %w", err)
	}

	legacy := filepath.Join(legacyDir, DBFilename)
	legacyInfo, err := os.Stat(legacy)
	if errors.Is(err, os.ErrNotExist) {
		return MigrationResult{Reason: "no legacy database to migrate"}, nil
	} else if err != nil {
		return MigrationResult{}, fmt.Errorf("stat legacy database: %w", err)
	}
	if legacyInfo.IsDir() {
		return MigrationResult{}, fmt.Errorf("legacy database path is a directory: %s", legacy)
	}

	// Let SQLite settle the legacy file before it is copied. If a previous
	// process died mid-write, opening it here replays or rolls back the
	// journal; copying the raw bytes without doing so could carry a torn
	// database across.
	legacyStore, err := Open(legacy)
	if err != nil {
		return MigrationResult{}, fmt.Errorf("open legacy database: %w", err)
	}
	legacyInitialized, err := legacyStore.IsInitialized(ctx)
	closeErr := legacyStore.Close()
	if err != nil {
		return MigrationResult{}, fmt.Errorf("read legacy database: %w", err)
	}
	if closeErr != nil {
		return MigrationResult{}, fmt.Errorf("close legacy database: %w", closeErr)
	}

	if err := os.MkdirAll(privateDir, 0o700); err != nil {
		return MigrationResult{}, fmt.Errorf("create private directory: %w", err)
	}

	temp := destination + ".migrating"
	if err := copyFileSynced(legacy, temp); err != nil {
		os.Remove(temp)
		return MigrationResult{}, fmt.Errorf("copy legacy database: %w", err)
	}
	// Within one filesystem, so this is atomic.
	if err := os.Rename(temp, destination); err != nil {
		os.Remove(temp)
		return MigrationResult{}, fmt.Errorf("install migrated database: %w", err)
	}

	// Validate through the real store contract, not by trusting the byte copy.
	migratedStore, err := Open(destination)
	if err != nil {
		os.Remove(destination)
		return MigrationResult{}, fmt.Errorf("open migrated database: %w", err)
	}
	migratedInitialized, err := migratedStore.IsInitialized(ctx)
	closeErr = migratedStore.Close()
	if err != nil || closeErr != nil || migratedInitialized != legacyInitialized {
		// The destination is unusable or disagrees with the source. Remove it
		// and leave the legacy database exactly where it is: it is still the
		// only thing that can verify the user's password.
		os.Remove(destination)
		if err != nil {
			return MigrationResult{}, fmt.Errorf("validate migrated database: %w", err)
		}
		if closeErr != nil {
			return MigrationResult{}, fmt.Errorf("close migrated database: %w", closeErr)
		}
		return MigrationResult{}, errors.New(
			"migrated database does not carry the credential the legacy one had")
	}

	result := MigrationResult{Migrated: true}
	if err := os.Remove(legacy); err == nil {
		result.LegacyRemoved = true
	}
	// Only sidecars of this exact database, by exact name. Nothing else in the
	// shared directory is touched.
	for _, suffix := range legacySidecarSuffixes {
		os.Remove(legacy + suffix)
	}
	return result, nil
}

// copyFileSynced copies src to dst and forces the result to disk.
//
// The fsync matters: without it a power loss between the rename and the flush
// could leave a present-but-empty destination, which would look like a
// successful migration of an empty credential store.
func copyFileSynced(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
