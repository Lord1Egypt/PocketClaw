package agent

import (
	"context"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/modelaccess"
)

// MenuActionSelectModel is the action a model button carries.
const MenuActionSelectModel = "model.select"

// MenuActionCancel closes a picker without changing anything.
const MenuActionCancel = "menu.cancel"

// selectableModel is one entry a user may choose.
type selectableModel struct {
	// Name is what the switch operation takes — the same string a user would
	// type after "/switch model to". It is a product-facing model name, not a
	// registry key, and it never contains a credential or an endpoint.
	Name string
	// Label is what the reader sees.
	Label   string
	Current bool
}

// listSelectableModels returns the models a user may switch to.
//
// Eligibility is modelaccess.IsSelectable, which builds on the same configured
// rule the Dashboard uses and then asks the stricter question a picker needs:
// can this be switched to now. Seeded provider templates are excluded whether
// they are keyless or probe-based, because every button here has to mean a
// model PocketClaw can actually select.
func listSelectableModels(cfg *config.Config, currentModel string) []selectableModel {
	if cfg == nil {
		return nil
	}

	seen := make(map[string]bool, len(cfg.ModelList))
	models := make([]selectableModel, 0, len(cfg.ModelList))

	for _, entry := range cfg.ModelList {
		if entry == nil || !modelaccess.IsSelectable(entry) {
			continue
		}
		name := strings.TrimSpace(entry.ModelName)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		models = append(models, selectableModel{
			Name:    name,
			Label:   name,
			Current: strings.EqualFold(name, strings.TrimSpace(currentModel)),
		})
	}

	sort.Slice(models, func(i, j int) bool {
		return strings.ToLower(models[i].Label) < strings.ToLower(models[j].Label)
	})
	return models
}

// buildModelMenu renders the picker: one model per row, then a cancel row.
//
// One column rather than two: configured model names are long enough that
// pairing them truncates both, and PocketClaw shows only configured models so
// the list is short. A provider tier would only be worth its extra tap if that
// stopped being true.
func buildModelMenu(models []selectableModel) *bus.InteractiveMenu {
	if len(models) == 0 {
		return nil
	}

	menu := &bus.InteractiveMenu{Rows: make([]bus.MenuRow, 0, len(models)+1)}
	for _, model := range models {
		label := model.Label
		if model.Current {
			label = "✓ " + label
		}
		menu.Rows = append(menu.Rows, bus.MenuRow{Buttons: []bus.MenuButton{{
			Label:   label,
			Action:  MenuActionSelectModel,
			Value:   model.Name,
			Current: model.Current,
		}}})
	}
	menu.Rows = append(menu.Rows, bus.MenuRow{Buttons: []bus.MenuButton{{
		Label:  "✕ Cancel",
		Action: MenuActionCancel,
	}}})
	return menu
}

// RunMenuAction performs an action a user tapped. It implements
// bus.MenuActionDelegate.
//
// The channel has already established who the user is; this re-derives the
// agent the same way an ordinary message would, and re-checks the request
// against current configuration. A menu built minutes ago is a claim about the
// past, so nothing it says is trusted here.
func (al *AgentLoop) RunMenuAction(ctx context.Context, req bus.MenuActionRequest) bus.MenuActionResult {
	switch req.Action {
	case MenuActionCancel:
		// The picker becomes a closed card rather than a headless one. Clearing
		// only the buttons left the body still saying "Choose a model:", which
		// reads as a live picker that has stopped working.
		return bus.MenuActionResult{
			Message: "Cancelled.",
			Text:    "✕ Model selection cancelled.",
		}
	case MenuActionSelectModel:
		return al.runSelectModelAction(req)
	default:
		return bus.MenuActionResult{Message: "That option is no longer available."}
	}
}

