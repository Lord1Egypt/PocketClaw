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
	// PrivateReady reports that the destination is usable as the authoritative
	// credential store. A caller that demanded private storage must refuse to
	// start when this is false: the alternative is reopening the shared store,
	// which is the attacker-writable state the override exists to escape.
	PrivateReady bool
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
		// No separate private location, so there is nothing to move and the
		// caller's existing store is the only one there is.
		return MigrationResult{
			PrivateReady: true,
			Reason:       "no separate private directory configured",
		}, nil
	}

	legacy := filepath.Join(legacyDir, DBFilename)
	destination := filepath.Join(privateDir, DBFilename)

	if _, err := os.Stat(destination); err == nil {
		// Private state is authoritative and is never replaced from shared
		// state — that path would be the rollback an attacker wants. But it
		// only earns that authority if it actually opens, so it is validated
		// rather than trusted for existing.
		if err := validateStore(ctx, destination); err != nil {
			// Do not delete the legacy database and do not promote it: a
			// caller demanding private storage must fail closed, with the old
			// file left intact for recovery.
			return MigrationResult{
				Reason: "private database present but unusable",
			}, fmt.Errorf("validate private database: %w", err)
		}
		// A stale shared copy must not sit there indefinitely as a rollback
		// artifact. Best effort: failing to remove it cannot move authority
		// back to it, because the private store is already authoritative.
		result := MigrationResult{PrivateReady: true, Reason: "private database already present"}
		result.LegacyRemoved = retireLegacyDatabase(legacy)
		return result, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return MigrationResult{}, fmt.Errorf("stat destination: %w", err)
	}

	legacyInfo, err := os.Stat(legacy)
	if errors.Is(err, os.ErrNotExist) {
		// Nothing to carry across; the store will be created privately.
		return MigrationResult{PrivateReady: true, Reason: "no legacy database to migrate"}, nil
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

	result := MigrationResult{Migrated: true, PrivateReady: true}
	result.LegacyRemoved = retireLegacyDatabase(legacy)
	return result, nil
}

// retireLegacyDatabase deletes the shared database and its sidecars once the
// private store is authoritative, so no rollback artifact is left behind.
//
// Best effort by design. It is only ever called after the private store has
// been validated, so a deletion failure leaves a file that is no longer
// consulted — it cannot move authority back to shared state.
//
// Only this exact database and its exact sidecar names. Nothing is matched by
// pattern, nothing recurses, and nothing else in the shared directory is
// touched: everything else there is the user's.
func retireLegacyDatabase(legacy string) bool {
	removed := os.Remove(legacy) == nil
	for _, suffix := range legacySidecarSuffixes {
		os.Remove(legacy + suffix)
	}
	return removed
}

// validateStore opens path through the real store contract and confirms it can
// answer the one question authentication depends on.
func validateStore(ctx context.Context, path string) error {
	store, err := Open(path)
	if err != nil {
		return err
	}
	_, readErr := store.IsInitialized(ctx)
	closeErr := store.Close()
	if readErr != nil {
		return readErr
	}
	return closeErr
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
