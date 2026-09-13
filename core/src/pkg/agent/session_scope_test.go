package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/common"
)

// PC-DEF-032, the half the provider tests cannot cover.
//
// A provider that knows how to send a conversation header is useless if nothing
// tells it which conversation the turn belongs to. These assert the agent puts
// the turn's scope on the options every request is built from.

type sessionScopeCaptureProvider struct {
	mu     sync.Mutex
	scopes []string
	seen   int
}

func (p *sessionScopeCaptureProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen++
	scope, _ := opts[common.SessionOptionKey].(string)
	p.scopes = append(p.scopes, scope)
	return &providers.LLMResponse{Content: "ok"}, nil
}

func (p *sessionScopeCaptureProvider) GetDefaultModel() string { return "test-model" }

func (p *sessionScopeCaptureProvider) snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.scopes...)
}

func sessionScopeConfig() *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
		ModelList: []*config.ModelConfig{{
			ModelName: "test-model",
			Model:     "openai/test-model",
		}},
	}
}

func sendTurn(t *testing.T, al *AgentLoop, chatID, content string) {
	t.Helper()
	if _, err := al.processMessage(context.Background(), bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  "pocketclaw",
			ChatID:   chatID,
			ChatType: "direct",
			SenderID: "pocketclaw-user",
		},
		Content: content,
	}); err != nil {
		t.Fatalf("processMessage(%q) error = %v", content, err)
	}
}

// Every turn carries a scope. Without one the provider falls back to the
// process identifier, which would put every conversation in one bucket.
func TestTurnOptionsCarryTheConversationScope(t *testing.T) {
	t.Setenv("PICOCLAW_BUILTIN_SKILLS", t.TempDir())
	capture := &sessionScopeCaptureProvider{}
	al := NewAgentLoop(sessionScopeConfig(), bus.NewMessageBus(), capture)

	sendTurn(t, al, "pocketclaw:scope-a", "hello")

	scopes := capture.snapshot()
	if len(scopes) == 0 {
		t.Fatal("the provider was never called")
	}
	if scopes[0] == "" {
		t.Fatalf("no %q on the turn options; a gateway that routes on "+
			"conversation identity has nothing to route on", common.SessionOptionKey)
	}
}

// Item 2: a follow-up in the same conversation is the same session.
func TestFollowUpTurnsShareOneConversationScope(t *testing.T) {
	t.Setenv("PICOCLAW_BUILTIN_SKILLS", t.TempDir())
	capture := &sessionScopeCaptureProvider{}
	al := NewAgentLoop(sessionScopeConfig(), bus.NewMessageBus(), capture)

	sendTurn(t, al, "pocketclaw:scope-a", "first")
	sendTurn(t, al, "pocketclaw:scope-a", "second")

	scopes := capture.snapshot()
	if len(scopes) < 2 {
		t.Fatalf("provider calls = %d, want at least 2", len(scopes))
	}
	if scopes[0] != scopes[1] {
		t.Fatalf("scopes %q and %q differ within one conversation", scopes[0], scopes[1])
	}
	if common.StableSessionID(scopes[0]) != common.StableSessionID(scopes[1]) {
		t.Fatal("two turns of one conversation would send two session ids")
	}
}

// Item 5: a different conversation is a different session.
func TestDifferentConversationsGetDifferentScopes(t *testing.T) {
	t.Setenv("PICOCLAW_BUILTIN_SKILLS", t.TempDir())
	capture := &sessionScopeCaptureProvider{}
	al := NewAgentLoop(sessionScopeConfig(), bus.NewMessageBus(), capture)

	sendTurn(t, al, "pocketclaw:scope-a", "hello")
	sendTurn(t, al, "pocketclaw:scope-b", "hello")

	scopes := capture.snapshot()
	if len(scopes) < 2 {
		t.Fatalf("provider calls = %d, want at least 2", len(scopes))
	}
	if scopes[0] == scopes[1] {
		t.Fatalf("two conversations shared the scope %q", scopes[0])
	}
	if common.StableSessionID(scopes[0]) == common.StableSessionID(scopes[1]) {
		t.Fatal("two conversations would send one session id")
	}
}
