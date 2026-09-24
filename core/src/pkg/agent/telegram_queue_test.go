package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/common"
)

type sentQueueNotice struct {
	lifecycleID string
	replyTo     string
	text        string
	id          string
}

// queueNoticeRecordingManager is a channel manager that supports queue
// notices and records them.
type queueNoticeRecordingManager struct {
	placeholderRecordingManager

	noticeMu sync.Mutex
	notices  []sentQueueNotice
	deleted  []string
	replies  []string
	// events orders deletes against replies: "delete:<id>" and "reply:<text>".
	events []string
	// hold, when set, blocks SendQueueNotice until it is closed.
	hold chan struct{}
}

func (m *queueNoticeRecordingManager) SendQueueNotice(
	ctx context.Context, channel, chatID, replyToMessageID, text string,
) string {
	if m.hold != nil {
		<-m.hold
	}
	m.noticeMu.Lock()
	defer m.noticeMu.Unlock()
	id := "notice-" + replyToMessageID
	m.notices = append(m.notices, sentQueueNotice{
		replyTo: replyToMessageID,
		text:    text,
		id:      id,
	})
	return id
}

// SendMessage records final replies; with a channel manager present the agent
// delivers through it rather than the bus.
func (m *queueNoticeRecordingManager) SendMessage(_ context.Context, msg bus.OutboundMessage) error {
	m.noticeMu.Lock()
	defer m.noticeMu.Unlock()
	m.replies = append(m.replies, msg.Content)
	m.events = append(m.events, "reply:"+msg.Content)
	return nil
}

func (m *queueNoticeRecordingManager) snapshotReplies() []string {
	m.noticeMu.Lock()
	defer m.noticeMu.Unlock()
	return append([]string(nil), m.replies...)
}

func (m *queueNoticeRecordingManager) DeleteQueueNotice(_ context.Context, _, _, messageID string) {
	m.noticeMu.Lock()
	defer m.noticeMu.Unlock()
	m.deleted = append(m.deleted, messageID)
	m.events = append(m.events, "delete:"+messageID)
}

func (m *queueNoticeRecordingManager) snapshotEvents() []string {
	m.noticeMu.Lock()
	defer m.noticeMu.Unlock()
	return append([]string(nil), m.events...)
}

func (m *queueNoticeRecordingManager) snapshotNotices() ([]sentQueueNotice, []string) {
	m.noticeMu.Lock()
	defer m.noticeMu.Unlock()
	return append([]sentQueueNotice(nil), m.notices...), append([]string(nil), m.deleted...)
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestQueuedTelegramMessagesAreToldTheirPosition(t *testing.T) {
	cm := &queueNoticeRecordingManager{}
	al := &AgentLoop{channelManager: cm}
	const sessionKey = "session-q"

	if _, claimed, _, _ := al.claimSessionMailboxPosition(sessionKey, telegramMessage("lc-1", "first")); !claimed {
		t.Fatal("first message did not claim the session")
	}
	for i, id := range []string{"lc-2", "lc-3", "lc-4"} {
		msg := telegramMessage(id, "more")
		_, claimed, queued, ahead := al.claimSessionMailboxPosition(sessionKey, msg)
		if claimed || !queued {
			t.Fatalf("%s: claimed=%v queued=%v", id, claimed, queued)
		}
		if ahead != i+1 {
			t.Fatalf("%s has %d ahead, want %d", id, ahead, i+1)
		}
		al.announceQueuedMessage(context.Background(), msg, ahead)
	}

	waitFor(t, "three queue notices", func() bool {
		notices, _ := cm.snapshotNotices()
		return len(notices) == 3
	})
	notices, _ := cm.snapshotNotices()
	want := map[string]string{
		"lc-2": "Queued — 1 message ahead.",
		"lc-3": "Queued — 2 messages ahead.",
		"lc-4": "Queued — 3 messages ahead.",
	}
	for _, n := range notices {
		if !strings.HasPrefix(n.text, want[n.replyTo]) {
			t.Fatalf("notice replying to %s says %q, want prefix %q", n.replyTo, n.text, want[n.replyTo])
		}
	}
}

// The notice goes as soon as the message is queued; the message may start
// before the Bot API has answered. Retiring must wait for the send, or the
// notice lands after the delete and stays in the chat.
func TestQueueNoticeIsDeletedEvenWhenTheMessageStartsFirst(t *testing.T) {
	cm := &queueNoticeRecordingManager{hold: make(chan struct{})}
	al := &AgentLoop{channelManager: cm}
	msg := telegramMessage("lc-race", "hi")

	al.announceQueuedMessage(context.Background(), msg, 1)
	retired := make(chan struct{})
	go func() {
		al.retireQueueNotice(context.Background(), msg)
		close(retired)
	}()

	select {
	case <-retired:
		t.Fatal("retired the notice before it had been sent")
	case <-time.After(100 * time.Millisecond):
	}
	close(cm.hold)
	<-retired

	_, deleted := cm.snapshotNotices()
	if len(deleted) != 1 || deleted[0] != "notice-lc-race" {
		t.Fatalf("deleted %v, want the one notice", deleted)
	}
}

func TestQueueNoticeNeedsTheCapability(t *testing.T) {
	al := &AgentLoop{channelManager: &placeholderRecordingManager{}}
	msg := telegramMessage("lc-plain", "hi")
	al.announceQueuedMessage(context.Background(), msg, 2)
	al.retireQueueNotice(context.Background(), msg)
	if _, ok := al.queueNotices.Load("lc-plain"); ok {
		t.Fatal("a manager without the capability still recorded a notice")
	}
}

// burstProvider holds the first turn open until released, then answers every
// request with the user message it was asked about. failFirst and panicFirst
// make that first turn fail instead; failText fails the turn for that one
// user message.
type burstProvider struct {
	mu         sync.Mutex
	calls      int
	entered    chan struct{}
	release    chan struct{}
	failFirst  bool
	panicFirst bool
	failText   string
}

func (p *burstProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	p.calls++
	first := p.calls == 1
	p.mu.Unlock()

	if first {
		close(p.entered)
		select {
		case <-p.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if p.panicFirst {
			panic("provider exploded")
		}
		if p.failFirst {
			return nil, &common.HTTPError{StatusCode: 401, BodyPreview: "invalid api key"}
		}
	}
	last := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			last = messages[i].Content
			break
		}
	}
	if p.failText != "" && last == p.failText {
		return nil, &common.HTTPError{StatusCode: 401, BodyPreview: "invalid api key"}
	}
	return &providers.LLMResponse{Content: "answer: " + last, FinishReason: "stop"}, nil
}

