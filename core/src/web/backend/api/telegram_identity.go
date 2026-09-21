package api

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// The Telegram bot's own identity, canonicalised once, at the source.
//
// PC-DEF-075. "Open chat" built its destination by interpolating a username
// straight into `https://t.me/<value>`, in three independent places, from a
// value supplied by the onboarding service and cached on the Android host. Two
// things went wrong with that and only one of them is about formatting:
//
//   - Nothing stripped a leading "@". Telegram's canonical bot link is
//     `https://t.me/name`; `https://t.me/@name` is a different, non-existent
//     username and Telegram answers "Username not found". The codebase could
//     not even agree on the shape it held — `TelegramOnboardingLauncher`
//     documented its return as "the paired bot's `@username`" while
//     `TelegramBotCredentials.toString` wrote "@$botUsername", adding one. Core
//     itself already defends against this internally, stripping "@" before it
//     compares usernames.
//   - The Android cache is written only by the *native* pairing launcher. A bot
//     paired through any other path — the Dashboard's own managed connect, or a
//     manual token save — never updates it, so the card can hold a username
//     from a bot that no longer exists while the configured bot works fine.
//     That is a stale destination, and no amount of formatting fixes it.
//
// This endpoint answers with the running bot's own identity, taken from the
// committed credential and canonicalised here so no client has to. It is the
// authority: a cached username that disagrees with it is stale by definition.

// telegramUsernamePattern is Telegram's username shape, applied strictly.
//
// Strict on purpose. The job is not to be generous about what a username may
// look like, it is to make sure a *display string* can never become a
// destination: this rejects a second "@", a path separator, a scheme, a dot, a
// space and every bidirectional control character, which is what stops RTL
// display formatting from reaching the URL.
var telegramUsernamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{3,30}[A-Za-z0-9]$`)

// canonicalTelegramUsername trims, removes exactly one optional leading "@",
// and validates. It returns the empty string when the value is not a usable
// Telegram username, so a caller cannot accidentally build a link from junk.
//
// Exactly one "@" is removed. Stripping repeatedly would quietly accept "@@name"
// — a value that is already evidence something upstream is wrong — and turn it
// into a working link, hiding the bug instead of refusing it.
func canonicalTelegramUsername(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "@")
	trimmed = strings.TrimSpace(trimmed)
	if !telegramUsernamePattern.MatchString(trimmed) {
		return ""
	}
	return trimmed
}

// canonicalTelegramBotURL is the one place a bot chat link is built.
// Empty means there is no link to offer, which callers must render as "no
// button" rather than as a broken one.
func canonicalTelegramBotURL(raw string) string {
	username := canonicalTelegramUsername(raw)
	if username == "" {
		return ""
	}
	return "https://t.me/" + username
}

func (h *Handler) registerTelegramIdentityRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/telegram/identity", h.handleTelegramIdentity)
}

// handleTelegramIdentity reports the configured bot's canonical username and
// chat link.
//
//	GET /api/telegram/identity
//
// Taken from getMe against the committed credential, so it describes the bot
// this install actually holds rather than whatever a client cached. The
// username is public — it is what a user types to find the bot, and it is
// already on screen — but the token never leaves this function and is not in
// the response, the log, or the error.
func (h *Handler) handleTelegramIdentity(w http.ResponseWriter, r *http.Request) {
	_, _, settings, err := h.loadTelegramConfigForUpdate()
	if err != nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]any{
			"error": "configuration_unreadable",
		})
		return
	}
	token := strings.TrimSpace(settings.Token.String())
	if token == "" {
		writeJSON(w, map[string]any{"configured": false})
		return
	}

	client, apiRoot, err := telegramValidationHTTP(settings.BaseURL, settings.Proxy)
	if err != nil {
		writeJSON(w, map[string]any{"configured": true})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), telegramCredentialValidationTimeout)
	defer cancel()

	me, err := telegramValidationCall(ctx, client, apiRoot, token, "getMe", nil)
	if err != nil || !me.OK {
		// Unreachable or refused is not the same as "no bot": say nothing about
		// the identity rather than inventing one, and let the client keep
		// whatever it is showing until this can be answered.
		writeJSON(w, map[string]any{"configured": true})
		return
	}
	var identity struct {
		Username string `json:"username"`
	}
	if json.Unmarshal(me.Result, &identity) != nil {
		writeJSON(w, map[string]any{"configured": true})
		return
	}
	username := canonicalTelegramUsername(identity.Username)
	if username == "" {
		writeJSON(w, map[string]any{"configured": true})
		return
	}
	writeJSON(w, map[string]any{
		"configured": true,
		"username":   username,
		"chat_url":   canonicalTelegramBotURL(username),
	})
}
