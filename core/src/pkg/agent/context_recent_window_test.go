package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

var errSummaryFailed = errors.New("summary provider unavailable")

func userMsg(text string) providers.Message {
	return providers.Message{Role: "user", Content: text}
}

func assistantMsg(text string) providers.Message {
	return providers.Message{Role: "assistant", Content: text}
}

// A tool interaction is three records: the assistant issuing the call, the tool
// result, and the assistant's reply that reads it. Only the last is
// conversational.
func toolInteraction(id string) []providers.Message {
	return []providers.Message{
		{Role: "assistant", ToolCalls: []protocoltypes.ToolCall{{ID: id}}},
		{Role: "tool", ToolCallID: id, Content: "result"},
	}
}

// conversation builds n alternating user/assistant messages.
func conversation(n int) []providers.Message {
	history := make([]providers.Message, 0, n)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			history = append(history, userMsg(fmt.Sprintf("u%d", i)))
		} else {
			history = append(history, assistantMsg(fmt.Sprintf("a%d", i)))
		}
	}
	return history
}

func TestProjectRecentHistoryKeepsEverythingUnderTheLimit(t *testing.T) {
	history := conversation(14)
	projected, evicted := projectRecentHistory(history, 14)
	if evicted != 0 {
		t.Fatalf("evicted %d messages when the history already fit", evicted)
	}
	if len(projected) != len(history) {
		t.Fatalf("projected %d messages, want all %d", len(projected), len(history))
	}
}

func TestProjectRecentHistoryBoundsALongTranscript(t *testing.T) {
	for _, total := range []int{16, 40, 101, 500} {
		history := conversation(total)
		projected, evicted := projectRecentHistory(history, 14)

		got := countConversationalMessages(projected)
		if got > 14 {
			t.Errorf("history of %d projected %d conversational messages, want <= 14",
				total, got)
		}
		if evicted == 0 {
			t.Errorf("history of %d evicted nothing", total)
		}
		if evicted+len(projected) != total {
			t.Errorf("history of %d: %d evicted + %d kept != %d",
				total, evicted, len(projected), total)
		}
		// The projection must not grow with the transcript.
		if len(projected) > 15 {
			t.Errorf("history of %d projected %d messages; the window is not bounded",
				total, len(projected))
		}
	}
}

// The window must keep as much as it is allowed to, not the bare minimum.
func TestProjectRecentHistoryKeepsAsMuchAsTheLimitAllows(t *testing.T) {
	history := conversation(100)
	projected, _ := projectRecentHistory(history, 14)

	got := countConversationalMessages(projected)
	if got < 13 || got > 14 {
		t.Fatalf("projected %d conversational messages, want 13 or 14 (turn-aligned)", got)
	}
	if projected[0].Role != "user" {
		t.Fatalf("projection starts with %q, want a user message: the cut must be a turn boundary",
			projected[0].Role)
	}
	// It must be the tail, not some interior slice.
	if projected[len(projected)-1].Content != history[len(history)-1].Content {
		t.Fatal("projection is not anchored to the end of the history")
	}
}

// A cut may never open with a tool result whose call was dropped.
func TestProjectRecentHistoryPreservesToolCallAtomicity(t *testing.T) {
	var history []providers.Message
	for i := 0; i < 30; i++ {
		history = append(history, userMsg(fmt.Sprintf("u%d", i)))
		history = append(history, toolInteraction(fmt.Sprintf("call-%d", i))...)
		history = append(history, assistantMsg(fmt.Sprintf("a%d", i)))
	}

	projected, evicted := projectRecentHistory(history, 14)
	if evicted == 0 {
		t.Fatal("expected the window to evict from a 120-message history")
	}
	if projected[0].Role != "user" {
		t.Fatalf("projection starts with %q, want user", projected[0].Role)
	}

	open := map[string]bool{}
	for _, msg := range projected {
		for _, call := range msg.ToolCalls {
			open[call.ID] = true
		}
		if msg.ToolCallID != "" {
			if !open[msg.ToolCallID] {
				t.Fatalf("tool result %q has no initiating call in the projection", msg.ToolCallID)
			}
			delete(open, msg.ToolCallID)
		}
	}
	if len(open) != 0 {
		t.Fatalf("%d tool calls left without results in the projection", len(open))
	}
	if got := countConversationalMessages(projected); got > 14 {
		t.Fatalf("projected %d conversational messages, want <= 14", got)
	}
}

