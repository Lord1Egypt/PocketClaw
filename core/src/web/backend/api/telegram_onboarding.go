package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/canonicalenv"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/telegramonboarding"
)

// Managed Telegram onboarding for a client with no Android host.
//
// PC-DEF-060. The native flow launches Telegram and writes the token itself, which a
// desktop browser cannot do. These endpoints let the Dashboard drive the same pairing
// through Core: the browser calls same origin, Core makes the server-to-server calls,
// and completion goes through writeTelegramCredentials — the same authoritative writer
// the Android bridge uses, so the owner contract is not reimplemented.
//
// What the browser never receives:
//
//   - the onboarding service's URL, so no cross-origin access and no hosting origin in
//     the user's navigation (the PC-DEF-052 requirement, kept for this client too);
//   - the poll token, which authorises status reads and the single token collection.
//     Core holds it against the pairing id and the browser only ever names the id;
//   - the bot token, at any point. It goes from the service into Core's configuration
//     and is never serialised back.
//
// These routes are not in the launcher auth allowlist, so they require a Dashboard
// session like every other /api path.

// OnboardingBaseURLEnv is where Core learns the onboarding service address.
//
// The Android host passes its compiled-in value here, read from the same dart-define
// the APK uses, so there is one place it is configured. A deployment without it simply
// reports managed onboarding as unavailable and the manual form remains.
const OnboardingBaseURLEnv = "POCKETCLAW_ONBOARDING_BASE_URL"

// onboardingPairingTTL bounds how long Core remembers a pairing it started.
//
// The service expires pairings on its own; this is the local ceiling so an abandoned
// browser tab cannot leave a poll token in memory indefinitely.
const onboardingPairingTTL = 30 * time.Minute

// telegramOnboardingSession is one in-flight pairing, held server-side.
type telegramOnboardingSession struct {
	pairing *telegramonboarding.Pairing
	created time.Time
}

// telegramOnboardingStore keeps the poll tokens out of the browser.
type telegramOnboardingStore struct {
	mu       sync.Mutex
	sessions map[string]*telegramOnboardingSession
}

func newTelegramOnboardingStore() *telegramOnboardingStore {
	return &telegramOnboardingStore{sessions: map[string]*telegramOnboardingSession{}}
}

func (s *telegramOnboardingStore) put(pairing *telegramonboarding.Pairing) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
	s.sessions[pairing.PairingID] = &telegramOnboardingSession{
		pairing: pairing,
		created: time.Now(),
	}
}

func (s *telegramOnboardingStore) get(id string) *telegramonboarding.Pairing {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
	session, ok := s.sessions[id]
	if !ok {
		return nil
	}
	return session.pairing
}

func (s *telegramOnboardingStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// pruneLocked drops sessions past the local ceiling. Called on every access rather
// than on a timer: the map holds at most a handful of entries and a background
// goroutine for that would outlive the work it does.
func (s *telegramOnboardingStore) pruneLocked() {
	cutoff := time.Now().Add(-onboardingPairingTTL)
	for id, session := range s.sessions {
		if session.created.Before(cutoff) {
			delete(s.sessions, id)
		}
	}
}

func (h *Handler) registerTelegramOnboardingRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/telegram/onboarding", h.handleTelegramOnboardingAvailability)
	mux.HandleFunc("POST /api/telegram/onboarding/pairings", h.handleTelegramOnboardingCreate)
	mux.HandleFunc("GET /api/telegram/onboarding/pairings/{id}", h.handleTelegramOnboardingStatus)
	mux.HandleFunc("POST /api/telegram/onboarding/pairings/{id}/complete",
		h.handleTelegramOnboardingComplete)
	mux.HandleFunc("DELETE /api/telegram/onboarding/pairings/{id}",
		h.handleTelegramOnboardingCancel)
}

// onboardingClient returns the configured client, or nil when this deployment has no
// onboarding endpoint.
func (h *Handler) onboardingClient() *telegramonboarding.Client {
	h.telegramOnboardingOnce.Do(func() {
		base := canonicalenv.Getenv(OnboardingBaseURLEnv)
		h.telegramOnboarding = telegramonboarding.NewClient(base, nil)
		h.telegramOnboardingStore = newTelegramOnboardingStore()
		if h.telegramOnboarding == nil {
			logger.InfoC("web",
				"Managed Telegram onboarding is unavailable: no onboarding endpoint is configured")
		}
	})
	return h.telegramOnboarding
}

// handleTelegramOnboardingAvailability tells the Dashboard whether to offer Connect.
//
//	GET /api/telegram/onboarding
func (h *Handler) handleTelegramOnboardingAvailability(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"available": h.onboardingClient() != nil})
}

// handleTelegramOnboardingCreate starts a pairing.
//
//	POST /api/telegram/onboarding/pairings
//
// The response deliberately omits the poll token. Everything it does carry is something
// Telegram is about to show the user anyway: the manager bot, the suggested username,
// and the deep link built from them.
func (h *Handler) handleTelegramOnboardingCreate(w http.ResponseWriter, r *http.Request) {
	client := h.onboardingClient()
	if client == nil {
		writeOnboardingUnavailable(w)
		return
	}

	pairing, err := client.CreatePairing(r.Context())
	if err != nil {
		writeOnboardingError(w, err)
		return
	}
	h.telegramOnboardingStore.put(pairing)

	logger.InfoCF("telegram", "Managed onboarding pairing created", map[string]any{
		"surface":            "dashboard",
		"suggested_username": pairing.SuggestedUsername,
	})

	writeJSON(w, map[string]any{
		"pairing_id":            pairing.PairingID,
		"suggested_username":    pairing.SuggestedUsername,
		"suggested_name":        pairing.SuggestedName,
		"deep_link":             pairing.DeepLink,
		"qr_payload":            pairing.QRPayload,
		"expires_at":            pairing.ExpiresAt.Format(time.RFC3339),
		"poll_interval_seconds": int(pairing.PollInterval / time.Second),
	})
}

