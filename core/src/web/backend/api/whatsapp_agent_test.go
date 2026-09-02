package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/whatsapp/pairing"
	"github.com/sipeed/picoclaw/pkg/whatsapp/session"
)

func newWhatsAppAgentMux(t *testing.T, configPath string) *http.ServeMux {
	t.Helper()
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestWhatsAppAgentStatus_ReportsUnavailableWithoutAHost(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	t.Setenv(pairing.EnvStateDir, "")

	rec := httptest.NewRecorder()
	newWhatsAppAgentMux(t, configPath).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, whatsAppAgentStatusPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp whatsAppAgentStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Available {
		t.Error("Available = true with no host directory")
	}
	if resp.State != string(pairing.StateUnavailable) {
		t.Errorf("State = %q, want %q", resp.State, pairing.StateUnavailable)
	}
}

// TestWhatsAppAgentStatus_NeverCarriesThePairingPayload is the reason the QR is
// served as an image. A code in this JSON would sit in the browser's memory, in
// any response cache, and in a devtools network log.
func TestWhatsAppAgentStatus_NeverCarriesThePairingPayload(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	dir := t.TempDir()
	t.Setenv(pairing.EnvStateDir, dir)

	const code = "2@secret-pairing-payload"
	if err := pairing.NewStoreAt(dir).PublishQR(code); err != nil {
		t.Fatalf("PublishQR: %v", err)
	}

	rec := httptest.NewRecorder()
	newWhatsAppAgentMux(t, configPath).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, whatsAppAgentStatusPath, nil))

	if bytes.Contains(rec.Body.Bytes(), []byte(code)) {
		t.Errorf("the status response carries the pairing payload: %s", rec.Body.String())
	}

	var resp whatsAppAgentStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.HasQR {
		t.Error("HasQR = false while a code is live")
	}
	if resp.State != string(pairing.StatePairing) {
		t.Errorf("State = %q, want %q", resp.State, pairing.StatePairing)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}
}

func TestWhatsAppAgentQR_RendersAPNGAndIsNotCacheable(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	dir := t.TempDir()
	t.Setenv(pairing.EnvStateDir, dir)

	const code = "2@secret-pairing-payload"
	if err := pairing.NewStoreAt(dir).PublishQR(code); err != nil {
		t.Fatalf("PublishQR: %v", err)
	}

	rec := httptest.NewRecorder()
	newWhatsAppAgentMux(t, configPath).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, whatsAppAgentQRPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", got)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
		t.Error("response is not a PNG")
	}
	// The rendered image must not embed the payload as readable bytes.
	if bytes.Contains(rec.Body.Bytes(), []byte(code)) {
		t.Error("the rendered PNG carries the pairing payload verbatim")
	}
	if got := rec.Header().Get("Cache-Control"); got == "" || !bytes.Contains([]byte(got), []byte("no-store")) {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestWhatsAppAgentQR_IsNotFoundWhenNoCodeIsLive(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	dir := t.TempDir()
	t.Setenv(pairing.EnvStateDir, dir)

	// Connected, not pairing: there is no code to serve.
	if err := pairing.NewStoreAt(dir).PublishState(pairing.StateConnected, ""); err != nil {
		t.Fatalf("PublishState: %v", err)
	}

	rec := httptest.NewRecorder()
	newWhatsAppAgentMux(t, configPath).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, whatsAppAgentQRPath, nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestWhatsAppAgentForget_ErasesTheSessionAndTheSnapshot(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	pairDir := t.TempDir()
	sessionDir := t.TempDir()
	t.Setenv(pairing.EnvStateDir, pairDir)
	t.Setenv(session.EnvStoreDir, sessionDir)

	if err := pairing.NewStoreAt(pairDir).PublishQR("2@code"); err != nil {
		t.Fatalf("PublishQR: %v", err)
	}
	dbPath := filepath.Join(sessionDir, "store.db")
	if err := os.WriteFile(dbPath, []byte("session-keys"), 0o600); err != nil {
		t.Fatalf("write session db: %v", err)
	}

	rec := httptest.NewRecorder()
	newWhatsAppAgentMux(t, configPath).ServeHTTP(
		rec, httptest.NewRequest(http.MethodPost, whatsAppAgentForgetPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Errorf("the session database survived forget (err=%v)", err)
	}
	if snap := pairing.NewStoreAt(pairDir).Read(); snap.HasQR() {
		t.Error("a pairing code survived forget")
	}
	// The directory itself stays: the host created it and owns its mode.
	if _, err := os.Stat(sessionDir); err != nil {
		t.Errorf("the host-owned session directory was removed: %v", err)
	}
}

// TestWhatsAppAgentForget_IgnoresAConfiguredSessionPath keeps this endpoint
// from becoming an arbitrary directory delete. Only the host-named directory is
// ever erased; a path in the config file is untrusted input on Android.
func TestWhatsAppAgentForget_IgnoresAConfiguredSessionPath(t *testing.T) {
	t.Setenv(session.EnvStoreDir, "")

	if err := removeWhatsAppSessionStore(); err != nil {
		t.Fatalf("removeWhatsAppSessionStore with no host directory: %v", err)
	}
}

func TestWhatsAppAgentForget_RejectedWithoutAHost(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	t.Setenv(pairing.EnvStateDir, "")

	rec := httptest.NewRecorder()
	newWhatsAppAgentMux(t, configPath).ServeHTTP(
		rec, httptest.NewRequest(http.MethodPost, whatsAppAgentForgetPath, nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// TestWhatsAppAgentIsInTheCatalogAsExperimental keeps the console entry present
// and marked, without restoring the retired "whatsapp" and "whatsapp_native"
// cards that asked a phone user for a bridge URL.
func TestWhatsAppAgentIsInTheCatalogAsExperimental(t *testing.T) {
	item, ok := findChannelCatalogItem("whatsapp_agent")
	if !ok {
		t.Fatal("whatsapp_agent is not in the channel catalog")
	}
	if item.ConfigKey != "whatsapp_native" {
		t.Errorf("ConfigKey = %q, want whatsapp_native", item.ConfigKey)
	}
	if item.Variant != "experimental" {
		t.Errorf("Variant = %q, want experimental", item.Variant)
	}
	for _, retired := range []string{"whatsapp", "whatsapp_native"} {
		if _, present := findChannelCatalogItem(retired); present {
			t.Errorf("the retired %q card is back in the catalog", retired)
		}
	}
}
