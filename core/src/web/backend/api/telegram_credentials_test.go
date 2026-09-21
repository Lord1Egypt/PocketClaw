package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
	"github.com/sipeed/picoclaw/pkg/status"
)

// PC-DEF-069. The candidate preflight is the replacement transaction's commit
// gate. getMe proves the token is valid; it does not prove PocketClaw can own
// the update stream.
//
// The getUpdates probe is deliberately non-consuming: it carries no offset, so
// nothing is confirmed or dropped. Telegram confirms an update only when a later
// getUpdates is called with an offset higher than its update_id, and a negative
// offset would forget all earlier updates -- exactly the pending first /start
// PC-DEF-061 exists to preserve. The fake below models that confirmation rule,
// so these tests prove the probe leaves pending updates intact.

type capturedCall struct {
	method string
	body   string
}

type fakeTelegramAPI struct {
	mu sync.Mutex

	calls      []capturedCall
	getMe      string
	webhook    string
	webhookSeq []string
	webhookIdx int
	updates    string
	updatesC   int
	pending    []int64
	// sendMessage, when set, is the raw envelope the fake answers the greeting
	// with. Empty means an ordinary success.
	sendMessage string
}

func (f *fakeTelegramAPI) server(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

		f.mu.Lock()
		f.calls = append(f.calls, capturedCall{method: method, body: string(raw)})
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch method {
		case "getMe":
			if f.getMe != "" {
				_, _ = w.Write([]byte(f.getMe))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":42,"is_bot":true,"username":"bot"}}`))

		case "getWebhookInfo":
			f.mu.Lock()
			url := f.webhook
			if len(f.webhookSeq) > 0 {
				idx := f.webhookIdx
				if idx >= len(f.webhookSeq) {
					idx = len(f.webhookSeq) - 1
				}
				url = f.webhookSeq[idx]
				f.webhookIdx++
			}
			f.mu.Unlock()
			_, _ = w.Write([]byte(`{"ok":true,"result":{"url":` + quoteJSON(url) + `,"pending_update_count":0}}`))

		case "getUpdates":
			var params struct {
				Offset *int64 `json:"offset"`
				Limit  int    `json:"limit"`
			}
			_ = json.Unmarshal(raw, &params)

			f.mu.Lock()
			// Telegram confirms (forgets) updates whose id is below the offset.
			if params.Offset != nil {
				off := *params.Offset
				kept := make([]int64, 0, len(f.pending))
				for _, id := range f.pending {
					if id >= off {
						kept = append(kept, id)
					}
				}
				f.pending = kept
			}
			status := f.updatesC
			rawResult := f.updates
			pending := append([]int64(nil), f.pending...)
			f.mu.Unlock()

			if status != 0 && status != http.StatusOK {
				w.WriteHeader(status)
			}
			if rawResult != "" {
				_, _ = w.Write([]byte(rawResult))
				return
			}
			limit := params.Limit
			if limit <= 0 {
				limit = 100
			}
			updates := make([]map[string]any, 0, limit)
			for i, id := range pending {
				if i >= limit {
					break
				}
				updates = append(updates, map[string]any{
					"update_id": id,
					"message": map[string]any{
						"message_id": int(id),
						"chat":       map[string]any{"id": id, "type": "private"},
						"from":       map[string]any{"id": id, "is_bot": false},
						"text":       "/start",
					},
				})
			}
			result, _ := json.Marshal(updates)
			_, _ = w.Write([]byte(`{"ok":true,"result":` + string(result) + `}`))

		case "sendMessage":
			if f.sendMessage != "" {
				_, _ = w.Write([]byte(f.sendMessage))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))

		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func (f *fakeTelegramAPI) count(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if c.method == method {
			n++
		}
	}
	return n
}

func (f *fakeTelegramAPI) firstBody(method string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if c.method == method {
			return c.body
		}
	}
	return ""
}

func (f *fakeTelegramAPI) pendingCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pending)
}

