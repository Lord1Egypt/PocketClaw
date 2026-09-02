// Package pairing carries WhatsApp Agent Channel pairing state from the Core
// gateway process, where the whatsmeow client runs, to the console backend
// process, which renders it for an authenticated operator.
//
// # Why a file and not stdout
//
// The upstream native transport prints its pairing QR to stdout with
// qrterminal. On Android that is wrong twice over: Core's stdout is captured
// into PocketClaw's persisted Logs screen, and a WhatsApp pairing QR is
// credential material — anyone who scans it before the user does links their
// own device to the account. The QR must never reach a log.
//
// # Why a file and not a socket
//
// The gateway and the console backend are two processes. A loopback port
// between them would be reachable by every other app on the device and would
// need a shared secret to be safe; an app-private directory is reachable only
// by this UID, so it needs neither. This mirrors the decision already recorded
// for the Core-to-host bridge in pkg/whatsapp/selfchat.
//
// The QR is therefore not held purely in memory — it cannot be, across a
// process boundary, without the socket that decision rejects. It is instead
// held as briefly as possible: written 0600 into an app-private, no-backup
// directory, and removed the moment pairing succeeds, the code expires, or the
// channel stops.
package pairing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

// EnvStateDir names the app-private directory the pairing snapshot lives in.
// The Android host sets it for both the console backend and the gateway it
// spawns. Unset — desktop, CI — pairing state is simply unavailable, and the
// console says so rather than failing obscurely.
const EnvStateDir = "POCKETCLAW_WHATSAPP_PAIR_DIR"

const stateFileName = "pairing.json"

// State is the channel's pairing lifecycle as the console presents it.
type State string

const (
	// StateUnavailable means no host directory was named, so nothing can be
	// reported. It is not an error; it is what desktop and CI look like.
	StateUnavailable State = "unavailable"
	// StateNotPaired means no WhatsApp device is linked yet.
	StateNotPaired State = "not_paired"
	// StatePairing means a QR is live and waiting to be scanned. This is the
	// only state that carries a code.
	StatePairing State = "pairing"
	// StateConnecting means the session exists and the socket is coming up.
	StateConnecting State = "connecting"
	// StateConnected means linked and receiving.
	StateConnected State = "connected"
	// StateDisconnected means a transient drop; the session is intact and
	// reconnect is running.
	StateDisconnected State = "disconnected"
	// StateLoggedOut means WhatsApp revoked the link from the phone. The
	// session is gone and reconnecting cannot fix it.
	StateLoggedOut State = "logged_out"
)

// Snapshot is what the console reads.
type Snapshot struct {
	State State `json:"state"`
	// QR is the raw pairing payload, present only while State is StatePairing.
	// It is never logged and never written under any other state.
	QR string `json:"qr,omitempty"`
	// Detail is a short non-sensitive reason, e.g. why a reconnect is running.
	// It never contains message content, phone numbers, or key material.
	Detail    string    `json:"detail,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HasQR reports whether a live code is waiting to be scanned.
func (s Snapshot) HasQR() bool { return s.State == StatePairing && s.QR != "" }

// Dir returns the configured state directory, or "" when there is no host.
func Dir() string { return strings.TrimSpace(os.Getenv(EnvStateDir)) }

// Available reports whether pairing state can be exchanged at all.
func Available() bool { return Dir() != "" }

// Store publishes and reads the pairing snapshot. The zero value is unusable;
// construct it with NewStore.
type Store struct {
	dir string
	mu  sync.Mutex
}

// NewStore binds a Store to the host-named directory. It returns nil when no
// host directory is configured, which callers treat as "nothing to publish".
func NewStore() *Store {
	dir := Dir()
	if dir == "" {
		return nil
	}
	return &Store{dir: dir}
}

// NewStoreAt binds a Store to an explicit directory. Tests use this; the
// running system uses NewStore so the path can only come from the host.
func NewStoreAt(dir string) *Store {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	return &Store{dir: dir}
}

func (s *Store) path() string { return filepath.Join(s.dir, stateFileName) }

// PublishQR records a live pairing code. This is the only call that ever puts
// a QR on disk.
func (s *Store) PublishQR(code string) error {
	if s == nil || strings.TrimSpace(code) == "" {
		return nil
	}
	return s.write(Snapshot{State: StatePairing, QR: code, UpdatedAt: time.Now().UTC()})
}

// PublishState records a lifecycle state. It never carries a QR: moving to any
// state other than StatePairing is exactly what retires the previous code.
func (s *Store) PublishState(state State, detail string) error {
	if s == nil {
		return nil
	}
	return s.write(Snapshot{State: state, Detail: detail, UpdatedAt: time.Now().UTC()})
}

// Clear removes the snapshot entirely, taking any live QR with it.
func (s *Store) Clear() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) write(snap Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 0700 before the atomic write, whose MkdirAll would otherwise create the
	// directory 0755. Creating it here is a no-op when it already exists, so
	// this cannot loosen a directory the host already made.
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(s.path(), data, 0o600)
}

// Read returns the current snapshot. A missing or unreadable file is reported
// as StateNotPaired rather than an error: the console has to render something,
// and "no device linked" is the truthful reading of "the gateway published
// nothing".
func (s *Store) Read() Snapshot {
	if s == nil {
		return Snapshot{State: StateUnavailable}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path())
	if err != nil {
		return Snapshot{State: StateNotPaired}
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{State: StateNotPaired}
	}
	// A QR that survived under a non-pairing state would be a stale credential
	// served to the console. Drop it on read as well as on write.
	if snap.State != StatePairing {
		snap.QR = ""
	}
	return snap
}
