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
// Eligibility comes from pkg/modelaccess, the same rule the Dashboard's model
// list uses, so a model offered here is one the Dashboard would also call
// configured. Entries seeded into model_list without credentials — the thirty
// keyless provider templates DefaultConfig ships — are configured by nobody and
// are not offered.
func listSelectableModels(cfg *config.Config, currentModel string) []selectableModel {
	if cfg == nil {
		return nil
	}

	seen := make(map[string]bool, len(cfg.ModelList))
	models := make([]selectableModel, 0, len(cfg.ModelList))

	for _, entry := range cfg.ModelList {
		if entry == nil || !entry.Enabled {
			continue
		}
		name := strings.TrimSpace(entry.ModelName)
		if name == "" || seen[name] {
			continue
		}
		if !modelaccess.IsConfigured(entry) {
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
		return bus.MenuActionResult{Message: "Cancelled."}
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
		// working provider for nothing.
		return bus.MenuActionResult{
			Message: "Already using " + chosen.Label + ".",
			Menu:    buildModelMenu(models),
		}
	}

	if _, err := switchAgentModel(cfg, agent, chosen.Name); err != nil {
		// The provider's own words are not shown: this path is reached by
		// tapping a button, and a failure here is PocketClaw's to explain.
		logMenuActionFailure(req, err)
		return bus.MenuActionResult{Message: "Could not switch to " + chosen.Label + "."}
	}

	return bus.MenuActionResult{
		Message: "✅ Switched to " + chosen.Label,
		Changed: true,
		Menu:    buildModelMenu(listSelectableModels(cfg, agent.Model)),
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
