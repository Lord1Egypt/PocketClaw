package commands

import (
	"context"
	"strings"
	"testing"
)

func modelRuntime(current string, choices ...ModelChoice) *Runtime {
	return &Runtime{
		GetModelPicker: func() *ModelPicker {
			return &ModelPicker{
				CurrentModel:    current,
				CurrentProvider: "openai",
				Choices:         choices,
			}
		},
	}
}

func runModelCommand(t *testing.T, rt *Runtime, withMenu bool) (string, *Menu) {
	t.Helper()
	var text string
	var menu *Menu
	req := Request{Text: "/model", Reply: func(s string) error { text = s; return nil }}
	if withMenu {
		req.ReplyMenu = func(s string, m *Menu) error { text, menu = s, m; return nil }
	}
	res := NewExecutor(NewRegistry(BuiltinDefinitions()), rt).Execute(context.Background(), req)
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/model outcome = %v, want handled", res.Outcome)
	}
	return text, menu
}

// A + D + E. One menu, the configured models on it, the active one marked.
func TestModelCommandOffersConfiguredModelsWithCurrentMarked(t *testing.T) {
	rt := modelRuntime("gemini-2.5-flash",
		ModelChoice{Name: "gemini-2.5-flash", Label: "gemini-2.5-flash", Current: true},
		ModelChoice{Name: "deepseek-chat", Label: "deepseek-chat"},
	)

	text, menu := runModelCommand(t, rt, true)
	if menu == nil {
		t.Fatal("/model produced no menu")
	}
	if !strings.Contains(text, "gemini-2.5-flash") {
		t.Fatalf("the header should name the current model: %q", text)
	}

	// One row per model, plus a cancel row.
	if len(menu.Rows) != 3 {
		t.Fatalf("expected 2 model rows and a cancel row, got %d", len(menu.Rows))
	}

	marked := 0
	labels := map[string]bool{}
	for _, row := range menu.Rows {
		for _, button := range row.Buttons {
			labels[button.Label] = true
			if button.Current {
				marked++
				if !strings.HasPrefix(button.Label, "✓") {
					t.Errorf("the active model is not visibly marked: %q", button.Label)
				}
			}
		}
	}
	if marked != 1 {
		t.Fatalf("exactly one model should be marked current, got %d", marked)
	}
	if !labels["✓ gemini-2.5-flash"] || !labels["deepseek-chat"] {
		t.Fatalf("unexpected labels: %v", labels)
	}
	if !labels["✕ Cancel"] {
		t.Fatal("the picker must offer a way out")
	}
}

// The buttons carry PocketClaw's own vocabulary, never configuration.
func TestModelMenuButtonsCarryNoConfiguration(t *testing.T) {
	rt := modelRuntime("m-one",
		ModelChoice{Name: "m-one", Label: "m-one", Current: true},
		ModelChoice{Name: "m-two", Label: "m-two"},
	)
	_, menu := runModelCommand(t, rt, true)

	for _, row := range menu.Rows {
		for _, button := range row.Buttons {
			switch button.Action {
			case ActionSelectModel, ActionCancelMenu:
			default:
				t.Fatalf("unexpected action %q", button.Action)
			}
			for _, forbidden := range []string{"http://", "https://", "sk-", "key", "token", "api_base"} {
				if strings.Contains(strings.ToLower(button.Value), forbidden) {
					t.Fatalf("button value carries configuration: %q", button.Value)
				}
			}
		}
	}
}

// G. A channel that cannot render choices still gets a useful answer.
func TestModelCommandDegradesToTextWithoutMenuSupport(t *testing.T) {
	rt := modelRuntime("m-one",
		ModelChoice{Name: "m-one", Label: "m-one", Current: true},
		ModelChoice{Name: "m-two", Label: "m-two"},
	)

	text, menu := runModelCommand(t, rt, false)
	if menu != nil {
		t.Fatal("a channel without menu support must not receive a menu")
	}
	for _, want := range []string{"m-one", "m-two", "/switch model to"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the text fallback is missing %q: %q", want, text)
		}
	}
}

// No configured models is a configuration problem, and saying so beats an
// empty picker.
func TestModelCommandExplainsWhenNothingIsConfigured(t *testing.T) {
	text, menu := runModelCommand(t, modelRuntime(""), true)
	if menu != nil {
		t.Fatal("an empty picker should not be sent")
	}
	if !strings.Contains(text, "No models are configured") {
		t.Fatalf("unhelpful empty state: %q", text)
	}
}

// /model is answered from local state, so it must not stage a Thinking
// placeholder as though a model were being consulted.
func TestModelCommandIsMarkedInstant(t *testing.T) {
	registry := NewRegistry(BuiltinDefinitions())
	if !registry.IsInstantCommand("/model") {
		t.Fatal("/model must be instant; otherwise a picker arrives behind a Thinking placeholder")
	}
	if registry.IsInstantCommand("/btw what is the time") {
		t.Fatal("/btw reaches the model and must keep its activity signals")
	}
}
