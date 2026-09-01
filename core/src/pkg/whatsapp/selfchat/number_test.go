package selfchat

import (
	"errors"
	"testing"
)

func TestNormalizeAcceptsInternationalForms(t *testing.T) {
	cases := map[string]string{
		"+20 101 234 5678":   "+201012345678",
		"+201012345678":      "+201012345678",
		"+90 (532) 123-4567": "+905321234567",
		"  +905321234567  ":  "+905321234567",
		"201012345678":       "+201012345678",
		"0020 101 234 5678":  "+201012345678",
		"+1.415.555.0123":    "+14155550123",
		"+20 101234567":      "+20101234567",
	}
	for input, want := range cases {
		got, err := Normalize(input)
		if err != nil {
			t.Fatalf("Normalize(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Errorf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeRejectsInvalidInput(t *testing.T) {
	cases := map[string]error{
		"":                  ErrNumberEmpty,
		"   ":               ErrNumberEmpty,
		"+":                 ErrNumberEmpty,
		"01012345678":       ErrNumberNotInternational,
		"0101 234 5678":     ErrNumberNotInternational,
		"+201234":           ErrNumberLength,
		"+2010123456789012": ErrNumberLength,
		"+20 101 CALL ME":   ErrNumberInvalidChars,
		"20+1012345678":     ErrNumberInvalidChars,
		"++201012345678":    ErrNumberInvalidChars,
		"+20101234567;8":    ErrNumberInvalidChars,
	}
	for input, want := range cases {
		got, err := Normalize(input)
		if !errors.Is(err, want) {
			t.Errorf("Normalize(%q) error = %v, want %v", input, err, want)
		}
		if got != "" {
			t.Errorf("Normalize(%q) returned %q alongside an error", input, got)
		}
	}
}

// A number that already carries "+" keeps its leading digits: "00" is only an
// international access code when the user did not write "+".
func TestNormalizeKeepsLeadingZerosAfterPlusAsInvalid(t *testing.T) {
	if _, err := Normalize("+0020101234567"); !errors.Is(err, ErrNumberNotInternational) {
		t.Fatalf("Normalize(+0020…) error = %v, want ErrNumberNotInternational", err)
	}
}

func TestDigitsStripsPlus(t *testing.T) {
	if got := Digits("+201012345678"); got != "201012345678" {
		t.Errorf("Digits() = %q", got)
	}
}
