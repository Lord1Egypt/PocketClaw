package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/common"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// bigOutputTool returns a fixed, large result, like the incident's curl call.
type bigOutputTool struct{ output string }

func (t *bigOutputTool) Name() string        { return "big_output" }
func (t *bigOutputTool) Description() string { return "returns a very large result" }
func (t *bigOutputTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func (t *bigOutputTool) Execute(context.Context, map[string]any) *tools.ToolResult {
	return tools.UserResult(t.output)
}

// budgetToolProvider asks for one tool call, then answers, and keeps every
// request it was sent.
type budgetToolProvider struct {
	mu       sync.Mutex
	requests [][]providers.Message
	toolDefs [][]providers.ToolDefinition
}

func (p *budgetToolProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	defs []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests = append(p.requests, append([]providers.Message(nil), messages...))
	p.toolDefs = append(p.toolDefs, defs)
	if len(p.requests) == 1 {
		return &providers.LLMResponse{ToolCalls: []providers.ToolCall{{
			ID: "call-1", Name: "big_output", Arguments: map[string]any{},
		}}}, nil
	}
	return &providers.LLMResponse{Content: "done", FinishReason: "stop"}, nil
}

func (p *budgetToolProvider) GetDefaultModel() string { return "scripted" }

func newBudgetTestLoop(t *testing.T, provider providers.LLMProvider, maxTokens, window int) (*AgentLoop, *AgentInstance) {
	t.Helper()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         maxTokens,
				ContextWindow:     window,
				MaxToolIterations: 10,
			},
		},
	}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)
	t.Cleanup(al.Close)
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	return al, agent
}

// textLines builds ordinary line-oriented output of about n bytes. A run of one
// repeated letter would not do: the registry already replaces base64-looking
// blobs before this budget ever sees them.
func textLines(n int) string {
	var b strings.Builder
	for i := 0; b.Len() < n; i++ {
		fmt.Fprintf(&b, "line %07d: {\"path\": \"src/file_%d.go\", \"size\": %d}\n", i, i, i*7)
	}
	return b.String()
}

func lastToolMessage(t *testing.T, messages []providers.Message) providers.Message {
	t.Helper()
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "tool" {
			return messages[i]
		}
	}
	t.Fatal("request carries no tool result")
	return providers.Message{}
}

// A2: a 3 MB runtime result must reach neither the next provider request nor
// the session whole.
func TestHugeToolResultIsBoundedBeforeTheNextProviderRequest(t *testing.T) {
	provider := &budgetToolProvider{}
	al, agent := newBudgetTestLoop(t, provider, 32768, 0)
	output := "curl exited 0 after 2140ms (completed)\n\nstdout:\n" +
		textLines(3_202_895) + "TAIL-MARKER\n"
	agent.Tools.Register(&bigOutputTool{output: output})

	resp, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey: "budget-1", Channel: "cli", ChatID: "direct",
		UserMessage: "fetch it", DefaultResponse: defaultResponse,
	})
	if err != nil {
		t.Fatalf("runAgentLoop: %v", err)
	}
	if resp != "done" {
		t.Fatalf("response = %q", resp)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider saw %d requests, want 2", len(provider.requests))
	}

	sent := lastToolMessage(t, provider.requests[1])
	if len(sent.Content) > config.DefaultToolMaxResultBytes {
		t.Fatalf("the next request carried %d bytes of tool output; budget %d",
			len(sent.Content), config.DefaultToolMaxResultBytes)
	}
	for _, want := range []string{"curl exited 0", "[OUTPUT TRUNCATED]", "original bytes: ", "TAIL-MARKER"} {
		if !strings.Contains(sent.Content, want) {
			t.Fatalf("bounded result lacks %q", want)
		}
	}

	stored := lastToolMessage(t, agent.Sessions.GetHistory("budget-1"))
	if len(stored.Content) > config.DefaultToolMaxResultBytes {
		t.Fatalf("session history kept %d bytes of the tool result", len(stored.Content))
	}
}

// A4: a request that grew past the window because of tool output is fitted
// before it is sent, not after the provider refuses it.
func TestPostToolPreflightFitsTheRequestBeforeSendingIt(t *testing.T) {
	const maxTokens = 1024
	output := textLines(60_000)
	run := func(window int) *budgetToolProvider {
		provider := &budgetToolProvider{}
		al, agent := newBudgetTestLoop(t, provider, maxTokens, window)
		agent.Tools.Register(&bigOutputTool{output: output})
		if _, err := al.runAgentLoop(context.Background(), agent, processOptions{
			SessionKey: "budget-2", Channel: "cli", ChatID: "direct",
			UserMessage: "fetch it", DefaultResponse: defaultResponse,
		}); err != nil {
			t.Fatalf("runAgentLoop: %v", err)
		}
		if len(provider.requests) != 2 {
			t.Fatalf("provider saw %d requests, want 2", len(provider.requests))
		}
		return provider
	}

	// Size the window from the real prompt: room for everything but most of
	// the tool result, so the post-tool request is over budget unless the
	// preflight acts.
	probe := run(1 << 30)
	base := estimateRequestTokens(probe.requests[0], probe.toolDefs[0], maxTokens)
	window := base + 8000
	if !isOverContextBudget(window, probe.requests[1], probe.toolDefs[1], maxTokens) {
		t.Fatal("test setup: the unshortened post-tool request already fits the window")
	}

	provider := run(window)
	second := provider.requests[1]
	if isOverContextBudget(window, second, provider.toolDefs[1], maxTokens) {
		t.Fatalf("post-tool request sent over budget: estimated %d tokens, window %d",
			estimateRequestTokens(second, provider.toolDefs[1], maxTokens), window)
	}
	sent := lastToolMessage(t, second)
	if len(sent.Content) >= len(output) || !strings.Contains(sent.Content, "[OUTPUT TRUNCATED]") {
		t.Fatalf("tool result was not shortened with a notice (len %d)", len(sent.Content))
	}
}

