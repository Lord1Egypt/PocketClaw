package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func toolResultMessage(toolCallID, content string) providers.Message {
	return providers.Message{Role: "tool", Content: content, ToolCallID: toolCallID}
}

// The guard reuses a result only for the exact same provider-supplied call id.
func TestCompletedToolResultReusedForSameToolCallID(t *testing.T) {
	ts := &turnState{}
	ts.recordCompletedToolResult("call_1", "git", toolResultMessage("call_1", "pushed"))

	prior, ok := ts.completedToolResultFor("call_1")
	if !ok {
		t.Fatal("a completed call must be found by its own id")
	}
	if prior.toolName != "git" || prior.message.Content != "pushed" {
		t.Fatalf("the recorded result must be returned verbatim, got %+v", prior)
	}
}

// This is the property that makes the guard safe to have. The same tool with
// identical arguments under a different call id is a genuinely new request:
// reading a file again after editing it, or retrying a fetch the model believes
// went stale. Collapsing those would silently change what the agent did.
func TestIdenticalToolAndArgsWithDifferentIDStillExecutes(t *testing.T) {
	ts := &turnState{}
	ts.recordCompletedToolResult("call_1", "read_file", toolResultMessage("call_1", "contents"))

	if _, ok := ts.completedToolResultFor("call_2"); ok {
		t.Fatal(
			"a different tool_call_id must not match. Deduplicating on tool name or " +
				"arguments would suppress a legitimate repeat request.",
		)
	}
}

// Without an id there is no identity, and inventing one from the arguments is
// exactly the fingerprint matching this design rejects.
func TestMissingToolCallIDIsNeverDeduplicated(t *testing.T) {
	ts := &turnState{}
	ts.recordCompletedToolResult("", "send_message", toolResultMessage("", "sent"))

	if ts.completedToolResults != nil && ts.completedToolResults.count() != 0 {
		t.Fatal("a call with no id must not be recorded")
	}
	if _, ok := ts.completedToolResultFor(""); ok {
		t.Fatal("an empty id must never match a recorded result")
	}
}

func TestWhitespaceOnlyToolCallIDIsNotIdentity(t *testing.T) {
	ts := &turnState{}
	ts.recordCompletedToolResult("   ", "send_message", toolResultMessage("   ", "sent"))
	if _, ok := ts.completedToolResultFor("   "); ok {
		t.Fatal("a blank id must not be treated as an identity")
	}
}

// A side-effecting tool recorded once must be answered from its result, never
// run again. This is the invariant the whole milestone protects.
func TestSideEffectingToolIsNotReplayed(t *testing.T) {
	ts := &turnState{}
	executions := 0

	execute := func(toolCallID, toolName string) providers.Message {
		if prior, reused := ts.completedToolResultFor(toolCallID); reused {
			return prior.message
		}
		executions++
		message := toolResultMessage(toolCallID, "pushed to origin")
		ts.recordCompletedToolResult(toolCallID, toolName, message)
		return message
	}

	first := execute("call_push", "git")
	// A provider failure between these two would previously have been the
	// dangerous moment; the second call stands in for any path that reaches the
	// same tool call again.
	second := execute("call_push", "git")

	if executions != 1 {
		t.Fatalf("a completed side effect must run exactly once, ran %d times", executions)
	}
	if first.Content != second.Content {
		t.Fatalf("the reused result must be identical: %q vs %q", first.Content, second.Content)
	}
}

func TestDistinctCallsAreRecordedIndependently(t *testing.T) {
	ts := &turnState{}
	ts.recordCompletedToolResult("call_1", "git", toolResultMessage("call_1", "one"))
	ts.recordCompletedToolResult("call_2", "git", toolResultMessage("call_2", "two"))

	if ts.completedToolResults.count() != 2 {
		t.Fatalf("expected two recorded calls, got %d", ts.completedToolResults.count())
	}
	first, _ := ts.completedToolResultFor("call_1")
	second, _ := ts.completedToolResultFor("call_2")
	if first.message.Content == second.message.Content {
		t.Fatal("distinct calls must keep distinct results")
	}
}

// The store is reached from the tool loop and from result recording, which can
// run under different goroutines for async tools.
func TestCompletedToolResultsAreConcurrencySafe(t *testing.T) {
	ts := &turnState{}
	done := make(chan struct{})

	for i := 0; i < 16; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			id := string(rune('a' + n))
			ts.recordCompletedToolResult(id, "tool", toolResultMessage(id, "result"))
			_, _ = ts.completedToolResultFor(id)
		}(i)
	}
	for i := 0; i < 16; i++ {
		<-done
	}
	if ts.completedToolResults.count() != 16 {
		t.Fatalf("expected 16 recorded calls, got %d", ts.completedToolResults.count())
	}
}
