package pairing

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(10 * time.Minute)
}

func create(t *testing.T, s *Store, username string) (Snapshot, string) {
	t.Helper()
	snap, token, err := s.Create(username, "PocketClaw Agent", "https://t.me/newbot/M/"+username)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return snap, token
}

func TestCreateProducesRandomUnguessableIdentifiers(t *testing.T) {
	s := newTestStore(t)
	ids := make(map[string]bool)
	tokens := make(map[string]bool)
	for i := 0; i < 200; i++ {
		snap, token := create(t, s, "pocketclaw_"+pad(i)+"_bot")
		if len(snap.ID) != 32 {
			t.Fatalf("pairing id %q is %d chars, want 32 hex chars", snap.ID, len(snap.ID))
		}
		if len(token) != 64 {
			t.Fatalf("poll token is %d chars, want 64 hex chars", len(token))
		}
		if strings.Contains(token, snap.ID) || strings.Contains(snap.ID, token) {
			t.Fatal("poll token and pairing id are derived from each other")
		}
		ids[snap.ID] = true
		tokens[token] = true
	}
	if len(ids) != 200 || len(tokens) != 200 {
		t.Fatalf("got %d ids and %d tokens out of 200; identifiers are not random", len(ids), len(tokens))
	}
}

func TestPollTokenIsNotStoredInPlaintext(t *testing.T) {
	s := newTestStore(t)
	snap, token := create(t, s, "pocketclaw_aaaaaaaa_bot")
	s.mu.Lock()
	session := s.byID[snap.ID]
	s.mu.Unlock()
	if strings.Contains(string(session.pollTokenHash[:]), token) {
		t.Fatal("the poll token is recoverable from the stored hash")
	}
	// The struct has no plaintext field to begin with; assert the hash differs
	// from the token bytes so a future refactor cannot quietly store it raw.
	if string(session.pollTokenHash[:]) == token {
		t.Fatal("the poll token is stored verbatim")
	}
}

func TestGetRejectsWrongPollToken(t *testing.T) {
	s := newTestStore(t)
	snap, _ := create(t, s, "pocketclaw_aaaaaaaa_bot")
	if _, err := s.Get(snap.ID, "0000000000000000000000000000000000000000000000000000000000000000"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Get with a wrong token returned %v, want ErrUnauthorized", err)
	}
}

func TestGetRejectsAnotherPairingsToken(t *testing.T) {
	s := newTestStore(t)
	first, _ := create(t, s, "pocketclaw_aaaaaaaa_bot")
	_, secondToken := create(t, s, "pocketclaw_bbbbbbbb_bot")
	if _, err := s.Get(first.ID, secondToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("polling one pairing with another's token returned %v, want ErrUnauthorized", err)
	}
}

func TestUnknownPairingIsNotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Get("deadbeef", "whatever"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get on an unknown id returned %v, want ErrNotFound", err)
	}
}

