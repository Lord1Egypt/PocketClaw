package agent

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type sessionMailbox struct {
	independent []bus.InboundMessage
}

func requiresIndependentResponseLifecycle(msg bus.InboundMessage) bool {
	return bus.ChannelUsesIndependentResponseLifecycle(msg.Channel)
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
	owner, claimed, queued, _ = al.claimSessionMailboxPosition(sessionKey, msg)
	return owner, claimed, queued
}

// claimSessionMailboxPosition is claimSessionMailbox that also reports, for a
// queued message, how many requests are ahead of it: the running turn plus
// every message queued before it. It is computed under the mailbox lock, so a
// burst of messages gets consecutive, accurate positions.
func (al *AgentLoop) claimSessionMailboxPosition(
	sessionKey string,
	msg bus.InboundMessage,
) (owner *sessionMailbox, claimed, queued bool, ahead int) {
	al.sessionMailboxMu.Lock()
	defer al.sessionMailboxMu.Unlock()

	if al.sessionMailboxes == nil {
		al.sessionMailboxes = make(map[string]*sessionMailbox)
	}
	mailbox, exists := al.sessionMailboxes[sessionKey]
	if !exists {
		mailbox = &sessionMailbox{}
		al.sessionMailboxes[sessionKey] = mailbox
		return mailbox, true, false, 0
	}
	if isControlPlaneMessage(msg) {
		// Control traffic is never queued. /stop exists to cancel the turn that
		// owns this mailbox, so waiting behind that turn would make it useless:
		// it would be dequeued only after the work it was meant to stop had
		// already completed. Reported as neither claimed nor queued, so the
		// dispatcher handles it out of band on the control path.
		return mailbox, false, false, 0
	}
	if !requiresIndependentResponseLifecycle(msg) {
		// clearActiveTurn runs while a panicking worker unwinds, just before its
		// mailbox defer. Do not mistake that stale mailbox for a live owner and
		// strand a new message in a steering queue with nobody left to drain it.
		if _, active := al.activeTurnStates.Load(sessionKey); !active {
			mailbox = &sessionMailbox{}
			al.sessionMailboxes[sessionKey] = mailbox
			return mailbox, true, false, 0
		}
		return mailbox, false, false, 0
	}
	mailbox.independent = append(mailbox.independent, msg)
	logger.DebugCF("agent", "Inbound response lifecycle queued", map[string]any{
		"channel":      msg.Channel,
		"lifecycle_id": bus.InboundLifecycleID(&msg.Context),
		"queue_depth":  len(mailbox.independent),
	})
	return mailbox, false, true, len(mailbox.independent)
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