func quoteJSON(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func TestTelegramPreflightAcceptsAHealthyBot(t *testing.T) {
	fake := &fakeTelegramAPI{}
	server := fake.server(t)

	if err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, ""); err != nil {
		t.Fatalf("healthy candidate rejected: %v", err)
	}
	if fake.count("getMe") != 1 || fake.count("getWebhookInfo") != 1 || fake.count("getUpdates") != 1 {
		t.Fatalf("preflight must call getMe, getWebhookInfo and the getUpdates probe: %v", fake.calls)
	}
}

// The ownership probe must never use a negative offset. offset=-1 asks Telegram
// for the last update and forgets all earlier ones, which would discard pending
// updates.
func TestTelegramPreflightNeverUsesANegativeOffset(t *testing.T) {
	fake := &fakeTelegramAPI{pending: []int64{101}}
	server := fake.server(t)

	if err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, ""); err != nil {
		t.Fatalf("healthy candidate rejected: %v", err)
	}

	body := fake.firstBody("getUpdates")
	if body == "" {
		t.Fatal("the preflight did not issue a getUpdates probe")
	}
	var params struct {
		Offset *int64 `json:"offset"`
	}
	if err := json.Unmarshal([]byte(body), &params); err != nil {
		t.Fatalf("probe body is not JSON: %v", err)
	}
	if params.Offset != nil && *params.Offset < 0 {
		t.Fatalf("probe used a negative offset %d; pending updates would be forgotten", *params.Offset)
	}
	if strings.Contains(body, "-1") {
		t.Fatalf("probe body carries a negative offset: %s", body)
	}
}

// Pending updates must survive the probe: it confirms nothing, so the real
// generation still receives the pending /start.
func TestTelegramPreflightPreservesPendingUpdates(t *testing.T) {
	fake := &fakeTelegramAPI{pending: []int64{101, 102}}
	server := fake.server(t)

	if err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, ""); err != nil {
		t.Fatalf("healthy candidate rejected: %v", err)
	}
	if got := fake.pendingCount(); got != 2 {
		t.Fatalf("pending updates after the probe = %d, want 2; the probe confirmed them", got)
	}

	// The real intake polls from an unset offset and must still receive them.
	probe, err := telegramValidationCall(
		context.Background(), http.DefaultClient, server.URL, "123:abc",
		"getUpdates", strings.NewReader(`{"limit":10,"timeout":0}`),
	)
	if err != nil || !probe.OK {
		t.Fatalf("second poll failed: %v", err)
	}
	var updates []struct {
		UpdateID int64 `json:"update_id"`
	}
	if err := json.Unmarshal(probe.Result, &updates); err != nil {
		t.Fatal(err)
	}
	if len(updates) != 2 || updates[0].UpdateID != 101 || updates[1].UpdateID != 102 {
		t.Fatalf("pending updates were not redelivered to the active generation: %+v", updates)
	}
}

func TestTelegramPreflightRejectsAnActiveWebhookWithoutDeletingIt(t *testing.T) {
	fake := &fakeTelegramAPI{webhook: "https://other-service.invalid/hook"}
	server := fake.server(t)

	err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, "")
	if !errors.Is(err, ErrTelegramWebhookConflict) {
		t.Fatalf("err = %v, want webhook conflict", err)
	}
	if fake.count("deleteWebhook") != 0 || fake.count("setWebhook") != 0 {
		t.Fatal("the preflight must never delete or replace a webhook")
	}
	if fake.count("getUpdates") != 0 {
		t.Fatal("an active webhook must be refused before the poll probe")
	}
}

func TestTelegramPreflightRejectsAnotherPoller(t *testing.T) {
	fake := &fakeTelegramAPI{
		updates: `{"ok":false,"error_code":409,"description":"Conflict: terminated by other getUpdates request"}`,
	}
	server := fake.server(t)

	err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, "")
	if !errors.Is(err, ErrTelegramBotInUse) {
		t.Fatalf("err = %v, want bot-in-use conflict", err)
	}
	if fake.count("deleteWebhook") != 0 || fake.count("setWebhook") != 0 {
		t.Fatal("the preflight must not mutate anything")
	}
}

