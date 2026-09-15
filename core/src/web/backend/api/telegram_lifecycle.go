package api

import (
	"net/http"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// The rest of the Telegram lifecycle, for clients that are not the Android host.
//
// PC-DEF-062. Managed pairing worked from a desktop browser, but once Telegram
// was configured the desktop had no way to undo it: the connected card offered
// Open chat and Reconnect, both of which need the native host, so the owner had
// to go back to the phone to remove a bot. Removal is Core's operation, not the
// host's, so it belongs here.
//
// This is the authoritative removal and the only one. It is the mirror image of
// writeTelegramCredentials -- same load, same save, same apply -- so the two
// cannot drift, and in particular so removal goes through PC-DEF-030's runtime
// apply and the running channel actually stops. A client-side delete that only
// cleared fields would leave the old bot polling.

func (h *Handler) registerTelegramLifecycleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("DELETE /api/telegram/configuration", h.handleTelegramDisconnect)
}

// handleTelegramDisconnect removes the configured bot.
//
//	DELETE /api/telegram/configuration
//
// Everything the pairing wrote is cleared together: the token, the owner
// allowlist, and the enabled flag. Leaving any one of them would be worse than
// leaving all three -- a disabled channel that still holds a token and an owner
// reads as "connected" to every surface that asks.
func (h *Handler) handleTelegramDisconnect(w http.ResponseWriter, _ *http.Request) {
	configured, err := h.telegramIsConfigured()
	if err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]any{
			"error": "configuration_unreadable",
		})
		return
	}

	applied, pending, err := h.clearTelegramCredentials()
	if err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]any{
			"error": "configuration_failed",
		})
		return
	}

	logger.InfoCF("telegram", "Telegram configuration removed", map[string]any{
		"surface":        "dashboard",
		"was_configured": configured,
		"applied":        applied,
		"pending":        pending,
	})

	writeJSON(w, map[string]any{
		"ok":      true,
		"applied": applied,
		"pending": pending,
	})
}

// clearTelegramCredentials is the one place a configured bot is removed.
//
// Deliberately the mirror of writeTelegramCredentials, down to the helper it
// loads with, so the two agree about what "the Telegram channel" is. The
// runtime apply is what stops the old bot: without it the previous token keeps
// polling until something else restarts the gateway, which is exactly the stale
// runtime the owner asked to rule out.
//
// Idempotent: removing an already-removed configuration writes the same empty
// state and reports the same result, so a repeated confirmation is harmless.
func (h *Handler) clearTelegramCredentials() (applied bool, pending bool, err error) {
	cfg, channel, settings, loadErr := h.loadTelegramConfigForUpdate()
	if loadErr != nil {
		return false, false, loadErr
	}

	settings.Token.Set("")
	channel.Enabled = false
	// The owner allowlist is part of the credential, not a preference: a
	// retained owner would silently authorise the next bot paired here.
	channel.AllowFrom = config.FlexibleStringSlice{}

	if saveErr := config.SaveConfig(h.configPath, cfg); saveErr != nil {
		return false, false, saveErr
	}

	applied, pending = h.applyTelegramConfigChange("telegram_removed")
	return applied, pending, nil
}
