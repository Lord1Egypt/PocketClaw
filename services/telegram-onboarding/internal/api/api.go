// Package api exposes the pairing HTTP contract the PocketClaw app talks to.
//
// Three endpoints, and nothing else:
//
//	POST /telegram/pairings              start a pairing, get the deep link
//	GET  /telegram/pairings/{id}         poll its state (never returns a token)
//	POST /telegram/pairings/{id}/token   collect the child token, exactly once
//
// Polling and token collection are separate on purpose. Polling is idempotent
// and happens many times; collection is a one-shot state transition that wipes
// the server's copy of the token. Folding the token into the poll response
// would mean either repeating it on every poll or making polling destructive.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/onboarding"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/pairing"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/ratelimit"
)

// Server routes the pairing API.
type Server struct {
	service *onboarding.Service
	store   *pairing.Store
	limiter *ratelimit.Limiter
	log     *slog.Logger

	// trustProxyHeader makes the limiter read X-Forwarded-For. Only enable it
	// behind a proxy that overwrites the header, or clients can forge it.
	trustProxyHeader bool
}

// New returns a Server.
func New(service *onboarding.Service, store *pairing.Store, limiter *ratelimit.Limiter, trustProxyHeader bool, log *slog.Logger) *Server {
	return &Server{
		service:          service,
		store:            store,
		limiter:          limiter,
		log:              log,
		trustProxyHeader: trustProxyHeader,
	}
}

// Handler returns the routed HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /telegram/pairings", s.createPairing)
	mux.HandleFunc("GET /telegram/pairings/{id}", s.getPairing)
	mux.HandleFunc("POST /telegram/pairings/{id}/token", s.collectToken)
	mux.HandleFunc("GET /healthz", s.health)
	return mux
}

type createResponse struct {
	PairingID           string `json:"pairing_id"`
	PollToken           string `json:"poll_token"`
	SuggestedUsername   string `json:"suggested_username"`
	SuggestedName       string `json:"suggested_name"`
	DeepLink            string `json:"deep_link"`
	QRPayload           string `json:"qr_payload"`
	ExpiresAt           string `json:"expires_at"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
}

type statusResponse struct {
	PairingID   string `json:"pairing_id"`
	State       string `json:"state"`
	Reason      string `json:"reason,omitempty"`
	BotUsername string `json:"bot_username,omitempty"`
	OwnerUserID int64  `json:"owner_user_id,omitempty"`
	ExpiresAt   string `json:"expires_at"`
}

type tokenResponse struct {
	BotToken    string `json:"bot_token"`
	BotUserID   int64  `json:"bot_user_id"`
	BotUsername string `json:"bot_username"`
	OwnerUserID int64  `json:"owner_user_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) createPairing(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.Allow(s.clientKey(r)) {
		writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: "rate_limited"})
		return
	}

	snap, pollToken, err := s.service.CreatePairing()
	if err != nil {
		s.log.Error("could not create a pairing", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "pairing_unavailable"})
		return
	}

	// deep_link and qr_payload are deliberately the same string. Telegram does
	// not define a distinct QR form, and the QR must carry no secret.
	writeJSON(w, http.StatusCreated, createResponse{
		PairingID:           snap.ID,
		PollToken:           pollToken,
		SuggestedUsername:   snap.SuggestedUsername,
		SuggestedName:       snap.SuggestedName,
		DeepLink:            snap.DeepLink,
		QRPayload:           snap.DeepLink,
		ExpiresAt:           snap.ExpiresAt.UTC().Format(time.RFC3339),
		PollIntervalSeconds: 2,
	})
}

func (s *Server) getPairing(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	token, ok := bearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing_poll_token"})
		return
	}

	snap, err := s.store.Get(id, token)
	if err != nil {
		s.writePairingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{
		PairingID:   snap.ID,
		State:       string(snap.State),
		Reason:      snap.Reason,
		BotUsername: snap.BotUsername,
		OwnerUserID: snap.OwnerUserID,
		ExpiresAt:   snap.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (s *Server) collectToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	token, ok := bearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing_poll_token"})
		return
	}

	delivery, err := s.store.Consume(id, token)
	if err != nil {
		s.writePairingError(w, err)
		return
	}
	s.log.Info("delivered a child bot token", "pairing_id", id, "bot_username", delivery.BotUsername)
	// The token appears here and nowhere else: not in a log line, not in a
	// poll response, not in the deep link.
	writeJSON(w, http.StatusOK, tokenResponse{
		BotToken:    delivery.BotToken,
		BotUserID:   delivery.BotUserID,
		BotUsername: delivery.BotUsername,
		OwnerUserID: delivery.OwnerUserID,
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "ok",
		"manager_username": s.service.ManagerUsername(),
		"live_pairings":    s.store.Len(),
	})
}

// writePairingError maps store errors to status codes without ever revealing
// whether a pairing exists to a caller holding the wrong token: unknown and
// unauthorized both answer 404.
func (s *Server) writePairingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pairing.ErrNotFound), errors.Is(err, pairing.ErrUnauthorized):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "pairing_not_found"})
	case errors.Is(err, pairing.ErrAlreadyConsumed):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "already_delivered"})
	case errors.Is(err, pairing.ErrNotReady):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "not_ready"})
	default:
		s.log.Error("unexpected pairing error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal_error"})
	}
}

func (s *Server) clientKey(r *http.Request) string {
	if s.trustProxyHeader {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			if first, _, found := strings.Cut(forwarded, ","); found {
				return strings.TrimSpace(first)
			}
			return strings.TrimSpace(forwarded)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", false
	}
	value, ok := strings.CutPrefix(header, "Bearer ")
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
