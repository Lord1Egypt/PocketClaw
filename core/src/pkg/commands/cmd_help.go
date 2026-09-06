package commands

import (
	"context"
	"fmt"
	"strings"
)

func helpCommand() Definition {
	return Definition{
		Name:        "help",
		Description: "Show what PocketClaw can do",
		Usage:       "/help",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			var defs []Definition
			if rt != nil && rt.ListDefinitions != nil {
				defs = rt.ListDefinitions()
			} else {
				defs = BuiltinDefinitions()
			}
			return req.Reply(formatHelpMessage(defs))
		},
	}
}

// helpOrder lists the commands worth putting in front of someone first, in the
// order they are most likely to want them. Anything registered but not named
// here still appears, after these, so a new command cannot go missing.
var helpOrder = []string{
	"stop", "clear", "context", "show", "list", "use", "btw", "subagents", "reload",
}

// formatHelpMessage renders the command overview a user actually reads.
//
// It deliberately does not print EffectiveUsage. That is where the angle
// brackets and pipes live — "/show [model|channel|agents|mcp <server>]" — and
// making someone parse command grammar to find out what PocketClaw can do is
// the thing this replaces. The precise syntax is still one command away: every
// command answers a bare invocation with its own guidance, and a wrong argument
// still gets the exact usage line.
func formatHelpMessage(defs []Definition) string {
	if len(defs) == 0 {
		return "No commands available."
	}

	byName := make(map[string]Definition, len(defs))
	for _, def := range defs {
		byName[def.Name] = def
	}

	lines := []string{"🦞 PocketClaw", ""}
	seen := make(map[string]bool, len(defs))

	appendCommand := func(def Definition) {
		if seen[def.Name] {
			return
		}
		seen[def.Name] = true
		desc := strings.TrimSpace(def.Description)
		if desc == "" {
			return
		}
		lines = append(lines, fmt.Sprintf("/%s — %s", def.Name, desc))
	}

	for _, name := range helpOrder {
		if def, ok := byName[name]; ok {
			appendCommand(def)
		}
	}
	// help and start are real commands but not what someone is looking for in
	// a list of things to do, so they are not promoted into the ordering above.
	for _, def := range defs {
		if def.Name == "help" || def.Name == "start" {
			continue
		}
		appendCommand(def)
	}

	if len(lines) == 2 {
		return "No commands available."
	}
	return strings.Join(lines, "\n")
}
