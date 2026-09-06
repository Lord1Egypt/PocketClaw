package commands

import (
	"context"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

type MCPServerInfo struct {
	Name      string
	Enabled   bool
	Deferred  bool
	Connected bool
	ToolCount int
}

type MCPToolParameterInfo struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

type MCPToolInfo struct {
	Name        string
	Description string
	Parameters  []MCPToolParameterInfo
}

// ContextStats describes current session context window usage.
type ContextStats struct {
	UsedTokens        int
	TotalTokens       int // model context window
	HistoryTokens     int // history-only tokens (what maybeSummarize checks)
	CompressAtTokens  int // hard budget compression threshold
	SummarizeAtTokens int // soft summarization trigger
	UsedPercent       int // 0-100
	MessageCount      int
}

// SubagentInfo is the user-facing view of one active turn.
//
// It exists so the agent's internal turn state cannot reach a chat window. The
// runtime used to hand this package the whole *agent.ActiveTurnInfo behind an
// `any`, and /subagents printed it with %+v — which put the user's own prompt,
// their session key and their chat id into Telegram. Those fields are absent
// here by construction rather than by remembering not to print them.
//
// There is deliberately no name field: the only human-readable label the turn
// carries is the user's message, which is exactly what must not be shown.
type SubagentInfo struct {
	// Status is a short lifecycle word such as "Running" or "Waiting".
	Status string
	// Duration is how long the turn has been running. Zero means the start
	// time was not set, and callers must omit it rather than render an epoch.
	Duration time.Duration
	// Depth is 0 for a root turn and greater for a nested one.
	Depth int
}

// StopResult describes the outcome of a stop request for the current session.
type StopResult struct {
	Stopped  bool
	TaskName string
}

// Runtime provides runtime dependencies to command handlers. It is constructed
// per-request by the agent loop so that per-request state (like session scope)
// can coexist with long-lived callbacks (like GetModelInfo).
type Runtime struct {
	Config             *config.Config
	GetModelInfo       func() (name, provider string)
	AskSideQuestion    func(ctx context.Context, question string) (string, error)
	ListAgentIDs       func() []string
	ListDefinitions    func() []Definition
	ListSkillNames     func() []string
	ListMCPServers     func(ctx context.Context) []MCPServerInfo
	ListMCPTools       func(ctx context.Context, serverName string) ([]MCPToolInfo, error)
	GetEnabledChannels func() []string
	// ListSubagents returns the safe view of the turns currently running. It
	// returns a commands-owned type rather than the agent's own struct so the
	// internals cannot be reached, let alone formatted, from here.
	ListSubagents   func() []SubagentInfo
	GetContextStats func() *ContextStats
	SwitchModel     func(value string) (oldModel string, err error)
	SwitchChannel   func(value string) error
	ClearHistory    func() error
	ReloadConfig    func() error
	StopActiveTurn  func() (StopResult, error)
}
