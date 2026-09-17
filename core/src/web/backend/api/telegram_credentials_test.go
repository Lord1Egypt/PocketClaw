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
// the update stream. These prove the preflight rejects an owned bot without
// mutating the other service, and that it never weakens the 401 path.

type fakeTelegramAPI struct {
	mu       sync.Mutex
	methods  []string
	getMe    string
	webhook  string
	updates  string
	updatesC int // status code for the getUpdates probe; 0 means 200
}

func (f *fakeTelegramAPI) server(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

		f.mu.Lock()
		f.methods = append(f.methods, method)
		status := f.updatesC
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
			_, _ = w.Write([]byte(`{"ok":true,"result":{"url":` + quoteJSON(f.webhook) + `,"pending_update_count":0}}`))
		case "getUpdates":
			if status != 0 && status != http.StatusOK {
				w.WriteHeader(status)
			}
			if f.updates != "" {
				_, _ = w.Write([]byte(f.updates))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
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
	for _, m := range f.methods {
		if m == method {
			n++
		}
	}
	return n
}

func quoteJSON(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

// A healthy candidate passes all three checks.
func TestTelegramPreflightAcceptsAHealthyBot(t *testing.T) {
	fake := &fakeTelegramAPI{}
	server := fake.server(t)

	if err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, ""); err != nil {
		t.Fatalf("healthy candidate rejected: %v", err)
	}
	if fake.count("getMe") != 1 || fake.count("getWebhookInfo") != 1 || fake.count("getUpdates") != 1 {
		t.Fatalf("preflight must call getMe, getWebhookInfo and the getUpdates probe: %v", fake.methods)
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

func TestTelegramPreflightClassifiesAWebhookConflictFromThePoll(t *testing.T) {
	fake := &fakeTelegramAPI{
		updates: `{"ok":false,"error_code":409,"description":"Conflict: can't use getUpdates method while webhook is active"}`,
	}
	server := fake.server(t)

	err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, "")
	if !errors.Is(err, ErrTelegramWebhookConflict) {
		t.Fatalf("err = %v, want webhook conflict", err)
	}
}

func TestTelegramPreflightKeepsInvalidCredentialsInvalid(t *testing.T) {
	fake := &fakeTelegramAPI{getMe: `{"ok":false,"error_code":401,"description":"Unauthorized"}`}
	server := fake.server(t)

	err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, "")
	if !errors.Is(err, ErrTelegramCredentialsInvalid) {
		t.Fatalf("err = %v, want invalid credentials", err)
	}
	// A 401 and a 409 are different outcomes and must not be conflated.
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

// An unexpected probe answer (a 5xx, malformed body) is not a conflict: a valid
// token must not be refused because of an unrelated upstream failure. The
// runtime getUpdates 409 path remains the backstop.
func TestTelegramPreflightDoesNotTreatAnUnexpectedProbeAnswerAsConflict(t *testing.T) {
	fake := &fakeTelegramAPI{updatesC: http.StatusInternalServerError, updates: "not json"}
	server := fake.server(t)

	if err := validateTelegramCredentials(context.Background(), "123:abc", server.URL, ""); err != nil {
		t.Fatalf("an unexpected probe answer must not reject a valid candidate: %v", err)
	}
}
