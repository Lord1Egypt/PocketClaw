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

// placeholderRecordingManager records every deferred placeholder the agent
// sends, with the lifecycle ID the channel manager would have correlated it by.
type placeholderRecordingManager struct {
	recordingChannelManager

	mu    sync.Mutex
	sends []placeholderSend
}

type placeholderSend struct {
	channel     string
	chatID      string
	lifecycleID string
}

func (m *placeholderRecordingManager) SendPlaceholder(
	ctx context.Context, channel, chatID string,
) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sends = append(m.sends, placeholderSend{
		channel:     channel,
		chatID:      chatID,
		lifecycleID: bus.LifecycleIDFromContext(ctx),
	})
	return true
}

func (m *placeholderRecordingManager) snapshot() []placeholderSend {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]placeholderSend(nil), m.sends...)
}

func (m *placeholderRecordingManager) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sends)
}

func telegramMessage(lifecycleID, content string) bus.InboundMessage {
	return bus.NormalizeInboundMessage(bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:   "telegram",
			ChatID:    "chat-1",
			ChatType:  "direct",
			SenderID:  "owner",
			MessageID: lifecycleID,
			Raw:       map[string]string{bus.LifecycleIDMetadataKey: lifecycleID},
		},
		Content: content,
	})
}

// The placeholder must carry the lifecycle ID of the message actually being
// executed, or the outbound edit would look for a placeholder under a key
// nothing ever recorded and send a second message instead.
func TestSendDeferredPlaceholderCorrelatesTheExecutingMessage(t *testing.T) {
	cm := &placeholderRecordingManager{}
	al := &AgentLoop{channelManager: cm}

	msg := telegramMessage("lc-42", "part one")
	al.sendDeferredPlaceholder(context.Background(), msg)

	sends := cm.snapshot()
	if len(sends) != 1 {
		t.Fatalf("sent %d placeholders, want 1", len(sends))
	}
	if got, want := sends[0].lifecycleID, bus.InboundLifecycleID(&msg.Context); got != want {
		t.Errorf("lifecycle_id = %q, want %q", got, want)
	}
	if sends[0].channel != "telegram" || sends[0].chatID != "chat-1" {
		t.Errorf("addressed %s/%s, want telegram/chat-1", sends[0].channel, sends[0].chatID)
	}
}

// Only the channels whose placeholder the channel layer deferred may receive
// one here; every other channel already sent its own on receipt.
func TestSendDeferredPlaceholderSkipsNonDeferringChannels(t *testing.T) {
	for _, channel := range []string{"discord", "slack", "pico", "cli"} {
		t.Run(channel, func(t *testing.T) {
			cm := &placeholderRecordingManager{}
			al := &AgentLoop{channelManager: cm}

			msg := telegramMessage("lc-1", "hello")
			msg.Context.Channel = channel
			msg.Channel = channel

			al.sendDeferredPlaceholder(context.Background(), msg)

			if cm.count() != 0 {
				t.Fatalf("%s received %d placeholders, want 0", channel, cm.count())
			}
		})
	}
}

// prepareInboundMessageForAgent sends the audio placeholder itself once
// transcription has produced real text. Sending one here as well would leave
// two "Thinking…" messages in the chat with only the second one recorded, and
// the first could never be edited away.
func TestSendDeferredPlaceholderSkipsAudioToAvoidDoubleSend(t *testing.T) {
	for _, content := range []string{"[voice]", "[audio: note.ogg]", "prefix [voice] suffix"} {
		cm := &placeholderRecordingManager{}
		al := &AgentLoop{channelManager: cm}

		al.sendDeferredPlaceholder(context.Background(), telegramMessage("lc-1", content))

		if cm.count() != 0 {
			t.Errorf("content %q sent %d placeholders, want 0", content, cm.count())
		}
	}
}

func TestSendDeferredPlaceholderToleratesNoChannelManager(t *testing.T) {
	al := &AgentLoop{}
	al.sendDeferredPlaceholder(context.Background(), telegramMessage("lc-1", "hi"))
}

