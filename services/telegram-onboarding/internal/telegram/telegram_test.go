package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const fakeToken = "123456:AAHfakemanagertokenvaluenotreal00000"

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := New(fakeToken, server.Client())
	c.SetBaseURL(server.URL)
	return c
}

func TestGetMeParsesCanManageBots(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/getMe") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"username":"PocketClawSetupBot","can_manage_bots":true}}`))
	})
	me, err := c.GetMe(context.Background())
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if !me.CanManageBots || me.Username != "PocketClawSetupBot" {
		t.Fatalf("GetMe = %+v", me)
	}
}

func TestGetUpdatesParsesManagedBotUpdate(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.FormValue("allowed_updates"); got != `["managed_bot"]` {
			t.Errorf("allowed_updates = %q, want [\"managed_bot\"]", got)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":7,"managed_bot":{"user":{"id":555,"first_name":"Ada"},"bot":{"id":9001,"is_bot":true,"username":"pocketclaw_abcd1234_bot"}}}]}`))
	})
	updates, err := c.GetUpdates(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("GetUpdates: %v", err)
	}
	if len(updates) != 1 || updates[0].ManagedBot == nil {
		t.Fatalf("updates = %+v", updates)
	}
	got := updates[0].ManagedBot
	if got.User.ID != 555 || got.Bot.ID != 9001 || got.Bot.Username != "pocketclaw_abcd1234_bot" {
		t.Fatalf("managed bot update = %+v", got)
	}
	if updates[0].UpdateID != 7 {
		t.Fatalf("update_id = %d", updates[0].UpdateID)
	}
}

func TestGetUpdatesIgnoresUnrelatedUpdateTypes(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":8,"message":{"message_id":1,"text":"hello"}}]}`))
	})
	updates, err := c.GetUpdates(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("GetUpdates: %v", err)
	}
	if len(updates) != 1 || updates[0].ManagedBot != nil {
		t.Fatalf("a non-managed-bot update decoded as one: %+v", updates)
	}
}

func TestGetManagedBotTokenSendsUserIDAndReturnsToken(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.FormValue("user_id"); got != "9001" {
			t.Errorf("user_id = %q, want 9001", got)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":"9001:CHILD-TOKEN"}`))
	})
	token, err := c.GetManagedBotToken(context.Background(), 9001)
	if err != nil {
		t.Fatalf("GetManagedBotToken: %v", err)
	}
	if token != "9001:CHILD-TOKEN" {
		t.Fatalf("token = %q", token)
	}
}

func TestGetManagedBotTokenRejectsEmptyResult(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":""}`))
	})
	if _, err := c.GetManagedBotToken(context.Background(), 9001); err == nil {
		t.Fatal("an empty token was accepted")
	}
}

func TestAPIErrorDoesNotEchoTheManagerToken(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		// Telegram sometimes quotes the request URL, which contains the token.
		body, _ := json.Marshal(map[string]any{
			"ok":          false,
			"error_code":  401,
			"description": "Unauthorized: bot" + fakeToken + " is invalid",
		})
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(body)
	})
	_, err := c.GetMe(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), fakeToken) {
		t.Fatalf("the error echoes the manager token: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("the error was not redacted: %v", err)
	}
}

func TestTransportErrorDoesNotEchoTheManagerToken(t *testing.T) {
	c := New(fakeToken, &http.Client{Timeout: 2 * time.Second})
	// A host that cannot resolve produces an error quoting the full URL.
	c.SetBaseURL("http://pocketclaw-onboarding-invalid.invalid")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := c.GetMe(ctx)
	if err == nil {
		t.Fatal("expected a transport error")
	}
	if strings.Contains(err.Error(), fakeToken) {
		t.Fatalf("the transport error echoes the manager token: %v", err)
	}
}

func TestRedact(t *testing.T) {
	secret := "123456:AAsecretpart"
	cases := []struct{ in, want string }{
		{"nothing to hide", "nothing to hide"},
		{"url https://api.telegram.org/bot" + secret + "/getMe", "url https://api.telegram.org/bot[REDACTED]/getMe"},
		// Only the secret half appears, which happens when Telegram echoes it.
		{"leaked AAsecretpart here", "leaked [REDACTED] here"},
	}
	for _, tc := range cases {
		if got := Redact(tc.in, secret); got != tc.want {
			t.Fatalf("Redact(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if got := Redact("anything", ""); got != "anything" {
		t.Fatalf("Redact with an empty secret changed the text: %q", got)
	}
}
