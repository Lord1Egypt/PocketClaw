package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// newCounterLoop returns an AgentLoop usable for counter assertions only.
func newCounterLoop() *AgentLoop {
	return &AgentLoop{}
}

func TestRecordTurnOutcomeCountsCompletedOnly(t *testing.T) {
	al := newCounterLoop()
	al.recordTurnOutcome(TurnEndStatusCompleted)

	activity := al.StatusActivity()
	if activity.Completed != 1 {
		t.Fatalf("Completed = %d, want 1", activity.Completed)
	}
	if activity.Failed != 0 || activity.Cancelled != 0 {
		t.Fatalf("Failed = %d, Cancelled = %d, want 0 and 0", activity.Failed, activity.Cancelled)
	}
	if activity.LastActivityUnix == 0 {
		t.Fatal("LastActivityUnix was not recorded for a completed turn")
	}
}

func TestRecordTurnOutcomeCountsFailedOnly(t *testing.T) {
	al := newCounterLoop()
	al.recordTurnOutcome(TurnEndStatusError)

	activity := al.StatusActivity()
	if activity.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", activity.Failed)
	}
	if activity.Completed != 0 || activity.Cancelled != 0 {
		t.Fatalf("Completed = %d, Cancelled = %d, want 0 and 0", activity.Completed, activity.Cancelled)
	}
}

func TestRecordTurnOutcomeCountsCancelledOnly(t *testing.T) {
	al := newCounterLoop()
	al.recordTurnOutcome(TurnEndStatusAborted)

	activity := al.StatusActivity()
	if activity.Cancelled != 1 {
		t.Fatalf("Cancelled = %d, want 1", activity.Cancelled)
	}
	if activity.Completed != 0 || activity.Failed != 0 {
		t.Fatalf("Completed = %d, Failed = %d, want 0 and 0", activity.Completed, activity.Failed)
	}
}

// TestRecordTurnOutcomeIgnoresUnknownStatus documents that an unrecognized
// terminal status is left uncounted rather than folded into a category it does
// not belong to.
func TestRecordTurnOutcomeIgnoresUnknownStatus(t *testing.T) {
	al := newCounterLoop()
	al.recordTurnOutcome(TurnEndStatus("something-new"))

	activity := al.StatusActivity()
	if activity.Completed != 0 || activity.Failed != 0 || activity.Cancelled != 0 {
		t.Fatalf("unknown status was counted: %+v", activity)
	}
	if activity.LastActivityUnix != 0 {
		t.Fatal("unknown status updated LastActivityUnix")
	}
}

func TestRecordToolOutcomeCounters(t *testing.T) {
	al := newCounterLoop()
	al.recordToolOutcome(true)
	al.recordToolOutcome(true)
	al.recordToolOutcome(false)

	activity := al.StatusActivity()
	if activity.ToolCalls != 3 {
		t.Fatalf("ToolCalls = %d, want 3", activity.ToolCalls)
	}
	if activity.ToolCallsFailed != 1 {
		t.Fatalf("ToolCallsFailed = %d, want 1", activity.ToolCallsFailed)
	}
}

// TestActiveTurnCountsSeparatesRootFromSubagent proves a spawned sub-turn is
// reported as an active subagent and never inflates the root turn count.
func TestActiveTurnCountsSeparatesRootFromSubagent(t *testing.T) {
	al := newCounterLoop()

	root := &turnState{sessionKey: "session-root"}
	child := &turnState{sessionKey: "subturn-1", depth: 1}
	grandchild := &turnState{sessionKey: "subturn-2", depth: 2}

	al.activeTurnStates.Store(root.sessionKey, root)
	al.activeTurnStates.Store(child.sessionKey, child)
	al.activeTurnStates.Store(grandchild.sessionKey, grandchild)

	activity := al.StatusActivity()
	if activity.ActiveTurns != 1 {
		t.Fatalf("ActiveTurns = %d, want 1", activity.ActiveTurns)
	}
	if activity.ActiveSubagents != 2 {
		t.Fatalf("ActiveSubagents = %d, want 2", activity.ActiveSubagents)
	}

	al.activeTurnStates.Delete(child.sessionKey)
	al.activeTurnStates.Delete(grandchild.sessionKey)
	al.activeTurnStates.Delete(root.sessionKey)

	if got := al.StatusActivity(); got.ActiveTurns != 0 || got.ActiveSubagents != 0 {
		t.Fatalf("gauges did not return to zero: %+v", got)
	}
}

