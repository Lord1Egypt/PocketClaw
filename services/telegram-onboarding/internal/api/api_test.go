package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/onboarding"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/pairing"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/ratelimit"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/telegram"
)

type fakeBot struct {
	token      string
	tokenCalls int
}

func (f *fakeBot) GetMe(context.Context) (telegram.User, error) {
	return telegram.User{ID: 1, Username: "PocketClawSetupBot", CanManageBots: true}, nil
}
func (f *fakeBot) GetUpdates(context.Context, int64, int) ([]telegram.Update, error) {
	return nil, nil
}
func (f *fakeBot) GetManagedBotToken(_ context.Context, _ int64) (string, error) {
	f.tokenCalls++
	return f.token, nil
}

type harness struct {
	server  *httptest.Server
	service *onboarding.Service
	store   *pairing.Store
	bot     *fakeBot
}

func newHarness(t *testing.T, burst int) *harness {
	t.Helper()
	bot := &fakeBot{token: "9001:CHILD-TOKEN"}
	store := pairing.NewStore(10 * time.Minute)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := onboarding.New(store, bot, "PocketClawSetupBot", log)
	api := New(service, store, ratelimit.New(burst, 1), false, log)
	server := httptest.NewServer(api.Handler())
	t.Cleanup(server.Close)
	return &harness{server: server, service: service, store: store, bot: bot}
}

func (h *harness) do(t *testing.T, method, path, bearer string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, h.server.URL+path, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := h.server.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, body
}

func (h *harness) createPairing(t *testing.T) createResponse {
	t.Helper()
	resp, body := h.do(t, http.MethodPost, "/telegram/pairings", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create returned %d: %s", resp.StatusCode, body)
	}
	var out createResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	return out
}

func TestCreatePairingResponseShape(t *testing.T) {
	h := newHarness(t, 10)
	got := h.createPairing(t)

	if len(got.PairingID) != 32 {
		t.Fatalf("pairing_id %q is not 32 hex chars", got.PairingID)
	}
	if len(got.PollToken) != 64 {
		t.Fatalf("poll_token is %d chars, want 64", len(got.PollToken))
	}
	if got.DeepLink != got.QRPayload {
		t.Fatalf("qr_payload %q differs from deep_link %q", got.QRPayload, got.DeepLink)
	}
	if !strings.HasPrefix(got.DeepLink, "https://t.me/newbot/PocketClawSetupBot/") {
		t.Fatalf("deep_link %q is not a managed-bot creation link", got.DeepLink)
	}
	if got.SuggestedName != "PocketClaw Agent" {
		t.Fatalf("suggested_name = %q", got.SuggestedName)
	}
	if _, err := time.Parse(time.RFC3339, got.ExpiresAt); err != nil {
		t.Fatalf("expires_at %q is not RFC3339: %v", got.ExpiresAt, err)
	}
	if got.PollIntervalSeconds <= 0 {
		t.Fatalf("poll_interval_seconds = %d", got.PollIntervalSeconds)
	}
}

func TestQRPayloadCarriesNoSecret(t *testing.T) {
	h := newHarness(t, 10)
	got := h.createPairing(t)
	for _, secret := range []string{got.PollToken, got.PairingID, "CHILD-TOKEN"} {
		if strings.Contains(got.QRPayload, secret) {
			t.Fatalf("qr_payload %q contains a secret", got.QRPayload)
		}
		if strings.Contains(got.DeepLink, secret) {
			t.Fatalf("deep_link %q contains a secret", got.DeepLink)
		}
	}
}

func TestPollingRequiresTheCorrectToken(t *testing.T) {
	h := newHarness(t, 10)
	created := h.createPairing(t)

	resp, _ := h.do(t, http.MethodGet, "/telegram/pairings/"+created.PairingID, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("polling without a token returned %d, want 401", resp.StatusCode)
	}

	// A wrong token must be indistinguishable from a missing pairing, so an
	// attacker cannot enumerate live pairing ids.
	resp, _ = h.do(t, http.MethodGet, "/telegram/pairings/"+created.PairingID, strings.Repeat("0", 64))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("polling with a wrong token returned %d, want 404", resp.StatusCode)
	}
	resp, _ = h.do(t, http.MethodGet, "/telegram/pairings/ffffffffffffffffffffffffffffffff", strings.Repeat("0", 64))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("polling an unknown pairing returned %d, want 404", resp.StatusCode)
	}
}

