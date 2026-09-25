package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// queueNoticeManager is the optional part of the channel manager that can tell
// a user their message is queued. It is optional so a manager that cannot do
// it — every test double, and any channel without the capability — simply
// sends nothing.
type queueNoticeManager interface {
	SendQueueNotice(ctx context.Context, channel, chatID, replyToMessageID, text string) string
	DeleteQueueNotice(ctx context.Context, channel, chatID, messageID string)
}

// queueNoticeTimeout bounds the Bot API call. The notice is sent from its own
// goroutine, so this delays nothing but the notice.
const queueNoticeTimeout = 10 * time.Second

// queueNotice is one message's notice, sent asynchronously. done closes once
// the send has finished, successfully or not, and only then is id valid.
type queueNotice struct {
	done chan struct{}
	id   string
}

// queueNoticeText is what a queued sender sees. ahead counts the running turn
// and every message queued before this one.
func queueNoticeText(ahead int) string {
	noun := "messages"
	if ahead == 1 {
		noun = "message"
	}
	return fmt.Sprintf("Queued — %d %s ahead. This will start automatically.", ahead, noun)
}

// announceQueuedMessage tells the sender of a message that has to wait that it
// was received and will run, rather than leaving a long-running turn ahead of
// it to look like a bot that stopped answering.
//
// One notice per queued message, sent as a reply to it. It is removed when the
// message begins executing, where the ordinary Thinking placeholder takes over,
// so a burst of messages leaves no trail of status messages behind.
func (al *AgentLoop) announceQueuedMessage(ctx context.Context, msg bus.InboundMessage, ahead int) {
	manager, ok := al.channelManager.(queueNoticeManager)
	if !ok || ahead <= 0 {
		return
	}
	lifecycleID := bus.InboundLifecycleID(&msg.Context)
	if lifecycleID == "" {
		return
	}
	notice := &queueNotice{done: make(chan struct{})}
	al.queueNotices.Store(lifecycleID, notice)
	logger.InfoCF("agent", "Inbound message queued behind a running turn", map[string]any{
		"channel":      msg.Channel,
		"lifecycle_id": lifecycleID,
		"ahead":        ahead,
	})

	go func() {
		defer close(notice.done)
		sendCtx, cancel := context.WithTimeout(ctx, queueNoticeTimeout)
		defer cancel()
		notice.id = manager.SendQueueNotice(
			sendCtx, msg.Channel, msg.ChatID, msg.Context.MessageID, queueNoticeText(ahead),
		)
	}()
}

// retireQueueNotice removes the notice for a message that is now executing.
//
// It waits for the send to finish first: a message dequeued the moment it was
// queued would otherwise race its own notice, and the notice would arrive after
// the delete and stay in the chat for good.
func (al *AgentLoop) retireQueueNotice(ctx context.Context, msg bus.InboundMessage) {
	lifecycleID := bus.InboundLifecycleID(&msg.Context)
	if lifecycleID == "" {
		return
	}
	value, ok := al.queueNotices.LoadAndDelete(lifecycleID)
	if !ok {
		return
	}
	notice := value.(*queueNotice)
	select {
	case <-notice.done:
	case <-ctx.Done():
		return
	}
	logger.InfoCF("agent", "Queued message started", map[string]any{
		"channel":      msg.Channel,
		"lifecycle_id": lifecycleID,
	})
	if notice.id == "" {
		return
	}
	if manager, ok := al.channelManager.(queueNoticeManager); ok {
		manager.DeleteQueueNotice(ctx, msg.Channel, msg.ChatID, notice.id)
	}
}

// runMailboxTurn runs one message from a session mailbox and confines a panic
// to that message.
//
// Without this a panicking turn unwound the whole worker, and the mailbox it
// owned was released with every message still queued behind it: those
// messages were dropped without a reply or a log line. Now the sender of the
// failed message is told, and the worker moves on to the next one.
func (al *AgentLoop) runMailboxTurn(ctx context.Context, msg bus.InboundMessage) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		logger.RecoverPanicNoExit(r)
		logger.ErrorCF("agent", "Turn panicked; continuing with the next queued message", map[string]any{
			"channel":      msg.Channel,
			"lifecycle_id": bus.InboundLifecycleID(&msg.Context),
			"panic":        fmt.Sprintf("%v", r),
		})
		err = fmt.Errorf("turn panicked: %v", r)
		if publishErr := al.publishResponseForContext(
			ctx, msg.Channel, msg.ChatID, msg.SessionKey, &msg.Context,
			"Something went wrong while handling this message. Please send it again.",
		); publishErr != nil {
			logger.WarnCF("agent", "Could not report a panicked turn", map[string]any{
				"channel": msg.Channel,
				"error":   publishErr.Error(),
			})
		}
	}()
	return al.runTurnWithDeferredActivity(ctx, msg)
}
