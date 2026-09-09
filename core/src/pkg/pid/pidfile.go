package pid

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"

	"github.com/sipeed/picoclaw/pkg/canonicalenv"
)

var errInvalidPidFile = errors.New("invalid pid file")

// PidFileData is the JSON structure stored in the PID file.
//
// Everything here except Token is discovery metadata: which process, which
// build, where to reach it. Token is a credential, and `omitempty` is what lets
// the record be written without one — see writeTokenSink.
type PidFileData struct {
	PID     int    `json:"pid"`
	Token   string `json:"token,omitempty"`
	Version string `json:"version"`
	Port    int    `json:"port"`
	Host    string `json:"host"`
}

// writeTokenSink writes the bearer credential to the path named by
// EnvGatewayTokenFile and reports whether it took ownership of it.
//
// The PID record lives in PICOCLAW_HOME. On Android that is a user-visible
// directory on shared external storage, where the 0600 the record is written
// with is synthesised by the filesystem rather than enforced — so any app with
// storage access can read whatever is in it. The credential therefore goes to a
// separate file the host places somewhere only the app can reach.
//
// Returns false when no sink is configured, which is the desktop and server
// case: PICOCLAW_HOME is already private there and the token stays in the
// record, exactly as before.
func writeTokenSink(token string) bool {
	path := strings.TrimSpace(canonicalenv.Getenv(config.EnvGatewayTokenFile))
	if path == "" {
		return false
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		logger.Warnf("gateway token sink directory unavailable: %v", err)
		return false
	}
	// Written and renamed rather than truncated in place, so a reader never
	// sees a half-written credential.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(token), 0o600); err != nil {
		logger.Warnf("failed to write gateway token file: %v", err)
		return false
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		logger.Warnf("failed to install gateway token file: %v", err)
		return false
	}
	return true
}

// removeTokenSink deletes the private credential file, if one is configured.
func removeTokenSink() {
	path := strings.TrimSpace(canonicalenv.Getenv(config.EnvGatewayTokenFile))
	if path == "" {
		return
	}
	os.Remove(path)
}

var pidMu sync.Mutex

// pidFilePath returns the absolute path of the record this build writes.
//
// Only the canonical name. The legacy record is discovered through
// resolvePidRecords and is never a write target.
func pidFilePath(homePath string) string {
	return filepath.Join(homePath, CanonicalPidFileName)
}

// generateToken creates a cryptographically random 32-character hex token.
func generateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to something pseudo-random if crypto/rand fails
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// WritePidFile creates (or overwrites) the PID file atomically.
// It returns an error if another gateway instance appears to be running
// (a valid PID file exists with a live process).
func WritePidFile(homePath, host string, port int) (*PidFileData, error) {
	pidMu.Lock()
	defer pidMu.Unlock()

	pidPath := pidFilePath(homePath)

	// Singleton enforcement across both record names. An older build's Gateway
	// is still a running Gateway, so its record blocks a duplicate start just
	// as the canonical one does — and its file is left where it is, because
	// that process will remove it itself and knows it by no other name.
	canonical, legacy := resolvePidRecords(homePath, runningAndOwned)
	active, err := activeRecord(canonical, legacy)
	if err != nil {
		// Two live Gateways, two records, and nothing on disk to choose
		// between them. Killing one, overwriting a record or starting a third
		// are all worse than reporting it.
		return nil, fmt.Errorf("%w: %s names PID %d and %s names PID %d",
			err, canonical.path, canonical.data.PID, legacy.path, legacy.data.PID)
	}
	for _, record := range []pidRecord{canonical, legacy} {
		if record.state == recordStale {
			logger.Warnf("not running (PID: %d) so will remove the pid file: %s",
				record.data.PID, record.path)
		}
	}
	// Superseded records go whether or not we go on to start, so a rejected
	// start still leaves one record for one Gateway. Our own is overwritten
	// below rather than removed first, so there is no window with none at all.
	cleanSupersededRecords(active, canonical, legacy)

	if active.live() && active.data.PID != os.Getpid() {
		return nil, fmt.Errorf("gateway is already running (PID: %d, version: %s)",
			active.data.PID, active.data.Version)
	}

	data := &PidFileData{
		PID:     os.Getpid(),
		Version: config.GetVersion(),
		Port:    port,
		Host:    host,
	}

	token := generateToken()
	data.Token = token

	// The caller needs the token in memory to configure the health server. The
	// record on disk is a different question: when a private sink is
	// configured the credential goes there and is cleared from the copy that
	// gets serialised, so nothing under PICOCLAW_HOME ever carries it.
	record := *data
	if writeTokenSink(token) {
		record.Token = ""
	}

	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pid file: %w", err)
	}

	// Ensure parent directory exists.
	dir := filepath.Dir(pidPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create pid directory: %w", err)
	}

	// Write atomically via temp file + rename.
	tmp := pidPath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write pid file: %w", err)
	}
	if err := os.Rename(tmp, pidPath); err != nil {
		os.Remove(tmp)
		return nil, fmt.Errorf("failed to rename pid file: %w", err)
	}
	logger.Debugf("wrote pid file: %s success", pidPath)

	return data, nil
}

