package commands

import (
	"context"
	"strings"
)

type Handler func(ctx context.Context, req Request, rt *Runtime) error

type Request struct {
	Channel  string
	ChatID   string
	SenderID string
	Text     string
	Reply    func(text string) error
	// ReplyMenu answers with text plus a set of choices to tap. It is optional:
	// a caller that cannot render choices leaves it nil, and a command that
	// wants a menu falls back to its text form. That is what lets /model work
	// on a channel without buttons instead of failing there.
	ReplyMenu func(text string, menu *Menu) error
}

// Menu is a set of tappable choices a command can offer.
//
// It is declared here rather than taken from pkg/bus so this package stays
// independent of the transport, as it already is for everything else. The agent
// translates it on the way out.
type Menu struct {
	Rows []MenuRow
}

// MenuRow is one row of buttons.
type MenuRow struct {
	Buttons []MenuButton
}

// MenuButton is one choice. Action and Value are PocketClaw's own terms for
// what tapping means; neither is a credential, an endpoint or a registry key.
type MenuButton struct {
	Label   string
	Action  string
	Value   string
	Current bool
}

// CanReplyMenu reports whether this request can render choices.
func (r Request) CanReplyMenu() bool { return r.ReplyMenu != nil }

const unavailableMsg = "Command unavailable in current context."

var commandPrefixes = []string{"/", "!"}

// parseCommandName accepts "/name", "!name", and Telegram's "/name@bot", then
// normalizes to lowercase command names.
func parseCommandName(input string) (string, bool) {
	token := nthToken(input, 0)
	if token == "" {
		return "", false
	}

	name, ok := trimCommandPrefix(token)
	if !ok {
		return "", false
	}
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}
	name = normalizeCommandName(name)
	if name == "" {
		return "", false
	}
	return name, true
}

// CommandName returns the normalized command name for an input if present.
func CommandName(input string) (string, bool) {
	return parseCommandName(input)
}

func trimCommandPrefix(token string) (string, bool) {
	for _, prefix := range commandPrefixes {
		if strings.HasPrefix(token, prefix) {
			return strings.TrimPrefix(token, prefix), true
		}
	}
	return "", false
}

// HasCommandPrefix returns true if the input starts with a recognized
// command prefix (e.g. "/" or "!").
func HasCommandPrefix(input string) bool {
	token := nthToken(input, 0)
	if token == "" {
		return false
	}
	_, ok := trimCommandPrefix(token)
	return ok
}

// nthToken returns the 0-indexed token from whitespace-split input.
func nthToken(input string, n int) string {
	parts := strings.Fields(strings.TrimSpace(input))
	if n >= len(parts) {
		return ""
	}
	return parts[n]
}

func normalizeCommandName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
