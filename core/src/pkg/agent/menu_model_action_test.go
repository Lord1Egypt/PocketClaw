package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func menuTestConfig(t *testing.T, current string, entries ...*config.ModelConfig) *config.Config {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.ModelName = current
	cfg.ModelList = entries
	return cfg
}

func configuredEntry(name, provider, model string) *config.ModelConfig {
	return &config.ModelConfig{
		ModelName: name,
		Provider:  provider,
		Model:     model,
		Enabled:   true,
		APIKeys:   config.SecureStrings{config.NewSecureString("key-" + name)},
	}
}

func unconfiguredEntry(name, provider, model string) *config.ModelConfig {
	return &config.ModelConfig{
		ModelName: name,
		Provider:  provider,
		Model:     model,
		Enabled:   true,
	}
}

// A + B + D. The picker offers configured models, marks the active one, and
// leaves keyless entries out.
func TestSelectableModelsAreConfiguredOnly(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha",
		configuredEntry("Alpha", "openai", "gpt-alpha"),
		configuredEntry("Beta", "gemini", "gemini-beta"),
		unconfiguredEntry("Seeded", "groq", "groq-seeded"),
	)

	models := listSelectableModels(cfg, "Alpha")

	names := map[string]bool{}
	current := 0
	for _, model := range models {
		names[model.Name] = true
		if model.Current {
			current++
		}
	}
	if !names["Alpha"] || !names["Beta"] {
		t.Fatalf("configured models missing: %v", names)
	}
	if names["Seeded"] {
		t.Fatal("a keyless seeded entry was offered")
	}
	if current != 1 {
		t.Fatalf("exactly one model should be current, got %d", current)
	}
}

// O + P. A tap performs the same switch the textual command performs, on the
// same agent state.
func TestMenuTapSwitchesTheAgentModel(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha",
		configuredEntry("Alpha", "openai", "gpt-alpha"),
		configuredEntry("Beta", "openai", "gpt-beta"),
	)
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()

	agent := loop.registry.GetDefaultAgent()
	if agent.Model != "Alpha" {
		t.Fatalf("starting model = %q, want Alpha", agent.Model)
	}

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionSelectModel, Value: "Beta",
	})

	if !result.Changed {
		t.Fatalf("the tap reported no change: %+v", result)
	}
	if agent.Model != "Beta" {
		t.Fatalf("agent model = %q, want Beta — the tap did not use the switch operation", agent.Model)
	}
	if !strings.Contains(result.Message, "Beta") {
		t.Fatalf("confirmation does not name the model: %q", result.Message)
	}
	// The replacement picker marks the new selection.
	if result.Menu == nil {
		t.Fatal("no replacement picker; the old one would still claim Alpha")
	}
	marked := ""
	for _, row := range result.Menu.Rows {
		for _, button := range row.Buttons {
			if button.Current {
				marked = button.Value
			}
		}
	}
	if marked != "Beta" {
		t.Fatalf("replacement picker marks %q as current, want Beta", marked)
	}
}

// R. Tapping the active model is harmless and does not rebind a working
// provider for nothing.
func TestTappingTheActiveModelIsHarmless(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha",
		configuredEntry("Alpha", "openai", "gpt-alpha"),
		configuredEntry("Beta", "openai", "gpt-beta"),
	)
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()

	agent := loop.registry.GetDefaultAgent()
	before := agent.Provider

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionSelectModel, Value: "Alpha",
	})

	if result.Changed {
		t.Fatal("re-selecting the active model reported a change")
	}
	if agent.Provider != before {
		t.Fatal("re-selecting the active model rebuilt its provider")
	}
	if strings.Contains(strings.ToLower(result.Message), "error") ||
		strings.Contains(strings.ToLower(result.Message), "could not") {
		t.Fatalf("re-selecting should not read as a failure: %q", result.Message)
	}
}

// S. Eligibility is re-checked at tap time. A menu is a claim about the past.
func TestModelThatBecameUnconfiguredCannotBeSelected(t *testing.T) {
	beta := configuredEntry("Beta", "openai", "gpt-beta")
	cfg := menuTestConfig(t, "Alpha", configuredEntry("Alpha", "openai", "gpt-alpha"), beta)
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()
	agent := loop.registry.GetDefaultAgent()

	// The picker was built while Beta was configured; its credentials are then
	// removed before the tap arrives.
	beta.APIKeys = nil

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionSelectModel, Value: "Beta",
	})

	if result.Changed || agent.Model != "Alpha" {
		t.Fatalf("a model that lost its configuration was still selected; model=%q", agent.Model)
	}
	if !strings.Contains(result.Message, "no longer available") {
		t.Fatalf("unhelpful rejection: %q", result.Message)
	}
}

// A value that was never offered cannot be smuggled in through the action.
func TestUnknownModelValueIsRejected(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha", configuredEntry("Alpha", "openai", "gpt-alpha"))
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()
	agent := loop.registry.GetDefaultAgent()

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionSelectModel, Value: "../../etc/passwd",
	})
	if result.Changed || agent.Model != "Alpha" {
		t.Fatal("an unoffered value changed the model")
	}
}

// T. Cancel changes nothing.
func TestCancelChangesNoModelState(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha",
		configuredEntry("Alpha", "openai", "gpt-alpha"),
		configuredEntry("Beta", "openai", "gpt-beta"),
	)
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()
	agent := loop.registry.GetDefaultAgent()

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionCancel,
	})
	if result.Changed || agent.Model != "Alpha" {
		t.Fatal("cancel changed model state")
	}
	// Repeating it is equally harmless.
	if again := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionCancel,
	}); again.Changed {
		t.Fatal("a repeated cancel reported a change")
	}
}

