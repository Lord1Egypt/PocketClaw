package deeplink

import (
	"net/url"
	"strings"
	"testing"
)

func TestNewBotMatchesTelegramFormat(t *testing.T) {
	got, err := NewBot("PocketClawSetupBot", "pocketclaw_k7m2x9aa_bot", "PocketClaw Agent")
	if err != nil {
		t.Fatalf("NewBot: %v", err)
	}
	want := "https://t.me/newbot/PocketClawSetupBot/pocketclaw_k7m2x9aa_bot?name=PocketClaw%20Agent"
	if got != want {
		t.Fatalf("NewBot() = %q, want %q", got, want)
	}
}

func TestNewBotEncodesSpaceAsPercent20NotPlus(t *testing.T) {
	// A "+" here would reach Telegram as a literal plus in the bot's name.
	got, err := NewBot("PocketClawSetupBot", "pocketclaw_k7m2x9aa_bot", "PocketClaw Agent")
	if err != nil {
		t.Fatalf("NewBot: %v", err)
	}
	if strings.Contains(got, "+") {
		t.Fatalf("NewBot() = %q, encodes a space as \"+\"", got)
	}
	if !strings.Contains(got, "%20") {
		t.Fatalf("NewBot() = %q, does not percent-encode the space", got)
	}
}

func TestNewBotParsesAndRoundTripsTheName(t *testing.T) {
	link, err := NewBot("PocketClawSetupBot", "pocketclaw_k7m2x9aa_bot", "PocketClaw Agent")
	if err != nil {
		t.Fatalf("NewBot: %v", err)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("the generated link does not parse: %v", err)
	}
	if parsed.Host != "t.me" || !strings.HasPrefix(parsed.Path, "/newbot/") {
		t.Fatalf("link %q is not a t.me/newbot link", link)
	}
	if name := parsed.Query().Get("name"); name != "PocketClaw Agent" {
		t.Fatalf("name round-tripped as %q, want %q", name, "PocketClaw Agent")
	}
}

func TestNewBotAcceptsManagerWithAtPrefix(t *testing.T) {
	got, err := NewBot("@PocketClawSetupBot", "pocketclaw_k7m2x9aa_bot", "PocketClaw Agent")
	if err != nil {
		t.Fatalf("NewBot: %v", err)
	}
	if strings.Contains(got, "@") {
		t.Fatalf("NewBot() = %q, leaked the @ prefix into the link", got)
	}
}

func TestNewBotRejectsBadInput(t *testing.T) {
	cases := []struct {
		name                    string
		manager, child, display string
	}{
		{"empty manager", "", "pocketclaw_k7m2x9aa_bot", "PocketClaw Agent"},
		{"invalid child username", "PocketClawSetupBot", "not-a-bot", "PocketClaw Agent"},
		{"child without bot suffix", "PocketClawSetupBot", "pocketclaw_k7m2x9aa", "PocketClaw Agent"},
		{"forbidden display name", "PocketClawSetupBot", "pocketclaw_k7m2x9aa_bot", "Hermes Agent"},
		{"empty display name", "PocketClawSetupBot", "pocketclaw_k7m2x9aa_bot", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewBot(tc.manager, tc.child, tc.display); err == nil {
				t.Fatalf("NewBot(%q, %q, %q) accepted invalid input", tc.manager, tc.child, tc.display)
			}
		})
	}
}

func TestBotChat(t *testing.T) {
	got, err := BotChat("@pocketclaw_k7m2x9aa_bot")
	if err != nil {
		t.Fatalf("BotChat: %v", err)
	}
	if got != "https://t.me/pocketclaw_k7m2x9aa_bot" {
		t.Fatalf("BotChat() = %q", got)
	}
}