// ReadPidFileWithCheck reads the PID file and additionally checks if
// the recorded process is still alive. Returns nil if the file is
// missing, unreadable, or the process has exited.
func ReadPidFileWithCheck(homePath string) *PidFileData {
	pidMu.Lock()
	defer pidMu.Unlock()

	// Liveness here is "the PID is alive", which is what this API asked before
	// the migration and what the console reports on. Startup applies the
	// stricter ownership rule; both share the migration and cleanup policy.
	canonical, legacy := resolvePidRecords(homePath, runningOnly)
	active, err := activeRecord(canonical, legacy)
	if err != nil {
		// Reporting one of two live Gateways as "the" Gateway would be a guess.
		// Nothing is removed, and a caller that acts on this gets the same
		// conflict from WritePidFile rather than a silent third instance.
		logger.Warnf("conflicting gateway pid records: %s names PID %d and %s names PID %d",
			canonical.path, canonical.data.PID, legacy.path, legacy.data.PID)
		return nil
	}

	cleanSupersededRecords(active, canonical, legacy)
	if !active.live() {
		return nil
	}
	return active.data
}

// RemovePidFile deletes the PID file (e.g. on graceful shutdown).
func RemovePidFile(homePath string) {
	pidMu.Lock()
	defer pidMu.Unlock()

	pidPath := pidFilePath(homePath)
	// Only remove if the PID matches our own process (avoid deleting
	// a file that belongs to a newer gateway instance).
	if data, err := readPidFileUnlocked(pidPath); err == nil {
		if data.PID != os.Getpid() {
			return
		}
	}

	logger.Infof("remove pid file: %s", pidPath)
	os.Remove(pidPath)
	removeTokenSink()
}

// RemovePidFileIfPID deletes the PID file only when the recorded PID matches
// expectedPID. It returns true when the file is removed successfully.
func RemovePidFileIfPID(homePath string, expectedPID int) bool {
	if expectedPID <= 0 {
		return false
	}

	pidMu.Lock()
	defer pidMu.Unlock()

	// Whichever name the record carries. The console calls this after stopping
	// a Gateway it started, and one started by an older build left its record
	// under the legacy name — the process is gone either way.
	removed := false
	for _, path := range []string{
		filepath.Join(homePath, CanonicalPidFileName),
		filepath.Join(homePath, legacyPidFileName),
	} {
		data, err := readPidFileUnlocked(path)
		if err != nil || data.PID != expectedPID {
			continue
		}
		if err := os.Remove(path); err == nil {
			removed = true
		}
	}
	return removed
}

// readPidFileUnlocked reads the PID file without acquiring the lock.
// Caller must hold pidMu.
func readPidFileUnlocked(pidPath string) (*PidFileData, error) {
	raw, err := os.ReadFile(pidPath)
	if err != nil {
		return nil, err
	}

	var data PidFileData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("%w: %v", errInvalidPidFile, err)
	}

	// Validate PID is a positive integer.
	if data.PID <= 0 {
		return nil, fmt.Errorf("%w: pid=%d", errInvalidPidFile, data.PID)
	}

	return &data, nil
}
