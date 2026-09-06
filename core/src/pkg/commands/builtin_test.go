package commands

import (
	"context"
	"strings"
	"testing"
)

func findDefinitionByName(t *testing.T, defs []Definition, name string) Definition {
	t.Helper()
	for _, def := range defs {
		if def.Name == name {
			return def
		}
	}
	t.Fatalf("missing /%s definition", name)
	return Definition{}
}

func TestBuiltinHelpHandler_ReturnsFormattedMessage(t *testing.T) {
	defs := BuiltinDefinitions()
	helpDef := findDefinitionByName(t, defs, "help")
	if helpDef.Handler == nil {
		t.Fatalf("/help handler should not be nil")
	}

	var reply string
	err := helpDef.Handler(context.Background(), Request{
		Text: "/help",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	}, nil)
	if err != nil {
		t.Fatalf("/help handler error: %v", err)
	}
	// The overview names every command and what it does, and deliberately
	// carries no command grammar: someone reading it should not have to parse
	// angle brackets or pipes to find out what PocketClaw can do.
	for _, name := range []string{
		"/stop", "/clear", "/context", "/show", "/list", "/use", "/btw",
		"/subagents", "/reload", "/switch", "/check",
	} {
		if !strings.Contains(reply, name) {
			t.Fatalf("/help reply missing %s, got %q", name, reply)
		}
	}
	for _, grammar := range []string{"[model|", "[models|", "<name>", "<skill>", "<question>", "<server>"} {
		if strings.Contains(reply, grammar) {
			t.Fatalf("/help reply exposes command grammar %q, got %q", grammar, reply)
		}
	}
	if !strings.HasPrefix(reply, "🦞 PocketClaw") {
		t.Fatalf("/help reply should open with the product name, got %q", reply)
	}
}

func TestBuiltinBtwCommand_MissingQuestion(t *testing.T) {
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), &Runtime{
		AskSideQuestion: func(context.Context, string) (string, error) {
			return "", nil
		},
	})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/btw",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/btw outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	// A bare /btw is someone exploring, not someone making a syntax error, so
	// it gets a sentence rather than a usage expression.
	if strings.Contains(reply, "Usage:") || strings.Contains(reply, "<question>") {
		t.Fatalf("/btw reply exposes usage syntax: %q", reply)
	}
	if !strings.Contains(reply, "/btw") {
		t.Fatalf("/btw reply should still name the command, got %q", reply)
	}
}

func TestBuiltinBtwCommand_PreservesQuestionWhitespace(t *testing.T) {
	const want = "explain:\n    fmt.Println(\"hi\")"
	rt := &Runtime{
		AskSideQuestion: func(ctx context.Context, question string) (string, error) {
			if question != want {
				t.Fatalf("question=%q, want %q", question, want)
			}
			return "ok", nil
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	res := ex.Execute(context.Background(), Request{
		Text: "/btw " + want,
		Reply: func(text string) error {
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/btw outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
}
