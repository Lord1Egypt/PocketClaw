package commands

import (
	"context"
	"fmt"
)

func switchCommand() Definition {
	return Definition{
		Name:        "switch",
		Description: "Switch model or channel",
		// No pointer to /list models here on purpose: it reports the current
		// model and tells you to edit config.json rather than enumerating what
		// you could switch to, so sending someone there to browse would be a
		// dead end. A picker belongs to the interactive command UI.
		NoArgsHelp: "🔄 You can switch your model or your channel.\n\n" +
			"To change model, name it after 'to' — for example: " +
			"/switch model to gemini-2.5-flash\n" +
			"To change channel: /switch channel\n\n" +
			"A simpler model picker is coming in the interactive command UI.",
		SubCommands: []SubCommand{
			{
				Name:        "model",
				Description: "Switch to a different model",
				ArgsUsage:   "to <name>",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					if rt == nil || rt.SwitchModel == nil {
						return req.Reply(unavailableMsg)
					}
					// Parse: /switch model to <value>
					value := nthToken(req.Text, 3) // tokens: [/switch, model, to, <value>]
					if nthToken(req.Text, 2) != "to" || value == "" {
						return req.Reply("Usage: /switch model to <name>")
					}
					oldModel, err := rt.SwitchModel(value)
					if err != nil {
						return req.Reply(err.Error())
					}
					return req.Reply(fmt.Sprintf("Switched model from %s to %s", oldModel, value))
				},
			},
			{
				Name:        "channel",
				Description: "Moved to /check channel",
				Handler: func(_ context.Context, req Request, _ *Runtime) error {
					return req.Reply("This command has moved. Please use: /check channel <name>")
				},
			},
		},
	}
}