// Tool records ride with their turn. Counting them would let one tool-heavy
// turn evict the entire conversation.
func TestToolRecordsDoNotConsumeTheBudget(t *testing.T) {
	var history []providers.Message
	history = append(history, userMsg("u0"))
	for i := 0; i < 40; i++ {
		history = append(history, toolInteraction(fmt.Sprintf("call-%d", i))...)
	}
	history = append(history, assistantMsg("a0"))

	if got := countConversationalMessages(history); got != 2 {
		t.Fatalf("counted %d conversational messages in an 82-message turn, want 2", got)
	}
	projected, evicted := projectRecentHistory(history, 14)
	if evicted != 0 {
		t.Fatalf("evicted %d from a turn holding only 2 conversational messages", evicted)
	}
	if len(projected) != len(history) {
		t.Fatal("a single turn under the conversational limit was truncated")
	}
}

// A single turn larger than the whole budget must stay whole rather than be
// torn: the token-budget compression that runs afterwards still applies.
func TestProjectRecentHistoryKeepsAnOversizedFinalTurnWhole(t *testing.T) {
	history := []providers.Message{userMsg("old")}
	history = append(history, assistantMsg("old reply"))
	history = append(history, userMsg("big"))
	for i := 0; i < 30; i++ {
		history = append(history, assistantMsg(fmt.Sprintf("a%d", i)))
	}

	projected, evicted := projectRecentHistory(history, 14)
	if evicted == 0 {
		t.Fatal("expected the earlier turn to be evicted")
	}
	if projected[0].Role != "user" || projected[0].Content != "big" {
		t.Fatalf("projection starts at %q/%q, want the user message that opens the last turn",
			projected[0].Role, projected[0].Content)
	}
}

func TestProjectRecentHistoryDisabledByANonPositiveLimit(t *testing.T) {
	history := conversation(100)
	for _, limit := range []int{0, -1} {
		projected, evicted := projectRecentHistory(history, limit)
		if evicted != 0 || len(projected) != len(history) {
			t.Fatalf("limit %d bounded the history; it must disable the window", limit)
		}
	}
}

func TestRecentContextLimitIsTelegramOnly(t *testing.T) {
	agent := &AgentInstance{TelegramRecentContextMessages: 15}

	for _, channel := range []string{"telegram", "Telegram", " TELEGRAM "} {
		if got := recentContextLimit(agent, channel); got != 15 {
			t.Errorf("%q limit = %d, want 15", channel, got)
		}
	}
	for _, channel := range []string{"pocketclaw", "discord", "slack", "matrix", "cli", "web", ""} {
		if got := recentContextLimit(agent, channel); got != 0 {
			t.Errorf("%q limit = %d, want 0: only Telegram is bounded", channel, got)
		}
	}
	if got := recentContextLimit(nil, "telegram"); got != 0 {
		t.Errorf("nil agent limit = %d, want 0", got)
	}
	if got := recentContextLimit(&AgentInstance{}, "telegram"); got != 0 {
		t.Errorf("unset limit = %d, want 0", got)
	}
}

