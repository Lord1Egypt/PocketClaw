// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// completedToolResults records, for the active turn only, the result of every
// tool call that has already run to completion.
//
// This is a defensive guard, not the mechanism that provides correctness. The
// agent loop never rewinds past a completed tool execution: results are
// committed to the turn and the session before the next provider request, and
// a provider retry re-sends those committed results rather than replaying the
// call. The guard exists so that if some future change did allow a turn to be
// re-entered, a completed side effect would not happen twice.
//
// Identity is the provider's own tool_call_id and nothing else. It is
// deliberately not a fingerprint of the tool name and arguments: asking for the
// same command twice in one turn is legitimate — reading a file again after
// editing it, retrying a fetch the model believes was stale — and collapsing
// those into one execution would silently change what the agent did.
type completedToolResults struct {
	mu      sync.Mutex
	results map[string]*completedToolResult
}

type completedToolResult struct {
	toolName string
	message  providers.Message
}

func newCompletedToolResults() *completedToolResults {
	return &completedToolResults{results: make(map[string]*completedToolResult)}
}

// record stores a finished tool result under its provider-supplied call id.
//
// A call with no id is not recorded. Inventing an identity for it, or falling
// back to the arguments, would reintroduce exactly the fingerprint matching
// this design rejects.
func (c *completedToolResults) record(toolCallID, toolName string, message providers.Message) {
	toolCallID = strings.TrimSpace(toolCallID)
	if toolCallID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[toolCallID] = &completedToolResult{toolName: toolName, message: message}
}

// lookup returns a previously completed result for this exact call id.
func (c *completedToolResults) lookup(toolCallID string) (*completedToolResult, bool) {
	toolCallID = strings.TrimSpace(toolCallID)
	if toolCallID == "" {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.results[toolCallID]
	return result, ok
}

// count is the number of distinct completed calls recorded for the turn.
func (c *completedToolResults) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.results)
}

// recordCompletedToolResult stores a finished tool result on the turn.
func (ts *turnState) recordCompletedToolResult(
	toolCallID, toolName string,
	message providers.Message,
) {
	ts.completedToolResultsOnce.Do(func() {
		ts.completedToolResults = newCompletedToolResults()
	})
	ts.completedToolResults.record(toolCallID, toolName, message)
}

// completedToolResultFor returns an already-recorded result for this call id.
func (ts *turnState) completedToolResultFor(toolCallID string) (*completedToolResult, bool) {
	if ts.completedToolResults == nil {
		return nil, false
	}
	return ts.completedToolResults.lookup(toolCallID)
}