// The 409 subtype comes from a fresh getWebhookInfo, not from the English
// description: here the first webhook check reads empty and the re-check finds a
// URL, so it is webhook_active despite a generic 409 description.
func TestTelegramPreflightClassifiesA409ByRecheckingTheWebhook(t *testing.T) {
	fake := &fakeTelegramAPI{
		webhookSeq: []string{"", "https://other-service.invalid/hook"},
		updates:    `{"ok":false,"error_code":409,"description":"Conflict: terminated by other getUpdates request"}`,
	}
	server := fake.server(t)

	err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, "")
	if !errors.Is(err, ErrTelegramWebhookConflict) {
		t.Fatalf("err = %v, want webhook conflict from the re-check", err)
	}
}

func TestTelegramPreflightKeepsInvalidCredentialsInvalid(t *testing.T) {
	fake := &fakeTelegramAPI{getMe: `{"ok":false,"error_code":401,"description":"Unauthorized"}`}
	server := fake.server(t)

	err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, "")
	if !errors.Is(err, ErrTelegramCredentialsInvalid) {
		t.Fatalf("err = %v, want invalid credentials", err)
	}
	if errors.Is(err, ErrTelegramWebhookConflict) || errors.Is(err, ErrTelegramBotInUse) {
		t.Fatal("a 401 must never be classified as a conflict")
	}
}

func TestTelegramPreflightTreatsAProbeUnauthorizedAsInvalid(t *testing.T) {
	fake := &fakeTelegramAPI{
		updates: `{"ok":false,"error_code":401,"description":"Unauthorized"}`,
	}
	server := fake.server(t)

	err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, "")
	if !errors.Is(err, ErrTelegramCredentialsInvalid) {
		t.Fatalf("err = %v, want invalid credentials", err)
	}
}

func TestTelegramPreflightDoesNotTreatAnUnexpectedProbeAnswerAsConflict(t *testing.T) {
	fake := &fakeTelegramAPI{updatesC: http.StatusInternalServerError, updates: "not json"}
	server := fake.server(t)

	if err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, ""); err != nil {
		t.Fatalf("an unexpected probe answer must not reject a valid candidate: %v", err)
	}
}

// PC-DEF-073. PocketClaw must never probe ownership against its own active
// generation.
//
// The candidate preflight ends in a real getUpdates call and Telegram allows one
// long-polling consumer per bot, so aiming it at a bot this install is already
// polling makes PocketClaw collide with itself: Telegram answers 409 to one of
// the two, and the runtime treats a 409 as terminal with no retry. A healthy
// generation would be retired on evidence PocketClaw manufactured.
//
// Reachability could not be ruled out from the repository: the candidate token
// comes from an external pairing service through CollectCredentials, so whether
// it can ever be a bot this install already holds is not decidable here. These
// pin the guard instead of the assumption.

