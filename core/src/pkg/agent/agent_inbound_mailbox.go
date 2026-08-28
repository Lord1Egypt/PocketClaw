package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type sessionMailbox struct {
	independent []bus.InboundMessage
}

func requiresIndependentResponseLifecycle(msg bus.InboundMessage) bool {
	return strings.EqualFold(strings.TrimSpace(msg.Channel), "telegram")
}

// claimSessionMailbox serializes ownership of a routed session. A Telegram
// message that arrives while the session is busy is retained as a complete
// inbound message, including its safe lifecycle ID and reply context, so it
// receives its own provider call and final delivery. Other channels keep the
// existing steering behavior.
func (al *AgentLoop) claimSessionMailbox(
	sessionKey string,
	msg bus.InboundMessage,
) (owner *sessionMailbox, claimed, queued bool) {
	al.sessionMailboxMu.Lock()
	defer al.sessionMailboxMu.Unlock()

	if al.sessionMailboxes == nil {
		al.sessionMailboxes = make(map[string]*sessionMailbox)
	}
	mailbox, exists := al.sessionMailboxes[sessionKey]
	if !exists {
		mailbox = &sessionMailbox{}
		al.sessionMailboxes[sessionKey] = mailbox
		return mailbox, true, false
	}
	if !requiresIndependentResponseLifecycle(msg) {
		// clearActiveTurn runs while a panicking worker unwinds, just before its
		// mailbox defer. Do not mistake that stale mailbox for a live owner and
		// strand a new message in a steering queue with nobody left to drain it.
		if _, active := al.activeTurnStates.Load(sessionKey); !active {
			mailbox = &sessionMailbox{}
			al.sessionMailboxes[sessionKey] = mailbox
			return mailbox, true, false
		}
		return mailbox, false, false
	}
	mailbox.independent = append(mailbox.independent, msg)
	logger.DebugCF("agent", "Inbound response lifecycle queued", map[string]any{
		"channel":      msg.Channel,
		"lifecycle_id": bus.InboundLifecycleID(&msg.Context),
		"queue_depth":  len(mailbox.independent),
	})
	return mailbox, false, true
}

func (al *AgentLoop) takeNextSessionMessage(
	sessionKey string,
	owner *sessionMailbox,
) (bus.InboundMessage, bool) {
	al.sessionMailboxMu.Lock()
	defer al.sessionMailboxMu.Unlock()

	mailbox := al.sessionMailboxes[sessionKey]
	if mailbox != owner {
		return bus.InboundMessage{}, false
	}
	if len(mailbox.independent) == 0 {
		delete(al.sessionMailboxes, sessionKey)
		return bus.InboundMessage{}, false
	}
	next := mailbox.independent[0]
	mailbox.independent[0] = bus.InboundMessage{}
	mailbox.independent = mailbox.independent[1:]
	return next, true
}

func (al *AgentLoop) releaseSessionMailbox(sessionKey string, owner *sessionMailbox) {
	al.sessionMailboxMu.Lock()
	if al.sessionMailboxes[sessionKey] == owner {
		delete(al.sessionMailboxes, sessionKey)
	}
	al.sessionMailboxMu.Unlock()
}
