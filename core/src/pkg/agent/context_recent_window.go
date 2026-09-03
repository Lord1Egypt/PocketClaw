package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// DefaultTelegramRecentContextMessages is how many conversational messages a
// Telegram turn may carry into the model, counting the current inbound message.
//
// It bounds the prompt without touching what is stored: the transcript on disk
// and the conversation in Telegram are untouched, and only the projection sent
// to the provider is capped. Anything older is represented by the rolling
// summary the context manager already maintains.
//
// A future Settings surface offering 10 / 15 / 20 / Custom sets
// `agents.defaults.telegram_recent_context_messages`; nothing in the context
// engine needs to change for that.
const DefaultTelegramRecentContextMessages = 15

// recentContextLimit is the number of conversational messages this turn may
// carry, or 0 for no limit.
//
// The cap is Telegram-only on purpose. Telegram is the channel where a burst of
// separate messages each becomes its own queued request, so its transcripts grow
// in a way the other channels' do not. Every other channel keeps the unbounded
// history it has today, and the budget-driven compression that already governs
// all of them is unchanged.
func recentContextLimit(agent *AgentInstance, channel string) int {
	if agent == nil {
		return 0
	}
	if !strings.EqualFold(strings.TrimSpace(channel), "telegram") {
		return 0
	}
	if agent.TelegramRecentContextMessages <= 0 {
		return 0
	}
	return agent.TelegramRecentContextMessages
}

// isConversationalMessage reports whether a stored message counts against the
// recent-context budget.
//
// A tool call and its result are machinery belonging to the assistant turn that
// issued them, not conversation, so they ride along with the turn they belong to
// and do not consume budget of their own. Counting them would let a single
// tool-heavy turn evict the whole conversation.
func isConversationalMessage(msg providers.Message) bool {
	switch msg.Role {
	case "user":
		return true
	case "assistant":
		return len(msg.ToolCalls) == 0 && msg.ToolCallID == ""
	default:
		return false
	}
}

func countConversationalMessages(history []providers.Message) int {
	count := 0
	for _, msg := range history {
		if isConversationalMessage(msg) {
			count++
		}
	}
	return count
}

// projectRecentHistory returns the tail of history holding at most limit
// conversational messages, together with how many messages were left behind.
//
// The cut always lands on a turn boundary — a user message — which is the same
// rule forceCompression and summarizeSession already use. That is what keeps a
// tool interaction whole: a slice that begins at a user message can never open
// with a tool result whose call was dropped, and can never end mid-sequence
// because it runs to the end of the history.
//
// Among the boundaries that fit, the earliest is chosen, so the model keeps as
// much conversation as the limit allows rather than the bare minimum.
func projectRecentHistory(
	history []providers.Message,
	limit int,
) (projected []providers.Message, evicted int) {
	if limit <= 0 || len(history) == 0 {
		return history, 0
	}
	if countConversationalMessages(history) <= limit {
		return history, 0
	}

	for _, start := range parseTurnBoundaries(history) {
		if countConversationalMessages(history[start:]) <= limit {
			return history[start:], start
		}
	}

	// No boundary produces a small enough tail — a single turn already exceeds
	// the limit on its own. Keep that last turn whole rather than tearing it:
	// an over-budget but valid turn is recoverable, a torn one is not, and the
	// token-budget compression that runs after this still applies.
	if starts := parseTurnBoundaries(history); len(starts) > 0 {
		start := starts[len(starts)-1]
		return history[start:], start
	}
	return history, 0
}

// projectRecentContext bounds what this turn sends to the model.
//
// It runs after Assemble and before the prompt is built, which is the only
// point where the whole assembled history is in hand and nothing has been
// written back. It is a projection and nothing else: the session store is not
// touched, no message is deleted, edited or retracted, and the transcript the
// user sees in Telegram is unaffected.
//
// The current inbound message is not in history yet — the prompt builder
// appends it — so the budget reserves a slot for it.
func (ts *turnState) projectRecentContext(
	history []providers.Message,
	summary string,
) []providers.Message {
	limit := recentContextLimit(ts.agent, ts.opts.Channel)
	if limit <= 0 {
		return history
	}

	historyBudget := limit - 1
	if historyBudget < 0 {
		historyBudget = 0
	}

	projected, evicted := projectRecentHistory(history, historyBudget)
	if evicted == 0 {
		return projected
	}

	// Counts and sizes only. No message content, no prompt, no reasoning.
	logger.DebugCF("agent", "Bounded Telegram model context", map[string]any{
		"channel":                 ts.opts.Channel,
		"session_key":             ts.sessionKey,
		"history_total":           len(history),
		"recent_context_messages": countConversationalMessages(projected) + 1,
		"summarized_or_evicted":   evicted,
		"summary_present":         strings.TrimSpace(summary) != "",
		"summary_chars":           len(summary),
		"limit":                   limit,
	})
	return projected
}
