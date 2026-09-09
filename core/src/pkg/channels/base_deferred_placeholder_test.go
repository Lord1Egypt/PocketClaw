package channels

import (
	"context"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// deferredPlaceholderChannel is a channel that can do everything the inbound
// path checks for: typing, reactions and placeholders. Each is counted so a
// test can assert that deferring the placeholder left the other two alone.
type deferredPlaceholderChannel struct {
	*BaseChannel

	mu                sync.Mutex
	placeholdersSent  int
	typingStarts      int
	reactionsApplied  int
	lastPlaceholderID string
}

func (c *deferredPlaceholderChannel) Send(
	context.Context, bus.OutboundMessage,
) ([]string, error) {
	return nil, nil
}
func (c *deferredPlaceholderChannel) Start(context.Context) error { return nil }
func (c *deferredPlaceholderChannel) Stop(context.Context) error  { return nil }

func (c *deferredPlaceholderChannel) SendPlaceholder(
	_ context.Context, _ string,
) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.placeholdersSent++
	c.lastPlaceholderID = "ph-1"
	return c.lastPlaceholderID, nil
}

func (c *deferredPlaceholderChannel) StartTyping(
	_ context.Context, _ string,
) (func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.typingStarts++
	return func() {}, nil
}

func (c *deferredPlaceholderChannel) ReactToMessage(
	_ context.Context, _, _ string,
) (func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reactionsApplied++
	return func() {}, nil
}

func (c *deferredPlaceholderChannel) counts() (placeholders, typing, reactions int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.placeholdersSent, c.typingStarts, c.reactionsApplied
}

// recordingPlaceholderRecorder captures what the inbound path registered.
type recordingPlaceholderRecorder struct {
	mu               sync.Mutex
	lifecycleRecords []string
	plainRecords     []string
}

func (r *recordingPlaceholderRecorder) RecordPlaceholder(_, _, placeholderID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plainRecords = append(r.plainRecords, placeholderID)
}

func (r *recordingPlaceholderRecorder) RecordTypingStop(_, _ string, _ func()) {}

func (r *recordingPlaceholderRecorder) RecordReactionUndo(_, _ string, _ func()) {}

func (r *recordingPlaceholderRecorder) RecordPlaceholderForLifecycle(
	_, _, lifecycleID, _ string,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lifecycleRecords = append(r.lifecycleRecords, lifecycleID)
}

func (r *recordingPlaceholderRecorder) RecordTypingStopForLifecycle(
	_, _, _ string, _ func(),
) {
}

func (r *recordingPlaceholderRecorder) RecordReactionUndoForLifecycle(
	_, _, _ string, _ func(),
) {
}

func (r *recordingPlaceholderRecorder) recorded() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.lifecycleRecords) + len(r.plainRecords)
}

func newDeferredPlaceholderChannel(
	t *testing.T,
	name string,
) (*deferredPlaceholderChannel, *recordingPlaceholderRecorder) {
	t.Helper()

	base := NewBaseChannel(name, nil, bus.NewMessageBus(), nil)
	ch := &deferredPlaceholderChannel{BaseChannel: base}
	base.SetOwner(ch)
	recorder := &recordingPlaceholderRecorder{}
	base.SetPlaceholderRecorder(recorder)
	return ch, recorder
}

func handleInbound(ch *deferredPlaceholderChannel, content string) {
	ch.HandleMessageWithContext(
		context.Background(),
		"chat-1",
		content,
		nil,
		bus.InboundContext{
			Channel:   ch.BaseChannel.name,
			ChatID:    "chat-1",
			ChatType:  "direct",
			SenderID:  "owner",
			MessageID: "m-1",
		},
	)
}

// A Telegram message may wait in the session FIFO behind a turn that runs for
// minutes. Sending "Thinking…" on receipt would claim the agent had started a
// request it has not dequeued, so the channel must leave it to the agent.
func TestTelegramDefersPlaceholderToTheAgent(t *testing.T) {
	ch, recorder := newDeferredPlaceholderChannel(t, "telegram")

	handleInbound(ch, "a long report, part one")

	placeholders, typing, reactions := ch.counts()
	if placeholders != 0 {
		t.Fatalf("telegram sent %d placeholders on receipt, want 0", placeholders)
	}
	if recorder.recorded() != 0 {
		t.Fatalf("recorded %d placeholders on receipt, want 0", recorder.recorded())
	}
	// The typing indicator is a repeating chat-action loop owned per received
	// message. Starting one here for a request that may wait minutes in the
	// FIFO gave a deep queue as many concurrent loops as it had entries, which
	// is what drove Telegram into rate limiting. It is deferred with the
	// placeholder.
	if typing != 0 {
		t.Errorf("typing starts = %d, want 0: Telegram defers typing to execution", typing)
	}
	// The reaction is a one-shot acknowledgement that the message arrived, not
	// a claim that work is under way, so it stays at receipt.
	if reactions != 1 {
		t.Errorf("reactions = %d, want 1: receipt acknowledgement must be preserved", reactions)
	}
}

// Every other channel merges a message arriving mid-turn into the running turn
// as steering input, so its placeholder is still correct at receipt.
func TestNonTelegramChannelStillSendsPlaceholderOnReceipt(t *testing.T) {
	for _, name := range []string{"discord", "slack", "matrix", "pocketclaw"} {
		t.Run(name, func(t *testing.T) {
			ch, recorder := newDeferredPlaceholderChannel(t, name)

			handleInbound(ch, "hello")

			placeholders, typing, reactions := ch.counts()
			if placeholders != 1 {
				t.Fatalf("%s sent %d placeholders, want 1", name, placeholders)
			}
			if recorder.recorded() != 1 {
				t.Fatalf("%s recorded %d placeholders, want 1", name, recorder.recorded())
			}
			if typing != 1 || reactions != 1 {
				t.Errorf("%s typing=%d reactions=%d, want 1 and 1: unchanged", name, typing, reactions)
			}
		})
	}
}

// Audio was already deferred, to after transcription. That must not change, and
// the deferral must not be applied twice.
func TestAudioMessageStillDefersPlaceholder(t *testing.T) {
	for _, name := range []string{"telegram", "discord"} {
		t.Run(name, func(t *testing.T) {
			ch, recorder := newDeferredPlaceholderChannel(t, name)

			handleInbound(ch, "[voice] transcribe me")

			placeholders, typing, _ := ch.counts()
			if placeholders != 0 {
				t.Fatalf("%s sent %d placeholders for audio, want 0", name, placeholders)
			}
			wantTyping := 1
			if name == "telegram" {
				wantTyping = 0
			}
			if typing != wantTyping {
				t.Fatalf("%s audio typing starts = %d, want %d", name, typing, wantTyping)
			}
			if recorder.recorded() != 0 {
				t.Fatalf("%s recorded %d placeholders for audio, want 0", name, recorder.recorded())
			}
		})
	}
}

// The channel layer and the agent must not disagree about which channels defer,
// or a queued message would either show a placeholder it never earned or run
// without one at all.
func TestChannelUsesIndependentResponseLifecycle(t *testing.T) {
	for _, name := range []string{"telegram", "Telegram", "TELEGRAM", " telegram "} {
		if !bus.ChannelUsesIndependentResponseLifecycle(name) {
			t.Errorf("%q should use an independent response lifecycle", name)
		}
	}
	for _, name := range []string{"discord", "slack", "pocketclaw", "cli", "", "telegramish"} {
		if bus.ChannelUsesIndependentResponseLifecycle(name) {
			t.Errorf("%q should not use an independent response lifecycle", name)
		}
	}
}
