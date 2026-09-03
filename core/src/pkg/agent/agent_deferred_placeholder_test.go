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

	mu           sync.Mutex
	sends        []placeholderSend
	typingStarts []string
	typingStops  []string
	typingActive map[string]bool
}

type placeholderSend struct {
	channel     string
	chatID      string
	lifecycleID string
}

// StartTyping and InvokeTypingStopForLifecycle are recorded per lifecycle so a
// test can count how many typing loops a session owns at once.
func (m *placeholderRecordingManager) StartTyping(
	ctx context.Context, channel, chatID string,
) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	lifecycleID := bus.LifecycleIDFromContext(ctx)
	m.typingStarts = append(m.typingStarts, lifecycleID)
	if m.typingActive == nil {
		m.typingActive = make(map[string]bool)
	}
	m.typingActive[lifecycleID] = true
	return true
}

func (m *placeholderRecordingManager) InvokeTypingStopForLifecycle(
	channel, chatID, lifecycleID string,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.typingStops = append(m.typingStops, lifecycleID)
	delete(m.typingActive, lifecycleID)
}

func (m *placeholderRecordingManager) typingStartedFor() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.typingStarts...)
}

func (m *placeholderRecordingManager) typingStoppedFor() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.typingStops...)
}

// activeTypingLoops is the assertion the 429 burst reduces to: however deep the
// queue is, a session may own only one repeating chat-action loop at a time.
func (m *placeholderRecordingManager) activeTypingLoops() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.typingActive)
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
		if got := cm.activeTypingLoops(); got != 0 {
			t.Fatalf("typing started while the worker was waiting for a slot (%d loops)", got)
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

	started := cm.typingStartedFor()
	if len(started) != 1 || started[0] != "lc-sem" {
		t.Fatalf("typing starts = %v, want exactly [lc-sem] once the slot was held", started)
	}
	if got := cm.activeTypingLoops(); got != 1 {
		t.Fatalf("%d typing loops active during the turn, want exactly 1", got)
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

// Only the channels whose activity signals the channel layer deferred may have
// typing started here.
func TestStartDeferredTypingOnlyForIndependentLifecycleChannels(t *testing.T) {
	for _, channel := range []string{"telegram", "discord", "slack", "pico", "cli"} {
		cm := &placeholderRecordingManager{}
		al := &AgentLoop{channelManager: cm}

		msg := telegramMessage("lc-1", "hello")
		msg.Context.Channel = channel
		msg.Channel = channel

		al.startDeferredTyping(context.Background(), msg)

		want := 0
		if channel == "telegram" {
			want = 1
		}
		if got := len(cm.typingStartedFor()); got != want {
			t.Errorf("%s started %d typing loops, want %d", channel, got, want)
		}
	}
}

// Typing is deferred for audio too. The placeholder is not — it waits for
// transcription — so the two must not be gated by the same condition, and a
// voice message must get exactly one of each and no duplicate.
func TestAudioGetsTypingButDefersItsPlaceholder(t *testing.T) {
	cm := &placeholderRecordingManager{}
	al := &AgentLoop{channelManager: cm}

	msg := telegramMessage("lc-voice", "[voice]")
	al.startDeferredTyping(context.Background(), msg)
	al.sendDeferredPlaceholder(context.Background(), msg)

	if got := len(cm.typingStartedFor()); got != 1 {
		t.Errorf("audio started %d typing loops, want 1", got)
	}
	if got := cm.count(); got != 0 {
		t.Errorf("audio sent %d placeholders here, want 0: transcription sends it", got)
	}
}

// The structural cause of the 429 burst: however deep the queue is, the session
// owns exactly one repeating chat-action loop, and it belongs to the request
// that is running.
func TestQueuedRequestsDoNotMultiplyTypingLoops(t *testing.T) {
	cm := &placeholderRecordingManager{}
	al := &AgentLoop{channelManager: cm}
	const sessionKey = "session-1"

	first := telegramMessage("lc-1", "part one")
	owner, claimed, _ := al.claimSessionMailbox(sessionKey, first)
	if !claimed {
		t.Fatal("first message did not claim the session")
	}
	queuedIDs := []string{"lc-2", "lc-3", "lc-4", "lc-5", "lc-6", "lc-7", "lc-8"}
	for _, lifecycleID := range queuedIDs {
		al.claimSessionMailbox(sessionKey, telegramMessage(lifecycleID, "part"))
	}

	// Seven requests are waiting. None of them may be signalling activity.
	if got := cm.activeTypingLoops(); got != 0 {
		t.Fatalf("%d typing loops active before any turn began, want 0", got)
	}

	current := first
	for i := 0; ; i++ {
		al.startDeferredTyping(context.Background(), current)

		if got := cm.activeTypingLoops(); got != 1 {
			t.Fatalf("while running %s there were %d active typing loops, want exactly 1",
				bus.InboundLifecycleID(&current.Context), got)
		}

		al.stopDeferredTyping(current)
		if got := cm.activeTypingLoops(); got != 0 {
			t.Fatalf("after %s finished there were %d active typing loops, want 0",
				bus.InboundLifecycleID(&current.Context), got)
		}

		next, ok := al.takeNextSessionMessage(sessionKey, owner)
		if !ok {
			if i != len(queuedIDs) {
				t.Fatalf("ran %d requests, want %d", i+1, len(queuedIDs)+1)
			}
			break
		}
		current = next
	}

	started := cm.typingStartedFor()
	wantOrder := append([]string{"lc-1"}, queuedIDs...)
	if len(started) != len(wantOrder) {
		t.Fatalf("started %d typing loops for %d requests", len(started), len(wantOrder))
	}
	for i, want := range wantOrder {
		if started[i] != want {
			t.Fatalf("typing loop %d belonged to %s, want %s: FIFO order changed",
				i, started[i], want)
		}
	}
	if len(cm.typingStoppedFor()) != len(wantOrder) {
		t.Fatalf("stopped %d typing loops for %d starts",
			len(cm.typingStoppedFor()), len(wantOrder))
	}
}

// The stop is deferred, so it must fire even when the turn fails or the
// context is cancelled rather than only when a response is delivered.
func TestDeferredTypingStopsOnEveryTerminalPath(t *testing.T) {
	cm := &placeholderRecordingManager{}
	al := &AgentLoop{channelManager: cm}

	msg := telegramMessage("lc-term", "hello")

	// Simulates runTurnWithDeferredActivity's own body: whatever the turn does,
	// the deferred stop runs.
	func() {
		defer al.stopDeferredTyping(msg)
		al.startDeferredTyping(context.Background(), msg)
		// turn returns an error, panics, or is cancelled — all unwind through here
	}()

	if got := cm.activeTypingLoops(); got != 0 {
		t.Fatalf("%d typing loops still active after the turn ended, want 0", got)
	}
	if got := cm.typingStoppedFor(); len(got) != 1 || got[0] != "lc-term" {
		t.Fatalf("typing stops = %v, want exactly [lc-term]", got)
	}
}

func TestDeferredTypingStopSurvivesAPanickingTurn(t *testing.T) {
	cm := &placeholderRecordingManager{}
	al := &AgentLoop{channelManager: cm}
	msg := telegramMessage("lc-panic", "hello")

	func() {
		defer func() { _ = recover() }()
		defer al.stopDeferredTyping(msg)
		al.startDeferredTyping(context.Background(), msg)
		panic("turn exploded")
	}()

	if got := cm.activeTypingLoops(); got != 0 {
		t.Fatalf("%d typing loops leaked past a panic, want 0", got)
	}
}
