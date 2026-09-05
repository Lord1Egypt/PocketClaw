package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/commands"
)

// controlPlaneCommands are the commands whose whole purpose is to act on a turn
// that is already running. Queueing one behind that turn makes it a no-op: by
// the time it is dequeued the operation it was meant to cancel has finished.
var controlPlaneCommands = map[string]struct{}{
	"stop": {},
}

// isControlPlaneMessage reports whether an inbound message is control traffic
// rather than a conversational turn.
func isControlPlaneMessage(msg bus.InboundMessage) bool {
	name, ok := commands.CommandName(msg.Content)
	if !ok {
		return false
	}
	_, control := controlPlaneCommands[strings.ToLower(strings.TrimSpace(name))]
	return control
}

// activeTurnDeliveryTarget returns the channel, chat and inbound context of the
// turn currently running for a session.
//
// The inbound context is what makes cancellation cleanup work on Telegram: the
// typing indicator and the "Thinking…" placeholder are both recorded per
// inbound lifecycle, so they can only be reached through the lifecycle of the
// turn being cancelled — never through the lifecycle of the /stop message that
// cancels it.
func (al *AgentLoop) activeTurnDeliveryTarget(sessionKey string) (
	channel string, chatID string, inbound *bus.InboundContext, ok bool,
) {
	ts := al.getActiveTurnState(sessionKey)
	if ts == nil {
		return "", "", nil, false
	}
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	inbound = ts.opts.Dispatch.InboundContext
	return ts.channel, ts.chatID, inbound, true
}
