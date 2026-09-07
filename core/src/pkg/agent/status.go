package agent

import (
	"time"

	"github.com/sipeed/picoclaw/pkg/status"
)

// recordTurnOutcome counts one terminal turn.
//
// It is called from runTurn's terminal defer, which is the single place that
// owns the turn's final classification, so every path that ends a turn —
// normal finish, provider error, hook abort, hard abort, setup failure — is
// counted exactly once and in the same category the turn.end event reports.
func (al *AgentLoop) recordTurnOutcome(turnStatus TurnEndStatus) {
	if al == nil {
		return
	}
	switch turnStatus {
	case TurnEndStatusCompleted:
		al.turnsCompleted.Add(1)
	case TurnEndStatusError:
		al.turnsFailed.Add(1)
	case TurnEndStatusAborted:
		al.turnsCancelled.Add(1)
	default:
		// An unrecognized status is left uncounted rather than folded into a
		// category it does not belong to. A total that silently absorbs an
		// unknown outcome is worse than one that visibly omits it.
		return
	}
	al.lastActivityUnix.Store(time.Now().Unix())
}

// recordToolOutcome counts one tool execution that reached a result.
//
// Only the success flag is read. The tool's name, arguments and result are
// deliberately not passed in: the counter has no use for them, and not
// accepting them is what keeps tool data out of the Status payload by
// construction rather than by remembering to strip it later.
func (al *AgentLoop) recordToolOutcome(success bool) {
	if al == nil {
		return
	}
	al.toolCalls.Add(1)
	if !success {
		al.toolCallsFailed.Add(1)
	}
}

// activeTurnCounts reports how many root turns and sub-turns are executing.
//
// Root turns and sub-turns share activeTurnStates but are registered under
// distinct keys, and a sub-turn records its nesting in depth, so the two are
// separable without any extra bookkeeping. They are counted separately
// because they are different things: one is a user's question in flight, the
// other is work that question spawned.
func (al *AgentLoop) activeTurnCounts() (root, sub int) {
	if al == nil {
		return 0, 0
	}
	al.activeTurnStates.Range(func(_, value any) bool {
		ts, ok := value.(*turnState)
		if !ok {
			return true
		}
		if ts.currentDepth() > 0 {
			sub++
		} else {
			root++
		}
		return true
	})
	return root, sub
}

// queuedMessageCount reports how many inbound messages are parked in session
// mailboxes waiting for the turn that owns their session to finish.
//
// Only lengths are read; no message, session key or mailbox identity leaves
// this function.
func (al *AgentLoop) queuedMessageCount() int {
	if al == nil {
		return 0
	}
	al.sessionMailboxMu.Lock()
	defer al.sessionMailboxMu.Unlock()

	total := 0
	for _, mailbox := range al.sessionMailboxes {
		if mailbox == nil {
			continue
		}
		total += len(mailbox.independent)
	}
	return total
}

// StatusActivity reports the current activity snapshot.
func (al *AgentLoop) StatusActivity() status.Activity {
	if al == nil {
		return status.Activity{}
	}
	root, sub := al.activeTurnCounts()
	return status.Activity{
		ActiveTurns:      root,
		ActiveSubagents:  sub,
		Waiting:          al.queuedMessageCount(),
		Completed:        al.turnsCompleted.Load(),
		Failed:           al.turnsFailed.Load(),
		Cancelled:        al.turnsCancelled.Load(),
		ToolCalls:        al.toolCalls.Load(),
		ToolCallsFailed:  al.toolCallsFailed.Load(),
		LastActivityUnix: al.lastActivityUnix.Load(),
	}
}

// StatusModel reports which model is answering and which one is configured.
//
// ActiveModel is the agent instance's live model and ConfiguredModel the
// configured default; the advanced "/switch model to <name>" command moves the
// first without the second. Status reports both so a divergence is visible,
// and changes neither: model selection belongs to the Dashboard.
func (al *AgentLoop) StatusModel() status.Model {
	if al == nil {
		return status.Model{}
	}

	snapshot := status.Model{}
	if cfg := al.GetConfig(); cfg != nil {
		snapshot.ConfiguredModel = cfg.Agents.Defaults.GetModelName()
	}
	if al.registry == nil {
		return snapshot
	}
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return snapshot
	}

	snapshot.ActiveModel = agent.Model
	snapshot.FallbackCount = len(agent.Candidates)
	defaultProvider := ""
	if cfg := al.GetConfig(); cfg != nil {
		defaultProvider = cfg.Agents.Defaults.Provider
	}
	// Same resolution the /show command uses, so Status and chat never
	// disagree about which provider is answering.
	snapshot.Provider = resolvedCandidateProvider(agent.Candidates, defaultProvider)
	return snapshot
}
