package pid

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// The Gateway's discovery record, and the one place either filename is spelled.
//
// The record is deliberately shared, discoverable metadata — host, pid, port,
// version, and never a credential — so this is a rename, not a relocation. It
// stays in POCKETCLAW_HOME beside the workspace, exactly where it was.
const (
	// CanonicalPidFileName is what this build writes. Nothing else.
	CanonicalPidFileName = ".pocketclaw.pid"

	// legacyPidFileName is LEGACY READ-ONLY MIGRATION.
	//
	// Read to discover a Gateway an older build started, and deleted once that
	// Gateway is gone. Never written, never recreated, never symlinked, and
	// never copied back to. A running old build owns this file and will remove
	// it itself on shutdown; nothing here writes into a live process's record.
	legacyPidFileName = ".picoclaw.pid"
)

// errPidConflict is returned when both records name different live Gateways.
var errPidConflict = errors.New("conflicting gateway pid records")

// recordState is what one PID file on disk turned out to be.
type recordState int

const (
	// recordAbsent: no such file.
	recordAbsent recordState = iota
	// recordInvalid: present but not a usable record.
	recordInvalid
	// recordStale: parsed, but nothing worth honouring is behind it.
	recordStale
	// recordLive: parsed, and the process it names should be honoured.
	recordLive
)

// pidRecord is one file's contribution to the decision.
type pidRecord struct {
	path  string
	state recordState
	data  *PidFileData
}

func (r pidRecord) live() bool { return r.state == recordLive }

// removable reports whether this file can be deleted without destroying
// something another process still depends on.
//
// A live record is never removable. That is the whole reason the two records
// are resolved together rather than one being renamed onto the other: an old
// build that is still running holds the legacy path in its own shutdown code,
// and moving the file out from under it would leave it removing a name that no
// longer exists while its record lingered forever.
func (r pidRecord) removable() bool {
	return r.state == recordInvalid || r.state == recordStale
}

// livenessFunc decides whether a parsed record names a process to honour.
//
// There are deliberately two of these. Startup asks whether the PID is a live
// Core executable, because wrongly honouring a recycled PID wedges the Gateway
// behind a record for a process that is not it. The status API asks only
// whether the PID is alive, which is what it asked before this migration; the
// stricter rule there would change what the console reports, and N4G is a
// filename migration. The difference is in the predicate, never in the
// migration, cleanup or precedence policy below.
type livenessFunc func(data *PidFileData) bool

// runningAndOwned is startup's rule: a live process that is one of our Core
// executables. PID 1 is a container leftover on a shared volume — the host's
// init — and is always stale.
func runningAndOwned(data *PidFileData) bool {
	if data.PID == 1 {
		return false
	}
	return isProcessRunning(data.PID) && isPicoclawProcess(data.PID)
}

// runningOnly is the status API's pre-existing rule, unchanged.
func runningOnly(data *PidFileData) bool {
	if data.PID == 1 && os.Getpid() != 1 {
		return false
	}
	return isProcessRunning(data.PID)
}

// classifyRecord reads one file and says what it is. Caller must hold pidMu.
func classifyRecord(path string, live livenessFunc) pidRecord {
	data, err := readPidFileUnlocked(path)
	switch {
	case err == nil && live(data):
		return pidRecord{path: path, state: recordLive, data: data}
	case err == nil:
		return pidRecord{path: path, state: recordStale, data: data}
	case os.IsNotExist(err):
		return pidRecord{path: path, state: recordAbsent}
	case errors.Is(err, errInvalidPidFile):
		// Malformed discovery metadata must not wedge startup permanently.
		// There is no credential in this file and nothing to preserve, so it is
		// cleaned by the same stale policy as any other unusable record.
		return pidRecord{path: path, state: recordInvalid}
	default:
		// An unreadable file says nothing about who owns the PID, and deleting
		// it on an I/O error would throw away a record that may be perfectly
		// good. Treated as invalid for selection but left on disk.
		logger.Debugf("failed to read pid file %s: %v", path, err)
		return pidRecord{path: path, state: recordAbsent}
	}
}

// resolvePidRecords reads both records under one policy.
//
// Every caller — startup, the status API, shutdown and the web console's
// cleanup — comes through here, so there is one migration policy, one
// precedence rule and one stale rule rather than each caller trying two
// filenames its own way. Caller must hold pidMu.
func resolvePidRecords(homePath string, live livenessFunc) (canonical, legacy pidRecord) {
	canonical = classifyRecord(filepath.Join(homePath, CanonicalPidFileName), live)
	legacy = classifyRecord(filepath.Join(homePath, legacyPidFileName), live)
	return canonical, legacy
}

// activeRecord picks the record to honour, and reports a conflict rather than
// choosing between two live Gateways.
//
// Canonical wins when both name the same process: it is this build's name for
// the same thing, and the legacy file is then redundant rather than
// authoritative. Two different live PIDs is a split-brain discovery state that
// nothing on disk can resolve — killing one, overwriting a record or starting a
// third are all worse than saying so.
func activeRecord(canonical, legacy pidRecord) (pidRecord, error) {
	switch {
	case canonical.live() && legacy.live():
		if canonical.data.PID == legacy.data.PID {
			return canonical, nil
		}
		return pidRecord{}, errPidConflict
	case canonical.live():
		return canonical, nil
	case legacy.live():
		return legacy, nil
	default:
		return pidRecord{}, nil
	}
}

// cleanSupersededRecords deletes what is safe to delete once active is chosen.
//
// Stale and malformed records go, and so does a live record that names the same
// process the active one does — two names for one Gateway, of which this build
// writes only the canonical one.
//
// A live record naming a *different* process is never removed. That is an older
// build's only way to find its own Gateway again, and deleting it would leave
// that process running with nothing pointing at it.
func cleanSupersededRecords(active pidRecord, records ...pidRecord) {
	for _, record := range records {
		if record.path == active.path {
			continue
		}
		redundant := record.live() && active.live() &&
			record.data.PID == active.data.PID
		if !record.removable() && !redundant {
			continue
		}
		logger.Debugf("removing superseded pid file: %s", record.path)
		_ = os.Remove(record.path)
	}
}
