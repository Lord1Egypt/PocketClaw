package channels

import (
	"context"
	"testing"
)

type queueNoticeChannel struct {
	mockChannel
	notices []string
	replyTo []string
	deleted []string
}

func (c *queueNoticeChannel) SendQueueNotice(_ context.Context, _, replyTo, text string) (string, error) {
	c.notices = append(c.notices, text)
	c.replyTo = append(c.replyTo, replyTo)
	return "notice-1", nil
}

func (c *queueNoticeChannel) DeleteMessage(_ context.Context, _, messageID string) error {
	c.deleted = append(c.deleted, messageID)
	return nil
}

// A queue notice must never pass through the channel's Send: preSend and the
// Telegram Send path would take any plain text for the running turn's answer.
func TestQueueNoticeBypassesTheOutboundPath(t *testing.T) {
	ch := &queueNoticeChannel{}
	mgr := newTestManager()
	mgr.channels["telegram"] = ch

	id := mgr.SendQueueNotice(context.Background(), "telegram", "chat-1", "42", "Queued — 1 message ahead.")
	if id != "notice-1" {
		t.Fatalf("SendQueueNotice returned %q", id)
	}
	if len(ch.notices) != 1 || ch.replyTo[0] != "42" {
		t.Fatalf("notices=%v replyTo=%v", ch.notices, ch.replyTo)
	}
	if len(ch.sentMessages) != 0 {
		t.Fatalf("the notice went through Send: %v", ch.sentMessages)
	}
	if _, loaded := mgr.placeholders.Load("telegram:chat-1"); loaded {
		t.Fatal("the notice was recorded as a placeholder and would be edited into an answer")
	}

	mgr.DeleteQueueNotice(context.Background(), "telegram", "chat-1", id)
	if len(ch.deleted) != 1 || ch.deleted[0] != "notice-1" {
		t.Fatalf("deleted = %v", ch.deleted)
	}
}

func TestQueueNoticeIsSkippedWithoutTheCapability(t *testing.T) {
	mgr := newTestManager()
	mgr.channels["plain"] = &mockChannel{}
	if id := mgr.SendQueueNotice(context.Background(), "plain", "c", "1", "x"); id != "" {
		t.Fatalf("a channel without the capability sent notice %q", id)
	}
	if id := mgr.SendQueueNotice(context.Background(), "missing", "c", "1", "x"); id != "" {
		t.Fatalf("an unknown channel sent notice %q", id)
	}
	mgr.DeleteQueueNotice(context.Background(), "plain", "c", "")
}
