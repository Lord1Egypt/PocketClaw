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

func TestPublishPairing_CarriesBothCredentials(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreAt(dir)

	if err := store.PublishPairing("2@qr-payload", "ABCD1234"); err != nil {
		t.Fatalf("PublishPairing: %v", err)
	}

	snap := store.Read()
	if !snap.HasQR() || snap.QR != "2@qr-payload" {
		t.Errorf("QR = %q, want the published payload", snap.QR)
	}
	if !snap.HasCode() || snap.Code != "ABCD1234" {
		t.Errorf("Code = %q, want the published code", snap.Code)
	}
}

// TestPublishPairing_KeepsTheCodeAcrossQRRefreshes is the reason an empty
// argument means "keep". whatsmeow regenerates the QR every twenty seconds or
// so; if that dropped the companion code, it would change under a user who is
// part-way through typing it into WhatsApp.
func TestPublishPairing_KeepsTheCodeAcrossQRRefreshes(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreAt(dir)

	if err := store.PublishPairing("2@first-qr", "ABCD1234"); err != nil {
		t.Fatalf("PublishPairing: %v", err)
	}
	if err := store.PublishQR("2@second-qr"); err != nil {
		t.Fatalf("PublishQR: %v", err)
	}

	snap := store.Read()
	if snap.QR != "2@second-qr" {
		t.Errorf("QR = %q, want the refreshed payload", snap.QR)
	}
	if snap.Code != "ABCD1234" {
		t.Errorf("Code = %q, want the original code to survive the refresh", snap.Code)
	}
}

// TestPublishState_RetiresBothCredentials covers pair, cancel, timeout, logout
// and disconnect at once: every one of them is a state change away from
// StatePairing, and that is what must erase the credentials.
func TestPublishState_RetiresBothCredentials(t *testing.T) {
	for _, state := range []State{
		StateConnected, StateNotPaired, StateConnecting, StateDisconnected, StateLoggedOut,
	} {
		t.Run(string(state), func(t *testing.T) {
			dir := t.TempDir()
			store := NewStoreAt(dir)

			if err := store.PublishPairing("2@qr-payload", "ABCD1234"); err != nil {
				t.Fatalf("PublishPairing: %v", err)
			}
			if err := store.PublishState(state, ""); err != nil {
				t.Fatalf("PublishState: %v", err)
			}

			snap := store.Read()
			if snap.QR != "" || snap.Code != "" || snap.HasQR() || snap.HasCode() {
				t.Errorf("credentials survived the move to %q: qr=%q code=%q", state, snap.QR, snap.Code)
			}

			raw, err := os.ReadFile(filepath.Join(dir, stateFileName))
			if err != nil {
				t.Fatalf("read snapshot: %v", err)
			}
			for _, credential := range []string{"2@qr-payload", "ABCD1234"} {
				if containsCode(t, raw, credential) {
					t.Errorf("%q is still on disk after moving to %q", credential, state)
				}
			}
		})
	}
}

func TestClear_RemovesBothCredentials(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreAt(dir)

	if err := store.PublishPairing("2@qr-payload", "ABCD1234"); err != nil {
		t.Fatalf("PublishPairing: %v", err)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	snap := store.Read()
	if snap.HasQR() || snap.HasCode() {
		t.Error("a credential survived Clear")
	}
}

func TestRead_DropsACodeThatSurvivedUnderANonPairingState(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreAt(dir)

	forged, err := json.Marshal(Snapshot{State: StateConnected, Code: "ABCD1234"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, stateFileName), forged, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if snap := store.Read(); snap.Code != "" || snap.HasCode() {
		t.Errorf("Read returned a pairing code under state %q", snap.State)
	}
}

func TestPublishPairing_IgnoresTwoEmptyArguments(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreAt(dir)

	if err := store.PublishPairing("  ", ""); err != nil {
		t.Fatalf("PublishPairing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, stateFileName)); !os.IsNotExist(err) {
		t.Error("an empty publish created a snapshot")
	}
}

func TestNilStoreAcceptsPairingCredentials(t *testing.T) {
	var store *Store
	if err := store.PublishPairing("2@qr", "ABCD1234"); err != nil {
		t.Errorf("PublishPairing on a nil store: %v", err)
	}
}
