package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func (al *AgentLoop) tryHandleStopCommand(
	ctx context.Context,
	msg bus.InboundMessage,
	sessionKey string,
) bool {
	cmdName, ok := commands.CommandName(msg.Content)
	if !ok || cmdName != "stop" {
		return false
	}

	// Resolve the running turn's delivery target before cancelling it: once the
	// turn is aborted its state is released and the lifecycle that owns the
	// typing indicator and the "Thinking…" placeholder is no longer reachable.
	targetChannel, targetChatID, targetInbound, hadActiveTurn := al.activeTurnDeliveryTarget(sessionKey)
	if hadActiveTurn {
		traceRequestLifecycle("cancel_requested", targetInbound, map[string]any{
			"requested_by": bus.InboundLifecycleID(&msg.Context),
		})
	}

	result, err := al.stopActiveTurnForSession(sessionKey)

	// This function is only called when loaded=true (another turn already
	// claimed this session). If stopActiveTurnForSession found a pending
	// placeholder but didn't stop it, that placeholder belongs to the other
	// message's worker which hasn't started yet — arm a pending stop so the
	// worker will bail when it checks before running.
	if err == nil && !result.Stopped {
		if ts := al.getActiveTurnState(sessionKey); ts != nil {
			snap := ts.snapshot()
			if strings.HasPrefix(snap.TurnID, pendingTurnPrefix) {
				al.markPendingStop(sessionKey)
				result.Stopped = true
			}
		}
	}

	reply := commands.FormatStopReply(result)
	if err != nil {
		reply = "Failed to stop task: " + err.Error()
	}

	al.finalizeCancelledTurnActivity(msg, targetChannel, targetChatID, targetInbound, hadActiveTurn)
	al.resetMessageToolRound(sessionKey)

	// Deliver the acknowledgement on the cancelled turn's lifecycle so the
	// channel layer edits its stale "Thinking…" placeholder into this reply
	// instead of leaving it in the chat until its TTL expires. Falls back to the
	// /stop message's own context when no turn was running.
	deliverChannel, deliverChatID, deliverInbound := msg.Channel, msg.ChatID, (*bus.InboundContext)(nil)
	if hadActiveTurn && targetChannel == msg.Channel && targetChatID == msg.ChatID {
		deliverInbound = targetInbound
	}
	if publishErr := al.publishResponseForContext(
		ctx, deliverChannel, deliverChatID, sessionKey, deliverInbound, reply,
	); publishErr != nil {
		logger.ErrorCF("agent", "Cancellation acknowledgement delivery failed", map[string]any{
			"channel": deliverChannel,
			"error":   publishErr.Error(),
		})
	}
	if hadActiveTurn {
		traceRequestLifecycle("cancelled", targetInbound, map[string]any{
			"stopped": result.Stopped,
		})
	}
	return true
}

// finalizeCancelledTurnActivity clears the transient chat activity a cancelled
// turn leaves behind. It is idempotent: every stop primitive it calls tolerates
// being invoked when nothing is recorded, so repeated /stop commands and a
// concurrent natural turn completion cannot conflict.
func (al *AgentLoop) finalizeCancelledTurnActivity(
	msg bus.InboundMessage,
	targetChannel, targetChatID string,
	targetInbound *bus.InboundContext,
	hadActiveTurn bool,
) {
	if al.channelManager == nil {
		return
	}
	// The /stop message's own indicator, keyed without a lifecycle.
	al.channelManager.InvokeTypingStop(msg.Channel, msg.ChatID)
	if !hadActiveTurn || targetChannel == "" || targetChatID == "" {
		return
	}
	al.channelManager.InvokeTypingStop(targetChannel, targetChatID)
	// Telegram records the indicator per inbound lifecycle, so the bare key
	// above cannot reach the one the cancelled turn started.
	if lifecycleID := bus.InboundLifecycleID(targetInbound); lifecycleID != "" {
		al.channelManager.InvokeTypingStopForLifecycle(targetChannel, targetChatID, lifecycleID)
	}
}

func (al *AgentLoop) stopActiveTurnForSession(sessionKey string) (commands.StopResult, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return commands.StopResult{}, fmt.Errorf("session key is required")
	}

	result := commands.StopResult{}
	cleared := al.clearSteeringMessagesForScope(sessionKey)
	al.clearPendingSkills(sessionKey)

	ts := al.getActiveTurnState(sessionKey)
	if ts == nil {
		result.Stopped = cleared > 0
		return result, nil
	}

	snap := ts.snapshot()
	result.TaskName = snap.UserMessage

	if strings.HasPrefix(snap.TurnID, pendingTurnPrefix) {
		// A pending placeholder means this session is either idle (our own
		// placeholder from the /stop command) or another message is queued but
		// hasn't started yet. In both cases, we don't arm a pending stop here;
		// the caller (tryHandleStopCommand) handles the "another message queued"
		// case explicitly, since it knows loaded=true.
		return result, nil
	}

	if err := al.HardAbort(sessionKey); err != nil {
		if al.getActiveTurnState(sessionKey) == nil {
			result.Stopped = cleared > 0
			return result, nil
		}
		return commands.StopResult{}, err
	}

	result.Stopped = true
	return result, nil
}

func (al *AgentLoop) markPendingStop(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	al.pendingStops.Store(sessionKey, struct{}{})
}

func (al *AgentLoop) takePendingStop(sessionKey string) bool {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return false
	}
	_, ok := al.pendingStops.LoadAndDelete(sessionKey)
	return ok
}

func (al *AgentLoop) resetMessageToolRound(sessionKey string) {
	if strings.TrimSpace(sessionKey) == "" {
		return
	}
	if registry := al.GetRegistry(); registry != nil {
		if agent := registry.GetDefaultAgent(); agent != nil {
			if tool, ok := agent.Tools.Get("message"); ok {
				if resetter, ok := tool.(interface{ ResetSentInRound(sessionKey string) }); ok {
					resetter.ResetSentInRound(sessionKey)
				}
			}
		}
	}
}
