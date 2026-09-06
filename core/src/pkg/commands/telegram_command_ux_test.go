package commands

import (
	"context"
	"strings"
	"testing"
	"time"
)

// forbiddenInChatOutput are the shapes that must never reach a chat window,
// whichever command produced them.
var forbiddenInChatOutput = []string{
	"&{",          // a Go struct printed with %v / %+v
	"TurnID:",     // internal turn identity
	"AgentID:",    // internal agent identity
	"SessionKey:", // routing identity
	"ChatID:",     // routing identity
	"UserMessage", // the user's own prompt
	"0001-01-01",  // a zero-value timestamp rendered as a year 1 date
	"ParentTurnID",
	"ChildTurnIDs",
	"Iteration:",
}

func assertNoInternalsLeaked(t *testing.T, reply string) {
	t.Helper()
	for _, forbidden := range forbiddenInChatOutput {
		if strings.Contains(reply, forbidden) {
			t.Fatalf("output leaked %q:\n%s", forbidden, reply)
		}
	}
}

func runCommand(t *testing.T, rt *Runtime, text string) string {
	t.Helper()
	var reply string
	res := NewExecutor(NewRegistry(BuiltinDefinitions()), rt).Execute(
		context.Background(),
		Request{
			Text: text,
			Reply: func(s string) error {
				reply = s
				return nil
			},
		},
	)
	if res.Outcome != OutcomeHandled {
		t.Fatalf("%s outcome = %v, want handled", text, res.Outcome)
	}
	return reply
}

// The reported defect: /subagents printed the whole active-turn struct, which
// carried the user's own message, their session key and their chat id.
func TestSubagentsNeverPrintsInternalTurnState(t *testing.T) {
	rt := &Runtime{
		ListSubagents: func() []SubagentInfo {
			return []SubagentInfo{
				{Status: "running", Duration: 34 * time.Second},
				{Status: "tools", Duration: 12 * time.Second, Depth: 1},
			}
		},
	}

	reply := runCommand(t, rt, "/subagents")
	assertNoInternalsLeaked(t, reply)

	if !strings.Contains(reply, "Active subagents: 2") {
		t.Fatalf("missing the count: %q", reply)
	}
	for _, want := range []string{"Subagent 1", "Subagent 2", "Running", "Working", "34s", "12s"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("missing %q in:\n%s", want, reply)
		}
	}
}

// The type system, not the formatter, is what makes the leak impossible: the
// fields are not reachable from this package at all.
func TestSubagentInfoCarriesNoIdentifyingFields(t *testing.T) {
	// A SubagentInfo with every field set to something recognisable must still
	// render nothing sensitive, because there is nothing sensitive to set.
	reply := formatSubagents([]SubagentInfo{{
		Status:   "running",
		Duration: time.Second,
		Depth:    2,
	}})
	assertNoInternalsLeaked(t, reply)
}

func TestSubagentsEmptyStateIsClean(t *testing.T) {
	rt := &Runtime{ListSubagents: func() []SubagentInfo { return nil }}
	reply := runCommand(t, rt, "/subagents")

	if reply != "🤖 No active subagents." {
		t.Fatalf("empty state = %q", reply)
	}
	assertNoInternalsLeaked(t, reply)
}

// A turn with no recorded start time has no duration. The old output rendered
// that zero instant as a year 1 date.
func TestSubagentsOmitsUnknownDurationRatherThanPrintingZero(t *testing.T) {
	reply := formatSubagents([]SubagentInfo{{Status: "setup"}})

	assertNoInternalsLeaked(t, reply)
	if strings.Contains(reply, "Duration") {
		t.Fatalf("an unknown duration must be omitted, got:\n%s", reply)
	}
	if !strings.Contains(reply, "Starting") {
		t.Fatalf("the status should still show, got:\n%s", reply)
	}
}

// An unrecognised lifecycle token is dropped rather than passed through as an
// internal word.
func TestSubagentsDropsUnknownStatusTokens(t *testing.T) {
	reply := formatSubagents([]SubagentInfo{{Status: "some_internal_phase"}})
	if strings.Contains(reply, "some_internal_phase") {
		t.Fatalf("an internal status token reached the user:\n%s", reply)
	}
}

