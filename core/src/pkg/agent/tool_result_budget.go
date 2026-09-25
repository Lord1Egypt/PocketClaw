package agent

import (
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// boundToolContentForLLM applies the configured per-result budget and records
// the outcome. Sizes are logged; content never is, because a tool's output is
// exactly where a fetched credential would appear.
func (al *AgentLoop) boundToolContentForLLM(source, content string) tools.ResultBudget {
	maxBytes := al.cfg.Tools.GetMaxResultBytes()
	budget := tools.BoundResultForLLM(content, maxBytes)
	if budget.Truncated {
		logger.InfoCF("agent", "Tool result bounded for the model", map[string]any{
			"tool":            source,
			"original_bytes":  budget.OriginalBytes,
			"delivered_bytes": budget.DeliveredBytes,
			"budget_bytes":    maxBytes,
		})
	}
	return budget
}
