package agent

import (
	"context"
	"errors"
	"net/http"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/common"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// CodeContextBudgetExceeded is raised when the provider refused a request as
// too large and the one compact-and-resend could not make it fit. It is a
// stable code; see UserFacingError.
const CodeContextBudgetExceeded = "PC-E-CTX-001"

func newContextBudgetExceeded(cause error) *UserFacingError {
	return &UserFacingError{
		Code: CodeContextBudgetExceeded,
		Message: "This request no longer fits the AI model's context window, even after " +
			"PocketClaw shortened tool output and older history. Send /clear to start a " +
			"fresh conversation, or ask for a smaller part of the task.",
		cause: cause,
	}
}

// minShrunkToolResultBytes is the smallest a tool result is cut to while
// making a request fit. Below this a result stops saying anything useful, and
// the remaining overflow belongs to history compaction instead.
const minShrunkToolResultBytes = 2 * 1024

// isProviderContextOverflow reports whether a failed request was refused for
// being too large, so that compacting and re-sending it could succeed.
//
// A 400 carries no meaning of its own: malformed tool schemas, unknown
// parameters and bad model names all arrive as 400 too, and re-sending a
// shorter version of a malformed request only hides the real error. So a
// status that means something else — authentication, billing, quota, rate
// limiting, a missing model, a server fault — is never read as overflow, and
// a 400 counts only when the provider's own words say the request was too big.
// 413 is the one status that says so by itself.
func isProviderContextOverflow(err error, failErr *providers.FailoverError) bool {
	if err == nil {
		return false
	}
	status := 0
	if failErr != nil {
		status = failErr.Status
	}
	var httpErr *common.HTTPError
	if errors.As(err, &httpErr) && httpErr != nil {
		status = httpErr.StatusCode
	}
	switch status {
	case http.StatusRequestEntityTooLarge:
		return true
	case 0, http.StatusBadRequest:
	default:
		return false
	}
	if failErr != nil {
		switch failErr.Reason {
		case providers.FailoverAuth, providers.FailoverBilling, providers.FailoverHardQuota,
			providers.FailoverRateLimit:
			return false
		}
	}
	return providers.IsContextOverflowMessage(err.Error())
}

// estimateRequestTokens is the same estimate isOverContextBudget compares.
func estimateRequestTokens(
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	maxTokens int,
) int {
	total := EstimateToolDefsTokens(toolDefs) + maxTokens
	for _, m := range messages {
		total += EstimateMessageTokens(m)
	}
	return total
}

// shrinkToolResultsToFit halves the largest tool result, repeatedly, until
// fits reports true or no result is left above minShrunkToolResultBytes.
//
// Only role "tool" messages are touched: they are the one part of a request
// that is machine output and can be cut without changing what anybody said.
// Each cut goes through tools.BoundResultForLLM, so a shrunk result still
// states that it was truncated and keeps its beginning and end. It returns
// how many cuts were made and whether the messages now fit.
func shrinkToolResultsToFit(messages []providers.Message, fits func() bool) (int, bool) {
	cuts := 0
	for !fits() {
		largest := -1
		for i := range messages {
			if messages[i].Role != "tool" || len(messages[i].Content) <= minShrunkToolResultBytes {
				continue
			}
			if largest < 0 || len(messages[i].Content) > len(messages[largest].Content) {
				largest = i
			}
		}
		if largest < 0 {
			return cuts, false
		}
		target := len(messages[largest].Content) / 2
		if target < minShrunkToolResultBytes {
			target = minShrunkToolResultBytes
		}
		messages[largest].Content = tools.BoundResultForLLM(messages[largest].Content, target).Content
		cuts++
	}
	return cuts, true
}

// preflightContextBudget checks that the request about to be sent fits the
// model's context window, and makes it fit when it does not.
//
// It runs before every provider request, not only when a turn starts: a turn
// that calls tools grows between requests, and a single large tool result
// used to reach the provider unchecked. The remedies are applied cheapest
// first — cut tool output, then compact and trim history — and the request is
// re-measured after each.
//
// What is left after every remedy is irreducible: the system prompt, the
// user's message, the model's own tool calls and results already cut to
// minShrunkToolResultBytes. That is measured against an estimate of a window
// that is itself often a heuristic (four times max_tokens when unset), and
// such requests were always sent and routinely accepted. So it is sent, with a
// warning, and the provider's answer decides; a genuine overflow then gets the
// single compact-and-resend in CallLLM and, failing that, PC-E-CTX-001.
func (p *Pipeline) preflightContextBudget(
	ctx context.Context,
	ts *turnState,
	exec *turnExecution,
	iteration int,
) {
	window := ts.agent.ContextWindow
	if window <= 0 {
		return
	}
	fits := func() bool {
		return !isOverContextBudget(window, exec.callMessages, exec.providerToolDefs, ts.agent.MaxTokens)
	}
	if fits() {
		return
	}

	before := estimateRequestTokens(exec.callMessages, exec.providerToolDefs, ts.agent.MaxTokens)
	fields := map[string]any{
		"agent_id":         ts.agent.ID,
		"iteration":        iteration,
		"estimated_tokens": before,
		"context_window":   window,
		"max_tokens":       ts.agent.MaxTokens,
	}
	logger.WarnCF("agent", "Context preflight: request exceeds the context budget", fields)

	if cuts, ok := shrinkToolResultsToFit(exec.callMessages, fits); ok {
		fields["tool_results_cut"] = cuts
		fields["estimated_tokens_after"] = estimateRequestTokens(
			exec.callMessages, exec.providerToolDefs, ts.agent.MaxTokens)
		logger.InfoCF("agent", "Context preflight: fitted by shortening tool results", fields)
		return
	}

	// Iteration 1 was compacted by SetupTurn when it was over budget; later
	// iterations compact at most once per turn.
	if !ts.opts.NoHistory && iteration > 1 && !exec.historyCompactedForBudget {
		exec.historyCompactedForBudget = true
		p.rebuildContextWithinBudget(ctx, ts, exec, ContextCompressReasonProactive, iteration)
		if !fits() {
			shrinkToolResultsToFit(exec.callMessages, fits)
		}
		if fits() {
			fields["estimated_tokens_after"] = estimateRequestTokens(
				exec.callMessages, exec.providerToolDefs, ts.agent.MaxTokens)
			logger.InfoCF("agent", "Context preflight: fitted by compacting history", fields)
			return
		}
	}

	fields["estimated_tokens_after"] = estimateRequestTokens(
		exec.callMessages, exec.providerToolDefs, ts.agent.MaxTokens)
	logger.WarnCF("agent", "Context preflight: only irreducible content remains over budget; "+
		"sending and letting the provider decide", fields)
}

// rebuildContextWithinBudget compacts the session, re-assembles the prompt and
// trims the oldest complete turns until the request fits. The active turn —
// the user message and every tool call and result since — is never dropped;
// only history before it is. It reports whether the rebuilt request fits.
func (p *Pipeline) rebuildContextWithinBudget(
	ctx context.Context,
	ts *turnState,
	exec *turnExecution,
	reason ContextCompressReason,
	attempt int,
) bool {
	maxMediaSize := p.Cfg.Agents.Defaults.GetMaxMediaSize()
	if compactErr := p.ContextManager.Compact(ctx, &CompactRequest{
		SessionKey: ts.sessionKey,
		Reason:     reason,
		Budget:     ts.agent.ContextWindow,
	}); compactErr != nil {
		logger.WarnCF("agent", "Context compaction failed", map[string]any{
			"session_key": ts.sessionKey,
			"reason":      string(reason),
			"error":       compactErr.Error(),
		})
	}
	ts.refreshRestorePointFromSession(ts.agent)
	if asmResp, asmErr := p.ContextManager.Assemble(ctx, &AssembleRequest{
		SessionKey: ts.sessionKey,
		Budget:     ts.agent.ContextWindow,
		MaxTokens:  ts.agent.MaxTokens,
	}); asmErr == nil && asmResp != nil {
		exec.history = asmResp.History
		exec.summary = asmResp.Summary
	}
	contextualSkills := ts.activeSkills
	if ts.agent.ContextBuilder != nil {
		contextualSkills = ts.agent.ContextBuilder.ResolveActiveSkillsForContext(ts.activeSkills)
	}
	ts.recordSkillContextSnapshot(skillContextTriggerContextRetryRebuild, contextualSkills)
	stableHistory, protectedTurnTail := splitHistoryForActiveTurn(
		exec.history,
		ts.persistedMessagesSnapshot(),
	)
	buildMessages := func(trimmedHistory []providers.Message) []providers.Message {
		fullHistory := append(append([]providers.Message(nil), trimmedHistory...), protectedTurnTail...)
		rebuildPromptReq := promptBuildRequestForTurn(ts, fullHistory, exec.summary, "", nil, p.Cfg)
		rebuildPromptReq.ActiveSkills = append([]string(nil), contextualSkills...)
		rebuilt := ts.agent.ContextBuilder.BuildMessagesFromPrompt(rebuildPromptReq)
		return resolveMediaRefs(
			rebuilt,
			p.MediaStore,
			maxMediaSize,
			len(rebuilt)-len(protectedTurnTail),
		)
	}
	originalHistoryCount := len(exec.history)
	var fit bool
	var trimmedStableHistory []providers.Message
	trimmedStableHistory, exec.callMessages, fit = trimHistoryToFitContextWindow(
		stableHistory,
		func(trimmedHistory []providers.Message) []providers.Message {
			rebuilt := buildMessages(trimmedHistory)
			if exec.gracefulTerminal {
				return append(append([]providers.Message(nil), rebuilt...), ts.interruptHintMessage())
			}
			return rebuilt
		},
		ts.agent.ContextWindow,
		exec.providerToolDefs,
		ts.agent.MaxTokens,
	)
	exec.history = append(trimmedStableHistory, protectedTurnTail...)
	exec.messages = buildMessages(trimmedStableHistory)
	exec.currentTurnStart = len(exec.messages) - len(protectedTurnTail)
	if exec.gracefulTerminal {
		msgs := append([]providers.Message(nil), exec.messages...)
		exec.callMessages = append(msgs, ts.interruptHintMessage())
	} else {
		// callMessages and messages must stay one slice, so a tool result cut
		// to fit this request stays cut for the rest of the turn.
		exec.callMessages = exec.messages
	}
	if dropped := originalHistoryCount - len(exec.history); dropped > 0 {
		logger.WarnCF("agent", "Trimmed rebuilt history to fit the context window", map[string]any{
			"session_key":     ts.sessionKey,
			"reason":          string(reason),
			"attempt":         attempt,
			"dropped_msgs":    dropped,
			"remaining_msgs":  len(exec.history),
			"context_window":  ts.agent.ContextWindow,
			"max_tokens":      ts.agent.MaxTokens,
			"still_overlimit": !fit,
		})
	} else if !fit {
		logger.WarnCF("agent", "Context still exceeds budget after compaction rebuild", map[string]any{
			"session_key":         ts.sessionKey,
			"reason":              string(reason),
			"attempt":             attempt,
			"history_msgs":        len(exec.history),
			"protected_turn_msgs": len(protectedTurnTail),
			"context_window":      ts.agent.ContextWindow,
			"max_tokens":          ts.agent.MaxTokens,
		})
	}
	return fit
}
