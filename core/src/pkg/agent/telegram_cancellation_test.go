package agent

import (
	"context"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// cancellationWaitBudget bounds every "wait for progress" loop in this file.
//
// It is deliberately generous. These tests assert that something happens, never
// that it happens within a particular time, so the only thing a tight deadline
// buys is a failure when the machine is busy — which is how this file produced
// one unreproducible red run. A passing assertion returns as soon as its
// condition holds and never spends this budget. Waits that assert the absence
// of something stay short, and are not written in terms of this constant.
const cancellationWaitBudget = 15 * time.Second

// cancelBlockingProvider holds a turn open until the test releases it, so /stop
// can be delivered while a turn is genuinely in flight rather than racing setup.
type cancelBlockingProvider struct {
	entered  chan struct{}
	release  chan struct{}
	calls    atomic.Int32
	response string

	mu       sync.Mutex
	enteredN int
}

func newCancelBlockingProvider(response string) *cancelBlockingProvider {
	return &cancelBlockingProvider{
		entered:  make(chan struct{}, 16),
		release:  make(chan struct{}),
		response: response,
	}
}

func (p *cancelBlockingProvider) Chat(
	ctx context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.calls.Add(1)
	p.mu.Lock()
	p.enteredN++
	p.mu.Unlock()
	select {
	case p.entered <- struct{}{}:
	default:
	}
	select {
	case <-p.release:
		return &providers.LLMResponse{Content: p.response, FinishReason: "stop"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *cancelBlockingProvider) GetDefaultModel() string { return "blocking-model" }

func (p *cancelBlockingProvider) releaseAll() {
	select {
	case <-p.release:
	default:
		close(p.release)
	}
}

// typingTestChannel records typing start/stop so cancellation cleanup is
// observable, and counts stop calls to prove idempotency.
type typingTestChannel struct {
	lifecycleTestChannel

	typingMu    sync.Mutex
	typingStart int
	typingStop  int
}

func (c *typingTestChannel) StartTyping(_ context.Context, _ string) (func(), error) {
	c.typingMu.Lock()
	c.typingStart++
	c.typingMu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			c.typingMu.Lock()
			c.typingStop++
			c.typingMu.Unlock()
		})
	}, nil
}

func (c *typingTestChannel) typingCounts() (started, stopped int) {
	c.typingMu.Lock()
	defer c.typingMu.Unlock()
	return c.typingStart, c.typingStop
}

type cancellationHarness struct {
	bus      *bus.MessageBus
	loop     *AgentLoop
	manager  *channels.Manager
	channel  *typingTestChannel
	provider *cancelBlockingProvider
}

func newCancellationHarness(t *testing.T, response string) *cancellationHarness {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = harnessWorkspace(t)
	cfg.Agents.Defaults.MaxParallelTurns = 1
	cfg.Agents.Defaults.MaxLLMRetries = 0

	msgBus := bus.NewMessageBus()
	provider := newCancelBlockingProvider(response)
	delivery := &typingTestChannel{}
	manager := newStartedTestChannelManager(t, msgBus, nil, "telegram", delivery)
	loop := NewAgentLoop(cfg, msgBus, provider)
	loop.SetChannelManager(manager)

	runCtx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- loop.Run(runCtx) }()
	t.Cleanup(func() {
		provider.releaseAll()
		cancel()
		select {
		case err := <-runDone:
			if err != nil {
				t.Errorf("Run() error = %v", err)
			}
		case <-time.After(cancellationWaitBudget):
			t.Error("Run() did not stop")
		}
		loop.Close()
	})

	return &cancellationHarness{
		bus: msgBus, loop: loop, manager: manager, channel: delivery, provider: provider,
	}
}

func (h *cancellationHarness) send(t *testing.T, lifecycleID, content string) {
	t.Helper()
	if err := h.bus.PublishInbound(context.Background(), bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  "telegram",
			ChatID:   "chat",
			ChatType: "direct",
			SenderID: "test-user",
			Raw:      map[string]string{bus.LifecycleIDMetadataKey: lifecycleID},
		},
		Content: content,
	}); err != nil {
		t.Fatalf("PublishInbound(%s) error = %v", lifecycleID, err)
	}
}

func (h *cancellationHarness) waitForTurnStart(t *testing.T) {
	t.Helper()
	select {
	case <-h.provider.entered:
	case <-time.After(cancellationWaitBudget):
		t.Fatal("turn never reached the provider")
	}
}

func waitForSends(t *testing.T, ch *typingTestChannel, count int) []string {
	t.Helper()
	deadline := time.Now().Add(cancellationWaitBudget)
	for time.Now().Before(deadline) {
		_, sends := ch.snapshot()
		if len(sends) >= count {
			return sends
		}
		time.Sleep(5 * time.Millisecond)
	}
	edits, sends := ch.snapshot()
	t.Fatalf("expected %d sends, got sends=%v edits=%v", count, sends, edits)
	return nil
}