func TestIsConversationalMessage(t *testing.T) {
	cases := []struct {
		name string
		msg  providers.Message
		want bool
	}{
		{"user", userMsg("hi"), true},
		{"assistant", assistantMsg("hello"), true},
		{"assistant with tool call", providers.Message{
			Role: "assistant", ToolCalls: []protocoltypes.ToolCall{{ID: "c"}},
		}, false},
		{"tool result", providers.Message{Role: "tool", ToolCallID: "c"}, false},
		{"system", providers.Message{Role: "system", Content: "prompt"}, false},
	}
	for _, tc := range cases {
		if got := isConversationalMessage(tc.msg); got != tc.want {
			t.Errorf("%s: isConversationalMessage = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// The turn-level projection reserves a slot for the inbound message the prompt
// builder appends, so the model sees at most the configured total. Past the
// hold ceiling the cap is enforced even though coverage never advanced.
func TestProjectRecentContextReservesTheCurrentMessage(t *testing.T) {
	ts := &turnState{
		agent:      &AgentInstance{TelegramRecentContextMessages: 15},
		sessionKey: "s1",
		opts:       processOptions{Channel: "telegram"},
	}

	history := conversation(100)
	projected := ts.projectRecentContext(context.Background(), nil, history, "an existing summary")

	got := countConversationalMessages(projected)
	if got > 14 {
		t.Fatalf("projected %d history messages; with the current message that is %d, want <= 15",
			got, got+1)
	}
	if len(projected) == len(history) {
		t.Fatal("the Telegram window did not bound a 100-message history")
	}
}

func TestProjectRecentContextLeavesOtherChannelsAlone(t *testing.T) {
	history := conversation(100)
	for _, channel := range []string{"pocketclaw", "discord", "slack", "matrix", "cli"} {
		ts := &turnState{
			agent:      &AgentInstance{TelegramRecentContextMessages: 15},
			sessionKey: "s1",
			opts:       processOptions{Channel: channel},
		}
		projected := ts.projectRecentContext(context.Background(), nil, history, "")
		if len(projected) != len(history) {
			t.Errorf("%s projected %d of %d messages; it must be unchanged",
				channel, len(projected), len(history))
		}
	}
}

// A summarizer that never lands must not grow the prompt without limit. Past
// the hold ceiling the cap is enforced, with the loss logged rather than
// hidden. This is the documented degradation.
func TestProjectRecentContextBoundsEvenWithNoSummary(t *testing.T) {
	ts := &turnState{
		agent:      &AgentInstance{TelegramRecentContextMessages: 15},
		sessionKey: "s1",
		opts:       processOptions{Channel: "telegram"},
	}

	history := conversation(200)
	projected := ts.projectRecentContext(context.Background(), nil, history, "")

	if got := countConversationalMessages(projected); got > 14 {
		t.Fatalf("with no summary the projection was %d messages, want <= 14", got)
	}
}

// The projection is read-only: it must return a view of the input and never
// rewrite it.
func TestProjectRecentContextDoesNotMutateStoredHistory(t *testing.T) {
	ts := &turnState{
		agent:      &AgentInstance{TelegramRecentContextMessages: 15},
		sessionKey: "s1",
		opts:       processOptions{Channel: "telegram"},
	}

	history := conversation(100)
	before := make([]providers.Message, len(history))
	copy(before, history)

	ts.projectRecentContext(context.Background(), nil, history, "")

	if len(history) != len(before) {
		t.Fatalf("history length changed from %d to %d", len(before), len(history))
	}
	for i := range before {
		if history[i].Content != before[i].Content || history[i].Role != before[i].Role {
			t.Fatalf("history[%d] was mutated by the projection", i)
		}
	}
}

// The recent-context window must not become a way for an earlier turn to see
// later messages. Queued Telegram messages live in the session mailbox and are
// written to the session store only when their own turn starts, so a history
// assembled while A is running cannot contain B, C or D — whatever the window
// keeps. This pins that separation rather than the window itself.
func TestQueuedMessagesAreNotVisibleToAnEarlierTurn(t *testing.T) {
	al := &AgentLoop{}
	const sessionKey = "session-1"

	a := telegramMessage("lc-a", "A")
	owner, claimed, _ := al.claimSessionMailbox(sessionKey, a)
	if !claimed {
		t.Fatal("A did not claim the session")
	}
	for _, later := range []string{"B", "C", "D"} {
		_, _, queued := al.claimSessionMailbox(sessionKey, telegramMessage("lc-"+later, later))
		if !queued {
			t.Fatalf("%s was not queued behind A", later)
		}
	}

	// The history A's turn assembles holds only what was persisted before it:
	// A itself and whatever preceded it. B, C and D are still in the mailbox.
	historyDuringA := []providers.Message{userMsg("older"), assistantMsg("older reply")}
	ts := &turnState{
		agent:      &AgentInstance{TelegramRecentContextMessages: 15},
		sessionKey: sessionKey,
		opts:       processOptions{Channel: "telegram"},
	}
	projected := ts.projectRecentContext(context.Background(), nil, historyDuringA, "")

	for _, msg := range projected {
		switch msg.Content {
		case "B", "C", "D":
			t.Fatalf("A's context contained %q, a message queued after it", msg.Content)
		}
	}

	// They are still queued, in order, waiting for their own turns.
	for _, want := range []string{"lc-B", "lc-C", "lc-D"} {
		next, ok := al.takeNextSessionMessage(sessionKey, owner)
		if !ok {
			t.Fatalf("mailbox drained early, expected %s", want)
		}
		if got := bus.InboundLifecycleID(&next.Context); got != want {
			t.Fatalf("dequeued %s, want %s: FIFO order changed", got, want)
		}
	}
}

// recordingContextManager captures the coverage requests the window makes
// without running a summarizer.
type recordingContextManager struct {
	compacts []CompactRequest
	err      error
}

func (m *recordingContextManager) Assemble(context.Context, *AssembleRequest) (*AssembleResponse, error) {
	return &AssembleResponse{}, nil
}

func (m *recordingContextManager) Compact(_ context.Context, req *CompactRequest) error {
	m.compacts = append(m.compacts, *req)
	return m.err
}

func (m *recordingContextManager) Ingest(context.Context, *IngestRequest) error { return nil }

func (m *recordingContextManager) Clear(context.Context, string) error { return nil }

func telegramTurn(limit int) *turnState {
	return &turnState{
		agent:      &AgentInstance{TelegramRecentContextMessages: limit, ContextWindow: 128000},
		sessionKey: "s1",
		opts:       processOptions{Channel: "telegram"},
	}
}

// THE GAP. A fresh session with 16 conversational messages and no summary: the
// oldest must not vanish from the model with nothing representing it.
func TestFreshSessionDoesNotDropUncoveredHistory(t *testing.T) {
	cm := &recordingContextManager{}
	ts := telegramTurn(15)
	history := conversation(16)

	projected := ts.projectRecentContext(
		context.Background(), &Pipeline{ContextManager: cm}, history, "",
	)

	if len(projected) != len(history) {
		t.Fatalf("dropped %d uncovered messages from a fresh session; first was %q",
			len(history)-len(projected), history[0].Content)
	}
	if len(cm.compacts) != 1 {
		t.Fatalf("made %d coverage requests, want 1", len(cm.compacts))
	}
	if cm.compacts[0].Reason != ContextCompressReasonSummarize {
		t.Errorf("coverage request reason = %q, want summarize", cm.compacts[0].Reason)
	}
	if cm.compacts[0].MessageThreshold != 15 {
		t.Errorf("coverage threshold = %d, want 15: it must lead the window, not the generic 20",
			cm.compacts[0].MessageThreshold)
	}
}

// Once the summarizer has landed, the store itself is small and the window is
// simply satisfied — summary plus the recent conversation.
func TestCoveredHistoryProjectsSummaryPlusRecent(t *testing.T) {
	cm := &recordingContextManager{}
	ts := telegramTurn(15)

	// summarizeSession truncates what it summarized, so a covered session's
	// persisted history is the short tail that survived.
	history := conversation(4)
	projected := ts.projectRecentContext(
		context.Background(), &Pipeline{ContextManager: cm}, history, "a persisted rolling summary",
	)

	if len(projected) != len(history) {
		t.Fatalf("projected %d of %d covered messages, want all", len(projected), len(history))
	}
	if got := countConversationalMessages(projected); got > 14 {
		t.Fatalf("projected %d conversational messages, want <= 14", got)
	}
	if len(cm.compacts) != 0 {
		t.Fatalf("requested coverage %d times for a session already within the window",
			len(cm.compacts))
	}
}

// While summarization lags, the cutoff waits rather than outrunning coverage.
func TestProjectionDoesNotOutrunCoverageWhileSummarizationLags(t *testing.T) {
	cm := &recordingContextManager{}
	ts := telegramTurn(15)

	for _, total := range []int{16, 20, 24, 30} {
		cm.compacts = nil
		history := conversation(total)
		projected := ts.projectRecentContext(
			context.Background(), &Pipeline{ContextManager: cm}, history, "stale summary",
		)
		if len(projected) != len(history) {
			t.Fatalf("history of %d: dropped %d uncovered messages while summarization lagged",
				total, len(history)-len(projected))
		}
		if len(cm.compacts) != 1 {
			t.Fatalf("history of %d: %d coverage requests, want 1", total, len(cm.compacts))
		}
	}
}

// A Compact that errors must not advance coverage, must not restore an
// unbounded transcript, and must not stop the turn.
func TestSummaryFailureDoesNotAdvanceCoverageOrUnboundContext(t *testing.T) {
	cm := &recordingContextManager{err: errSummaryFailed}
	ts := telegramTurn(15)

	held := ts.projectRecentContext(
		context.Background(), &Pipeline{ContextManager: cm}, conversation(20), "previous valid summary",
	)
	if len(held) != 20 {
		t.Fatalf("a failed coverage request dropped %d uncovered messages", 20-len(held))
	}

	// Past the ceiling the cap is enforced rather than growing forever.
	bounded := ts.projectRecentContext(
		context.Background(), &Pipeline{ContextManager: cm}, conversation(200), "previous valid summary",
	)
	if got := countConversationalMessages(bounded); got > 14 {
		t.Fatalf("a permanently failing summarizer left %d messages in context, want <= 14", got)
	}
}

// The hold is bounded. Below the ceiling it waits; above it the cap wins.
func TestUncoveredHoldIsBounded(t *testing.T) {
	cm := &recordingContextManager{}
	ts := telegramTurn(15)
	ceiling := 15 * uncoveredHoldFactor

	below := conversation(ceiling)
	if got := ts.projectRecentContext(
		context.Background(), &Pipeline{ContextManager: cm}, below, ""); len(got) != len(below) {
		t.Fatalf("history of %d (at the ceiling) was cut; the hold must still apply", ceiling)
	}

	above := conversation(ceiling + 2)
	got := ts.projectRecentContext(context.Background(), &Pipeline{ContextManager: cm}, above, "")
	if countConversationalMessages(got) > 14 {
		t.Fatalf("history of %d (past the ceiling) was not bounded", ceiling+2)
	}
}

// A nil manager must not panic or change the outcome of the hold.
func TestCoverageRequestToleratesNoContextManager(t *testing.T) {
	ts := telegramTurn(15)
	history := conversation(16)
	if got := ts.projectRecentContext(context.Background(), nil, history, ""); len(got) != len(history) {
		t.Fatal("the hold must still apply when no context manager is available")
	}
	if got := ts.projectRecentContext(
		context.Background(), &Pipeline{}, history, ""); len(got) != len(history) {
		t.Fatal("the hold must still apply when the pipeline has no context manager")
	}
}

// Other channels never request coverage and are never held or cut.
func TestOtherChannelsNeitherHeldNorCoverageRequested(t *testing.T) {
	for _, channel := range []string{"pocketclaw", "discord", "slack", "matrix", "cli"} {
		cm := &recordingContextManager{}
		ts := telegramTurn(15)
		ts.opts.Channel = channel

		history := conversation(100)
		projected := ts.projectRecentContext(
			context.Background(), &Pipeline{ContextManager: cm}, history, "",
		)
		if len(projected) != len(history) {
			t.Errorf("%s history was modified", channel)
		}
		if len(cm.compacts) != 0 {
			t.Errorf("%s requested summary coverage; only Telegram may", channel)
		}
	}
}