// selfCollisionEnv commits a Telegram bot, points the real validator at a fake
// Bot API, and makes the gateway report whatever channel state the case needs.
func selfCollisionEnv(
	t *testing.T,
	fake *fakeTelegramAPI,
	channels []status.Channel,
	committedToken string,
) *Handler {
	t.Helper()
	handler, _, configPath := onboardingTestEnv(t)
	botAPI := fake.server(t)

	// Commit the bot exactly as a pairing would, and aim the validator at the
	// fake Bot API by way of the channel's own base URL.
	cfg, channel, settings, err := handler.loadTelegramConfigForUpdate()
	if err != nil {
		t.Fatalf("loadTelegramConfigForUpdate: %v", err)
	}
	settings.Token.Set(committedToken)
	settings.BaseURL = botAPI.URL
	channel.Enabled = true
	channel.Type = config.ChannelTelegram
	channel.AllowFrom = config.FlexibleStringSlice{"424242"}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// The real validator, so a probe it makes is a call the fake records.
	handler.SetTelegramCredentialValidator(validateTelegramCredentials)

	const bearer = "gateway-bearer-token"
	server := fakeGatewayHealth(t, bearer, channels)
	host, port := splitHostPortForTest(t, server.URL)
	gateway.mu.Lock()
	previous := gateway.pidData
	gateway.pidData = &ppid.PidFileData{PID: 1, Host: host, Port: port, Token: bearer}
	gateway.mu.Unlock()
	t.Cleanup(func() {
		gateway.mu.Lock()
		gateway.pidData = previous
		gateway.mu.Unlock()
	})

	previousProbe := gatewayRunningProbe
	gatewayRunningProbe = func(*Handler) bool { return true }
	t.Cleanup(func() { gatewayRunningProbe = previousProbe })

	return handler
}

// runningTelegramChannel is the gateway saying it owns the update stream now.
func runningTelegramChannel() []status.Channel {
	generation := uint64(11)
	return []status.Channel{{
		Name:              "telegram",
		Configured:        true,
		Running:           true,
		PollingGeneration: &generation,
	}}
}

const selfCollisionToken = "123456789:committed-and-polling"

// The defect: re-pairing the bot this install is already polling must not issue
// a competing getUpdates.
func TestSameAuthoritativeTokenIssuesNoCompetingProbe(t *testing.T) {
	fake := &fakeTelegramAPI{}
	handler := selfCollisionEnv(t, fake, runningTelegramChannel(), selfCollisionToken)

	if _, _, err := handler.writeTelegramCredentials(selfCollisionToken, 424242); err != nil {
		t.Fatalf("re-pairing the authoritative bot must succeed: %v", err)
	}

	if got := fake.count("getUpdates"); got != 0 {
		t.Fatalf("getUpdates probes = %d, want 0: PocketClaw probed its own active generation", got)
	}
}

// A different candidate is a genuine ownership question and still gets the full
// three-call validation. The guard must not become a way to skip it.
func TestDifferentCandidateStillReceivesFullValidation(t *testing.T) {
	fake := &fakeTelegramAPI{}
	handler := selfCollisionEnv(t, fake, runningTelegramChannel(), selfCollisionToken)

	if _, _, err := handler.writeTelegramCredentials("987654321:a-different-bot", 424242); err != nil {
		t.Fatalf("a healthy replacement must be accepted: %v", err)
	}

	if fake.count("getMe") != 1 || fake.count("getWebhookInfo") != 1 || fake.count("getUpdates") != 1 {
		t.Fatalf("a different candidate must take the full validation: %v", fake.calls)
	}
}

// A webhook on a different candidate is still a conflict, and the committed bot
// is still left alone.
func TestWebhookConflictStillDetectedAlongsideTheGuard(t *testing.T) {
	fake := &fakeTelegramAPI{webhook: `{"ok":true,"result":{"url":"https://other.invalid/hook"}}`}
	handler := selfCollisionEnv(t, fake, runningTelegramChannel(), selfCollisionToken)

	_, _, err := handler.writeTelegramCredentials("987654321:a-different-bot", 424242)
	if !errors.Is(err, ErrTelegramWebhookConflict) {
		t.Fatalf("err = %v, want ErrTelegramWebhookConflict", err)
	}
	if fake.count("deleteWebhook") != 0 {
		t.Fatal("the webhook check must stay non-destructive")
	}
	assertCommittedTokenUnchanged(t, handler, selfCollisionToken)
}

// Another poller on a different candidate is still terminal.
func TestExternalPollerConflictStillTerminalAlongsideTheGuard(t *testing.T) {
	fake := &fakeTelegramAPI{
		updates: `{"ok":false,"error_code":409,"description":"Conflict: terminated by other getUpdates request"}`,
	}
	handler := selfCollisionEnv(t, fake, runningTelegramChannel(), selfCollisionToken)

	_, _, err := handler.writeTelegramCredentials("987654321:a-different-bot", 424242)
	if !errors.Is(err, ErrTelegramBotInUse) {
		t.Fatalf("err = %v, want ErrTelegramBotInUse", err)
	}
	assertCommittedTokenUnchanged(t, handler, selfCollisionToken)
}