func waitForDeliveryCount(t *testing.T, ch *typingTestChannel, count int) ([]lifecycleDelivery, []string) {
	t.Helper()
	deadline := time.Now().Add(cancellationWaitBudget)
	for time.Now().Before(deadline) {
		edits, sends := ch.snapshot()
		if len(edits)+len(sends) >= count {
			return edits, sends
		}
		time.Sleep(5 * time.Millisecond)
	}
	edits, sends := ch.snapshot()
	t.Fatalf("expected %d deliveries, got edits=%v sends=%v", count, edits, sends)
	return nil, nil
}

// 1. Ordinary conversational messages keep strict FIFO per conversation.
func TestTelegramNormalMessagesPreserveFIFO(t *testing.T) {
	h := newCancellationHarness(t, "answer")
	h.provider.releaseAll()

	manager := h.manager
	for _, id := range []string{"A", "B", "C"} {
		manager.RecordPlaceholderForLifecycle("telegram", "chat", id, "ph-"+id)
		h.send(t, id, "message "+id)
		// Sequential publication is what the ordering guarantee is about; the
		// mailbox must preserve arrival order once the session is claimed.
		time.Sleep(5 * time.Millisecond)
	}

	deadline := time.Now().Add(cancellationWaitBudget)
	var edits []lifecycleDelivery
	for time.Now().Before(deadline) {
		edits, _ = h.channel.snapshot()
		if len(edits) >= 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(edits) < 3 {
		t.Fatalf("expected 3 deliveries, got %v", edits)
	}
	want := []string{"ph-A", "ph-B", "ph-C"}
	for i, id := range want {
		if edits[i].messageID != id {
			t.Fatalf("delivery[%d] = %s, want FIFO order %v", i, edits[i].messageID, want)
		}
	}
}

// 2 + 3. /stop must not wait behind the turn it exists to cancel: it is
// processed while that turn is still inside the provider call.
func TestTelegramStopBypassesQueuedWorkAndCancelsActiveTurn(t *testing.T) {
	h := newCancellationHarness(t, "should never be delivered")
	manager := h.manager
	manager.RecordPlaceholderForLifecycle("telegram", "chat", "A", "ph-A")

	h.send(t, "A", "long running request")
	h.waitForTurnStart(t)

	// A normal message queued behind the running turn; it must stay queued.
	h.send(t, "B", "queued conversational message")

	h.send(t, "S", "/stop")

	edits, sends := waitForDeliveryCount(t, h.channel, 1)
	all := append([]string(nil), sends...)
	for _, e := range edits {
		all = append(all, e.content)
	}
	joined := strings.Join(all, " | ")
	if !strings.Contains(strings.ToLower(joined), "stopped") {
		t.Fatalf("expected a cancellation acknowledgement, got %q", joined)
	}
	if strings.Contains(joined, "should never be delivered") {
		t.Fatalf("cancelled turn delivered its answer: %q", joined)
	}
	if h.provider.calls.Load() == 0 {
		t.Fatal("the turn never started, so nothing was actually cancelled")
	}
}

// 4. A cancelled turn must never publish its final answer afterwards.
func TestTelegramCancelledTurnDeliversNoStaleAnswer(t *testing.T) {
	h := newCancellationHarness(t, "stale answer")
	manager := h.manager
	manager.RecordPlaceholderForLifecycle("telegram", "chat", "A", "ph-A")

	h.send(t, "A", "long running request")
	h.waitForTurnStart(t)
	h.send(t, "S", "/stop")
	waitForDeliveryCount(t, h.channel, 1)

	// Release the provider: if the turn were still alive it would answer now.
	h.provider.releaseAll()
	time.Sleep(150 * time.Millisecond)

	edits, sends := h.channel.snapshot()
	for _, e := range edits {
		if strings.Contains(e.content, "stale answer") {
			t.Fatalf("cancelled turn delivered a stale answer via edit: %+v", e)
		}
	}
	for _, s := range sends {
		if strings.Contains(s, "stale answer") {
			t.Fatalf("cancelled turn delivered a stale answer via send: %q", s)
		}
	}
}

// 5 + 6. The "Thinking…" placeholder and the typing indicator are both
// lifecycle-scoped state that a cancelled turn would otherwise leave behind.
func TestTelegramCancellationClearsThinkingAndTypingState(t *testing.T) {
	h := newCancellationHarness(t, "never")
	manager := h.manager
	manager.RecordPlaceholderForLifecycle("telegram", "chat", "A", "ph-A")

	h.send(t, "A", "long running request")
	h.waitForTurnStart(t)

	deadline := time.Now().Add(cancellationWaitBudget)
	for time.Now().Before(deadline) {
		if started, _ := h.channel.typingCounts(); started > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	started, _ := h.channel.typingCounts()
	if started == 0 {
		t.Fatal("no typing indicator was started, nothing to assert about cleanup")
	}

	h.send(t, "S", "/stop")
	waitForDeliveryCount(t, h.channel, 1)

	deadline = time.Now().Add(cancellationWaitBudget)
	for time.Now().Before(deadline) {
		if _, stopped := h.channel.typingCounts(); stopped >= started {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	started, stopped := h.channel.typingCounts()
	if stopped < started {
		t.Fatalf("typing indicator left running: started=%d stopped=%d", started, stopped)
	}

	// The stale "Thinking…" placeholder must be consumed by the acknowledgement
	// rather than stranded until its TTL.
	edits, _ := h.channel.snapshot()
	sawPlaceholderEdit := false
	for _, e := range edits {
		if e.messageID == "ph-A" {
			sawPlaceholderEdit = true
		}
	}
	if !sawPlaceholderEdit {
		t.Fatalf("cancellation left the Thinking placeholder unfinalized; edits=%+v", edits)
	}
}

// 7. Cancellation must leave the conversation usable.
func TestTelegramMessageAfterCancellationRunsNormally(t *testing.T) {
	h := newCancellationHarness(t, "answer")
	manager := h.manager
	manager.RecordPlaceholderForLifecycle("telegram", "chat", "A", "ph-A")

	h.send(t, "A", "long running request")
	h.waitForTurnStart(t)
	h.send(t, "S", "/stop")
	waitForDeliveryCount(t, h.channel, 1)

	h.provider.releaseAll()
	manager.RecordPlaceholderForLifecycle("telegram", "chat", "N", "ph-N")
	h.send(t, "N", "next question")

	deadline := time.Now().Add(cancellationWaitBudget)
	for time.Now().Before(deadline) {
		edits, _ := h.channel.snapshot()
		for _, e := range edits {
			if e.messageID == "ph-N" && e.content == "answer" {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	edits, sends := h.channel.snapshot()
	t.Fatalf("next message after cancellation never completed; edits=%+v sends=%v", edits, sends)
}

// 8. Repeated /stop is safe; the second one simply reports nothing to stop.
func TestTelegramDuplicateStopIsIdempotent(t *testing.T) {
	h := newCancellationHarness(t, "never")
	manager := h.manager
	manager.RecordPlaceholderForLifecycle("telegram", "chat", "A", "ph-A")

	h.send(t, "A", "long running request")
	h.waitForTurnStart(t)

	h.send(t, "S1", "/stop")
	waitForDeliveryCount(t, h.channel, 1)
	h.send(t, "S2", "/stop")
	waitForDeliveryCount(t, h.channel, 2)

	edits, sends := h.channel.snapshot()
	if len(edits)+len(sends) < 2 {
		t.Fatalf("second /stop produced no reply; edits=%+v sends=%v", edits, sends)
	}
}

// 9. /stop on an idle conversation is harmless and answers plainly.
func TestTelegramStopWithNoActiveTurnIsHarmless(t *testing.T) {
	h := newCancellationHarness(t, "answer")
	h.provider.releaseAll()

	h.send(t, "S", "/stop")
	sends := waitForSends(t, h.channel, 1)
	if !strings.Contains(strings.ToLower(sends[0]), "no active task") &&
		!strings.Contains(strings.ToLower(sends[0]), "stopped") {
		t.Fatalf("unexpected idle /stop reply: %q", sends[0])
	}
}

// 10. The 45s Telegram Bot API HTTP timeout must never bound an agent turn.
func TestTelegramHTTPTimeoutIsNotAnAgentTurnTimeout(t *testing.T) {
	cfg := config.DefaultConfig()
	if got := cfg.Agents.Defaults.MaxToolIterations; got == 0 {
		t.Fatal("expected a tool iteration bound to exist")
	}
	h := newCancellationHarness(t, "slow answer")
	manager := h.manager
	manager.RecordPlaceholderForLifecycle("telegram", "chat", "A", "ph-A")

	h.send(t, "A", "slow request")
	h.waitForTurnStart(t)

	// Hold the turn open well past any per-HTTP-call budget, then complete it.
	time.Sleep(300 * time.Millisecond)
	h.provider.releaseAll()

	deadline := time.Now().Add(cancellationWaitBudget)
	for time.Now().Before(deadline) {
		edits, _ := h.channel.snapshot()
		for _, e := range edits {
			if e.messageID == "ph-A" && e.content == "slow answer" {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	edits, sends := h.channel.snapshot()
	t.Fatalf("a turn held open past the HTTP budget did not complete; edits=%+v sends=%v", edits, sends)
}

// harnessWorkspace gives the loop a scratch directory that is not torn down
// under it.
//
// t.TempDir removes its directory as a cleanup and fails the test if anything
// is still there. A turn released during teardown finishes and writes its
// session, which races that removal — so the suite went red on a filesystem
// detail while the behaviour under test had already passed. Removal here is
// best-effort for the same reason.
func harnessWorkspace(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "pocketclaw-telegram-harness-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
