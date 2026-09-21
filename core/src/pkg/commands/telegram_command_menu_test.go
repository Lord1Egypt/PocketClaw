package commands

import (
	"slices"
	"testing"
)

// The Telegram command menu is published from this registry, so the registry is
// where "the expected command set" is pinned. The menu itself cannot be read
// back from inside the process: only Telegram knows what it is showing. What can
// be held here is that the authoritative set is complete, that every entry in it
// is publishable, and that nothing quietly narrows it to a handful.

// expectedTelegramCommands is the command set PocketClaw publishes. It is
// written out rather than derived from BuiltinDefinitions so that removing a
// command is a deliberate edit to this list, not a silently passing test.
var expectedTelegramCommands = []string{
	"start",
	"help",
	"stop",
	"show",
	"list",
	"use",
	"btw",
	"switch",
	"model",
	"check",
	"clear",
	"context",
	"subagents",
	"reload",
}

func TestBuiltinDefinitionsMatchTheExpectedTelegramCommandSet(t *testing.T) {
	var got []string
	for _, def := range BuiltinDefinitions() {
		got = append(got, def.Name)
	}

	if !slices.Equal(got, expectedTelegramCommands) {
		t.Fatalf("command set = %v, want %v: the Telegram menu is published from "+
			"this list, so a change here changes what the bot exposes", got, expectedTelegramCommands)
	}
}

// The count the owner verifies on the device.
func TestBuiltinDefinitionsCount(t *testing.T) {
	if got := len(BuiltinDefinitions()); got != len(expectedTelegramCommands) {
		t.Fatalf("definitions = %d, want %d", got, len(expectedTelegramCommands))
	}
}

// /start is the one command Telegram itself shows to a new chat, so its absence
// is what makes the menu look empty even when the bot answers normally.
func TestStartIsInTheRegisteredCommandSet(t *testing.T) {
	for _, def := range BuiltinDefinitions() {
		if def.Name == "start" {
			if def.Description == "" {
				t.Fatal("/start has no description, so Telegram will not publish it")
			}
			return
		}
	}
	t.Fatal("/start is missing from the command set")
}

// Telegram requires a name and a description for every command, and an entry
// missing either is dropped on the way out. A dropped entry is a command absent
// from the menu for a reason no screenshot can show, so no definition may have
// an empty one.
func TestEveryBuiltinDefinitionIsPublishableToTelegram(t *testing.T) {
	for _, def := range BuiltinDefinitions() {
		if def.Name == "" {
			t.Errorf("a definition has no name: %+v", def)
			continue
		}
		if def.Description == "" {
			t.Errorf("/%s has no description, so Telegram will not publish it", def.Name)
		}
	}
}

// Descriptions are what the user reads in the menu. Telegram rejects a
// description longer than 256 characters, and rejecting one command rejects the
// whole setMyCommands call -- so a single over-long string empties the menu.
func TestBuiltinDescriptionsAreWithinTelegramLimits(t *testing.T) {
	const maxCommandLength = 32
	const maxDescriptionLength = 256
	for _, def := range BuiltinDefinitions() {
		if len(def.Name) > maxCommandLength {
			t.Errorf("/%s: name is %d characters, Telegram allows %d",
				def.Name, len(def.Name), maxCommandLength)
		}
		if len(def.Description) > maxDescriptionLength {
			t.Errorf("/%s: description is %d characters, Telegram allows %d",
				def.Name, len(def.Description), maxDescriptionLength)
		}
	}
}

// Telegram command names may only contain lowercase letters, digits and
// underscores. An invalid name fails the whole call, taking every other command
// down with it.
func TestBuiltinCommandNamesAreValidForTelegram(t *testing.T) {
	for _, def := range BuiltinDefinitions() {
		for _, r := range def.Name {
			valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_'
			if !valid {
				t.Errorf("/%s contains %q, which Telegram does not accept in a command name",
					def.Name, r)
				break
			}
		}
	}
}
