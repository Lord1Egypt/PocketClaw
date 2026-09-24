package channels

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// windowedEditor is a channel whose edits are silent, like Telegram.
type windowedEditor struct {
	mockMessageEditor
	window  time.Duration
	deleted []string
}

func (w *windowedEditor) FinalEditWindow() time.Duration { return w.window }

func (w *windowedEditor) DeleteMessage(_ context.Context, _ string, messageID string) error {
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

func backdatePlaceholders(m *Manager, age time.Duration) {
	m.placeholders.Range(func(key, value any) bool {
		entry := value.(placeholderEntry)
		entry.createdAt = time.Now().Add(-age)
		m.placeholders.Store(key, entry)
		return true
	})
}

func TestPreSend_LongTurnAnswerIsSentFreshAndThePlaceholderDeleted(t *testing.T) {
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
	if edits != 0 {
		t.Fatalf("the stale placeholder was edited %d time(s)", edits)
	}
	if len(ch.deleted) != 1 || ch.deleted[0] != "456" {
		t.Fatalf("deleted = %v, want the placeholder 456", ch.deleted)
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