func (al *AgentLoop) runSelectModelAction(req bus.MenuActionRequest) bus.MenuActionResult {
	cfg := al.GetConfig()
	if cfg == nil {
		return bus.MenuActionResult{Message: "Configuration is not available right now."}
	}

	agent := al.agentForMenuAction(req)
	if agent == nil {
		return bus.MenuActionResult{Message: "No agent is available to change."}
	}

	// Re-check eligibility now rather than trusting the menu. A model can be
	// disabled or have its credentials removed between building the picker and
	// tapping it, and the tap must not resurrect it.
	models := listSelectableModels(cfg, agent.Model)
	var chosen *selectableModel
	for i := range models {
		if strings.EqualFold(models[i].Name, strings.TrimSpace(req.Value)) {
			chosen = &models[i]
			break
		}
	}
	if chosen == nil {
		return bus.MenuActionResult{Message: "That model is no longer available. Send /model to see the current list."}
	}

	if chosen.Current {
		// Tapping what is already active is not an error and must not rebind a
		// working provider for nothing. The picker is left exactly as it is.
		return bus.MenuActionResult{Message: "Already using " + chosen.Label}
	}

	if _, err := switchAgentModel(cfg, agent, chosen.Name); err != nil {
		// The provider's own words are not shown: this path is reached by
		// tapping a button, and a failure here is PocketClaw's to explain.
		logMenuActionFailure(req, err)
		return bus.MenuActionResult{Message: "Could not switch to " + chosen.Label + "."}
	}

	// The picker updates in place: its header names the new model and the tick
	// moves. No second chat message — this is configuration, and a switch does
	// not belong in the conversation as a separate entry.
	updated := listSelectableModels(cfg, agent.Model)
	return bus.MenuActionResult{
		Message: "Switched to " + chosen.Label,
		Changed: true,
		Text:    modelPickerText(agent, cfg, updated),
		Menu:    buildModelMenu(updated),
	}
}

// agentForMenuAction resolves the agent a tapped button applies to, using the
// same routing an ordinary message from that chat would take.
func (al *AgentLoop) agentForMenuAction(req bus.MenuActionRequest) *AgentInstance {
	_, agent, err := al.resolveMessageRoute(bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  req.Channel,
			ChatID:   req.ChatID,
			SenderID: req.SenderID,
		},
		Channel:  req.Channel,
		ChatID:   req.ChatID,
		SenderID: req.SenderID,
	})
	if err != nil || agent == nil {
		if registry := al.GetRegistry(); registry != nil {
			return registry.GetDefaultAgent()
		}
		return nil
	}
	return agent
}

// logMenuActionFailure records why a tap failed, for the developer log only.
// The user is told something short and safe; the detail stays here.
func logMenuActionFailure(req bus.MenuActionRequest, err error) {
	logger.WarnCF("agent", "Menu action failed", map[string]any{
		"channel": req.Channel,
		"action":  req.Action,
		"error":   err.Error(),
	})
}

// publishMenuResponse delivers a command's answer together with its choices.
//
// It goes out on the same outbound path as any other reply, carrying the menu
// as an optional field. A channel that cannot render choices sends the text and
// ignores the rest, which is why /model degrades rather than fails elsewhere.
func (al *AgentLoop) publishMenuResponse(
	ctx context.Context,
	inbound bus.InboundMessage,
	text string,
	menu *commands.Menu,
) error {
	return al.publishResponseWithMenu(
		ctx, inbound.Channel, inbound.ChatID, inbound.SessionKey,
		&inbound.Context, text, busMenuFromCommandMenu(menu),
	)
}

// busMenuFromCommandMenu translates the command package's transport-independent
// menu into the bus payload.
func busMenuFromCommandMenu(menu *commands.Menu) *bus.InteractiveMenu {
	if menu == nil || len(menu.Rows) == 0 {
		return nil
	}
	out := &bus.InteractiveMenu{Rows: make([]bus.MenuRow, 0, len(menu.Rows))}
	for _, row := range menu.Rows {
		buttons := make([]bus.MenuButton, 0, len(row.Buttons))
		for _, button := range row.Buttons {
			buttons = append(buttons, bus.MenuButton{
				Label:   button.Label,
				Action:  button.Action,
				Value:   button.Value,
				Current: button.Current,
			})
		}
		out.Rows = append(out.Rows, bus.MenuRow{Buttons: buttons})
	}
	return out
}

// modelPickerText renders the picker body, so an updated picker reads exactly
// like a freshly requested one.
func modelPickerText(agent *AgentInstance, cfg *config.Config, models []selectableModel) string {
	header := "🤖 Current model\n" + agent.Model
	if provider := resolvedCandidateProvider(agent.Candidates, cfg.Agents.Defaults.Provider); provider != "" {
		header += "\nProvider: " + provider
	}
	if len(models) == 0 {
		return header
	}
	return header + "\n\nChoose a model:"
}

// MenuTextPickerClosed retires a picker that a newer one has replaced. It is
// stated here, with the rest of the menu wording, rather than in the channel.
const MenuTextPickerClosed = "Model selection closed."
