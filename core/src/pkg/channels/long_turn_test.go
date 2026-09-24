package channels

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// windowedEditor is a channel whose edits are silent, like Telegram.
type windowedEditor struct {
	mockMessageEditor
	window    time.Duration
	deleted   []string
	deleteErr error
	// events orders every platform call: "send:<text>", "edit:<id>:<text>"
	// and "delete:<id>".
	events []string
}

func (w *windowedEditor) FinalEditWindow() time.Duration { return w.window }

func (w *windowedEditor) DeleteMessage(_ context.Context, _ string, messageID string) error {
	w.events = append(w.events, "delete:"+messageID)
	if w.deleteErr != nil {
		return w.deleteErr
	}
	w.deleted = append(w.deleted, messageID)
	return nil
}

func newWindowedEditor(t *testing.T, edits *int) *windowedEditor {
	t.Helper()
	return &windowedEditor{
		mockMessageEditor: mockMessageEditor{
			editFn: func(context.Context, string, string, string) error {
				*edits++
				return nil
			},
		},
		window: LongTurnEditWindow,
	}
}

// newRecordingWindowedEditor also records sends and edits in events; send
// decides each Send attempt's result.
func newRecordingWindowedEditor(send func(attempt int) error) *windowedEditor {
	w := &windowedEditor{window: LongTurnEditWindow}
	attempts := 0
	w.sendFn = func(_ context.Context, msg bus.OutboundMessage) error {
		attempts++
		w.events = append(w.events, "send:"+msg.Content)
		return send(attempts)
	}
	w.editFn = func(_ context.Context, _, messageID, content string) error {
		w.events = append(w.events, "edit:"+messageID+":"+content)
		return nil
	}
	return w
}

func immediateWorker(ch Channel) *channelWorker {
	return &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}
}

func sendSucceeds(int) error { return nil }

func backdatePlaceholders(m *Manager, age time.Duration) {
	m.placeholders.Range(func(key, value any) bool {
		entry := value.(placeholderEntry)
		entry.createdAt = time.Now().Add(-age)
		m.placeholders.Store(key, entry)
		return true
	})
}

func TestPreSend_LongTurnAnswerFallsThroughToAFreshSend(t *testing.T) {
	m := newTestManager()
	var edits int
	ch := newWindowedEditor(t, &edits)
	m.RecordPlaceholder("test", "123", "456")
	backdatePlaceholders(m, LongTurnEditWindow+time.Second)

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "answer"})
	ids, handled := m.preSend(context.Background(), "test", msg, ch)

	if handled || ids != nil {
		t.Fatalf("preSend handled the answer (%v); it must fall through to a fresh Send", ids)
	}
	if edits != 0 || len(ch.deleted) != 0 {
		t.Fatalf("edits = %d, deleted = %v; the placeholder must wait for the send", edits, ch.deleted)
	}
}

func TestLongTurn_AnswerIsSentBeforeThePlaceholderIsDeleted(t *testing.T) {
	m := newTestManager()
	ch := newRecordingWindowedEditor(sendSucceeds)
	m.RecordPlaceholder("test", "123", "456")
	backdatePlaceholders(m, LongTurnEditWindow+time.Second)

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "answer"})
	if _, ok := m.sendWithRetry(context.Background(), "test", immediateWorker(ch), msg); !ok {
		t.Fatal("the answer was not delivered")
	}
	if got := strings.Join(ch.events, " | "); got != "send:answer | delete:456" {
		t.Fatalf("events = %q; want the fresh send, then the delete", got)
	}
}

