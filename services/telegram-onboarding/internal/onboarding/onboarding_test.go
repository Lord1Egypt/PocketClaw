package onboarding

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/pairing"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/telegram"
)

// fakeBot stands in for Telegram. No test in this package touches a network or
// a real token.
type fakeBot struct {
	mu sync.Mutex

	me           telegram.User
	meErr        error
	token        string
	tokenErr     error
	tokenCalls   []int64
	updateQueues [][]telegram.Update
	updatesErr   error
}

func (f *fakeBot) GetMe(context.Context) (telegram.User, error) { return f.me, f.meErr }

func (f *fakeBot) GetUpdates(_ context.Context, _ int64, _ int) ([]telegram.Update, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updatesErr != nil {
		return nil, f.updatesErr
	}
	if len(f.updateQueues) == 0 {
		return nil, nil
	}
	batch := f.updateQueues[0]
	f.updateQueues = f.updateQueues[1:]
	return batch, nil
}

func (f *fakeBot) GetManagedBotToken(_ context.Context, userID int64) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokenCalls = append(f.tokenCalls, userID)
	return f.token, f.tokenErr
}

func newService(t *testing.T, bot *fakeBot) (*Service, *pairing.Store) {
	t.Helper()
	store := pairing.NewStore(10 * time.Minute)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(store, bot, "PocketClawSetupBot", log), store
}

func TestCreatePairingProducesPocketClawBrandedLink(t *testing.T) {
	svc, _ := newService(t, &fakeBot{})
	snap, pollToken, err := svc.CreatePairing()
	if err != nil {
		t.Fatalf("CreatePairing: %v", err)
	}
	if !strings.HasPrefix(snap.SuggestedUsername, "pocketclaw_") || !strings.HasSuffix(snap.SuggestedUsername, "_bot") {
		t.Fatalf("suggested username %q does not follow pocketclaw_<random>_bot", snap.SuggestedUsername)
	}
	if snap.SuggestedName != "PocketClaw Agent" {
		t.Fatalf("suggested name = %q, want %q", snap.SuggestedName, "PocketClaw Agent")
	}
	want := "https://t.me/newbot/PocketClawSetupBot/" + snap.SuggestedUsername + "?name=PocketClaw%20Agent"
	if snap.DeepLink != want {
		t.Fatalf("deep link = %q, want %q", snap.DeepLink, want)
	}
	if pollToken == "" {
		t.Fatal("CreatePairing returned an empty poll token")
	}
	if snap.State != pairing.StatePending {
		t.Fatalf("new pairing state = %q", snap.State)
	}
}

func TestHandleUpdateCompletesThePairing(t *testing.T) {
	bot := &fakeBot{token: "9001:CHILD-TOKEN"}
	svc, store := newService(t, bot)
	snap, pollToken, err := svc.CreatePairing()
	if err != nil {
		t.Fatalf("CreatePairing: %v", err)
	}

	svc.HandleUpdate(context.Background(), telegram.Update{
		UpdateID: 1,
		ManagedBot: &telegram.ManagedBotUpdated{
			User: telegram.User{ID: 555},
			Bot:  telegram.User{ID: 9001, IsBot: true, Username: snap.SuggestedUsername},
		},
	})

	got, err := store.Get(snap.ID, pollToken)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.State != pairing.StateReady {
		t.Fatalf("state = %q, want %q", got.State, pairing.StateReady)
	}
	if got.OwnerUserID != 555 {
		t.Fatalf("owner user id = %d, want 555", got.OwnerUserID)
	}

	delivery, err := store.Consume(snap.ID, pollToken)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if delivery.BotToken != "9001:CHILD-TOKEN" {
		t.Fatalf("delivered token = %q", delivery.BotToken)
	}
	if len(bot.tokenCalls) != 1 || bot.tokenCalls[0] != 9001 {
		t.Fatalf("getManagedBotToken called with %v, want [9001]", bot.tokenCalls)
	}
}

