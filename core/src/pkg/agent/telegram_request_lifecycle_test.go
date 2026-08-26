package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type lifecycleScriptProvider struct {
	mu        sync.Mutex
	responses []string
}

func (p *lifecycleScriptProvider) Chat(
	context.Context,
	[]providers.Message,
	[]providers.ToolDefinition,
	string,
	map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	response := ""
	if len(p.responses) > 0 {
		response = p.responses[0]
		p.responses = p.responses[1:]
	}
	return &providers.LLMResponse{Content: response}, nil
}

func (p *lifecycleScriptProvider) GetDefaultModel() string { return "test-model" }

type lifecycleDelivery struct {
	messageID string
	content   string
}

type lifecycleTestChannel struct {
	mu           sync.Mutex
	edits        []lifecycleDelivery
	sends        []string
	deletes      []string
	editFail     bool
	editFailures int
	sendErr      error
}

func (c *lifecycleTestChannel) Name() string                        { return "telegram" }
func (c *lifecycleTestChannel) Start(context.Context) error         { return nil }
func (c *lifecycleTestChannel) Stop(context.Context) error          { return nil }
func (c *lifecycleTestChannel) IsRunning() bool                     { return true }
func (c *lifecycleTestChannel) IsAllowed(string) bool               { return true }
func (c *lifecycleTestChannel) IsAllowedSender(bus.SenderInfo) bool { return true }
func (c *lifecycleTestChannel) ReasoningChannelID() string          { return "" }
func (c *lifecycleTestChannel) Send(_ context.Context, msg bus.OutboundMessage) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sends = append(c.sends, msg.Content)
	if c.sendErr != nil {
		return nil, c.sendErr
	}
	return []string{"sent"}, nil
}
func (c *lifecycleTestChannel) EditMessage(
	_ context.Context,
	_ string,
	messageID string,
	content string,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.editFail || c.editFailures > 0 {
		if c.editFailures > 0 {
			c.editFailures--
		}
		return context.DeadlineExceeded
	}
	c.edits = append(c.edits, lifecycleDelivery{messageID: messageID, content: content})
	return nil
}

func (c *lifecycleTestChannel) snapshot() ([]lifecycleDelivery, []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]lifecycleDelivery(nil), c.edits...), append([]string(nil), c.sends...)
}

func (c *lifecycleTestChannel) DeleteMessage(
	_ context.Context,
	_ string,
	messageID string,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deletes = append(c.deletes, messageID)
	return nil
}

func (c *lifecycleTestChannel) deletedSnapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.deletes...)
}

func waitForLifecycleDeliveries(t *testing.T, channel *lifecycleTestChannel, count int) []lifecycleDelivery {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		edits, _ := channel.snapshot()
		if len(edits) >= count {
			return edits
		}
		time.Sleep(5 * time.Millisecond)
	}
	edits, sends := channel.snapshot()
	t.Fatalf("deliveries timed out: edits=%v sends=%v", edits, sends)
	return nil
}

func TestTelegramRequestsCompleteIndependentlyAfterEmptyProviderResponse(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.MaxParallelTurns = 1
	cfg.Agents.Defaults.MaxLLMRetries = 1

	msgBus := bus.NewMessageBus()
	provider := &lifecycleScriptProvider{responses: []string{"", "response B", "response C"}}
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

	publish := func(id string) {
		manager.RecordPlaceholderForLifecycle("telegram", "chat", id, "ph-"+id)
		err := msgBus.PublishInbound(context.Background(), bus.InboundMessage{
			Context: bus.InboundContext{
				Channel:  "telegram",
				ChatID:   "chat",
				ChatType: "direct",
				SenderID: "test-user",
				Raw:      map[string]string{bus.LifecycleIDMetadataKey: id},
			},
			Content: "test input",
		})
		if err != nil {
			t.Fatalf("PublishInbound(%s) error = %v", id, err)
		}
	}

	// A must finish while the channel is otherwise idle; B is not a wake-up.
	publish("A")
	edits := waitForLifecycleDeliveries(t, delivery, 1)
	if edits[0].messageID != "ph-A" || edits[0].content != defaultResponse {
		t.Fatalf("request A delivery = %+v", edits[0])
	}

	// Close arrivals are queued as complete inbound requests, not coalesced
	// steering, and each final edits only its own placeholder in FIFO order.
	publish("B")
	publish("C")
	edits = waitForLifecycleDeliveries(t, delivery, 3)
	want := []lifecycleDelivery{
		{messageID: "ph-A", content: defaultResponse},
		{messageID: "ph-B", content: "response B"},
		{messageID: "ph-C", content: "response C"},
	}
	for i := range want {
		if edits[i] != want[i] {
			t.Fatalf("delivery[%d] = %+v, want %+v", i, edits[i], want[i])
		}
	}
}

func TestTelegramPlaceholderEditFailureFallsBackToSend(t *testing.T) {
	msgBus := bus.NewMessageBus()
	delivery := &lifecycleTestChannel{editFail: true}
	manager := newStartedTestChannelManager(t, msgBus, nil, "telegram", delivery)
	manager.RecordPlaceholderForLifecycle("telegram", "chat", "A", "ph-A")

	err := manager.SendMessage(context.Background(), bus.OutboundMessage{
		Context: bus.InboundContext{
			Channel: "telegram",
			ChatID:  "chat",
			Raw:     map[string]string{bus.LifecycleIDMetadataKey: "A"},
		},
		Content: "final response",
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	_, sends := delivery.snapshot()
	if len(sends) != 1 || sends[0] != "final response" {
		t.Fatalf("fallback sends = %v, want one final response", sends)
	}
	deletes := delivery.deletedSnapshot()
	if len(deletes) != 1 || deletes[0] != "ph-A" {
		t.Fatalf("fallback placeholder cleanup = %v, want ph-A", deletes)
	}
}

func TestTelegramSendFailureRetriesPlaceholderAsFinalDelivery(t *testing.T) {
	msgBus := bus.NewMessageBus()
	delivery := &lifecycleTestChannel{
		editFailures: 1,
		sendErr:      channels.ErrSendFailed,
	}
	manager := newStartedTestChannelManager(t, msgBus, nil, "telegram", delivery)
	manager.RecordPlaceholderForLifecycle("telegram", "chat", "A", "ph-A")

	err := manager.SendMessage(context.Background(), bus.OutboundMessage{
		Context: bus.InboundContext{
			Channel: "telegram",
			ChatID:  "chat",
			Raw:     map[string]string{bus.LifecycleIDMetadataKey: "A"},
		},
		Content: "final response",
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	edits, sends := delivery.snapshot()
	if len(sends) != 1 {
		t.Fatalf("normal send attempts = %v, want one", sends)
	}
	if len(edits) != 1 || edits[0] != (lifecycleDelivery{messageID: "ph-A", content: "final response"}) {
		t.Fatalf("terminal placeholder edit = %v", edits)
	}
}

var _ channels.Channel = (*lifecycleTestChannel)(nil)
var _ channels.MessageEditor = (*lifecycleTestChannel)(nil)
var _ channels.MessageDeleter = (*lifecycleTestChannel)(nil)