// Before, the placeholder was deleted first: a failed fresh send then left the
// owner with neither the status message nor the answer.
func TestLongTurn_FailedFreshSendEditsTheAnswerIntoThePlaceholder(t *testing.T) {
	m := newTestManager()
	ch := newRecordingWindowedEditor(func(int) error { return fmt.Errorf("chat gone: %w", ErrSendFailed) })
	m.RecordPlaceholder("test", "123", "456")
	backdatePlaceholders(m, LongTurnEditWindow+time.Second)

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "answer"})
	ids, ok := m.sendWithRetry(context.Background(), "test", immediateWorker(ch), msg)
	if !ok || len(ids) != 1 || ids[0] != "456" {
		t.Fatalf("ids = %v, ok = %v; the answer must land in the placeholder", ids, ok)
	}
	if got := strings.Join(ch.events, " | "); got != "send:answer | edit:456:answer" {
		t.Fatalf("events = %q; want one failed send, then the answer edited in", got)
	}
}

// A placeholder that cannot be deleted must not stay "Thinking…" under a
// delivered answer; the shared fallback replaces it with the answer.
func TestLongTurn_UndeletablePlaceholderStopsSayingThinking(t *testing.T) {
	m := newTestManager()
	ch := newRecordingWindowedEditor(sendSucceeds)
	ch.deleteErr = errors.New("message can't be deleted")
	m.RecordPlaceholder("test", "123", "456")
	backdatePlaceholders(m, LongTurnEditWindow+time.Second)

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "answer"})
	if _, ok := m.sendWithRetry(context.Background(), "test", immediateWorker(ch), msg); !ok {
		t.Fatal("the answer was not delivered")
	}
	if got := strings.Join(ch.events, " | "); got != "send:answer | delete:456 | edit:456:answer" {
		t.Fatalf("events = %q", got)
	}
}

func TestLongTurn_TemporaryFailureSendsOneAnswerAndDeletesOnce(t *testing.T) {
	m := newTestManager()
	ch := newRecordingWindowedEditor(func(attempt int) error {
		if attempt == 1 {
			return fmt.Errorf("timeout: %w", ErrTemporary)
		}
		return nil
	})
	m.RecordPlaceholder("test", "123", "456")
	backdatePlaceholders(m, LongTurnEditWindow+time.Second)

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "answer"})
	if _, ok := m.sendWithRetry(context.Background(), "test", immediateWorker(ch), msg); !ok {
		t.Fatal("the answer was not delivered")
	}
	// A later message in the same chat finds nothing left to retire.
	next := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "next"})
	m.sendWithRetry(context.Background(), "test", immediateWorker(ch), next)

	want := "send:answer | send:answer | delete:456 | send:next"
	if got := strings.Join(ch.events, " | "); got != want {
		t.Fatalf("events = %q, want %q", got, want)
	}
}

func TestLongTurn_NoStatusMessageMeansOnePlainSend(t *testing.T) {
	m := newTestManager()
	ch := newRecordingWindowedEditor(sendSucceeds)

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "answer"})
	if _, ok := m.sendWithRetry(context.Background(), "test", immediateWorker(ch), msg); !ok {
		t.Fatal("the answer was not delivered")
	}
	if got := strings.Join(ch.events, " | "); got != "send:answer" {
		t.Fatalf("events = %q; with no status message the answer is just sent", got)
	}
}

// A message queued behind a long turn gets its own placeholder when its own
// turn starts; that one is young, so its answer edits it, and the long turn's
// placeholder is neither touched by it nor edited by the long answer.
func TestLongTurn_QueuedTurnEditsItsOwnYoungPlaceholder(t *testing.T) {
	m := newTestManager()
	ch := newRecordingWindowedEditor(sendSucceeds)
	m.RecordPlaceholderForLifecycle("test", "123", "lc-long", "456")
	backdatePlaceholders(m, LongTurnEditWindow+time.Second)
	m.RecordPlaceholderForLifecycle("test", "123", "lc-queued", "789")

	answer := func(lifecycleID, content string) bus.OutboundMessage {
		msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: content})
		msg.Context.Raw = map[string]string{bus.LifecycleIDMetadataKey: lifecycleID}
		return msg
	}
	m.sendWithRetry(context.Background(), "test", immediateWorker(ch), answer("lc-long", "long answer"))
	m.sendWithRetry(context.Background(), "test", immediateWorker(ch), answer("lc-queued", "queued answer"))

	want := "send:long answer | delete:456 | edit:789:queued answer"
	if got := strings.Join(ch.events, " | "); got != want {
		t.Fatalf("events = %q, want %q", got, want)
	}
}

