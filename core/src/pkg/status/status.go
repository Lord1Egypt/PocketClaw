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
	System    System    `json:"system"`
	Activity  Activity  `json:"activity"`
	Model     Model     `json:"model"`
	Channels  []Channel `json:"channels"`
	Resources Resources `json:"resources"`
}

// System reports gateway-process facts.
type System struct {
	// UptimeSeconds is how long the gateway process has been serving, in whole
	// seconds.
	//
	// It is a number with a stated unit, not a preformatted string. The legacy
	// /health "uptime" field is Go's Duration.String() — "27.707765309s",
	// "1h24m0s" — which no consumer parsed: it was carried across three
	// boundaries as opaque text and printed verbatim, so the screen showed
	// nanosecond precision in an English-only format that no locale could
	// render properly. That field keeps its exact shape for the launcher and
	// the host, which already depend on it; this one is what the Status screen
	// reads, and the formatting decision belongs to the UI.
	UptimeSeconds int64 `json:"uptime_seconds"`
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
	// FallbackCount is how many fallback choices back the active model — the
	// resolved candidate list excluding the model being used, not the length of
	// that list. An agent with no fallbacks reports 0.
	FallbackCount int `json:"fallback_count"`
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

	// CommandsRegistered is deliberately three-valued. Absent means this
	// channel publishes no command menu, which is not the same as a menu that
	// has not been published yet -- and a readiness gate that could not tell
	// those apart would wait forever on a channel that was never going to
	// report. False means not yet, true means the platform accepted it.
	//
	// PC-DEF-061. "Connected" has to mean ready for the owner's first message,
	// and registration is asynchronous, so its outcome has to be observable.
	CommandsRegistered *bool `json:"commands_registered,omitempty"`

	// PollingGeneration is the local id of the polling owner this snapshot
	// describes, when the channel publishes one. Absent means the channel does
	// not poll. PC-DEF-061: readiness must name the generation it authorized so
	// a superseded generation's success cannot be read as the current one's.
	// It is a process-local counter, never an identity.
	PollingGeneration *uint64 `json:"polling_generation,omitempty"`

	// RuntimeFailure is a sanitized terminal code retained after a generation
	// has stopped. Empty means no terminal failure is known.
	RuntimeFailure string `json:"runtime_failure,omitempty"`
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