// budgetOverflowProvider fails its first `failures` calls with err, then answers.
type budgetOverflowProvider struct {
	mu       sync.Mutex
	calls    int
	failures int
	err      error
}

func (p *budgetOverflowProvider) Chat(
	context.Context, []providers.Message, []providers.ToolDefinition, string, map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.calls <= p.failures {
		return nil, p.err
	}
	return &providers.LLMResponse{Content: "recovered", FinishReason: "stop"}, nil
}

func (p *budgetOverflowProvider) GetDefaultModel() string { return "overflow" }

func http400(body string) error {
	return &common.HTTPError{StatusCode: 400, BodyPreview: body}
}

func runOverflowTurn(t *testing.T, provider *budgetOverflowProvider) (string, error) {
	t.Helper()
	al, agent := newBudgetTestLoop(t, provider, 4096, 0)
	agent.Sessions.SetHistory("overflow", []providers.Message{
		{Role: "user", Content: "old 1"}, {Role: "assistant", Content: "reply 1"},
		{Role: "user", Content: "old 2"}, {Role: "assistant", Content: "reply 2"},
	})
	return al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey: "overflow", Channel: "cli", ChatID: "direct",
		UserMessage: "go", DefaultResponse: defaultResponse,
	})
}

const overflowBody = `{"type":"error","error":{"type":"invalid_request_error",` +
	`"message":"prompt is too long: 950000 tokens > 200000 maximum"}}`

// A5: a 400 whose body says the request was too large is compacted and resent.
func TestContextOverflow400IsRecoveredByOneResend(t *testing.T) {
	provider := &budgetOverflowProvider{failures: 1, err: http400(overflowBody)}
	resp, err := runOverflowTurn(t, provider)
	if err != nil {
		t.Fatalf("turn failed: %v", err)
	}
	if resp != "recovered" || provider.calls != 2 {
		t.Fatalf("resp=%q calls=%d, want recovered after exactly 2 calls", resp, provider.calls)
	}
}

// A5: recovery is attempted once. The old path compacted and resent up to
// MaxLLMRetries times.
func TestContextOverflowIsRecoveredAtMostOnce(t *testing.T) {
	provider := &budgetOverflowProvider{failures: 99, err: http400(overflowBody)}
	_, err := runOverflowTurn(t, provider)
	if err == nil {
		t.Fatal("expected the turn to fail")
	}
	if provider.calls != 2 {
		t.Fatalf("provider called %d times, want 2 (the request and one resend)", provider.calls)
	}
	userFacing, ok := AsUserFacingError(err)
	if !ok || userFacing.Code != CodeContextBudgetExceeded {
		t.Fatalf("error %v is not %s", err, CodeContextBudgetExceeded)
	}
	if !strings.Contains(formatProcessingError(err), "/clear") {
		t.Fatalf("user sees %q, which does not say what to do", formatProcessingError(err))
	}
}

// A5: an ordinary 400 is not a context error, however it is worded otherwise.
func TestNonContext400IsNotRetriedAsContext(t *testing.T) {
	for name, body := range map[string]string{
		"schema":     `{"error":{"type":"invalid_request_error","message":"tools.0.function.parameters: unknown field"}}`,
		"max_tokens": `{"error":{"type":"invalid_request_error","message":"max_tokens: 64000 > 8192, the maximum allowed"}}`,
		"parameter":  `{"error":{"code":"InvalidParameter","message":"temperature must be <= 1"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			provider := &budgetOverflowProvider{failures: 99, err: http400(body)}
			_, err := runOverflowTurn(t, provider)
			if err == nil {
				t.Fatal("expected the turn to fail")
			}
			if provider.calls != 1 {
				t.Fatalf("a non-context 400 was sent %d times, want 1", provider.calls)
			}
			if userFacing, ok := AsUserFacingError(err); ok && userFacing.Code == CodeContextBudgetExceeded {
				t.Fatalf("a non-context 400 was reported as %s", CodeContextBudgetExceeded)
			}
		})
	}
}

func TestIsProviderContextOverflow(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"400 prompt too long", http400(overflowBody), true},
		{"400 maximum context length", http400("This model's maximum context length is 131072 tokens"), true},
		{"400 openai string too long", http400("Invalid 'messages[3].content': string too long. " +
			"Expected a string with maximum length 10485760"), true},
		{"413 without a body", &common.HTTPError{StatusCode: 413}, true},
		{"unwrapped context_length_exceeded", errors.New("context_length_exceeded"), true},
		{"400 schema error", http400("tools.0.function.parameters: unknown field"), false},
		{"400 max_tokens parameter", http400("max_tokens: 64000 > 8192"), false},
		{"401 even if it says too long", &common.HTTPError{StatusCode: 401, BodyPreview: "prompt is too long"}, false},
		{"404 model not found", &common.HTTPError{StatusCode: 404, BodyPreview: "model not found"}, false},
		{"429 rate limit", &common.HTTPError{StatusCode: 429, BodyPreview: "request too large for tokens per min"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failErr := providers.ClassifyError(tc.err, "p", "m")
			if got := isProviderContextOverflow(tc.err, failErr); got != tc.want {
				t.Fatalf("isProviderContextOverflow = %v, want %v", got, tc.want)
			}
		})
	}
}