func TestStatusMessageTooOldToEdit_Threshold(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	windowed := &windowedEditor{window: LongTurnEditWindow}
	for _, tc := range []struct {
		name   string
		ch     Channel
		sentAt time.Time
		want   bool
	}{
		{"just below the window", windowed, now.Add(-LongTurnEditWindow + time.Nanosecond), false},
		{"exactly the window", windowed, now.Add(-LongTurnEditWindow), true},
		{"above the window", windowed, now.Add(-LongTurnEditWindow - time.Second), true},
		{"unknown send time", windowed, time.Time{}, false},
		{"channel without a window", &mockMessageEditor{}, now.Add(-time.Hour), false},
	} {
		if got := statusMessageTooOldToEdit(tc.ch, tc.sentAt, now); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestPreSend_ShortTurnAnswerStillEditsThePlaceholder(t *testing.T) {
	m := newTestManager()
	var edits int
	ch := newWindowedEditor(t, &edits)
	m.RecordPlaceholder("test", "123", "456")

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "answer"})
	if _, handled := m.preSend(context.Background(), "test", msg, ch); !handled {
		t.Fatal("a short turn must keep editing its placeholder")
	}
	if edits != 1 || len(ch.deleted) != 0 {
		t.Fatalf("edits = %d, deleted = %v; want one edit and no delete", edits, ch.deleted)
	}
}

func TestPreSend_OldPlaceholderStillBecomesToolProgress(t *testing.T) {
	// A status-to-status edit is not an answer; nothing is gained by a new
	// message, which would only add noise.
	m := newTestManager()
	var edits int
	ch := newWindowedEditor(t, &edits)
	m.RecordPlaceholder("test", "123", "456")
	backdatePlaceholders(m, LongTurnEditWindow+time.Second)

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "🔧 `exec`",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw:     map[string]string{"message_kind": "tool_feedback"},
		},
	})
	if _, handled := m.preSend(context.Background(), "test", msg, ch); !handled {
		t.Fatal("tool progress must still be edited into the placeholder")
	}
	if edits != 1 || len(ch.deleted) != 0 {
		t.Fatalf("edits = %d, deleted = %v; want one edit and no delete", edits, ch.deleted)
	}
}

func TestPreSend_ChannelsWithoutAWindowKeepEditing(t *testing.T) {
	m := newTestManager()
	var edits int
	ch := &mockMessageEditor{
		editFn: func(context.Context, string, string, string) error {
			edits++
			return nil
		},
	}
	m.RecordPlaceholder("test", "123", "456")
	backdatePlaceholders(m, time.Hour)

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "answer"})
	if _, handled := m.preSend(context.Background(), "test", msg, ch); !handled || edits != 1 {
		t.Fatalf("handled = %v, edits = %d; a channel without a window keeps editing", handled, edits)
	}
}

func TestToolFeedbackAnimator_AgeSurvivesProgressUpdates(t *testing.T) {
	clock := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	a := NewToolFeedbackAnimator(func(context.Context, string, string, string) error { return nil })
	a.now = func() time.Time { return clock }
	defer a.StopAll()

	a.Record("chat", "1", "🔧 `read_file`")
	clock = clock.Add(90 * time.Second)
	if _, _, err := a.Update(context.Background(), "chat", "🔧 `exec`"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	clock = clock.Add(60 * time.Second)

	age, ok := a.Age("chat")
	if !ok || age != 150*time.Second {
		t.Fatalf("Age = %v, %v; want 150s measured from the first send", age, ok)
	}

	// A different message is a different send.
	a.Record("chat", "2", "🔧 `grep`")
	if age, ok := a.Age("chat"); !ok || age != 0 {
		t.Fatalf("Age after a new message = %v, %v; want 0", age, ok)
	}
}
