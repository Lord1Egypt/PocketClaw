// Package status carries the operational snapshot shown on the PocketClaw
// Status screen.
//
// Every type here is a flat DTO of scalars, built by explicit field-by-field
// mapping from runtime state. Nothing in this package embeds, wraps or
// reflects over an internal struct, so a field added to a turn, a mailbox or a
// channel later cannot reach the Status screen by default. That is the same
// boundary commands.SubagentInfo draws for /subagents, drawn again here
// because Status has a wider audience than one chat window.
//
// What must never appear: user messages, session keys, chat ids, turn ids,
// prompts, replies, reasoning, tool arguments, tool results, raw provider
// errors, API endpoints, keys, tokens, or filesystem paths.
package status

// Snapshot is the whole Status payload.
type Snapshot struct {
	Activity  Activity  `json:"activity"`
	Model     Model     `json:"model"`
	Channels  []Channel `json:"channels"`
	Resources Resources `json:"resources"`
}

// Activity reports turn, subagent and tool activity.
//
// The three terminal counters and both tool counters are cumulative since the
// gateway process started and reset when it restarts; the UI labels them so.
// The three gauges are instantaneous.
type Activity struct {
	// ActiveTurns counts root agent turns executing now (depth 0).
	ActiveTurns int `json:"active_turns"`
	// ActiveSubagents counts sub-turns executing now (depth greater than 0).
	ActiveSubagents int `json:"active_subagents"`
	// Waiting counts inbound messages parked in session mailboxes because the
	// session that owns them is still busy.
	Waiting int `json:"waiting"`

	// Completed, Failed and Cancelled count turns by terminal status.
	Completed uint64 `json:"completed"`
	Failed    uint64 `json:"failed"`
	Cancelled uint64 `json:"cancelled"`

	// ToolCalls counts tool executions that reached a result; ToolCallsFailed
	// counts the subset whose result was an error. Skipped and replayed calls
	// are excluded because they never ran.
	ToolCalls       uint64 `json:"tool_calls"`
	ToolCallsFailed uint64 `json:"tool_calls_failed"`

	// LastActivityUnix is when a turn last ended, in Unix seconds. Zero means
	// no turn has ended in this gateway process, which the UI renders as "—"
	// rather than as the epoch. It carries no channel, session or content.
	LastActivityUnix int64 `json:"last_activity_unix"`
}

// Model reports which model is answering.
//
// ActiveModel is the agent instance's runtime model and ConfiguredModel is the
// configured default. They diverge when the advanced "/switch model to <name>"
// command is used, and Status reports that divergence rather than resolving
// it: model selection belongs to the Dashboard, and Status is read-only.
//
// Provider is a resolved provider name such as "anthropic". No endpoint, key,
// credential id or model config reaches this struct.
type Model struct {
	ActiveModel     string `json:"active_model"`
	ConfiguredModel string `json:"configured_model"`
	Provider        string `json:"provider"`
	FallbackCount   int    `json:"fallback_count"`
}

// Channel reports one configured channel.
//
// The three booleans are distinct facts, not synonyms: Configured means the
// manager holds the channel, Started means Start returned without error and a
// worker exists, and Running is the channel's own flag. A channel that failed
// to start stays configured and is neither started nor running, so it can
// never be presented as running.
//
// There is deliberately no reachability field. Nothing here probes the
// network, so "connected", "healthy" and "reachable" cannot be stated
// truthfully and are not offered.
type Channel struct {
	// Name is the display name, after the pico to pocketclaw mapping. It is a
	// channel identity, never a bot username, account or chat id.
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
	Started    bool   `json:"started"`
	Running    bool   `json:"running"`
}

// Resources reports the Core process's own usage.
//
// Both values are read from the gateway's own /proc entry, so no other
// process is inspected and no Android API is involved. CPUSeconds is
// cumulative processor time since the process started, not a percentage:
// a percentage would need two samples taken apart in time, and Status runs no
// sampler.
type Resources struct {
	MemoryRSSBytes uint64  `json:"memory_rss_bytes"`
	CPUSeconds     float64 `json:"cpu_seconds"`
}

// ReadResources reports the calling process's own memory and CPU usage, or
// zeroes where that is not obtainable.
func ReadResources() Resources { return readResources() }