func TestFullHappyPathStateTransitions(t *testing.T) {
	s := newTestStore(t)
	snap, token := create(t, s, "pocketclaw_aaaaaaaa_bot")
	if snap.State != StatePending {
		t.Fatalf("new pairing state = %q, want %q", snap.State, StatePending)
	}

	matched, ok := s.MatchUsername("pocketclaw_aaaaaaaa_bot")
	if !ok || matched.ID != snap.ID {
		t.Fatal("MatchUsername did not find the pairing by its suggested username")
	}

	if err := s.MarkCreated(snap.ID, 4242, 777, "pocketclaw_aaaaaaaa_bot"); err != nil {
		t.Fatalf("MarkCreated: %v", err)
	}
	got, err := s.Get(snap.ID, token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.State != StateCreated || got.OwnerUserID != 4242 || got.BotUserID != 777 {
		t.Fatalf("after MarkCreated: %+v", got)
	}

	if err := s.MarkReady(snap.ID, "777:REDACTED-TEST-TOKEN"); err != nil {
		t.Fatalf("MarkReady: %v", err)
	}
	got, err = s.Get(snap.ID, token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.State != StateReady {
		t.Fatalf("state = %q, want %q", got.State, StateReady)
	}

	delivery, err := s.Consume(snap.ID, token)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if delivery.BotToken != "777:REDACTED-TEST-TOKEN" || delivery.OwnerUserID != 4242 || delivery.BotUsername != "pocketclaw_aaaaaaaa_bot" {
		t.Fatalf("delivery = %+v", delivery)
	}
}

func TestSnapshotNeverCarriesTheBotToken(t *testing.T) {
	s := newTestStore(t)
	snap, token := create(t, s, "pocketclaw_aaaaaaaa_bot")
	mustMarkReady(t, s, snap.ID)

	got, err := s.Get(snap.ID, token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// A Snapshot is what the polling endpoint serialises. If a token field is
	// ever added to it, the app would receive the token from plain polling and
	// the single-use guarantee would be gone.
	for _, field := range []string{got.ID, got.SuggestedUsername, got.DeepLink, string(got.State), got.Reason, got.BotUsername} {
		if strings.Contains(field, "REDACTED-TEST-TOKEN") {
			t.Fatalf("snapshot field %q carries the bot token", field)
		}
	}
	if strings.Contains(got.DeepLink, "REDACTED-TEST-TOKEN") {
		t.Fatal("the deep link carries the bot token")
	}
}

func TestConsumeIsSingleUse(t *testing.T) {
	s := newTestStore(t)
	snap, token := create(t, s, "pocketclaw_aaaaaaaa_bot")
	mustMarkReady(t, s, snap.ID)

	if _, err := s.Consume(snap.ID, token); err != nil {
		t.Fatalf("first Consume: %v", err)
	}
	// The session is gone entirely after delivery, so a replay cannot even
	// identify it.
	if _, err := s.Consume(snap.ID, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Consume returned %v, want ErrNotFound", err)
	}
	if s.Len() != 0 {
		t.Fatalf("store still holds %d sessions after delivery", s.Len())
	}
}

func TestConsumeRejectsWrongPollToken(t *testing.T) {
	s := newTestStore(t)
	snap, _ := create(t, s, "pocketclaw_aaaaaaaa_bot")
	mustMarkReady(t, s, snap.ID)
	if _, err := s.Consume(snap.ID, "0000"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Consume with a wrong token returned %v, want ErrUnauthorized", err)
	}
	// The token must survive a failed attempt, not be burned by it.
	if s.Len() != 1 {
		t.Fatal("a failed Consume discarded the session")
	}
}

func TestConsumeBeforeReadyIsRefused(t *testing.T) {
	s := newTestStore(t)
	snap, token := create(t, s, "pocketclaw_aaaaaaaa_bot")
	if _, err := s.Consume(snap.ID, token); !errors.Is(err, ErrNotReady) {
		t.Fatalf("Consume while pending returned %v, want ErrNotReady", err)
	}
}

func TestExpiryRemovesSessionAndToken(t *testing.T) {
	s := NewStore(5 * time.Minute)
	now := time.Now()
	s.SetClock(func() time.Time { return now })
	snap, token := create(t, s, "pocketclaw_aaaaaaaa_bot")
	mustMarkReady(t, s, snap.ID)

	now = now.Add(6 * time.Minute)
	if _, err := s.Get(snap.ID, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after expiry returned %v, want ErrNotFound", err)
	}
	if _, err := s.Consume(snap.ID, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Consume after expiry returned %v, want ErrNotFound", err)
	}
	if s.Len() != 0 {
		t.Fatal("expired session was not swept")
	}
}

func TestExpiredUsernameStopsMatching(t *testing.T) {
	s := NewStore(5 * time.Minute)
	now := time.Now()
	s.SetClock(func() time.Time { return now })
	create(t, s, "pocketclaw_aaaaaaaa_bot")
	now = now.Add(6 * time.Minute)
	if _, ok := s.MatchUsername("pocketclaw_aaaaaaaa_bot"); ok {
		t.Fatal("an expired pairing still matches its username")
	}
}

func TestLiveUsernameCannotBeReused(t *testing.T) {
	s := newTestStore(t)
	create(t, s, "pocketclaw_aaaaaaaa_bot")
	if _, _, err := s.Create("pocketclaw_aaaaaaaa_bot", "PocketClaw Agent", "link"); !errors.Is(err, ErrUsernameInUse) {
		t.Fatalf("reusing a live username returned %v, want ErrUsernameInUse", err)
	}
}

func TestMatchUsernameIsCaseInsensitiveAndIgnoresAt(t *testing.T) {
	s := newTestStore(t)
	snap, _ := create(t, s, "pocketclaw_aaaaaaaa_bot")
	for _, probe := range []string{"POCKETCLAW_AAAAAAAA_BOT", "@pocketclaw_aaaaaaaa_bot"} {
		got, ok := s.MatchUsername(probe)
		if !ok || got.ID != snap.ID {
			t.Fatalf("MatchUsername(%q) did not match", probe)
		}
	}
}

func TestMarkFailedClearsAnyHeldToken(t *testing.T) {
	s := newTestStore(t)
	snap, token := create(t, s, "pocketclaw_aaaaaaaa_bot")
	mustMarkReady(t, s, snap.ID)
	if err := s.MarkFailed(snap.ID, ReasonTokenRetrievalFailed); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	got, err := s.Get(snap.ID, token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.State != StateFailed || got.Reason != ReasonTokenRetrievalFailed {
		t.Fatalf("after MarkFailed: %+v", got)
	}
	if _, err := s.Consume(snap.ID, token); !errors.Is(err, ErrNotReady) {
		t.Fatalf("Consume after failure returned %v, want ErrNotReady", err)
	}
}

func TestMarkReadyRefusesEmptyToken(t *testing.T) {
	s := newTestStore(t)
	snap, _ := create(t, s, "pocketclaw_aaaaaaaa_bot")
	if err := s.MarkReady(snap.ID, ""); err == nil {
		t.Fatal("MarkReady accepted an empty token")
	}
}

func mustMarkReady(t *testing.T, s *Store, id string) {
	t.Helper()
	if err := s.MarkCreated(id, 4242, 777, "pocketclaw_aaaaaaaa_bot"); err != nil {
		t.Fatalf("MarkCreated: %v", err)
	}
	if err := s.MarkReady(id, "777:REDACTED-TEST-TOKEN"); err != nil {
		t.Fatalf("MarkReady: %v", err)
	}
}

func pad(i int) string {
	s := "aaaaaaaa" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26))
	return s[len(s)-8:]
}
