package anthropicmessages

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const bearerTestKey = "oc-test-key-not-a-real-secret"

// The Messages provider requires an explicit max_tokens, so every request in
// these tests carries the same minimal option set.
func chatOptions() map[string]any {
	return map[string]any{"max_tokens": 64}
}

func messagesResponse() string {
	return `{"id":"msg_1","type":"message","role":"assistant","model":"m",
		"content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn",
		"usage":{"input_tokens":1,"output_tokens":1}}`
}

// Default behavior must be unchanged for api.anthropic.com: X-API-Key only,
// with no Authorization header.
func TestDefaultSendsAnthropicKeyOnly(t *testing.T) {
	var gotKey, gotAuth, gotVersion, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		gotAuth = r.Header.Get("Authorization")
		gotVersion = r.Header.Get("Anthropic-Version")
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(messagesResponse()))
	}))
	defer srv.Close()

	p := NewProviderWithTimeout(bearerTestKey, srv.URL+"/v1", "PocketClaw/test", 0)
	if _, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-x", chatOptions()); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if gotKey != bearerTestKey {
		t.Errorf("X-API-Key = %q, want the configured key", gotKey)
	}
	if gotAuth != "" {
		t.Errorf("Authorization = %q, want it unset by default", gotAuth)
	}
	if gotVersion == "" {
		t.Error("Anthropic-Version must always be sent")
	}
	if gotPath != "/v1/messages" {
		t.Errorf("path = %q, want %q", gotPath, "/v1/messages")
	}
}

// A gateway that fronts the Messages protocol with a bearer account key gets
// both header forms, so one OpenCode key satisfies either convention.
func TestWithBearerAuthSendsBothHeaders(t *testing.T) {
	var gotKey, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(messagesResponse()))
	}))
	defer srv.Close()

	p := NewProviderWithTimeout(bearerTestKey, srv.URL+"/zen/v1", "", 0, WithBearerAuth())
	if _, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-x", chatOptions()); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if gotKey != bearerTestKey {
		t.Errorf("X-API-Key = %q, want the configured key", gotKey)
	}
	if gotAuth != "Bearer "+bearerTestKey {
		t.Errorf("Authorization = %q, want a bearer token", gotAuth)
	}
}

// The OpenCode bases survive the strip-then-append "/v1" normalization and
// resolve to the official messages endpoints.
func TestOpenCodeBasesResolveToOfficialMessagesEndpoint(t *testing.T) {
	cases := map[string]string{
		"/zen/v1":     "/zen/v1/messages",
		"/zen/go/v1":  "/zen/go/v1/messages",
		"/zen/v1/":    "/zen/v1/messages",
		"/zen/go/v1/": "/zen/go/v1/messages",
	}

	for base, wantPath := range cases {
		var gotPath string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			_, _ = w.Write([]byte(messagesResponse()))
		}))

		p := NewProviderWithTimeout(bearerTestKey, srv.URL+base, "", 0, WithBearerAuth())
		_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-x", chatOptions())
		srv.Close()

		if err != nil {
			t.Errorf("base %q: Chat() error = %v", base, err)
			continue
		}
		if gotPath != wantPath {
			t.Errorf("base %q: path = %q, want %q", base, gotPath, wantPath)
		}
	}
}

func TestBearerErrorsDoNotLeakAPIKey(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusNotFound} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"denied"}`))
		}))

		p := NewProviderWithTimeout(bearerTestKey, srv.URL+"/zen/v1", "", 0, WithBearerAuth())
		_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-x", chatOptions())
		srv.Close()

		if err == nil {
			t.Errorf("status %d: expected an error", status)
			continue
		}
		if strings.Contains(err.Error(), bearerTestKey) {
			t.Errorf("status %d: error leaks the API key: %v", status, err)
		}
	}
}