// Queueing must stay exactly as it was: a Telegram message arriving on a busy
// session is retained whole, in arrival order, and drained one at a time.
func TestSessionMailboxKeepsTelegramMessagesInFIFOOrder(t *testing.T) {
	al := &AgentLoop{}
	const sessionKey = "session-1"

	first := telegramMessage("lc-1", "part one")
	owner, claimed, queued := al.claimSessionMailbox(sessionKey, first)
	if !claimed || queued {
		t.Fatalf("first message: claimed=%v queued=%v, want true and false", claimed, queued)
	}

	for i, lifecycleID := range []string{"lc-2", "lc-3", "lc-4", "lc-5", "lc-6", "lc-7"} {
		_, claimed, queued := al.claimSessionMailbox(sessionKey, telegramMessage(lifecycleID, "part"))
		if claimed || !queued {
			t.Fatalf("message %d: claimed=%v queued=%v, want false and true", i+2, claimed, queued)
		}
	}

	al.sessionMailboxMu.Lock()
	depth := len(al.sessionMailboxes[sessionKey].independent)
	al.sessionMailboxMu.Unlock()
	if depth != 6 {
		t.Fatalf("queue depth = %d, want 6", depth)
	}

	want := []string{"lc-2", "lc-3", "lc-4", "lc-5", "lc-6", "lc-7"}
	for _, wantID := range want {
		next, ok := al.takeNextSessionMessage(sessionKey, owner)
		if !ok {
			t.Fatalf("mailbox drained early, expected %s", wantID)
		}
		if got := bus.InboundLifecycleID(&next.Context); got != wantID {
			t.Fatalf("dequeued %s, want %s: FIFO order changed", got, wantID)
		}
	}
	if _, ok := al.takeNextSessionMessage(sessionKey, owner); ok {
		t.Fatal("mailbox returned a message after the queue was drained")
	}
}

// The whole point of the change: a message sitting in the FIFO has no
// persistent "Thinking…" placeholder. Only the message the worker is about to
// run gets one, and it gets exactly one.
func TestOnlyTheExecutingMessageGetsAPlaceholder(t *testing.T) {
	cm := &placeholderRecordingManager{}
	al := &AgentLoop{channelManager: cm}
	const sessionKey = "session-1"

	first := telegramMessage("lc-1", "part one")
	owner, claimed, _ := al.claimSessionMailbox(sessionKey, first)
	if !claimed {
		t.Fatal("first message did not claim the session")
	}
	queuedIDs := []string{"lc-2", "lc-3", "lc-4", "lc-5", "lc-6"}
	for _, lifecycleID := range queuedIDs {
		al.claimSessionMailbox(sessionKey, telegramMessage(lifecycleID, "part"))
	}

	// The worker is about to run the first message and nothing else.
	al.sendDeferredPlaceholder(context.Background(), first)

	sends := cm.snapshot()
	if len(sends) != 1 {
		t.Fatalf("sent %d placeholders while five messages waited, want 1", len(sends))
	}
	if sends[0].lifecycleID != "lc-1" {
		t.Fatalf("placeholder went to %s, want the executing message lc-1", sends[0].lifecycleID)
	}

	// Draining the queue gives each message its own placeholder, in order, and
	// only as it becomes current.
	current := first
	for i, wantID := range queuedIDs {
		next, ok := al.takeNextSessionMessage(sessionKey, owner)
		if !ok {
			t.Fatalf("mailbox drained early at %s", wantID)
		}
		current = next
		al.sendDeferredPlaceholder(context.Background(), current)

		sends = cm.snapshot()
		if len(sends) != i+2 {
			t.Fatalf("after dequeuing %s there were %d placeholders, want %d",
				wantID, len(sends), i+2)
		}
		if sends[len(sends)-1].lifecycleID != wantID {
			t.Fatalf("placeholder %d went to %s, want %s",
				len(sends), sends[len(sends)-1].lifecycleID, wantID)
		}
	}

	if len(cm.snapshot()) != 6 {
		t.Fatalf("six requests produced %d placeholders, want exactly one each", len(cm.snapshot()))
	}
}

