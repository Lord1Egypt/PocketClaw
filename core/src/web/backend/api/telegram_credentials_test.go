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