func TestPollingNeverReturnsTheBotToken(t *testing.T) {
	h := newHarness(t, 10)
	created := h.createPairing(t)
	h.completePairing(t, created)

	resp, body := h.do(t, http.MethodGet, "/telegram/pairings/"+created.PairingID, created.PollToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("poll returned %d: %s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "CHILD-TOKEN") {
		t.Fatalf("the poll response leaked the bot token: %s", body)
	}
	var status statusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if status.State != string(pairing.StateReady) {
		t.Fatalf("state = %q, want ready", status.State)
	}
	if status.BotUsername != created.SuggestedUsername {
		t.Fatalf("bot_username = %q, want %q", status.BotUsername, created.SuggestedUsername)
	}
	if status.OwnerUserID != 555 {
		t.Fatalf("owner_user_id = %d, want 555", status.OwnerUserID)
	}
}

func TestTokenCollectionIsSingleUse(t *testing.T) {
	h := newHarness(t, 10)
	created := h.createPairing(t)
	h.completePairing(t, created)

	resp, body := h.do(t, http.MethodPost, "/telegram/pairings/"+created.PairingID+"/token", created.PollToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("collect returned %d: %s", resp.StatusCode, body)
	}
	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if token.BotToken != "9001:CHILD-TOKEN" {
		t.Fatalf("bot_token = %q", token.BotToken)
	}
	if token.OwnerUserID != 555 || token.BotUserID != 9001 {
		t.Fatalf("token response = %+v", token)
	}

	resp, _ = h.do(t, http.MethodPost, "/telegram/pairings/"+created.PairingID+"/token", created.PollToken)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("replaying token collection returned %d, want 404", resp.StatusCode)
	}
}

func TestTokenCollectionRequiresTheCorrectToken(t *testing.T) {
	h := newHarness(t, 10)
	created := h.createPairing(t)
	h.completePairing(t, created)

	resp, body := h.do(t, http.MethodPost, "/telegram/pairings/"+created.PairingID+"/token", strings.Repeat("0", 64))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("collecting with a wrong token returned %d, want 404", resp.StatusCode)
	}
	if strings.Contains(string(body), "CHILD-TOKEN") {
		t.Fatalf("an unauthorized response leaked the token: %s", body)
	}

	// The failed attempt must not have burned the delivery.
	resp, _ = h.do(t, http.MethodPost, "/telegram/pairings/"+created.PairingID+"/token", created.PollToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the rightful owner got %d after a failed attempt by someone else", resp.StatusCode)
	}
}

func TestTokenCollectionBeforeReadyConflicts(t *testing.T) {
	h := newHarness(t, 10)
	created := h.createPairing(t)
	resp, body := h.do(t, http.MethodPost, "/telegram/pairings/"+created.PairingID+"/token", created.PollToken)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("collecting while pending returned %d, want 409", resp.StatusCode)
	}
	if strings.Contains(string(body), "CHILD-TOKEN") {
		t.Fatalf("response leaked a token: %s", body)
	}
}

func TestExpiredPairingPollsAsNotFound(t *testing.T) {
	h := newHarness(t, 10)
	created := h.createPairing(t)
	now := time.Now()
	h.store.SetClock(func() time.Time { return now.Add(11 * time.Minute) })

	resp, _ := h.do(t, http.MethodGet, "/telegram/pairings/"+created.PairingID, created.PollToken)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("polling an expired pairing returned %d, want 404", resp.StatusCode)
	}
}

func TestPairingCreationIsRateLimited(t *testing.T) {
	h := newHarness(t, 2)
	for i := 0; i < 2; i++ {
		resp, _ := h.do(t, http.MethodPost, "/telegram/pairings", "")
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("request %d returned %d, want 201", i+1, resp.StatusCode)
		}
	}
	resp, body := h.do(t, http.MethodPost, "/telegram/pairings", "")
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("the third request returned %d, want 429", resp.StatusCode)
	}
	if !strings.Contains(string(body), "rate_limited") {
		t.Fatalf("429 body = %s", body)
	}
}

func TestHealthReportsManagerWithoutSecrets(t *testing.T) {
	h := newHarness(t, 10)
	resp, body := h.do(t, http.MethodGet, "/healthz", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health returned %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["manager_username"] != "PocketClawSetupBot" {
		t.Fatalf("health payload = %s", body)
	}
	for _, forbidden := range []string{"token", "TOKEN", "secret"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("health payload mentions %q: %s", forbidden, body)
		}
	}
}

func TestResponsesAreNotCacheable(t *testing.T) {
	h := newHarness(t, 10)
	created := h.createPairing(t)
	resp, _ := h.do(t, http.MethodGet, "/telegram/pairings/"+created.PairingID, created.PollToken)
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

// completePairing drives the pairing to ready the way a real Telegram update
// would, without touching a network.
func (h *harness) completePairing(t *testing.T, created createResponse) {
	t.Helper()
	h.service.HandleUpdate(context.Background(), telegram.Update{
		UpdateID: 1,
		ManagedBot: &telegram.ManagedBotUpdated{
			User: telegram.User{ID: 555},
			Bot:  telegram.User{ID: 9001, IsBot: true, Username: created.SuggestedUsername},
		},
	})
}
