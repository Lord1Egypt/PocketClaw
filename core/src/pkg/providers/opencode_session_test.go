package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers/common"
)

// PC-DEF-032, on the wire.
//
// OpenCode Go refuses a request with no x-opencode-session:
//
//	HTTP 400 {"error":{"type":"MissingSessionID","message":"Error from provider
//	 (Console Go): Request is missing x-opencode-session and cannot be routed
//	 efficiently..."}}
//
// The same request with the header answers 200. These tests assert what leaves
// PocketClaw, against a server that records it, because the defect was that
// nothing left at all.

type capturedRequest struct {
	mu      sync.Mutex
	headers []http.Header
	paths   []string
}

func (c *capturedRequest) record(r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers = append(c.headers, r.Header.Clone())
	c.paths = append(c.paths, r.URL.Path)
}

func (c *capturedRequest) snapshot() ([]http.Header, []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]http.Header(nil), c.headers...), append([]string(nil), c.paths...)
}

// chatCompletionsServer answers the minimum an OpenAI-compatible turn needs,
// and records every request it was asked to answer.
func chatCompletionsServer(t *testing.T) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.record(r)

		// A gateway that requires the session header refuses without it, which
		// is what makes the assertions below meaningful rather than decorative.
		if r.Header.Get("x-opencode-session") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"MissingSessionID",` +
				`"message":"Request is missing x-opencode-session"}}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"c1","object":"chat.completion","choices":` +
			`[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}]}`))
	}))
	t.Cleanup(server.Close)
	return server, captured
}

func openCodeModelConfig(provider, apiBase, model string) *config.ModelConfig {
	return &config.ModelConfig{
		ModelName: "opencode-test",
		Provider:  provider,
		Model:     model,
		APIBase:   apiBase,
		APIKeys:   config.SimpleSecureStrings("sk-test-key-value-0123456789"),
	}
}

func turnOptions(scope string) map[string]any {
	return map[string]any{
		"max_tokens":            256,
		common.SessionOptionKey: scope,
	}
}

// Item 1 and item 8: a non-streaming OpenCode Go turn carries the header, and
// therefore succeeds against a gateway that demands it.
func TestOpenCodeGoChatSendsTheSessionHeader(t *testing.T) {
	server, captured := chatCompletionsServer(t)

	provider, _, err := CreateProviderFromConfig(
		openCodeModelConfig("opencode_go", server.URL, "deepseek-v4.1-flash"))
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}

	resp, err := provider.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil, "deepseek-v4.1-flash",
		turnOptions("telegram:123456789"))
	if err != nil {
		t.Fatalf("Chat() error = %v — the gateway refused the request", err)
	}
	if resp == nil || resp.Content != "OK" {
		t.Fatalf("Chat() response = %+v, want content OK", resp)
	}

	headers, paths := captured.snapshot()
	if len(headers) != 1 {
		t.Fatalf("requests = %d, want 1", len(headers))
	}
	if paths[0] != "/chat/completions" {
		t.Fatalf("path = %q, want /chat/completions", paths[0])
	}
	if got := headers[0].Get("x-opencode-session"); got == "" {
		t.Fatal("x-opencode-session was not sent; this is the defect")
	}
	if got := headers[0].Get("User-Agent"); got != OpenCodeUserAgent {
		t.Fatalf("User-Agent = %q, want %q", got, OpenCodeUserAgent)
	}
}

// Item 9: a streamed turn is the same conversation and must carry the same
// header. It was a separate request-building site, which is how one of these
// gets missed.
func TestOpenCodeGoStreamSendsTheSameSessionHeader(t *testing.T) {
	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.record(r)
		if r.Header.Get("x-opencode-session") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"type":"MissingSessionID","message":"missing"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	provider, _, err := CreateProviderFromConfig(
		openCodeModelConfig("opencode_go", server.URL, "deepseek-v4.1-flash"))
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	streamer, ok := provider.(StreamingProvider)
	if !ok {
		t.Skip("this provider does not stream")
	}

	const scope = "telegram:123456789"
	if _, err := streamer.ChatStream(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil, "deepseek-v4.1-flash",
		turnOptions(scope), func(string) {}); err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	headers, _ := captured.snapshot()
	if len(headers) == 0 {
		t.Fatal("no streaming request was made")
	}
	if got := headers[0].Get("x-opencode-session"); got != common.StableSessionID(scope) {
		t.Fatalf("streamed x-opencode-session = %q, want the conversation's id", got)
	}
}

// Items 2, 3 and 4 as the runtime actually produces them: several requests in
// one conversation — a second turn, a tool-call continuation, a retry — all
// carry one value, because they all carry the turn's scope.
func TestOpenCodeSessionIsIdenticalAcrossOneConversation(t *testing.T) {
	server, captured := chatCompletionsServer(t)

	provider, _, err := CreateProviderFromConfig(
		openCodeModelConfig("opencode_go", server.URL, "kimi-k3"))
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}

	const scope = "telegram:123456789"
	conversation := []struct {
		name     string
		messages []Message
	}{
		{"first turn", []Message{{Role: "user", Content: "hi"}}},
		{"second turn", []Message{{Role: "user", Content: "again"}}},
		{"tool continuation", []Message{
			{Role: "user", Content: "run it"},
			{Role: "assistant", Content: ""},
			{Role: "tool", Content: "result", ToolCallID: "call_1"},
		}},
		{"retry of the same turn", []Message{{Role: "user", Content: "hi"}}},
	}

	for _, step := range conversation {
		if _, err := provider.Chat(context.Background(), step.messages, nil, "kimi-k3",
			turnOptions(scope)); err != nil {
			t.Fatalf("%s: Chat() error = %v", step.name, err)
		}
	}

	headers, _ := captured.snapshot()
	if len(headers) != len(conversation) {
		t.Fatalf("requests = %d, want %d", len(headers), len(conversation))
	}
	first := headers[0].Get("x-opencode-session")
	if first == "" {
		t.Fatal("no session header on the first request")
	}
	for i, header := range headers {
		if got := header.Get("x-opencode-session"); got != first {
			t.Fatalf("%s sent session %q, want %q: one conversation is one session",
				conversation[i].name, got, first)
		}
	}
}

// Item 5.
func TestOpenCodeSessionDiffersForANewConversation(t *testing.T) {
	server, captured := chatCompletionsServer(t)

	provider, _, err := CreateProviderFromConfig(
		openCodeModelConfig("opencode_go", server.URL, "kimi-k3"))
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}

	for _, scope := range []string{"telegram:111", "telegram:222"} {
		if _, err := provider.Chat(context.Background(),
			[]Message{{Role: "user", Content: "hi"}}, nil, "kimi-k3",
			turnOptions(scope)); err != nil {
			t.Fatalf("Chat(%s) error = %v", scope, err)
		}
	}

	headers, _ := captured.snapshot()
	if headers[0].Get("x-opencode-session") == headers[1].Get("x-opencode-session") {
		t.Fatal("two conversations shared one session id")
	}
}

// Item 6, end to end: whatever else is on the wire, the key is only ever in the
// Authorization header, never in the routing identity.
func TestOpenCodeSessionHeaderNeverCarriesTheAPIKey(t *testing.T) {
	server, captured := chatCompletionsServer(t)

	const apiKey = "sk-test-key-value-0123456789"
	provider, _, err := CreateProviderFromConfig(
		openCodeModelConfig("opencode_go", server.URL, "kimi-k3"))
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if _, err := provider.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil, "kimi-k3",
		turnOptions("telegram:1")); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	headers, _ := captured.snapshot()
	session := headers[0].Get("x-opencode-session")
	if session == "" || session == apiKey {
		t.Fatalf("x-opencode-session = %q", session)
	}
	lowered := strings.ToLower(session)
	for _, fragment := range []string{"sk-", apiKey, "0123456789"} {
		if strings.Contains(lowered, strings.ToLower(fragment)) {
			t.Fatalf("credential fragment %q appeared in the session id %q", fragment, session)
		}
	}
}

// Item 7. The header is an OpenCode compatibility behaviour and belongs to
// nothing else: a provider that never asked for it must not receive it.
func TestSessionHeaderReachesNoOtherProvider(t *testing.T) {
	for _, provider := range []string{"deepseek", "openrouter", "groq", "custom-openai", "zhipu"} {
		t.Run(provider, func(t *testing.T) {
			server, captured := chatCompletionsServer(t)

			built, _, err := CreateProviderFromConfig(
				openCodeModelConfig(provider, server.URL, "some-model"))
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}
			// The stub answers 400 without the header, so this call is expected
			// to fail; the assertion is about what was sent, not the outcome.
			_, _ = built.Chat(context.Background(),
				[]Message{{Role: "user", Content: "hi"}}, nil, "some-model",
				turnOptions("telegram:1"))

			headers, _ := captured.snapshot()
			if len(headers) == 0 {
				t.Fatal("no request was made")
			}
			if got := headers[0].Get("x-opencode-session"); got != "" {
				t.Fatalf("%s received x-opencode-session = %q; the header is "+
					"OpenCode's and must not be injected elsewhere", provider, got)
			}
			if got := headers[0].Get("User-Agent"); got == OpenCodeUserAgent {
				t.Fatalf("%s received OpenCode's user agent", provider)
			}
		})
	}
}
