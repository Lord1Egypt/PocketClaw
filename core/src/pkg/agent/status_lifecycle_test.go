package agent

import (
	"context"
	"strings"
	"testing"
)

// The tests here drive the real runTurn so that the counters are proved
// against the function that owns turn classification, not against a
// re-implementation of it in the test.

func runTurnForStatus(t *testing.T, al *AgentLoop, agent *AgentInstance, sessionKey string) {
	t.Helper()
	pipeline := NewPipeline(al)
	ts := newTurnState(agent, makeTestProcessOpts(sessionKey), turnEventScope{
		turnID:  "turn-" + sessionKey,
		context: newTurnContext(nil, nil, nil),
	})
	_, _ = al.runTurn(context.Background(), ts, pipeline)
}

func TestRunTurnCountsCompletedTurn(t *testing.T) {
	al, agent, cleanup := newTurnCoordTestLoop(t, &simpleConvProvider{})
	defer cleanup()

	runTurnForStatus(t, al, agent, "completed-session")

	activity := al.StatusActivity()
	if activity.Completed != 1 {
		t.Fatalf("Completed = %d, want 1", activity.Completed)
	}
	if activity.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", activity.Failed)
	}
	if activity.Cancelled != 0 {
		t.Fatalf("Cancelled = %d, want 0", activity.Cancelled)
	}
	if activity.ActiveTurns != 0 {
		t.Fatalf("ActiveTurns = %d after the turn ended, want 0", activity.ActiveTurns)
	}
}

func TestRunTurnCountsProviderFailureAsFailed(t *testing.T) {
	al, agent, cleanup := newTurnCoordTestLoop(t, &errorProvider{})
	defer cleanup()

	runTurnForStatus(t, al, agent, "failed-session")

	activity := al.StatusActivity()
	if activity.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", activity.Failed)
	}
	if activity.Completed != 0 {
		t.Fatalf("Completed = %d, want 0", activity.Completed)
	}
	if activity.Cancelled != 0 {
		t.Fatalf("Cancelled = %d, want 0", activity.Cancelled)
	}
}

// A hard abort is what /stop drives, so it must land in Cancelled and never in
// Failed: a user who stopped their own answer has not hit an error.
func TestRunTurnCountsHardAbortAsCancelled(t *testing.T) {
	al, agent, cleanup := newTurnCoordTestLoop(t, &simpleConvProvider{})
	defer cleanup()

	pipeline := NewPipeline(al)
	ts := newTurnState(agent, makeTestProcessOpts("cancelled-session"), turnEventScope{
		turnID:  "turn-cancelled",
		context: newTurnContext(nil, nil, nil),
	})
	if !ts.requestHardAbort() {
		t.Fatal("requestHardAbort did not take effect")
	}
	_, _ = al.runTurn(context.Background(), ts, pipeline)

	activity := al.StatusActivity()
	if activity.Cancelled != 1 {
		t.Fatalf("Cancelled = %d, want 1", activity.Cancelled)
	}
	if activity.Completed != 0 {
		t.Fatalf("Completed = %d, want 0", activity.Completed)
	}
	if activity.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", activity.Failed)
	}
}

// TestRunTurnClassifiesSetupFailureAsFailed pins the SetupTurn error branch.
//
// SetupTurn has a single return today and cannot actually fail, so this branch
// is unreachable at runtime and no behavioural test can enter it. The defect
// it guards is real all the same: runTurn initialises turnStatus optimistically
// to Completed, and that branch used to return without correcting it, so the
// first time SetupTurn gains an error return a failed setup would have been
// counted — and reported in turn.end — as a completed turn.
//
// The assertion is therefore structural: it reads the source of the branch and
// requires that it assign an error classification before returning. Deleting
// the assignment fails this test, which is the point.
func TestRunTurnClassifiesSetupFailureAsFailed(t *testing.T) {
	source := readAgentSourceFile(t, "turn_coord.go")

	const anchor = "exec, err := pipeline.SetupTurn(turnCtx, ts)"
	start := strings.Index(source, anchor)
	if start < 0 {
		t.Fatalf("SetupTurn call not found; this test must be updated with the call site")
	}
	rest := source[start+len(anchor):]
	end := strings.Index(rest, "\n\t}")
	if end < 0 {
		t.Fatal("could not delimit the SetupTurn error branch")
	}
	branch := rest[:end]

	if !strings.Contains(branch, "turnStatus = TurnEndStatusError") {
		t.Fatalf(
			"the SetupTurn error branch returns without classifying the turn as failed, "+
				"so a setup failure would be counted as completed; branch was:\n%s",
			branch,
		)
	}
}