func (p *burstProvider) GetDefaultModel() string { return "test-model" }

// runTelegramBurst sends one message that runs long and three more while it
// runs, and returns every final reply in the order it was published.
func runTelegramBurst(t *testing.T, provider *burstProvider) ([]string, *queueNoticeRecordingManager) {
	t.Helper()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         1024,
				MaxToolIterations: 1,
				MaxParallelTurns:  1,
			},
		},
	}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)
	cm := &queueNoticeRecordingManager{}
	al.channelManager = cm

	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		_ = al.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-runDone
		al.Close()
	})

	publish := func(id, text string) {
		if err := msgBus.PublishInbound(ctx, telegramMessage(id, text)); err != nil {
			t.Fatalf("PublishInbound(%s): %v", id, err)
		}
	}
	publish("lc-1", "research this repository")
	select {
	case <-provider.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the first turn never started")
	}
	publish("lc-2", "link two")
	publish("lc-3", "link three")
	publish("lc-4", "link four")

	waitFor(t, "a notice for every queued message", func() bool {
		notices, _ := cm.snapshotNotices()
		return len(notices) == 3
	})
	close(provider.release)

	waitFor(t, "four replies", func() bool { return len(cm.snapshotReplies()) >= 4 })
	// Anything published after the fourth reply would be a duplicate.
	time.Sleep(200 * time.Millisecond)
	return cm.snapshotReplies(), cm
}

func assertQueuedRepliesInOrder(t *testing.T, replies []string) {
	t.Helper()
	want := []string{"answer: link two", "answer: link three", "answer: link four"}
	if len(replies) != 4 {
		t.Fatalf("got %d replies, want exactly 4 (no drops, no duplicates): %q", len(replies), replies)
	}
	for i, w := range want {
		if replies[i+1] != w {
			t.Fatalf("reply %d = %q, want %q; order or content changed: %q", i+2, replies[i+1], w, replies)
		}
	}
}

func assertEveryNoticeRetired(t *testing.T, cm *queueNoticeRecordingManager) {
	t.Helper()
	waitFor(t, "every notice deleted", func() bool {
		_, deleted := cm.snapshotNotices()
		return len(deleted) == 3
	})
}

