package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
)

// PC-DEF-020, the second half. Making the host's decision authoritative for the
// listener is not enough on its own: the Config page still read and wrote
// launcher-config.json's `public` field directly, so it could display a value
// the running listener contradicts, and saving the page rewrote the stale value
// that caused the defect in the first place.
//
// Where the host owns the decision the page is not an authority and must not
// present itself as one. Where it does not — desktop, which has no native
// toggle — the stored field is the only way to set Public Mode at all, and that
// must keep working.

func writeStoredLauncherPublic(t *testing.T, configPath string, public bool) {
	t.Helper()
	body := `{"port":18800,"public":false}`
	if public {
		body = `{"port":18800,"public":true}`
	}
	if err := os.WriteFile(launcherconfig.PathForAppConfig(configPath), []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func getLauncherConfigPayload(t *testing.T, h *Handler) launcherConfigPayload {
	t.Helper()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/system/launcher-config", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got launcherConfigPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return got
}

func putLauncherConfigPublic(t *testing.T, h *Handler, public bool) launcherConfigPayload {
	t.Helper()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	body := `{"port":18800,"public":false,"allowed_cidrs":[]}`
	if public {
		body = `{"port":18800,"public":true,"allowed_cidrs":[]}`
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/system/launcher-config", strings.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got launcherConfigPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return got
}

// The reported value follows the listener, not the file, once the host has
// spoken. This is the display half of the defect: the toggle said OFF, the
// listener was loopback, and this page still said LAN access was on.
func TestLauncherConfigReportsTheEffectiveModeWhenTheHostOwnsIt(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeStoredLauncherPublic(t, configPath, true)

	h := NewHandler(configPath)
	h.SetServerOptions(18800, false, true, nil) // explicit -public=false

	if got := getLauncherConfigPayload(t, h); got.Public {
		t.Fatal("GET reported public=true for an explicit host decision of off")
	}
}

// A runtime rebind outranks the startup flag, so the page must follow the
// controller rather than the value the process started with.
func TestLauncherConfigFollowsARuntimeRebind(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeStoredLauncherPublic(t, configPath, false)

	h := NewHandler(configPath)
	h.SetServerOptions(18800, false, true, nil)
	controller := &fakeNetworkModeController{}
	h.SetLauncherNetworkModeController(controller)

	if got := getLauncherConfigPayload(t, h); got.Public {
		t.Fatal("GET reported public=true before any rebind")
	}

	if err := controller.ApplyPublicMode(true); err != nil {
		t.Fatalf("ApplyPublicMode() error = %v", err)
	}
	if got := getLauncherConfigPayload(t, h); !got.Public {
		t.Fatal("GET did not follow the runtime rebind to public")
	}

	if err := controller.ApplyPublicMode(false); err != nil {
		t.Fatalf("ApplyPublicMode() error = %v", err)
	}
	if got := getLauncherConfigPayload(t, h); got.Public {
		t.Fatal("GET did not follow the runtime rebind back to loopback")
	}
}

// Saving the page cannot set a value the host did not choose, and — the part
// that matters for an installation that already drifted — saving repairs a
// stale stored true instead of preserving it.
func TestSavingLauncherConfigCannotOverrideAHostOwnedDecision(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeStoredLauncherPublic(t, configPath, true)

	h := NewHandler(configPath)
	h.SetServerOptions(18800, false, true, nil) // explicit -public=false

	if got := putLauncherConfigPublic(t, h, true); got.Public {
		t.Fatal("PUT accepted public=true against an explicit host decision of off")
	}

	stored, err := launcherconfig.Load(launcherconfig.PathForAppConfig(configPath), launcherconfig.Default())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if stored.Public {
		t.Fatal("a stale stored public=true survived the save; the next start would reopen the LAN listener")
	}
}

// The host-override precedence is preserved here too: with -host set the public
// decision is meaningless, so the page reports off whatever is stored.
func TestLauncherConfigReportsOffUnderAnExplicitHost(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeStoredLauncherPublic(t, configPath, true)

	h := NewHandler(configPath)
	h.SetServerOptions(18800, true, true, nil)
	h.SetServerBindHost("127.0.0.1", true)

	if got := getLauncherConfigPayload(t, h); got.Public {
		t.Fatal("GET reported public=true under an explicit host override")
	}
}

// Desktop. No flag was supplied, so the stored field is the authority and the
// page must still be able to read and write it. Removing this would delete the
// only way to turn Public Mode on where there is no native toggle.
func TestLauncherConfigStaysWritableWhenNoHostOwnsTheDecision(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeStoredLauncherPublic(t, configPath, false)

	h := NewHandler(configPath)
	h.SetServerOptions(18800, false, false, nil) // no -public on the command line

	if got := putLauncherConfigPublic(t, h, true); !got.Public {
		t.Fatal("PUT did not accept public=true with no host-owned decision")
	}
	stored, err := launcherconfig.Load(launcherconfig.PathForAppConfig(configPath), launcherconfig.Default())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !stored.Public {
		t.Fatal("public=true was not persisted with no host-owned decision")
	}
	if got := getLauncherConfigPayload(t, h); !got.Public {
		t.Fatal("GET did not report the stored public=true")
	}
}
