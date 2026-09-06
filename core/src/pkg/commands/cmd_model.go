package commands

import "context"

// modelInfoMessage is the whole of /model.
//
// Model selection is a Dashboard operation. The Dashboard owns the configured
// default in config.json, and a chat command that changed it would give one
// setting two places to be set from — which is the confusion this command
// exists to prevent rather than repeat. So /model points at the Dashboard and
// does nothing else: it reads no model state, writes none, and names no model,
// provider or endpoint.
const modelInfoMessage = "🤖 Model selection is managed from PocketClaw Settings."

func modelCommand() Definition {
	return Definition{
		Name:        "model",
		Description: "Manage models from PocketClaw Settings",
		Usage:       "/model",
		Handler: func(_ context.Context, req Request, _ *Runtime) error {
			// The Runtime is ignored on purpose. A command that reads nothing
			// has nothing to report that could disagree with the Dashboard, and
			// a fixed answer cannot leak a configured model or provider.
			return req.Reply(modelInfoMessage)
		},
	}
}
