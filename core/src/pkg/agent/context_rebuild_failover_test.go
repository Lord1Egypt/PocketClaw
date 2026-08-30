package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// This is the scenario the whole milestone exists to protect.
//
//	a tool executes successfully and its result is persisted
//	→ the next provider request overflows the context window
//	→ history is trimmed and the request rebuilt
//	→ the provider is retried or failed over
//	→ the tool result is still present, and the tool is NOT executed again
//
// The protection is splitHistoryForActiveTurn: the active turn's persisted
// messages become a protected tail that trimming may not touch. If that ever
// stopped holding, a failover after a `git push` would re-send a context
// missing the push result, and the model would reasonably ask for it again.
func TestCompletedToolResultSurvivesContextRebuildAndFailover(t *testing.T) {
	toolResult := providers.Message{
		Role:       "tool",
		Content:    "pushed 1 commit to origin/main",
		ToolCallID: "call_push",
	}
	assistantCall := providers.Message{
		Role:      "assistant",
		Content:   "",
		ToolCalls: []providers.ToolCall{{ID: "call_push", Name: "runtime"}},
	}
	userTurn := providers.Message{Role: "user", Content: "push my work"}

	// Old history that trimming is allowed to discard, then the active turn.
	history := []providers.Message{
		{Role: "user", Content: "something from an earlier turn"},
		{Role: "assistant", Content: "an earlier answer"},
		userTurn,
		assistantCall,
		toolResult,
	}
	persisted := []providers.Message{userTurn, assistantCall, toolResult}

	stable, protected := splitHistoryForActiveTurn(history, persisted)

	if len(protected) != len(persisted) {
		t.Fatalf("the whole active turn must be protected, got %d of %d",
			len(protected), len(persisted))
	}
	if !containsToolResult(protected, "call_push") {
		t.Fatal("the completed tool result must be in the protected tail")
	}
	for _, message := range stable {
		if message.ToolCallID == "call_push" {
			t.Fatal("the active turn's tool result must not be in the trimmable half")
		}
	}

	// Simulate the rebuild: trimming discards the stable half entirely, which
	// is the worst case a context overflow can produce.
	rebuilt := append([]providers.Message(nil), protected...)
	if !containsToolResult(rebuilt, "call_push") {
		t.Fatal("the tool result was lost by the rebuild; a failover would re-request the push")
	}

	// After the rebuild the guard still recognises the call as completed, so a
	// retry cannot re-execute it.
	ts := &turnState{}
	ts.recordCompletedToolResult("call_push", "runtime", toolResult)
	prior, reused := ts.completedToolResultFor("call_push")
	if !reused {
		t.Fatal("a completed call must stay recognised across a context rebuild")
	}
	if !strings.Contains(prior.message.Content, "pushed 1 commit") {
		t.Fatalf("the reused result must be the original, got %q", prior.message.Content)
	}
}

// When nothing of the active turn is persisted yet there is nothing to protect,
// and everything stays trimmable. Getting this wrong in the other direction
// would freeze all history and defeat compaction.
func TestNoProtectedTailWhenTurnHasNotPersistedYet(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "older"},
		{Role: "assistant", Content: "older answer"},
	}

	stable, protected := splitHistoryForActiveTurn(history, nil)
	if len(protected) != 0 {
		t.Fatalf("nothing should be protected, got %d", len(protected))
	}
	if len(stable) != len(history) {
		t.Fatalf("all history should stay trimmable, got %d of %d", len(stable), len(history))
	}
}

func containsToolResult(messages []providers.Message, toolCallID string) bool {
	for _, message := range messages {
		if message.ToolCallID == toolCallID {
			return true
		}
	}
	return false
}
