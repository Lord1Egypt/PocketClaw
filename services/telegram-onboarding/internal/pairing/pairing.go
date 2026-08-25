// Package pairing holds short-lived Telegram pairing sessions.
//
// A session exists only to connect one PocketClaw install to one bot the user
// creates in Telegram. It is kept in memory, never written to disk, and
// discarded as soon as the child token is delivered or the session expires.
// Nothing about the user's conversations, provider keys, or workspace is
// stored here; see README.md, "Privacy".
package pairing

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// State is the observable lifecycle of a pairing session.
type State string

const (
	// StatePending means the link has been issued and Telegram has not yet
	// reported a bot.
	StatePending State = "pending"
	// StateCreated means the manager bot saw the new child bot and PocketClaw
	// is retrieving its token.
	StateCreated State = "created"
	// StateReady means the child token is held and awaiting its single
	// delivery to the app that started the pairing.
	StateReady State = "ready"
	// StateExpired means the session outlived its TTL.
	StateExpired State = "expired"
	// StateFailed means the session cannot complete; Reason says why, in
	// non-secret terms.
	StateFailed State = "failed"
)

// Failure reasons. These are returned to the app and must never carry secrets
// or Telegram error text that might quote a token.
const (
	ReasonTokenRetrievalFailed = "token_retrieval_failed"
	ReasonManagerUnavailable   = "manager_unavailable"
)

var (
	// ErrNotFound is returned for an unknown or already-discarded pairing.
	ErrNotFound = errors.New("pairing not found")
	// ErrUnauthorized is returned when the poll token does not match.
	ErrUnauthorized = errors.New("poll token does not match this pairing")
	// ErrNotReady is returned when a token is requested before it exists.
	ErrNotReady = errors.New("pairing is not ready")
	// ErrAlreadyConsumed is returned when a token is requested twice.
	ErrAlreadyConsumed = errors.New("pairing token has already been delivered")
	// ErrUsernameInUse is returned when a suggested username collides with a
	// live pairing.
	ErrUsernameInUse = errors.New("suggested username is already pending")
)

// Session is one pairing. Callers outside this package receive copies via
// Snapshot, never the live struct, so the token cannot escape by aliasing.
type Session struct {
	ID                string
	SuggestedUsername string
	SuggestedName     string
	DeepLink          string
	State             State
	Reason            string
	CreatedAt         time.Time
	ExpiresAt         time.Time

	// OwnerUserID is the Telegram user that created the bot, known only once
	// Telegram reports it. It becomes the child bot's allow-list entry.
	OwnerUserID int64
	BotUserID   int64
	BotUsername string

	// pollTokenHash authenticates polling. The token itself is returned once,
	// at creation, and never stored.
	pollTokenHash [32]byte
	// botToken is held only between retrieval and its single delivery.
	botToken string
	consumed bool
}

// Snapshot is a read-only view of a session, with no token material.
type Snapshot struct {
	ID                string
	SuggestedUsername string
	SuggestedName     string
	DeepLink          string
	State             State
	Reason            string
	CreatedAt         time.Time
	ExpiresAt         time.Time
	OwnerUserID       int64
	BotUserID         int64
	BotUsername       string
}

func (s *Session) snapshot() Snapshot {
	return Snapshot{
		ID:                s.ID,
		SuggestedUsername: s.SuggestedUsername,
		SuggestedName:     s.SuggestedName,
		DeepLink:          s.DeepLink,
		State:             s.State,
		Reason:            s.Reason,
		CreatedAt:         s.CreatedAt,
		ExpiresAt:         s.ExpiresAt,
		OwnerUserID:       s.OwnerUserID,
		BotUserID:         s.BotUserID,
		BotUsername:       s.BotUsername,
	}
}

// Delivery is the one-time result handed to the app that owns the pairing.
type Delivery struct {
	BotToken    string
	BotUserID   int64
	BotUsername string
	OwnerUserID int64
}

// Store keeps live pairings in memory.
type Store struct {
	mu   sync.Mutex
	byID map[string]*Session
	// byUsername indexes live pairings by lowercased suggested username, which
	// is how a Telegram managed-bot update is matched back to its pairing.
	byUsername map[string]*Session

	ttl time.Duration
	now func() time.Time
}

// NewStore returns a store whose sessions live for ttl.
func NewStore(ttl time.Duration) *Store {
	return &Store{
		byID:       make(map[string]*Session),
		byUsername: make(map[string]*Session),
		ttl:        ttl,
		now:        time.Now,
	}
}

