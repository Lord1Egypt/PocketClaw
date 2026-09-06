package commands

import (
	"fmt"
	"strings"
)

// SubCommand defines a single sub-command within a parent command.
type SubCommand struct {
	Name        string
	Description string
	ArgsUsage   string // optional, e.g. "<session-id>"
	Handler     Handler
}

// Definition is the single-source metadata and behavior contract for a slash command.
//
// Design notes (phase 1):
//   - Every channel reads command shape from this type instead of keeping local copies.
//   - Visibility is global: all definitions are considered available to all channels.
//   - Platform menu registration (for example Telegram BotCommand) also derives from this
//     same definition so UI labels and runtime behavior stay aligned.
type Definition struct {
	Name        string
	Description string
	Usage       string // for simple commands; ignored when SubCommands is set
	Aliases     []string
	SubCommands []SubCommand // optional; when set, Executor routes to sub-command handlers
	Handler     Handler      // for simple commands without sub-commands

	// Instant marks a command that answers from local state without calling the
	// model. Those need no "Thinking…" placeholder, and showing one for a
	// configuration action makes a setting change look like a conversation.
	Instant bool

	// NoArgsHelp is what a user sees when they send the command with nothing
	// after it. That is the common case for someone exploring, and answering it
	// with "Usage: /show [model|channel|agents|mcp <server>]" asks them to read
	// command grammar before they can do anything.
	//
	// It replaces the usage line only for that case. Genuinely wrong advanced
	// syntax still gets the precise usage string, because there the grammar is
	// the answer.
	NoArgsHelp string
}

// NoArgsMessage returns what to send when the command arrives with no
// sub-command, preferring the human wording when a command supplies one.
func (d Definition) NoArgsMessage() string {
	if help := strings.TrimSpace(d.NoArgsHelp); help != "" {
		return help
	}
	return "Usage: " + d.EffectiveUsage()
}

// EffectiveUsage returns the usage string. When SubCommands are present,
// it is auto-generated from sub-command names so metadata and behavior
// cannot drift.
func (d Definition) EffectiveUsage() string {
	if len(d.SubCommands) == 0 {
		return d.Usage
	}
	names := make([]string, 0, len(d.SubCommands))
	for _, sc := range d.SubCommands {
		name := sc.Name
		if sc.ArgsUsage != "" {
			name += " " + sc.ArgsUsage
		}
		names = append(names, name)
	}
	return fmt.Sprintf("/%s [%s]", d.Name, strings.Join(names, "|"))
}