// The guard is about an *authoritative* runtime, not merely a matching token.
// Every state that is not "this install owns the stream right now" must fall
// back to the full validation -- otherwise a stale snapshot would let a
// genuinely contested bot through unchecked.
func TestSameTokenWithoutAnAuthoritativeGenerationStillValidates(t *testing.T) {
	zero := uint64(0)
	generation := uint64(11)
	cases := map[string][]status.Channel{
		"channel not running": {{
			Name: "telegram", Configured: true, Running: false,
			PollingGeneration: &generation,
		}},
		"no polling generation": {{
			Name: "telegram", Configured: true, Running: true,
		}},
		"generation zero": {{
			Name: "telegram", Configured: true, Running: true, PollingGeneration: &zero,
		}},
		"runtime failure latched": {{
			Name: "telegram", Configured: true, Running: true,
			PollingGeneration: &generation, RuntimeFailure: "conflict:bot_in_use",
		}},
		"gateway has no telegram channel": {},
	}
	for name, channels := range cases {
		t.Run(name, func(t *testing.T) {
			fake := &fakeTelegramAPI{}
			handler := selfCollisionEnv(t, fake, channels, selfCollisionToken)

			if _, _, err := handler.writeTelegramCredentials(selfCollisionToken, 424242); err != nil {
				t.Fatalf("write: %v", err)
			}
			if got := fake.count("getUpdates"); got != 1 {
				t.Fatalf("getUpdates probes = %d, want 1: no active generation vouches for this token", got)
			}
		})
	}
}

// A near-miss token is not the committed one. The comparison is constant time,
// and it is a whole-value comparison -- a shared prefix proves nothing.
func TestNearMissTokenIsNotTreatedAsAuthoritative(t *testing.T) {
	for _, candidate := range []string{
		selfCollisionToken + "x",
		selfCollisionToken[:len(selfCollisionToken)-1],
		"123456789:committed-and-pollinG",
	} {
		t.Run(candidate, func(t *testing.T) {
			fake := &fakeTelegramAPI{}
			handler := selfCollisionEnv(t, fake, runningTelegramChannel(), selfCollisionToken)

			if _, _, err := handler.writeTelegramCredentials(candidate, 424242); err != nil {
				t.Fatalf("write: %v", err)
			}
			if got := fake.count("getUpdates"); got != 1 {
				t.Fatalf("getUpdates probes = %d, want 1: a near-miss is a different bot", got)
			}
		})
	}
}

// A pending or in-flight apply means the running gateway may hold a different
// credential from the one on disk, so its snapshot cannot vouch for this token.
func TestPendingConfigApplyDisablesTheGuard(t *testing.T) {
	fake := &fakeTelegramAPI{}
	handler := selfCollisionEnv(t, fake, runningTelegramChannel(), selfCollisionToken)

	resetPendingConfigApplyForTest(t)
	markConfigApplyPending("test_pending")

	if _, _, err := handler.writeTelegramCredentials(selfCollisionToken, 424242); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := fake.count("getUpdates"); got != 1 {
		t.Fatalf("getUpdates probes = %d, want 1: a parked apply invalidates the snapshot", got)
	}
}

// The skip must not leave a half-written configuration: the committed bot is
// still exactly the authoritative one afterwards.
func assertCommittedTokenUnchanged(t *testing.T, handler *Handler, want string) {
	t.Helper()
	_, _, settings, err := handler.loadTelegramConfigForUpdate()
	if err != nil {
		t.Fatalf("loadTelegramConfigForUpdate: %v", err)
	}
	if got := strings.TrimSpace(settings.Token.String()); got != want {
		t.Fatalf("committed token changed; a rejected candidate must not displace it")
	}
}

