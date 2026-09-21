package api

import "testing"

// PC-DEF-075. "Open chat" produced https://t.me/@name and Telegram answered
// "Username not found", while the bot, token and channel were all fine.
// Telegram's canonical bot link carries no "@" in the path.

func TestCanonicalTelegramUsername(t *testing.T) {
	cases := map[string]string{
		"pocketclaw_ab12cd34_bot":    "pocketclaw_ab12cd34_bot",
		"@pocketclaw_ab12cd34_bot":   "pocketclaw_ab12cd34_bot",
		"  pocketclaw_ab12cd34_bot ": "pocketclaw_ab12cd34_bot",
		" @pocketclaw_ab12cd34_bot ": "pocketclaw_ab12cd34_bot",

		// Refused rather than quietly repaired: a doubled "@" is already
		// evidence that something upstream is wrong.
		"@@pocketclaw_ab12cd34_bot": "",

		// A display string must never become a destination.
		"‏pocketclaw_ab12cd34_bot":    "",
		"‫pocketclaw_bot‬":            "",
		"https://t.me/pocketclaw_bot": "",
		"pocketclaw/../evil":          "",
		"pocketclaw bot":              "",
		"pocketclaw.bot":              "",
		"javascript:alert(1)":         "",
		"//evil.example":              "",
		"":                            "",
		"@":                           "",
		"bot":                         "",
	}
	for raw, want := range cases {
		if got := canonicalTelegramUsername(raw); got != want {
			t.Errorf("canonicalTelegramUsername(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestCanonicalTelegramBotURL(t *testing.T) {
	const want = "https://t.me/pocketclaw_ab12cd34_bot"
	for _, raw := range []string{
		"pocketclaw_ab12cd34_bot",
		"@pocketclaw_ab12cd34_bot",
		"  @pocketclaw_ab12cd34_bot  ",
	} {
		got := canonicalTelegramBotURL(raw)
		if got != want {
			t.Fatalf("canonicalTelegramBotURL(%q) = %q, want %q", raw, got, want)
		}
		if containsAt(got) {
			t.Fatalf("canonicalTelegramBotURL(%q) = %q still carries an @", raw, got)
		}
	}
	if got := canonicalTelegramBotURL("@@x"); got != "" {
		t.Fatalf("an unusable username must produce no link, got %q", got)
	}
}

func containsAt(value string) bool {
	for _, r := range value {
		if r == '@' {
			return true
		}
	}
	return false
}