// SetClock replaces the store's clock. For tests only.
func (s *Store) SetClock(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

// TTL reports how long a new session lives.
func (s *Store) TTL() time.Duration { return s.ttl }

// Create registers a new pairing and returns its snapshot plus the poll token,
// which is the only time that token is ever produced.
func (s *Store) Create(suggestedUsername, suggestedName, deepLink string) (Snapshot, string, error) {
	id, err := randomHex(16)
	if err != nil {
		return Snapshot{}, "", err
	}
	pollToken, err := randomHex(32)
	if err != nil {
		return Snapshot{}, "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()

	key := strings.ToLower(suggestedUsername)
	if _, exists := s.byUsername[key]; exists {
		return Snapshot{}, "", ErrUsernameInUse
	}

	now := s.now()
	session := &Session{
		ID:                id,
		SuggestedUsername: suggestedUsername,
		SuggestedName:     suggestedName,
		DeepLink:          deepLink,
		State:             StatePending,
		CreatedAt:         now,
		ExpiresAt:         now.Add(s.ttl),
		pollTokenHash:     sha256.Sum256([]byte(pollToken)),
	}
	s.byID[id] = session
	s.byUsername[key] = session
	return session.snapshot(), pollToken, nil
}

// Get returns a pairing for an authenticated poller.
func (s *Store) Get(id, pollToken string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.authenticateLocked(id, pollToken)
	if err != nil {
		return Snapshot{}, err
	}
	return session.snapshot(), nil
}

// MatchUsername finds the live pairing that suggested a username. It is how an
// incoming Telegram update is bound to the app that asked for it.
func (s *Store) MatchUsername(username string) (Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	session, ok := s.byUsername[strings.ToLower(strings.TrimPrefix(username, "@"))]
	if !ok {
		return Snapshot{}, false
	}
	return session.snapshot(), true
}

// MarkCreated records that Telegram reported the child bot, before its token
// has been retrieved.
func (s *Store) MarkCreated(id string, ownerUserID, botUserID int64, botUsername string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.byID[id]
	if !ok {
		return ErrNotFound
	}
	session.State = StateCreated
	session.OwnerUserID = ownerUserID
	session.BotUserID = botUserID
	session.BotUsername = botUsername
	return nil
}

// MarkReady stores the retrieved child token, awaiting its single delivery.
func (s *Store) MarkReady(id, botToken string) error {
	if botToken == "" {
		return fmt.Errorf("refusing to mark pairing ready with an empty token")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.byID[id]
	if !ok {
		return ErrNotFound
	}
	session.botToken = botToken
	session.State = StateReady
	return nil
}

// MarkFailed records a non-secret failure reason.
func (s *Store) MarkFailed(id, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.byID[id]
	if !ok {
		return ErrNotFound
	}
	session.State = StateFailed
	session.Reason = reason
	session.botToken = ""
	return nil
}

// Consume delivers the child token exactly once and immediately discards the
// server-side copy along with the whole session.
func (s *Store) Consume(id, pollToken string) (Delivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.authenticateLocked(id, pollToken)
	if err != nil {
		return Delivery{}, err
	}
	if session.consumed {
		return Delivery{}, ErrAlreadyConsumed
	}
	if session.State != StateReady || session.botToken == "" {
		return Delivery{}, ErrNotReady
	}

	delivery := Delivery{
		BotToken:    session.botToken,
		BotUserID:   session.BotUserID,
		BotUsername: session.BotUsername,
		OwnerUserID: session.OwnerUserID,
	}
	session.consumed = true
	session.botToken = ""
	s.removeLocked(session)
	return delivery, nil
}

// Len reports how many sessions are live. For tests and the health endpoint.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	return len(s.byID)
}

func (s *Store) authenticateLocked(id, pollToken string) (*Session, error) {
	s.sweepLocked()
	session, ok := s.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	given := sha256.Sum256([]byte(pollToken))
	if subtle.ConstantTimeCompare(given[:], session.pollTokenHash[:]) != 1 {
		return nil, ErrUnauthorized
	}
	return session, nil
}

func (s *Store) sweepLocked() {
	now := s.now()
	for id, session := range s.byID {
		if now.After(session.ExpiresAt) {
			session.botToken = ""
			delete(s.byID, id)
			delete(s.byUsername, strings.ToLower(session.SuggestedUsername))
		}
	}
}

func (s *Store) removeLocked(session *Session) {
	delete(s.byID, session.ID)
	delete(s.byUsername, strings.ToLower(session.SuggestedUsername))
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