// PC-DEF-061 (managed onboarding). The owner presses Telegram's Start exactly
// once, and that press is what completes the pairing: the onboarding service
// holds a webhook on the child bot and receives the `/start` itself, which is
// how it learns `owner_user_id` at all (see services/README.md). A webhook
// delivery is terminal -- Telegram does not also queue the update for
// getUpdates -- so by the time PocketClaw owns the bot there is nothing left to
// receive. The built-in `/start` path is never reached because it has no input,
// which is why the owner saw their `/start` still sitting there with no
// "Hello! I am PocketClaw." under it, and why the next message worked.
//
// The writer that commits the credential is therefore the one place that can
// honour the press. It already holds the token and the owner, and the private
// chat provably exists because the press created it.

func telegramGreetingBody(t *testing.T, fake *fakeTelegramAPI) string {
	t.Helper()
	return fake.firstBody("sendMessage")
}

func TestManagedPairingGreetsTheOwnerOnce(t *testing.T) {
	fake := &fakeTelegramAPI{}
	handler := selfCollisionEnv(t, fake, nil, selfCollisionToken)

	if _, _, err := handler.writeTelegramCredentials("987654321:a-fresh-bot", 424242); err != nil {
		t.Fatalf("pairing write: %v", err)
	}

	if got := fake.count("sendMessage"); got != 1 {
		t.Fatalf("sendMessage calls = %d, want exactly 1: the Start press must be answered once", got)
	}

	var sent struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}
	if err := json.Unmarshal([]byte(telegramGreetingBody(t, fake)), &sent); err != nil {
		t.Fatalf("greeting body: %v", err)
	}
	if sent.Text != commands.StartReplyText {
		t.Fatalf("greeting = %q, want the built-in start reply %q", sent.Text, commands.StartReplyText)
	}
	if sent.ChatID != 424242 {
		t.Fatalf("greeting chat = %d, want the owner's private chat", sent.ChatID)
	}
}

// A candidate that never becomes the configured bot must not greet anyone: the
// greeting belongs to a committed pairing, not to an attempt.
func TestRejectedCandidateGreetsNobody(t *testing.T) {
	for name, fake := range map[string]*fakeTelegramAPI{
		"webhook conflict": {webhook: `{"ok":true,"result":{"url":"https://other.invalid/hook"}}`},
		"another poller": {
			updates: `{"ok":false,"error_code":409,"description":"Conflict: terminated by other getUpdates request"}`,
		},
		"invalid credentials": {getMe: `{"ok":false,"error_code":401,"description":"Unauthorized"}`},
	} {
		t.Run(name, func(t *testing.T) {
			handler := selfCollisionEnv(t, fake, nil, selfCollisionToken)
			if _, _, err := handler.writeTelegramCredentials("987654321:a-fresh-bot", 424242); err == nil {
				t.Fatal("the candidate should have been rejected")
			}
			if got := fake.count("sendMessage"); got != 0 {
				t.Fatalf("sendMessage calls = %d, want 0 for a rejected candidate", got)
			}
		})
	}
}

// A greeting that cannot be delivered is cosmetic, not a pairing failure: the
// credential is committed and the channel works either way.
func TestAFailedGreetingDoesNotFailThePairing(t *testing.T) {
	fake := &fakeTelegramAPI{
		sendMessage: `{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`,
	}
	handler := selfCollisionEnv(t, fake, nil, selfCollisionToken)

	applied, pending, err := handler.writeTelegramCredentials("987654321:a-fresh-bot", 424242)
	if err != nil {
		t.Fatalf("a failed greeting must not fail the pairing: %v", err)
	}
	_ = applied
	_ = pending
	if got := fake.count("sendMessage"); got != 1 {
		t.Fatalf("sendMessage attempts = %d, want exactly 1 and no retry", got)
	}
}