// TestQueuedMessageCountTracksMailboxes proves Waiting reflects messages
// parked in independent session mailboxes and drops back as they drain.
func TestQueuedMessageCountTracksMailboxes(t *testing.T) {
	al := newCounterLoop()

	if got := al.StatusActivity().Waiting; got != 0 {
		t.Fatalf("Waiting = %d on an empty loop, want 0", got)
	}

	owner := &sessionMailbox{}
	al.sessionMailboxMu.Lock()
	al.sessionMailboxes = map[string]*sessionMailbox{"session-a": owner}
	al.sessionMailboxMu.Unlock()

	if got := al.StatusActivity().Waiting; got != 0 {
		t.Fatalf("Waiting = %d for an empty mailbox, want 0", got)
	}

	msg := telegramMessage("lifecycle-1", "second message")
	if _, _, queued := al.claimSessionMailbox("session-a", msg); !queued {
		t.Fatal("expected the second message for a busy session to be queued")
	}
	if got := al.StatusActivity().Waiting; got != 1 {
		t.Fatalf("Waiting = %d after queueing one message, want 1", got)
	}

	if _, ok := al.takeNextSessionMessage("session-a", owner); !ok {
		t.Fatal("expected to drain the queued message")
	}
	if got := al.StatusActivity().Waiting; got != 0 {
		t.Fatalf("Waiting = %d after draining, want 0", got)
	}
}

// TestStatusSnapshotCannotLeakSensitiveRuntimeFields fills the runtime with
// values that would be damaging to publish and proves none of them can reach
// the serialized Status payload.
//
// The DTOs are built by explicit field-by-field mapping, so this test is what
// makes that discipline enforceable: widening a Status struct to carry a turn,
// a mailbox or a config object fails here rather than in production.
func TestStatusSnapshotCannotLeakSensitiveRuntimeFields(t *testing.T) {
	const (
		secretPrompt  = "PROMPT-please-summarize-my-medical-records"
		secretSession = "SESSIONKEY-telegram-448812733"
		secretChat    = "CHATID-448812733"
		secretTurn    = "TURNID-9f3c11a7"
		secretUser    = "SENDER-lord1egypt"
	)

	al := newCounterLoop()
	ts := &turnState{
		sessionKey:  secretSession,
		chatID:      secretChat,
		turnID:      secretTurn,
		userMessage: secretPrompt,
		channel:     "telegram",
		agentID:     secretUser,
	}
	al.activeTurnStates.Store(ts.sessionKey, ts)

	al.sessionMailboxMu.Lock()
	al.sessionMailboxes = map[string]*sessionMailbox{secretSession: {}}
	al.sessionMailboxMu.Unlock()

	al.recordTurnOutcome(TurnEndStatusCompleted)
	al.recordToolOutcome(false)

	raw, err := json.Marshal(al.StatusActivity())
	if err != nil {
		t.Fatalf("marshal activity: %v", err)
	}
	payload := string(raw)

	for _, forbidden := range []string{
		secretPrompt, secretSession, secretChat, secretTurn, secretUser, "telegram",
	} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("Status payload leaked %q: %s", forbidden, payload)
		}
	}

	// The counts themselves must still be present, or the test would pass on
	// an empty payload.
	if !strings.Contains(payload, `"active_turns":1`) {
		t.Fatalf("Status payload lost its counts: %s", payload)
	}
}