// A bare command is someone exploring. Answering with command grammar asks them
// to read a manual before they can do anything.
func TestNoArgumentCommandsGiveGuidanceNotUsageGrammar(t *testing.T) {
	for _, test := range []struct {
		command string
		wants   []string
	}{
		{command: "/switch", wants: []string{"model", "channel"}},
		{command: "/show", wants: []string{"model", "channel", "agents", "MCP"}},
		{command: "/list", wants: []string{"models", "channels", "agents", "skills", "MCP"}},
	} {
		t.Run(test.command, func(t *testing.T) {
			reply := runCommand(t, &Runtime{}, test.command)

			if strings.HasPrefix(reply, "Usage:") {
				t.Fatalf("%s still answers with raw usage: %q", test.command, reply)
			}
			for _, grammar := range []string{"[model|", "[models|", "<name>", "<server>"} {
				if strings.Contains(reply, grammar) {
					t.Fatalf("%s exposes grammar %q: %q", test.command, grammar, reply)
				}
			}
			for _, want := range test.wants {
				if !strings.Contains(reply, want) {
					t.Fatalf("%s guidance does not mention %q: %q", test.command, want, reply)
				}
			}
		})
	}
}

// Grammar is still the right answer when the argument is genuinely wrong, so
// usage support is narrowed rather than removed.
func TestInvalidSubcommandStillGetsExactUsage(t *testing.T) {
	reply := runCommand(t, &Runtime{}, "/show nonsense")

	if !strings.Contains(reply, "Usage:") {
		t.Fatalf("a wrong argument should still get usage, got %q", reply)
	}
	if !strings.Contains(reply, "Unknown option") {
		t.Fatalf("a wrong argument should be named, got %q", reply)
	}
}

// The advanced textual forms are unchanged; this phase only altered what an
// empty invocation answers.
func TestAdvancedTextualFormsStillRoute(t *testing.T) {
	var switched string
	rt := &Runtime{
		SwitchModel: func(value string) (string, error) {
			switched = value
			return "old-model", nil
		},
		GetModelInfo:       func() (string, string) { return "current-model", "openai" },
		GetEnabledChannels: func() []string { return []string{"telegram"} },
	}

	if reply := runCommand(t, rt, "/switch model to new-model"); reply == "" {
		t.Fatal("/switch model to <name> produced no reply")
	}
	if switched != "new-model" {
		t.Fatalf("SwitchModel received %q, want new-model", switched)
	}

	if reply := runCommand(t, rt, "/show model"); !strings.Contains(reply, "current-model") {
		t.Fatalf("/show model no longer reports the model: %q", reply)
	}
	if reply := runCommand(t, rt, "/list channels"); !strings.Contains(reply, "telegram") {
		t.Fatalf("/list channels no longer lists channels: %q", reply)
	}
}

// /stop is physically accepted behaviour; this phase must not have touched it.
func TestStopCommandBehaviourUnchanged(t *testing.T) {
	var called bool
	rt := &Runtime{
		StopActiveTurn: func() (StopResult, error) {
			called = true
			return StopResult{Stopped: true, TaskName: "a task"}, nil
		},
	}

	reply := runCommand(t, rt, "/stop")
	if !called {
		t.Fatal("/stop no longer reaches StopActiveTurn")
	}
	if !strings.Contains(reply, "stopped") && !strings.Contains(reply, "Stopped") {
		t.Fatalf("/stop reply = %q", reply)
	}
	assertNoInternalsLeaked(t, reply)
}

// Every command's description is what Telegram's native "/" menu shows, so none
// may be empty or written as syntax.
func TestCommandDescriptionsAreHumanReadable(t *testing.T) {
	for _, def := range BuiltinDefinitions() {
		if strings.TrimSpace(def.Description) == "" {
			t.Errorf("/%s has no description; Telegram's command menu would show nothing", def.Name)
			continue
		}
		for _, grammar := range []string{"<", "|", "["} {
			if strings.Contains(def.Description, grammar) {
				t.Errorf("/%s description contains grammar %q: %q", def.Name, grammar, def.Description)
			}
		}
	}
}
