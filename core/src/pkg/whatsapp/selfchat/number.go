// Package selfchat implements PocketClaw's WhatsApp Self-Chat surface: the
// user's own number in canonical international form, and the request that asks
// the Android host to open WhatsApp on that number with a message prepared.
//
// Nothing here talks to WhatsApp. There is no bridge, no session store, no
// scraping and no automation — the host only ever hands Android a deep link,
// and the user presses Send inside WhatsApp.
package selfchat

import (
	"errors"
	"strings"
)

// E.164 caps a subscriber number at 15 digits. Eight is the shortest number
// that still carries a country code plus a plausible subscriber part, and
// rejecting anything below it catches truncated input without inventing a
// country.
const (
	minNumberDigits = 8
	maxNumberDigits = 15
)

var (
	// ErrNumberEmpty is returned for input that holds no digits at all.
	ErrNumberEmpty = errors.New("whatsapp: self number is empty")

	// ErrNumberInvalidChars is returned when the input holds something that is
	// neither a digit nor an accepted separator.
	ErrNumberInvalidChars = errors.New("whatsapp: self number may contain only digits, spaces, +, -, ., ( and )")

	// ErrNumberNotInternational is returned for a national number: one that
	// keeps a trunk prefix and names no country. Guessing the country from the
	// device locale would silently message a stranger, so it is refused.
	ErrNumberNotInternational = errors.New("whatsapp: self number must be international, starting with the country code (for example +20…)")

	// ErrNumberLength is returned when the digit count is outside E.164 range.
	ErrNumberLength = errors.New("whatsapp: self number must be 8 to 15 digits including the country code")
)

// separators a person plausibly types into, or pastes into, a phone field.
const numberSeparators = " \t-.()\u00a0"

// Normalize converts user input into canonical international form: a leading
// "+" followed by digits only.
//
// It accepts "+20 101 234 5678", "0020-101-234-5678" and "201012345678", all
// of which name their country unambiguously. It refuses "0101 234 5678",
// because the leading trunk zero belongs to a national plan and the country
// would have to be guessed.
func Normalize(input string) (string, error) {
	var b strings.Builder
	plusSeen := false
	for i, r := range input {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+':
			// A "+" is only meaningful as the first non-space character; one
			// buried mid-number means the input is not a phone number.
			if plusSeen || strings.TrimLeft(input[:i], numberSeparators) != "" {
				return "", ErrNumberInvalidChars
			}
			plusSeen = true
		case strings.ContainsRune(numberSeparators, r):
		default:
			return "", ErrNumberInvalidChars
		}
	}

	digits := b.String()
	if digits == "" {
		return "", ErrNumberEmpty
	}

	// "00" is the international access code across most of the world, so
	// stripping it reads the country code the user already wrote rather than
	// inferring one. It is only an access code when no "+" was given.
	if !plusSeen && strings.HasPrefix(digits, "00") {
		digits = digits[2:]
	}

	if strings.HasPrefix(digits, "0") {
		return "", ErrNumberNotInternational
	}
	if len(digits) < minNumberDigits || len(digits) > maxNumberDigits {
		return "", ErrNumberLength
	}

	return "+" + digits, nil
}

// Digits returns a canonical number without its leading "+", the form
// wa.me links and WhatsApp JIDs use.
func Digits(canonical string) string {
	return strings.TrimPrefix(canonical, "+")
}