func TestTelegramBurstDuringALongTurnIsAnsweredInOrder(t *testing.T) {
	provider := &burstProvider{entered: make(chan struct{}), release: make(chan struct{})}
	replies, cm := runTelegramBurst(t, provider)
	if replies[0] != "answer: research this repository" {
		t.Fatalf("first reply = %q", replies[0])
	}
	assertQueuedRepliesInOrder(t, replies)
	assertEveryNoticeRetired(t, cm)
}

func TestFailedTurnDoesNotStrandTheMessagesQueuedBehindIt(t *testing.T) {
	provider := &burstProvider{
		entered: make(chan struct{}), release: make(chan struct{}), failFirst: true,
	}
	replies, cm := runTelegramBurst(t, provider)
	if strings.HasPrefix(replies[0], "answer:") {
		t.Fatalf("the failing turn produced an answer: %q", replies[0])
	}
	assertQueuedRepliesInOrder(t, replies)
	assertEveryNoticeRetired(t, cm)
}

// Before, a panic unwound the worker and released its mailbox with the three
// queued messages still in it: they were never answered.
func TestPanickedTurnDoesNotDropTheMessagesQueuedBehindIt(t *testing.T) {
	provider := &burstProvider{
		entered: make(chan struct{}), release: make(chan struct{}), panicFirst: true,
	}
	replies, cm := runTelegramBurst(t, provider)
	if !strings.Contains(replies[0], "Something went wrong") {
		t.Fatalf("the sender of the panicked message was not told: %q", replies[0])
	}
	assertQueuedRepliesInOrder(t, replies)
	assertEveryNoticeRetired(t, cm)
}

func TestMiddleTurnFailureDoesNotStrandTheMessageQueuedBehindIt(t *testing.T) {
	provider := &burstProvider{
		entered: make(chan struct{}), release: make(chan struct{}), failText: "link three",
	}
	replies, cm := runTelegramBurst(t, provider)
	if len(replies) != 4 {
		t.Fatalf("got %d replies, want exactly 4: %q", len(replies), replies)
	}
	if replies[1] != "answer: link two" || replies[3] != "answer: link four" {
		t.Fatalf("the turns around the failure were not answered in order: %q", replies)
	}
	if strings.HasPrefix(replies[2], "answer:") {
		t.Fatalf("the failing middle turn produced an answer: %q", replies[2])
	}
	assertEveryNoticeRetired(t, cm)
}

// Each queued message gets exactly one notice replying to it, that notice is
// the one deleted, it is deleted before that message is answered, and each
// turn's "Thinking…" placeholder carries the lifecycle of the message running.
func TestBurstKeepsEveryNoticeAndPlaceholderWithItsOwnMessage(t *testing.T) {
	provider := &burstProvider{entered: make(chan struct{}), release: make(chan struct{})}
	_, cm := runTelegramBurst(t, provider)
	assertEveryNoticeRetired(t, cm)

	notices, deleted := cm.snapshotNotices()
	seen := map[string]int{}
	for _, n := range notices {
		seen[n.replyTo]++
	}
	for _, id := range []string{"lc-2", "lc-3", "lc-4"} {
		if seen[id] != 1 {
			t.Fatalf("%s got %d queue notices, want 1: %+v", id, seen[id], notices)
		}
	}
	if seen["lc-1"] != 0 {
		t.Fatal("the message that ran at once was told it was queued")
	}
	gone := map[string]int{}
	for _, id := range deleted {
		gone[id]++
	}
	for _, id := range []string{"notice-lc-2", "notice-lc-3", "notice-lc-4"} {
		if gone[id] != 1 {
			t.Fatalf("%s deleted %d times, want 1: %v", id, gone[id], deleted)
		}
	}

	events := cm.snapshotEvents()
	index := func(event string) int {
		for i, e := range events {
			if e == event {
				return i
			}
		}
		t.Fatalf("no %q in %q", event, events)
		return -1
	}
	for id, text := range map[string]string{"lc-2": "link two", "lc-3": "link three", "lc-4": "link four"} {
		if index("delete:notice-"+id) > index("reply:answer: "+text) {
			t.Fatalf("the notice for %s outlived its answer: %q", id, events)
		}
	}

	var placeholders []string
	for _, send := range cm.snapshot() {
		placeholders = append(placeholders, send.lifecycleID)
	}
	if strings.Join(placeholders, ",") != "lc-1,lc-2,lc-3,lc-4" {
		t.Fatalf("placeholders went to %v, want one per message in turn order", placeholders)
	}
}
