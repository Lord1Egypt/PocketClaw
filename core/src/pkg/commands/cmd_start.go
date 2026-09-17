package commands

import "context"

// StartReplyText is the built-in /start reply. It is exported so the channel
// that delivered the command can recognise the exact reply it produced before
// it treats the user's /start as handled; the two must not drift.
const StartReplyText = "Hello! I am PocketClaw."

func startCommand() Definition {
	return Definition{
		Name:        "start",
		Description: "Start using PocketClaw",
		Usage:       "/start",
		Handler: func(_ context.Context, req Request, _ *Runtime) error {
			return req.Reply(StartReplyText)
		},
	}
}
