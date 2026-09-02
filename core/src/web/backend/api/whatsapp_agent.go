package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"rsc.io/qr"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/whatsapp/pairing"
	"github.com/sipeed/picoclaw/pkg/whatsapp/session"
)

// The WhatsApp Agent Channel is experimental. These endpoints are the only
// authenticated surface its pairing code is ever exposed on: the code is
// credential material, so it must not reach the gateway's stdout, which
// PocketClaw captures into a persisted Logs screen.
const (
	whatsAppAgentStatusPath = "/api/channels/whatsapp-agent/status"
	whatsAppAgentQRPath     = "/api/channels/whatsapp-agent/qr.png"
	whatsAppAgentForgetPath = "/api/channels/whatsapp-agent/forget"
)

// whatsAppAgentQRScale is the image pixels per QR module. WhatsApp's pairing
// payload is long enough to produce a dense symbol, and a 1:1 PNG is too small
// for a phone camera to resolve off a screen.
const whatsAppAgentQRScale = 8

type whatsAppAgentStatusResponse struct {
	// Available is false off Android, where no host directory is named and no
	// pairing state can be exchanged at all.
	Available bool   `json:"available"`
	Enabled   bool   `json:"enabled"`
	State     string `json:"state"`
	// HasQR reports that a code is waiting to be scanned. The payload itself is
	// deliberately not in this response: it is fetched as a rendered image from
	// whatsAppAgentQRPath, so the raw pairing credential never exists as a
	// string in the browser, in a JSON cache, or in a devtools network log.
	HasQR  bool   `json:"has_qr"`
	Detail string `json:"detail,omitempty"`
}

func (h *Handler) registerWhatsAppAgentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+whatsAppAgentStatusPath, h.handleWhatsAppAgentStatus)
	mux.HandleFunc("GET "+whatsAppAgentQRPath, h.handleWhatsAppAgentQR)
	mux.HandleFunc("POST "+whatsAppAgentForgetPath, h.handleWhatsAppAgentForget)
}

// whatsAppAgentEnabled reports whether the channel entry is enabled on disk.
func (h *Handler) whatsAppAgentEnabled() bool {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil || cfg == nil {
		return false
	}
	bc := cfg.Channels.Get(config.ChannelWhatsAppNative)
	return bc != nil && bc.Enabled
}

// handleWhatsAppAgentStatus returns pairing state for the console panel.
//
//	GET /api/channels/whatsapp-agent/status
func (h *Handler) handleWhatsAppAgentStatus(w http.ResponseWriter, r *http.Request) {
	resp := whatsAppAgentStatusResponse{
		Available: pairing.Available(),
		Enabled:   h.whatsAppAgentEnabled(),
		State:     string(pairing.StateUnavailable),
	}
	if resp.Available {
		snap := pairing.NewStore().Read()
		resp.State = string(snap.State)
		resp.Detail = snap.Detail
		resp.HasQR = snap.HasQR()
	}
	w.Header().Set("Content-Type", "application/json")
	// A live pairing code must not be cached by anything between here and the
	// console tab.
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleWhatsAppAgentQR renders the live pairing code as a PNG.
//
// Rendering server-side is what keeps the payload out of the browser: the
// console shows an image it cannot read back, and nothing logs a string that
// would link a device to the user's WhatsApp account.
//
//	GET /api/channels/whatsapp-agent/qr.png
func (h *Handler) handleWhatsAppAgentQR(w http.ResponseWriter, r *http.Request) {
	snap := pairing.NewStore().Read()
	if !snap.HasQR() {
		http.NotFound(w, r)
		return
	}

	code, err := qr.Encode(snap.QR, qr.M)
	if err != nil {
		logger.Errorf("Failed to encode WhatsApp pairing code: %v", err)
		http.Error(w, "Could not render the pairing code", http.StatusInternalServerError)
		return
	}
	code.Scale = whatsAppAgentQRScale

	w.Header().Set("Content-Type", "image/png")
	// The code is short-lived and single-use. Nothing between here and the tab
	// may keep a copy.
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	w.Header().Set("Pragma", "no-cache")
	_, _ = w.Write(code.PNG())
}

// handleWhatsAppAgentForget erases the local WhatsApp session and any pairing
// snapshot. The console calls it after disabling the channel and restarting the
// gateway, so nothing holds the database open.
//
//	POST /api/channels/whatsapp-agent/forget
func (h *Handler) handleWhatsAppAgentForget(w http.ResponseWriter, r *http.Request) {
	if !pairing.Available() {
		http.Error(w, "WhatsApp Agent Channel is only available inside the PocketClaw Android app", http.StatusBadRequest)
		return
	}

	_ = pairing.NewStore().Clear()

	if err := removeWhatsAppSessionStore(); err != nil {
		logger.Errorf("Failed to erase WhatsApp session store: %v", err)
		http.Error(w, "Could not erase the WhatsApp session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "forgotten"})
}

// removeWhatsAppSessionStore deletes the host-owned session directory.
//
// Only the host-named path is ever removed. A configured path is not consulted
// here for the same reason the transport ignores it: on Android it is untrusted
// input, and honouring it would turn this endpoint into an arbitrary directory
// delete driven by whatever wrote the config file.
func removeWhatsAppSessionStore() error {
	dir := strings.TrimSpace(os.Getenv(session.EnvStoreDir))
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