var _ channels.Channel = (*fakeChannel)(nil)

// blockingProvider holds a turn open until the test releases it, so a second
// message really does have to wait.
type blockingProvider struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (p *blockingProvider) Chat(
	ctx context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.once.Do(func() { close(p.entered) })
	select {
	case <-p.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &providers.LLMResponse{Content: "done"}, nil
}

func (p *blockingProvider) GetDefaultModel() string { return "test-model" }

// A claimed message still has to acquire the global worker semaphore, which is
// shared across sessions. Creating the placeholder at claim time would announce
// "Thinking…" while the worker was blocked on a slot it had not been given, so
// it must not appear until the slot is actually held.
func TestPlaceholderWaitsForTheWorkerSemaphore(t *testing.T) {
	workspace := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "test-model",
				MaxTokens:         1024,
				MaxToolIterations: 1,
				MaxParallelTurns:  1,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &blockingProvider{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	al := NewAgentLoop(cfg, msgBus, provider)
	cm := &placeholderRecordingManager{}
	al.channelManager = cm

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		_ = al.Run(ctx)
	}()
	t.Cleanup(func() {
		close(provider.release)
		cancel()
		<-runDone
	})

	// Occupy the only worker slot, exactly as a turn on another session would.
	select {
	case al.workerSem <- struct{}{}:
	case <-time.After(2 * time.Second):
		t.Fatal("could not occupy the worker semaphore")
	}

	if err := msgBus.PublishInbound(ctx, telegramMessage("lc-sem", "hello")); err != nil {
		t.Fatalf("PublishInbound() error = %v", err)
	}

	// Give the loop room to claim the session and block on the semaphore. No
	// placeholder may appear during that window.
	deadline := time.Now().Add(750 * time.Millisecond)
	for time.Now().Before(deadline) {
		if got := cm.count(); got != 0 {
			t.Fatalf("placeholder sent while the worker was waiting for a slot (%d sends)", got)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Hand the slot over; the worker may now start, and only now may it speak.
	<-al.workerSem

	select {
	case <-provider.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the turn never started after the worker slot was released")
	}

	sends := cm.snapshot()
	if len(sends) != 1 {
		t.Fatalf("after acquiring the slot there were %d placeholders, want 1", len(sends))
	}
	if sends[0].lifecycleID != "lc-sem" {
		t.Fatalf("placeholder lifecycle = %q, want lc-sem", sends[0].lifecycleID)
	}
}

// A worker whose context is cancelled while it waits for a slot returns before
// the turn loop, so it never speaks. Deferring the placeholder made this the
// normal case rather than something needing cleanup: before the change the
// message already carried a placeholder by this point, and there was nothing
// left to run that could edit it away.
func TestCancelledWorkerSendsNoPlaceholder(t *testing.T) {
	workspace := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "test-model",
				MaxTokens:         1024,
				MaxToolIterations: 1,
				MaxParallelTurns:  1,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &blockingProvider{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	al := NewAgentLoop(cfg, msgBus, provider)
	cm := &placeholderRecordingManager{}
	al.channelManager = cm

	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		_ = al.Run(ctx)
	}()
	t.Cleanup(func() {
		close(provider.release)
		cancel()
		<-runDone
	})

	// Hold the only slot so the worker must wait, then cancel underneath it.
	select {
	case al.workerSem <- struct{}{}:
	case <-time.After(2 * time.Second):
		t.Fatal("could not occupy the worker semaphore")
	}

	if err := msgBus.PublishInbound(ctx, telegramMessage("lc-cancel", "hello")); err != nil {
		t.Fatalf("PublishInbound() error = %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	cancel()

	time.Sleep(300 * time.Millisecond)
	if got := cm.count(); got != 0 {
		t.Fatalf("cancelled worker sent %d placeholders, want 0", got)
	}
}
