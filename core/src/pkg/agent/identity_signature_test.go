package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func kernelIdentityPart(t *testing.T, workspace string) PromptPart {
	t.Helper()
	for _, part := range NewContextBuilder(workspace).BuildSystemPromptParts() {
		if part.ID == "kernel.identity" {
			return part
		}
	}
	t.Fatal("system prompt has no kernel.identity part")
	return PromptPart{}
}

// The kernel identity is the one prompt part every agent on every channel
// receives, so anything decorative in it is an instruction to imitate. The
// lobster was read as a signature and mirrored at the end of replies.
func TestKernelIdentityNamesPocketClawWithoutTheLobsterEmoji(t *testing.T) {
	workspace := setupWorkspace(t, nil)
	defer cleanupWorkspace(t, workspace)

	identity := kernelIdentityPart(t, workspace).Content

	for _, required := range []string{
		"# PocketClaw (",
		"You are PocketClaw, a helpful AI assistant.",
	} {
		if !strings.Contains(identity, required) {
			t.Fatalf("kernel identity missing %q:\n%s", required, identity)
		}
	}
	if strings.Contains(identity, "\U0001F99E") {
		t.Fatalf("kernel identity carries the lobster emoji:\n%s", identity)
	}
}

// A model's reply reaches the channel exactly as the model wrote it. There is
// no formatter between the turn and delivery, and the fix for the mirrored
// lobster must not become one: stripping a character from generated text would
// also strip it from text the user asked for.
func TestFinalResponseIsDeliveredVerbatimIncludingEmoji(t *testing.T) {
	const reply = "Steamed \U0001F99E with drawn butter \U0001F9C8 — done \u2705"

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.MaxParallelTurns = 1
	cfg.Agents.Defaults.MaxLLMRetries = 1

	msgBus := bus.NewMessageBus()
	provider := &lifecycleScriptProvider{responses: []string{reply}}
	delivery := &lifecycleTestChannel{}
	manager := newStartedTestChannelManager(t, msgBus, nil, "telegram", delivery)
	agentLoop := NewAgentLoop(cfg, msgBus, provider)
	agentLoop.SetChannelManager(manager)
	defer agentLoop.Close()

	runCtx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- agentLoop.Run(runCtx) }()
	defer func() {
		cancel()
		select {
		case err := <-runDone:
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Run() did not stop")
		}
	}()

	manager.RecordPlaceholderForLifecycle("telegram", "chat", "A", "ph-A")
	err := msgBus.PublishInbound(context.Background(), bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  "telegram",
			ChatID:   "chat",
			ChatType: "direct",
			SenderID: "test-user",
			Raw:      map[string]string{bus.LifecycleIDMetadataKey: "A"},
		},
		Content: "how do I cook one",
	})
	if err != nil {
		t.Fatalf("PublishInbound() error = %v", err)
	}

	edits := waitForLifecycleDeliveries(t, delivery, 1)
	if edits[0].content != reply {
		t.Fatalf("delivered %q, want the model's text verbatim %q", edits[0].content, reply)
	}
}
