package canonicalenv

import (
	"os"
	"testing"
)

// The canonical/legacy pair used throughout. LOG_DIR is a real migrated
// variable and an A2 security boundary, so a regression here is not cosmetic.
const (
	canonicalKey = "POCKETCLAW_LOG_DIR"
	legacyKey    = "PICOCLAW_LOG_DIR"
)

func TestLookupPrecedence(t *testing.T) {
	cases := []struct {
		name        string
		canonical   *string
		legacy      *string
		wantValue   string
		wantPresent bool
	}{
		{
			name:        "canonical only",
			canonical:   ptr("/canonical"),
			wantValue:   "/canonical",
			wantPresent: true,
		},
		{
			name:        "legacy only remains supported",
			legacy:      ptr("/legacy"),
			wantValue:   "/legacy",
			wantPresent: true,
		},
		{
			name:        "both present, canonical wins",
			canonical:   ptr("/canonical"),
			legacy:      ptr("/legacy"),
			wantValue:   "/canonical",
			wantPresent: true,
		},
		{
			// Presence, not emptiness. A host that blanks a variable is saying
			// something; reading past it to a stale legacy value would restore
			// exactly what the host was turning off.
			name:        "empty canonical beats non-empty legacy",
			canonical:   ptr(""),
			legacy:      ptr("/legacy"),
			wantValue:   "",
			wantPresent: true,
		},
		{
			name:        "neither set",
			wantValue:   "",
			wantPresent: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setOrUnset(t, canonicalKey, tc.canonical)
			setOrUnset(t, legacyKey, tc.legacy)

			value, present := LookupEnv(legacyKey)
			if present != tc.wantPresent {
				t.Fatalf("LookupEnv presence = %v, want %v", present, tc.wantPresent)
			}
			if value != tc.wantValue {
				t.Fatalf("LookupEnv value = %q, want %q", value, tc.wantValue)
			}
			if got := Getenv(legacyKey); got != tc.wantValue {
				t.Fatalf("Getenv = %q, want %q", got, tc.wantValue)
			}
		})
	}
}

// An unmapped key must fall straight through, so the resolver is safe to use
// for any variable rather than only the migrated dozen.
func TestUnmappedKeyFallsThrough(t *testing.T) {
	const key = "POCKETCLAW_TEST_UNMAPPED_KEY"
	t.Setenv(key, "value")
	if got := Getenv(key); got != "value" {
		t.Fatalf("Getenv(%q) = %q, want %q", key, got, "value")
	}
}

// The mapping is the compatibility contract; handing out the live map would
// let any caller edit it for the whole process.
func TestAliasesIsACopy(t *testing.T) {
	first := Aliases()
	first["POCKETCLAW_HOME"] = "TAMPERED"
	if second := Aliases(); second["POCKETCLAW_HOME"] != "PICOCLAW_HOME" {
		t.Fatalf("Aliases() returned a shared map: got %q", second["POCKETCLAW_HOME"])
	}
}

// Every canonical name must itself be free of the legacy namespace — that is
// the entire point of the migration — and every legacy name must carry it,
// which is what makes the table a compatibility surface rather than a rename.
func TestAliasTableShape(t *testing.T) {
	for canonical, legacy := range Aliases() {
		if !hasPrefix(canonical, "POCKETCLAW_") {
			t.Errorf("canonical name %q is not in the POCKETCLAW_ namespace", canonical)
		}
		if containsFold(canonical, "PICO") {
			t.Errorf("canonical name %q still contains the legacy namespace", canonical)
		}
		if !hasPrefix(legacy, "PICOCLAW_") {
			t.Errorf("legacy name %q for %q is not a PICOCLAW_ name", legacy, canonical)
		}
	}
}

func ptr(s string) *string { return &s }

// setOrUnset applies a case's intent exactly: a nil pointer means the variable
// is absent, which is a different input from present-but-empty.
func setOrUnset(t *testing.T, key string, value *string) {
	t.Helper()
	if value == nil {
		// t.Setenv registers the restore; Unsetenv alone would not, so set
		// first to claim the cleanup and then remove it.
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		return
	}
	t.Setenv(key, *value)
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func containsFold(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if lower(a[i]) != lower(b[i]) {
			return false
		}
	}
	return true
}

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
