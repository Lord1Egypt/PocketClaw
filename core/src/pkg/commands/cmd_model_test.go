package commands

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// executeForTest runs one command line through the real executor and returns
// what the user would see. Going through the executor rather than calling a
// handler directly is what makes these tests say anything about /model being
// registered.
func executeForTest(t *testing.T, input string, rt *Runtime) (ExecuteResult, string) {
	t.Helper()
	var reply string
	result := NewExecutor(NewRegistry(BuiltinDefinitions()), rt).Execute(
		context.Background(),
		Request{
			Channel:  "telegram",
			ChatID:   "chat",
			SenderID: "user",
			Text:     input,
			Reply: func(text string) error {
				reply = text
				return nil
			},
		},
	)
	return result, reply
}

// 1. /model is registered and answers with the fixed informational message.
func TestModelIsRegisteredAndAnswersWithTheFixedMessage(t *testing.T) {
	result, reply := executeForTest(t, "/model", nil)

	if result.Outcome != OutcomeHandled {
		t.Fatalf("/model outcome = %v, want handled; an unregistered command falls through to the model", result.Outcome)
	}
	if result.Err != nil {
		t.Fatalf("/model returned an error: %v", result.Err)
	}
	if reply != "🤖 Model selection is managed from PocketClaw Settings." {
		t.Fatalf("/model replied %q, want the fixed Dashboard message", reply)
	}
}

// A nil Runtime is what a command gets when the agent cannot build one. /model
// must still answer, because it depends on nothing.
func TestModelAnswersWithoutARuntime(t *testing.T) {
	_, withRuntime := executeForTest(t, "/model", &Runtime{})
	_, withoutRuntime := executeForTest(t, "/model", nil)

	if withRuntime != withoutRuntime {
		t.Fatalf("/model varied with the runtime: %q vs %q", withRuntime, withoutRuntime)
	}
	if strings.Contains(withoutRuntime, unavailableMsg) {
		t.Fatal("/model reported itself unavailable; it must not depend on runtime capabilities")
	}
}

// 3 (package half). /model cannot alter model state: the only mutator reachable
// from a command is Runtime.SwitchModel, and the handler never receives it.
//
// Asserting on the reply would not show this — a command that switched the
// model and then printed this same sentence would pass that. So the runtime
// handed in has a SwitchModel that fails the test if it is ever called.
func TestModelNeverReachesTheModelSwitcher(t *testing.T) {
	switched := false
	rt := &Runtime{
		SwitchModel: func(string) (string, error) {
			switched = true
			return "", nil
		},
		GetModelInfo: func() (string, string) {
			t.Error("/model read the current model; a fixed message needs no model state")
			return "secret-model", "secret-provider"
		},
	}

	for _, input := range []string{"/model", "/model gpt-4o", "/model set gpt-4o", "!model"} {
		_, reply := executeForTest(t, input, rt)
		if switched {
			t.Fatalf("%q switched the model", input)
		}
		// Arguments are ignored rather than acted on: /model never becomes a
		// setter by accident.
		if reply != modelInfoMessage {
			t.Fatalf("%q replied %q; arguments must not change the answer", input, reply)
		}
	}
}

// The response must name no model, provider, endpoint or key. It is a constant,
// so this checks the constant itself rather than one rendering of it.
func TestModelResponseExposesNoIdentifiers(t *testing.T) {
	for _, forbidden := range []string{
		"gpt", "claude", "gemini", "openai", "anthropic", "ollama",
		"http://", "https://", "api_key", "api-key", "model_list", "/switch",
	} {
		if strings.Contains(strings.ToLower(modelInfoMessage), forbidden) {
			t.Fatalf("the /model response contains %q: %q", forbidden, modelInfoMessage)
		}
	}
}

// 4. /model carries no interactive menu or callback infrastructure. The command
// answers through Reply, and Request has no other way to answer — there is no
// menu field to populate and no button type to build one from.
func TestModelHasNoInteractiveSurface(t *testing.T) {
	def := findDefinition(t, "model")

	if len(def.SubCommands) != 0 {
		t.Fatalf("/model declares %d sub-commands; it is a fixed message", len(def.SubCommands))
	}
	if def.Handler == nil {
		t.Fatal("/model has no handler")
	}

	// Structural: nothing in Request offers a non-text answer. If a menu field
	// is ever added back, this fails and the reviewer has to justify it.
	requestType := reflect.TypeOf(Request{})
	for i := 0; i < requestType.NumField(); i++ {
		name := strings.ToLower(requestType.Field(i).Name)
		if strings.Contains(name, "menu") || strings.Contains(name, "button") || strings.Contains(name, "callback") {
			t.Fatalf("Request grew an interactive field %q; /model must stay text-only",
				requestType.Field(i).Name)
		}
	}
}

// 5. The advanced textual command is untouched by all of this.
func TestSwitchModelToStillWorks(t *testing.T) {
	var got string
	rt := &Runtime{
		SwitchModel: func(value string) (string, error) {
			got = value
			return "old-model", nil
		},
	}

	result, reply := executeForTest(t, "/switch model to new-model", rt)
	if result.Outcome != OutcomeHandled || result.Err != nil {
		t.Fatalf("/switch model to … outcome = %v err = %v", result.Outcome, result.Err)
	}
	if got != "new-model" {
		t.Fatalf("SwitchModel received %q, want new-model", got)
	}
	if !strings.Contains(reply, "old-model") || !strings.Contains(reply, "new-model") {
		t.Fatalf("/switch reply = %q, want the old and new model names", reply)
	}
}

// 6. /help lists /model, and does so with the Dashboard-first wording rather
// than presenting it as a way to change the model.
func TestHelpListsModelWithDashboardWording(t *testing.T) {
	help := formatHelpMessage(BuiltinDefinitions())

	if !strings.Contains(help, "/model — ") {
		t.Fatalf("/help does not list /model:\n%s", help)
	}
	if !strings.Contains(help, "/model — Manage models from PocketClaw Settings") {
		t.Fatalf("/help does not use the Dashboard-first wording for /model:\n%s", help)
	}
	if strings.Contains(help, "/switch — Switch model") {
		t.Fatalf("/help advertises /switch as model selection:\n%s", help)
	}
}

func findDefinition(t *testing.T, name string) Definition {
	t.Helper()
	for _, def := range BuiltinDefinitions() {
		if def.Name == name {
			return def
		}
	}
	t.Fatalf("command %q is not registered", name)
	return Definition{}
}
