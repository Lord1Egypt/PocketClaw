// Package naming generates the child bot identities PocketClaw suggests to
// Telegram, and enforces Telegram's username rules on them.
package naming

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
)

const (
	// DefaultDisplayName is the display name suggested for every child bot.
	DefaultDisplayName = "PocketClaw Agent"

	// UsernamePrefix marks every bot created through PocketClaw as ours.
	UsernamePrefix = "pocketclaw_"

	// UsernameSuffix is required by Telegram: bot usernames must end in "bot".
	UsernameSuffix = "_bot"

	// randomLen is the length of the random middle segment. With a 36-symbol
	// alphabet that is ~2.8e12 possibilities, which keeps collisions
	// negligible both against Telegram's global namespace and against the
	// other pairings alive at the same time.
	randomLen = 8

	// alphabet is lowercase alphanumerics only. Telegram allows uppercase and
	// underscores too, but a username a person may have to read aloud or type
	// is better off without case distinctions.
	alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

	minUsernameLen = 5
	maxUsernameLen = 32
)

// forbiddenSubstrings must never appear in a name PocketClaw suggests. These
// are other projects' identities; suggesting them would misattribute the bot.
var forbiddenSubstrings = []string{"hermes", "picoclaw", "sipeed"}

var usernamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// GenerateUsername returns a fresh child bot username of the form
// pocketclaw_<random>_bot, using crypto/rand.
func GenerateUsername() (string, error) {
	suffix, err := randomString(randomLen)
	if err != nil {
		return "", err
	}
	username := UsernamePrefix + suffix + UsernameSuffix
	if err := ValidateUsername(username); err != nil {
		// Unreachable given the constants above; treated as a hard failure
		// rather than shipping an invalid username to Telegram.
		return "", fmt.Errorf("generated an invalid username %q: %w", username, err)
	}
	return username, nil
}

// ValidateUsername reports whether a username satisfies Telegram's bot
// username rules and PocketClaw's own naming policy.
func ValidateUsername(username string) error {
	if len(username) < minUsernameLen || len(username) > maxUsernameLen {
		return fmt.Errorf("username %q must be %d-%d characters, got %d",
			username, minUsernameLen, maxUsernameLen, len(username))
	}
	if !usernamePattern.MatchString(username) {
		return fmt.Errorf("username %q must start with a letter and contain only letters, digits, and underscores", username)
	}
	if !strings.HasSuffix(strings.ToLower(username), "bot") {
		return fmt.Errorf("username %q must end in \"bot\"", username)
	}
	lower := strings.ToLower(username)
	for _, forbidden := range forbiddenSubstrings {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("username %q must not contain %q", username, forbidden)
		}
	}
	return nil
}

// ValidateDisplayName enforces the same identity policy on display names.
func ValidateDisplayName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("display name must not be empty")
	}
	if len(trimmed) > 64 {
		return fmt.Errorf("display name must be at most 64 characters, got %d", len(trimmed))
	}
	lower := strings.ToLower(trimmed)
	for _, forbidden := range forbiddenSubstrings {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("display name %q must not contain %q", trimmed, forbidden)
		}
	}
	return nil
}

func randomString(n int) (string, error) {
	// rand.Text-style rejection is unnecessary here: len(alphabet) is 36 and
	// we draw from a 256-value byte, so a modulo would bias. Draw extra bytes
	// and reject out-of-range values instead.
	out := make([]byte, 0, n)
	buf := make([]byte, n*2)
	for len(out) < n {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("read random bytes: %w", err)
		}
		for _, b := range buf {
			if len(out) == n {
				break
			}
			if int(b) < 252 { // 252 = 36*7, the largest multiple of 36 below 256
				out = append(out, alphabet[int(b)%len(alphabet)])
			}
		}
	}
	return string(out), nil
}
