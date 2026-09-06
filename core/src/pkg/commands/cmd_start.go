package commands

import "context"

func startCommand() Definition {
	return Definition{
		Name:        "start",
		Description: "Start using PocketClaw",
		Usage:       "/start",
		Handler: func(_ context.Context, req Request, _ *Runtime) error {
			return req.Reply("Hello! I am PocketClaw.")
		},
	}
}
