package commands

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func subagentsCommand() Definition {
	return Definition{
		Name:        "subagents",
		Description: "See what PocketClaw is working on",
		Usage:       "/subagents",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.ListSubagents == nil {
				return req.Reply(unavailableMsg)
			}
			return req.Reply(formatSubagents(rt.ListSubagents()))
		},
	}
}

// formatSubagents renders the active turns for a chat window.
//
// Every field it prints is one the user can act on. The turn also carries their
// own message, their session key and their chat id; those are not in
// SubagentInfo at all, so no amount of editing here can reintroduce them.
func formatSubagents(subagents []SubagentInfo) string {
	if len(subagents) == 0 {
		return "🤖 No active subagents."
	}

	lines := make([]string, 0, len(subagents)*4)
	lines = append(lines, fmt.Sprintf("🤖 Active subagents: %d", len(subagents)))

	for i, sub := range subagents {
		// Numbered rather than named: the only label a turn carries is the
		// user's own prompt, and repeating that back is what this command was
		// leaking. A number is honest and reveals nothing.
		lines = append(lines, "", fmt.Sprintf("%d. Subagent %d", i+1, i+1))
		if status := subagentStatusLabel(sub.Status); status != "" {
			lines = append(lines, "   Status: "+status)
		}
		if sub.Duration > 0 {
			lines = append(lines, "   Duration: "+formatSubagentDuration(sub.Duration))
		}
		if sub.Depth > 0 {
			lines = append(lines, "   Started by another subagent")
		}
	}

	return strings.Join(lines, "\n")
}

// subagentStatusLabel maps a lifecycle phase to a word a user understands, and
// drops anything it does not recognise rather than passing an internal token
// through.
func subagentStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "setup":
		return "Starting"
	case "running":
		return "Running"
	case "tools":
		return "Working"
	case "finalizing":
		return "Finishing"
	case "completed":
		return "Finished"
	case "aborted":
		return "Stopped"
	default:
		return ""
	}
}

// formatSubagentDuration keeps the number short: seconds under a minute, then
// minutes, then hours. A turn that has not started yet has no duration and is
// omitted by the caller rather than shown as a zero.
func formatSubagentDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
}
