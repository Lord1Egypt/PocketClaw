package commands

import (
	"context"
	"fmt"
	"strings"
)

// ActionSelectModel and ActionCancelMenu are the actions a picker's buttons
// carry. They are PocketClaw's own vocabulary, not platform data.
const (
	ActionSelectModel = "model.select"
	ActionCancelMenu  = "menu.cancel"
)

func modelCommand() Definition {
	return Definition{
		Name:        "model",
		Description: "Choose which model to use",
		Usage:       "/model",
		Instant:     true,
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.GetModelPicker == nil {
				return req.Reply(unavailableMsg)
			}
			picker := rt.GetModelPicker()
			if picker == nil {
				return req.Reply(unavailableMsg)
			}

			if len(picker.Choices) == 0 {
				// Being specific about why beats an empty list: the user has a
				// configuration problem, not a PocketClaw failure.
				return req.Reply("🤖 No models are configured yet.\n\n" +
					"Add a model with an API key in the PocketClaw dashboard, then try /model again.")
			}

			header := modelPickerHeader(picker)

			// A channel without buttons still gets a useful answer rather than
			// an error, and the textual form remains the way to act there.
			if !req.CanReplyMenu() {
				return req.Reply(header + "\n\n" + modelPickerTextFallback(picker))
			}
			return req.ReplyMenu(header+"\n\nChoose a model:", buildMenu(picker.Choices))
		},
	}
}

func modelPickerHeader(picker *ModelPicker) string {
	current := strings.TrimSpace(picker.CurrentModel)
	if current == "" {
		return "🤖 No model is selected yet."
	}
	header := "🤖 Current model\n" + current
	if provider := strings.TrimSpace(picker.CurrentProvider); provider != "" {
		header += "\nProvider: " + provider
	}
	return header
}

// modelPickerTextFallback lists the choices for a channel that cannot render
// buttons, naming the command that acts on them.
func modelPickerTextFallback(picker *ModelPicker) string {
	lines := make([]string, 0, len(picker.Choices)+2)
	lines = append(lines, "Available models:")
	for _, choice := range picker.Choices {
		marker := "•"
		if choice.Current {
			marker = "✓"
		}
		lines = append(lines, fmt.Sprintf("%s %s", marker, choice.Label))
	}
	lines = append(lines, "", "To change: /switch model to "+picker.Choices[0].Name)
	return strings.Join(lines, "\n")
}

// buildMenu renders one model per row, then a cancel row.
//
// One column, not two: configured model names are long enough that pairing them
// truncates both in Telegram, and the list is short because only configured
// models appear.
func buildMenu(choices []ModelChoice) *Menu {
	menu := &Menu{Rows: make([]MenuRow, 0, len(choices)+1)}
	for _, choice := range choices {
		label := choice.Label
		if choice.Current {
			label = "✓ " + label
		}
		menu.Rows = append(menu.Rows, MenuRow{Buttons: []MenuButton{{
			Label:   label,
			Action:  ActionSelectModel,
			Value:   choice.Name,
			Current: choice.Current,
		}}})
	}
	menu.Rows = append(menu.Rows, MenuRow{Buttons: []MenuButton{{
		Label:  "✕ Cancel",
		Action: ActionCancelMenu,
	}}})
	return menu
}
