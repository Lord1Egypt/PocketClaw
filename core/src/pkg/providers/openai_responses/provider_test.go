package openairesponses

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

const testKey = "oc-test-key-not-a-real-secret"

func userMessages() []Message {
	return []Message{{Role: "user", Content: "hello"}}
}

// The Responses endpoint is the base plus exactly one "responses" segment, and
// the model travels in the body. A gateway base with extra path segments must
// be preserved.
func TestChatPostsToResponsesPathWithBearerAuth(t *testing.T) {
	var gotPath, gotAuth, gotMethod, gotContentType string
	body := make(map[string]any)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = decodeJSON(r, &body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","output":[{"type":"message","role":"assistant",
			"content":[{"type":"output_text","text":"hi"}]}]}`))
	}))
	defer srv.Close()

	p := NewProvider(testKey, srv.URL+"/zen/v1", "", "PocketClaw/test", 0, nil)
	resp, err := p.Chat(t.Context(), userMessages(), nil, "gpt-5.4", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/zen/v1/responses" {
		t.Errorf("path = %q, want %q", gotPath, "/zen/v1/responses")
	}
	if gotAuth != "Bearer "+testKey {
		t.Errorf("Authorization header = %q, want a bearer token", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if body["model"] != "gpt-5.4" {
		t.Errorf("request body model = %v, want gpt-5.4", body["model"])
	}
	if resp.Content != "hi" {
		t.Errorf("parsed content = %q, want %q", resp.Content, "hi")
	}
}

func TestChatTrimsTrailingSlashFromBase(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"r","output":[]}`))
	}))
	defer srv.Close()

	p := NewProvider(testKey, srv.URL+"/zen/go/v1/", "", "", 0, nil)
	if _, err := p.Chat(t.Context(), userMessages(), nil, "gpt-5.4-codex", nil); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if gotPath != "/zen/go/v1/responses" {
		t.Errorf("path = %q, want %q", gotPath, "/zen/go/v1/responses")
	}
}

func TestChatSendsCustomHeaders(t *testing.T) {
	var gotCustom string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCustom = r.Header.Get("X-Source")
		_, _ = w.Write([]byte(`{"id":"r","output":[]}`))
	}))
	defer srv.Close()

	p := NewProvider(testKey, srv.URL, "", "", 0, map[string]string{"X-Source": "pocketclaw"})
	if _, err := p.Chat(t.Context(), userMessages(), nil, "gpt-5.4", nil); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if gotCustom != "pocketclaw" {
		t.Errorf("X-Source = %q, want %q", gotCustom, "pocketclaw")
	}
}

func TestChatRequiresAPIKeyAndBase(t *testing.T) {
	if _, err := NewProvider("", "https://example.invalid/v1", "", "", 0, nil).
		Chat(t.Context(), userMessages(), nil, "m", nil); err == nil {
		t.Error("expected an error when the API key is missing")
	}
	if _, err := NewProvider(testKey, "", "", "", 0, nil).
		Chat(t.Context(), userMessages(), nil, "m", nil); err == nil {
		t.Error("expected an error when the API base is missing")
	}
}

// An upstream failure must surface the status without echoing the credential.
func TestChatErrorsDoNotLeakAPIKey(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusNotFound} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
		}))

		_, err := NewProvider(testKey, srv.URL+"/zen/v1", "", "", 0, nil).
			Chat(t.Context(), userMessages(), nil, "gpt-5.4", nil)
		srv.Close()

		if err == nil {
			t.Errorf("status %d: expected an error", status)
			continue
		}
		if strings.Contains(err.Error(), testKey) {
			t.Errorf("status %d: error leaks the API key: %v", status, err)
		}
	}
}

func TestGetDefaultModelIsEmpty(t *testing.T) {
	if got := NewProvider(testKey, "https://example.invalid/v1", "", "", 0, nil).GetDefaultModel(); got != "" {
		t.Errorf("GetDefaultModel() = %q, want empty for a generic endpoint", got)
	}
}

var _ = protocoltypes.Message{}

func decodeJSON(r *http.Request, out *map[string]any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(out)
}
