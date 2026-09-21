package commands

import (
	"context"
	"strings"
	"testing"
)

// PC-DEF-066. Legacy visual branding must not reappear in anything a chat user
// sees.
//
// The lobster survived every existing gate: Zero-Pico is lexical and an emoji
// is not a Pico identity, the i18n suites cover the Dashboard and the Flutter
// app rather than Core's command text, and the command-menu tests assert names
// and counts. So /help greeted every Telegram user with pre-Aperture branding
// and nothing failed. This is the check that would have.
//
// Scope is deliberately the user-facing command surface -- descriptions, usage,
// the no-argument help of every command and subcommand, and the assembled
// /help output. The upstream CLI's own banner constant is out of scope: it is
// upstream terminal output and is not reachable from pkg/ or web/, so it never
// reaches a PocketClaw user.
var legacyBrandingMarkers = []string{
	// The pre-Aperture mascot.
	"\U0001F99E",
	// Legacy product identity, in any casing, in text a user reads.
	"picoclaw",
	"pico claw",
}

func assertNoLegacyBranding(t *testing.T, where, text string) {
	t.Helper()
	lowered := strings.ToLower(text)
	for _, marker := range legacyBrandingMarkers {
		if strings.Contains(lowered, strings.ToLower(marker)) {
			t.Errorf("%s carries legacy branding %q: %q", where, marker, text)
		}
	}
}

func TestHelpOutputCarriesNoLegacyBranding(t *testing.T) {
	help := formatHelpMessage(BuiltinDefinitions())
	assertNoLegacyBranding(t, "/help output", help)

	// The header is exactly the product name and a blank line. "Contains
	// PocketClaw" would also pass a substituted emoji or a tagline, and PC-DEF-066
	// removed the mark rather than replacing it.
	if !strings.HasPrefix(help, "PocketClaw\n\n") {
		t.Fatalf("/help must open with exactly %q, got %q", "PocketClaw\n\n", help)
	}
}

// The handler prefers the runtime's own definition list over the built-in one,
// so that is the list that actually reaches a chat user. The branding check has
// to cover it and not only BuiltinDefinitions().
func TestRuntimeHelpOutputCarriesNoLegacyBranding(t *testing.T) {
	defs := append(BuiltinDefinitions(),
		Definition{Name: "custom", Description: "A runtime-supplied command"},
	)
	helpDef := findDefinitionByName(t, defs, "help")

	var reply string
	err := helpDef.Handler(context.Background(), Request{
		Text: "/help",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	}, &Runtime{ListDefinitions: func() []Definition { return defs }})
	if err != nil {
		t.Fatalf("/help handler error: %v", err)
	}

	assertNoLegacyBranding(t, "runtime /help output", reply)
	if !strings.Contains(reply, "/custom") {
		t.Fatalf("the runtime-supplied command did not reach /help: %q", reply)
	}
}

func TestCommandTextCarriesNoLegacyBranding(t *testing.T) {
	for _, def := range BuiltinDefinitions() {
		where := "/" + def.Name
		assertNoLegacyBranding(t, where+" description", def.Description)
		assertNoLegacyBranding(t, where+" usage", def.Usage)
		assertNoLegacyBranding(t, where+" no-args help", def.NoArgsHelp)
		for _, sub := range def.SubCommands {
			subWhere := where + " " + sub.Name
			assertNoLegacyBranding(t, subWhere+" description", sub.Description)
			assertNoLegacyBranding(t, subWhere+" args usage", sub.ArgsUsage)
		}
	}
}
