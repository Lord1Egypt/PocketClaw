package naming

import (
	"strings"
	"testing"
)

func TestGenerateUsernameShape(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 500; i++ {
		username, err := GenerateUsername()
		if err != nil {
			t.Fatalf("GenerateUsername: %v", err)
		}
		if !strings.HasPrefix(username, UsernamePrefix) {
			t.Fatalf("username %q missing prefix %q", username, UsernamePrefix)
		}
		if !strings.HasSuffix(username, UsernameSuffix) {
			t.Fatalf("username %q missing suffix %q", username, UsernameSuffix)
		}
		if err := ValidateUsername(username); err != nil {
			t.Fatalf("generated username %q is invalid: %v", username, err)
		}
		middle := strings.TrimSuffix(strings.TrimPrefix(username, UsernamePrefix), UsernameSuffix)
		if len(middle) != randomLen {
			t.Fatalf("username %q random segment is %d chars, want %d", username, len(middle), randomLen)
		}
		for _, r := range middle {
			if !strings.ContainsRune(alphabet, r) {
				t.Fatalf("username %q contains %q, outside the alphabet", username, r)
			}
		}
		seen[username] = true
	}
	// 500 draws from ~2.8e12 possibilities: any repeat means the randomness is
	// broken, not unlucky.
	if len(seen) != 500 {
		t.Fatalf("generated %d distinct usernames out of 500; randomness is not unique", len(seen))
	}
}

func TestGenerateUsernameLengthWithinTelegramLimit(t *testing.T) {
	username, err := GenerateUsername()
	if err != nil {
		t.Fatalf("GenerateUsername: %v", err)
	}
	if len(username) > maxUsernameLen {
		t.Fatalf("username %q is %d chars, over Telegram's %d limit", username, len(username), maxUsernameLen)
	}
}

func TestValidateUsernameRejects(t *testing.T) {
	cases := []struct {
		name     string
		username string
	}{
		{"too short", "a_bot"[:4]},
		{"too long", strings.Repeat("a", 30) + "_bot"},
		{"missing bot suffix", "pocketclaw_abc12345"},
		{"leading digit", "1ocketclaw_abc_bot"},
		{"hyphen", "pocketclaw-abc_bot"},
		{"space", "pocketclaw abc_bot"},
		{"contains hermes", "hermes_abc12345_bot"},
		{"contains picoclaw", "picoclaw_abc1234_bot"},
		{"contains sipeed", "sipeed_abc12345_bot"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateUsername(tc.username); err == nil {
				t.Fatalf("ValidateUsername(%q) accepted an invalid username", tc.username)
			}
		})
	}
}

func TestValidateUsernameAccepts(t *testing.T) {
	for _, username := range []string{
		"pocketclaw_k7m2x9aa_bot",
		"pocketclaw_4n8q2c11_bot",
		"MyOwnBot",
	} {
		if err := ValidateUsername(username); err != nil {
			t.Fatalf("ValidateUsername(%q) rejected a valid username: %v", username, err)
		}
	}
}

func TestDefaultDisplayNameIsPocketClawBranded(t *testing.T) {
	if DefaultDisplayName != "PocketClaw Agent" {
		t.Fatalf("DefaultDisplayName = %q, want %q", DefaultDisplayName, "PocketClaw Agent")
	}
	if err := ValidateDisplayName(DefaultDisplayName); err != nil {
		t.Fatalf("the default display name is invalid: %v", err)
	}
}

func TestValidateDisplayNameRejectsOtherProjects(t *testing.T) {
	for _, name := range []string{"Hermes Agent", "PicoClaw Agent", "Sipeed Agent", "  "} {
		if err := ValidateDisplayName(name); err == nil {
			t.Fatalf("ValidateDisplayName(%q) accepted a forbidden name", name)
		}
	}
}