// An unrecognised action does nothing rather than falling through to something.
func TestUnknownMenuActionIsRejected(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha", configuredEntry("Alpha", "openai", "gpt-alpha"))
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()
	agent := loop.registry.GetDefaultAgent()

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: "something.else", Value: "Alpha",
	})
	if result.Changed || agent.Model != "Alpha" {
		t.Fatal("an unknown action changed state")
	}
}

// Q. The textual form still performs the same switch.
func TestTextualSwitchRemainsBackwardCompatible(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha",
		configuredEntry("Alpha", "openai", "gpt-alpha"),
		configuredEntry("Beta", "openai", "gpt-beta"),
	)
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()
	agent := loop.registry.GetDefaultAgent()

	old, err := switchAgentModel(cfg, agent, "Beta")
	if err != nil {
		t.Fatalf("textual switch failed: %v", err)
	}
	if old != "Alpha" || agent.Model != "Beta" {
		t.Fatalf("textual switch did not apply: old=%q now=%q", old, agent.Model)
	}
}

// A failed switch tells the user something safe, not the provider's words.
func TestFailedSwitchDoesNotLeakInternals(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha", configuredEntry("Alpha", "openai", "gpt-alpha"))
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionSelectModel, Value: "Missing",
	})
	for _, forbidden := range []string{"&{", "http://", "https://", "api_key", "sk-", "Error:"} {
		if strings.Contains(result.Message, forbidden) {
			t.Fatalf("failure message leaked %q: %q", forbidden, result.Message)
		}
	}
}

// A picker must not offer a local runtime nobody has established exists. The
// seeded probe templates ship disabled; only a materialized one is offered.
func TestSeededProbeModelsAreNotOfferedByThePicker(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.ModelList = append(cfg.ModelList, configuredEntry("Alpha", "openai", "gpt-alpha"))

	offered := map[string]bool{}
	for _, model := range listSelectableModels(cfg, "Alpha") {
		offered[model.Name] = true
	}

	for _, seeded := range []string{"llama3", "local-model", "lmstudio-local", "copilot-gpt-5.4"} {
		if offered[seeded] {
			t.Errorf("the picker offers %q, whose runtime nothing has proved exists", seeded)
		}
	}
	if !offered["Alpha"] {
		t.Fatal("a normal configured model disappeared; the exclusion is too broad")
	}
}

// A local runtime the user materialized stays offered.
func TestMaterializedLocalModelIsOfferedByThePicker(t *testing.T) {
	local := &config.ModelConfig{
		ModelName: "My Ollama", Provider: "ollama", Model: "llama3", Enabled: true,
	}
	cfg := menuTestConfig(t, "Alpha", configuredEntry("Alpha", "openai", "gpt-alpha"), local)

	offered := map[string]bool{}
	for _, model := range listSelectableModels(cfg, "Alpha") {
		offered[model.Name] = true
	}
	if !offered["My Ollama"] {
		t.Fatal("a deliberately materialized local model was excluded")
	}
}

// A model switch is configuration. It updates the picker in place and adds
// nothing to the conversation.
func TestSuccessfulSwitchUpdatesThePickerWithoutANewMessage(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha",
		configuredEntry("Alpha", "openai", "gpt-alpha"),
		configuredEntry("Beta", "openai", "gpt-beta"),
	)
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionSelectModel, Value: "Beta",
	})

	if !result.Changed {
		t.Fatal("the switch did not report a change")
	}
	// The replacement picker states the new model in its own body...
	if !strings.Contains(result.Text, "Beta") {
		t.Fatalf("the picker header does not name the new model: %q", result.Text)
	}
	if strings.Contains(result.Text, "Alpha") {
		t.Fatalf("the picker header still claims the old model: %q", result.Text)
	}
	// ...and moves the tick.
	if result.Menu == nil {
		t.Fatal("no replacement picker")
	}
	// The acknowledgement is a toast, not a chat message, so it carries no
	// decoration that would look odd in the conversation.
	if strings.HasPrefix(result.Message, "✅") {
		t.Fatalf("the toast is styled as a chat message: %q", result.Message)
	}
	if !strings.Contains(result.Message, "Beta") {
		t.Fatalf("the acknowledgement does not name the model: %q", result.Message)
	}
}

// Re-selecting the active model leaves the picker exactly as it is.
func TestAlreadySelectedLeavesThePickerUntouched(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha", configuredEntry("Alpha", "openai", "gpt-alpha"))
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionSelectModel, Value: "Alpha",
	})

	if result.Changed || result.Text != "" || result.Menu != nil {
		t.Fatalf("re-selecting rewrote the picker: %+v", result)
	}
	if !strings.Contains(result.Message, "Already using") {
		t.Fatalf("unexpected acknowledgement: %q", result.Message)
	}
}

// Cancel says so in the toast and asks for no rewrite of the body.
func TestCancelProducesNoPickerText(t *testing.T) {
	cfg := menuTestConfig(t, "Alpha", configuredEntry("Alpha", "openai", "gpt-alpha"))
	loop := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer loop.Close()

	result := loop.RunMenuAction(context.Background(), bus.MenuActionRequest{
		Channel: "telegram", ChatID: "chat-1", SenderID: "user-1",
		Action: MenuActionCancel,
	})
	if result.Text != "" || result.Menu != nil || result.Changed {
		t.Fatalf("cancel asked for a picker rewrite: %+v", result)
	}
}
