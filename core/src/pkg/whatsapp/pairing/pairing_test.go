package pairing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishQR_IsReadableAndMarkedPairing(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreAt(dir)

	if err := store.PublishQR("2@abc,def,ghi"); err != nil {
		t.Fatalf("PublishQR: %v", err)
	}

	snap := store.Read()
	if snap.State != StatePairing {
		t.Errorf("State = %q, want %q", snap.State, StatePairing)
	}
	if snap.QR != "2@abc,def,ghi" {
		t.Errorf("QR = %q, want the published code", snap.QR)
	}
	if !snap.HasQR() {
		t.Error("HasQR() = false for a live pairing code")
	}
}

func TestPublishState_RetiresThePreviousQR(t *testing.T) {
	// Moving off StatePairing is what makes a code stop being offered. If the
	// QR survived a state change the console would keep serving a credential
	// that already linked a device.
	dir := t.TempDir()
	store := NewStoreAt(dir)

	if err := store.PublishQR("2@live-code"); err != nil {
		t.Fatalf("PublishQR: %v", err)
	}
	if err := store.PublishState(StateConnected, ""); err != nil {
		t.Fatalf("PublishState: %v", err)
	}

	snap := store.Read()
	if snap.QR != "" {
		t.Errorf("QR = %q, want it dropped once pairing finished", snap.QR)
	}
	if snap.HasQR() {
		t.Error("HasQR() = true after the state left pairing")
	}

	raw, err := os.ReadFile(filepath.Join(dir, stateFileName))
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(raw) == "" {
		t.Fatal("snapshot is empty")
	}
	if containsCode(t, raw, "2@live-code") {
		t.Error("the retired pairing code is still on disk")
	}
}

// containsCode reports whether the serialized snapshot still carries the code
// in any field, not merely in the one Read happens to look at.
func containsCode(t *testing.T, raw []byte, code string) bool {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("snapshot is not JSON: %v", err)
	}
	for _, v := range fields {
		if s, ok := v.(string); ok && s == code {
			return true
		}
	}
	return false
}

func TestRead_DropsAQRThatSurvivedUnderANonPairingState(t *testing.T) {
	// A hand-edited or half-written file must not be able to serve a code under
	// a state that says pairing is over.
	dir := t.TempDir()
	store := NewStoreAt(dir)

	forged, err := json.Marshal(Snapshot{State: StateConnected, QR: "2@forged"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, stateFileName), forged, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if snap := store.Read(); snap.QR != "" || snap.HasQR() {
		t.Errorf("Read returned a QR under state %q", snap.State)
	}
}

func TestClear_RemovesTheSnapshot(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreAt(dir)

	if err := store.PublishQR("2@code"); err != nil {
		t.Fatalf("PublishQR: %v", err)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, stateFileName)); !os.IsNotExist(err) {
		t.Errorf("snapshot still present after Clear (err=%v)", err)
	}
	if snap := store.Read(); snap.State != StateNotPaired {
		t.Errorf("State after Clear = %q, want %q", snap.State, StateNotPaired)
	}
}

func TestClear_OnAMissingSnapshotIsNotAnError(t *testing.T) {
	if err := NewStoreAt(t.TempDir()).Clear(); err != nil {
		t.Errorf("Clear on a missing snapshot returned %v", err)
	}
}

func TestSnapshotIsOwnerOnlyOnDisk(t *testing.T) {
	// The snapshot carries a pairing credential. Another app on the device, or
	// another user, must not be able to read it.
	dir := t.TempDir()
	store := NewStoreAt(dir)

	if err := store.PublishQR("2@code"); err != nil {
		t.Fatalf("PublishQR: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, stateFileName))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("snapshot mode = %o, want 600", perm)
	}
}

func TestPublishCreatesTheDirectoryOwnerOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pairing")
	store := NewStoreAt(dir)

	if err := store.PublishState(StateNotPaired, ""); err != nil {
		t.Fatalf("PublishState: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("directory mode = %o, want 700", perm)
	}
}

func TestNewStore_IsNilWithoutAHostDirectory(t *testing.T) {
	t.Setenv(EnvStateDir, "")
	if NewStore() != nil {
		t.Error("NewStore() returned a store with no host directory")
	}
	if Available() {
		t.Error("Available() = true with no host directory")
	}
}

func TestNilStoreIsSafeToUse(t *testing.T) {
	// Off Android the transport still runs and still calls these. They must be
	// no-ops rather than a nil dereference in the message path.
	var store *Store
	if err := store.PublishQR("2@code"); err != nil {
		t.Errorf("PublishQR on a nil store: %v", err)
	}
	if err := store.PublishState(StateConnected, ""); err != nil {
		t.Errorf("PublishState on a nil store: %v", err)
	}
	if err := store.Clear(); err != nil {
		t.Errorf("Clear on a nil store: %v", err)
	}
	if snap := store.Read(); snap.State != StateUnavailable {
		t.Errorf("Read on a nil store = %q, want %q", snap.State, StateUnavailable)
	}
}

func TestPublishQR_IgnoresAnEmptyCode(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreAt(dir)

	if err := store.PublishQR("   "); err != nil {
		t.Fatalf("PublishQR: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, stateFileName)); !os.IsNotExist(err) {
		t.Error("an empty code created a snapshot")
	}
}