func TestHandleUpdateIgnoresUnrelatedBots(t *testing.T) {
	bot := &fakeBot{token: "9001:CHILD-TOKEN"}
	svc, store := newService(t, bot)
	snap, pollToken, _ := svc.CreatePairing()

	svc.HandleUpdate(context.Background(), telegram.Update{
		UpdateID: 1,
		ManagedBot: &telegram.ManagedBotUpdated{
			User: telegram.User{ID: 555},
			Bot:  telegram.User{ID: 4321, Username: "somebody_elses_bot"},
		},
	})

	got, _ := store.Get(snap.ID, pollToken)
	if got.State != pairing.StatePending {
		t.Fatalf("an unrelated bot advanced the pairing to %q", got.State)
	}
	if len(bot.tokenCalls) != 0 {
		t.Fatal("a token was fetched for a bot with no matching pairing")
	}
}

func TestHandleUpdateIgnoresNonManagedBotUpdates(t *testing.T) {
	bot := &fakeBot{}
	svc, _ := newService(t, bot)
	svc.HandleUpdate(context.Background(), telegram.Update{UpdateID: 1})
	if len(bot.tokenCalls) != 0 {
		t.Fatal("a non-managed-bot update triggered a token fetch")
	}
}

func TestTokenRetrievalFailureIsReportedWithoutSecrets(t *testing.T) {
	bot := &fakeBot{tokenErr: &telegram.APIError{Method: "getManagedBotToken", Code: 400, Description: "Bad Request"}}
	svc, store := newService(t, bot)
	snap, pollToken, _ := svc.CreatePairing()

	svc.HandleUpdate(context.Background(), telegram.Update{
		UpdateID: 1,
		ManagedBot: &telegram.ManagedBotUpdated{
			User: telegram.User{ID: 555},
			Bot:  telegram.User{ID: 9001, Username: snap.SuggestedUsername},
		},
	})

	got, err := store.Get(snap.ID, pollToken)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.State != pairing.StateFailed {
		t.Fatalf("state = %q, want %q", got.State, pairing.StateFailed)
	}
	if got.Reason != pairing.ReasonTokenRetrievalFailed {
		t.Fatalf("reason = %q, want %q", got.Reason, pairing.ReasonTokenRetrievalFailed)
	}
}

func TestVerifyManagerRejectsABotWithoutManagementRights(t *testing.T) {
	bot := &fakeBot{me: telegram.User{ID: 1, IsBot: true, Username: "PocketClawSetupBot", CanManageBots: false}}
	svc, _ := newService(t, bot)
	if _, err := svc.VerifyManager(context.Background()); err == nil {
		t.Fatal("VerifyManager accepted a bot without bot management enabled")
	} else if !strings.Contains(err.Error(), "Bot Management Mode") {
		t.Fatalf("error %q does not tell the operator what to enable", err)
	}
}

func TestVerifyManagerAcceptsAManagerBot(t *testing.T) {
	bot := &fakeBot{me: telegram.User{ID: 1, IsBot: true, Username: "PocketClawSetupBot", CanManageBots: true}}
	svc, _ := newService(t, bot)
	me, err := svc.VerifyManager(context.Background())
	if err != nil {
		t.Fatalf("VerifyManager: %v", err)
	}
	if me.Username != "PocketClawSetupBot" {
		t.Fatalf("manager username = %q", me.Username)
	}
}

func TestEachPairingGetsADistinctUsername(t *testing.T) {
	svc, _ := newService(t, &fakeBot{})
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		snap, _, err := svc.CreatePairing()
		if err != nil {
			t.Fatalf("CreatePairing: %v", err)
		}
		if seen[snap.SuggestedUsername] {
			t.Fatalf("username %q was suggested twice while both pairings were live", snap.SuggestedUsername)
		}
		seen[snap.SuggestedUsername] = true
	}
}