// handleTelegramOnboardingStatus reports a pairing's state.
//
//	GET /api/telegram/onboarding/pairings/{id}
func (h *Handler) handleTelegramOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	client := h.onboardingClient()
	if client == nil {
		writeOnboardingUnavailable(w)
		return
	}
	pairing := h.telegramOnboardingStore.get(r.PathValue("id"))
	if pairing == nil {
		// Unknown and expired are the same answer, as they are at the service.
		writeJSON(w, map[string]any{"state": string(telegramonboarding.StateExpired)})
		return
	}

	status, err := client.FetchStatus(r.Context(), pairing)
	if err != nil {
		writeOnboardingError(w, err)
		return
	}
	response := map[string]any{"state": string(status.State)}
	if status.BotUsername != "" {
		response["bot_username"] = status.BotUsername
	}
	if status.Reason != "" {
		response["reason"] = status.Reason
	}
	writeJSON(w, response)
}

// handleTelegramOnboardingComplete collects the token and configures Telegram.
//
//	POST /api/telegram/onboarding/pairings/{id}/complete
//
// The bot token never leaves this function: it goes from the service straight into
// writeTelegramCredentials, and the response says only what happened.
func (h *Handler) handleTelegramOnboardingComplete(w http.ResponseWriter, r *http.Request) {
	client := h.onboardingClient()
	if client == nil {
		writeOnboardingUnavailable(w)
		return
	}
	id := r.PathValue("id")
	pairing := h.telegramOnboardingStore.get(id)
	if pairing == nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{
			"error": string(telegramonboarding.KindPairingGone),
		})
		return
	}

	credentials, err := client.CollectCredentials(r.Context(), pairing)
	if err != nil {
		writeOnboardingError(w, err)
		return
	}
	// The service delivers the token exactly once, so the session is spent either
	// way from here.
	h.telegramOnboardingStore.delete(id)

	applied, pending, writeErr := h.writeTelegramCredentialsContext(
		r.Context(), credentials.Token, credentials.OwnerUserID)
	if writeErr != nil {
		// The pairing is gone and the token is not stored, so this is terminal for
		// this attempt. Say so rather than leaving the UI polling a dead session.
		logger.ErrorCF("telegram", "Managed onboarding could not configure Telegram",
			map[string]any{"surface": "dashboard", "error": writeErr.Error()})
		status := http.StatusInternalServerError
		kind := "configuration_failed"
		if errors.Is(writeErr, ErrTelegramCredentialsInvalid) {
			status = http.StatusUnauthorized
			kind = "invalid_credentials"
		}
		writeJSONStatus(w, status, map[string]any{"error": kind})
		return
	}

	logger.InfoCF("telegram", "Managed onboarding configured Telegram", map[string]any{
		"surface":      "dashboard",
		"bot_username": credentials.BotUsername,
		"applied":      applied,
		"pending":      pending,
	})

	writeJSON(w, map[string]any{
		"ok":           true,
		"bot_username": credentials.BotUsername,
		"applied":      applied,
		"pending":      pending,
	})
}

// handleTelegramOnboardingCancel forgets a pairing.
//
//	DELETE /api/telegram/onboarding/pairings/{id}
//
// Local only: the service expires it on its own schedule, and there is nothing to
// revoke because nothing was configured. Dropping the poll token is the point.
func (h *Handler) handleTelegramOnboardingCancel(w http.ResponseWriter, r *http.Request) {
	if h.onboardingClient() == nil {
		writeOnboardingUnavailable(w)
		return
	}
	h.telegramOnboardingStore.delete(r.PathValue("id"))
	writeJSON(w, map[string]any{"ok": true})
}

func writeOnboardingUnavailable(w http.ResponseWriter) {
	writeJSONStatus(w, http.StatusServiceUnavailable, map[string]any{
		"error": string(telegramonboarding.KindNotConfigured),
	})
}

// writeOnboardingError maps a classified failure to a status and a machine-readable
// kind. The service's own detail stays in the developer log: it can quote a request.
func writeOnboardingError(w http.ResponseWriter, err error) {
	kind := telegramonboarding.KindServiceError
	if classified, ok := telegramonboarding.AsError(err); ok {
		kind = classified.Kind
	}
	status := http.StatusBadGateway
	switch kind {
	case telegramonboarding.KindNotConfigured:
		status = http.StatusServiceUnavailable
	case telegramonboarding.KindRateLimited:
		status = http.StatusTooManyRequests
	case telegramonboarding.KindPairingGone:
		status = http.StatusNotFound
	}
	logger.WarnCF("telegram", "Managed onboarding request failed", map[string]any{
		"surface": "dashboard",
		"kind":    string(kind),
		"error":   err.Error(),
	})
	writeJSONStatus(w, status, map[string]any{"error": string(kind)})
}

func writeJSON(w http.ResponseWriter, body map[string]any) {
	writeJSONStatus(w, http.StatusOK, body)
}

func writeJSONStatus(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
