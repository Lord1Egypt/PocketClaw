package pcruntime

import (
	"strings"
	"testing"
)

func TestRedactsAuthorizationHeader(t *testing.T) {
	argv := RedactArgv("curl", []string{
		"-H", "Authorization: Bearer sk-live-abcdefghijklmnop", "https://example.com",
	})
	joined := strings.Join(argv, " ")
	if strings.Contains(joined, "sk-live-abcdefghijklmnop") {
		t.Fatalf("the credential survived redaction: %q", joined)
	}
	if !strings.Contains(joined, redactedMarker) {
		t.Fatalf("expected a redaction marker: %q", joined)
	}
	if !strings.Contains(joined, "https://example.com") {
		t.Fatalf("redaction removed the diagnostic value of the log: %q", joined)
	}
}

func TestRedactsInlineAuthorizationHeaderText(t *testing.T) {
	got := RedactText(`curl -H "Authorization: Bearer abcdefghijklmnopqrst" https://api.example.com`)
	if strings.Contains(got, "abcdefghijklmnopqrst") {
		t.Fatalf("bearer token survived: %q", got)
	}
}

func TestRedactsTelegramTokens(t *testing.T) {
	for _, input := range []string{
		"123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw",
		"https://api.telegram.org/bot123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw/getMe",
	} {
		got := RedactText(input)
		if strings.Contains(got, "AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw") {
			t.Fatalf("telegram token survived redaction: %q", got)
		}
	}
}

func TestRedactsGitHubTokens(t *testing.T) {
	for _, token := range []string{
		"ghp_1234567890abcdefghijklmnopqrstuvwx",
		"gho_1234567890abcdefghijklmnopqrstuvwx",
		"github_pat_11ABCDEFG0abcdefghijklmnopqrstuvwxyz",
	} {
		if got := RedactText("token is " + token); strings.Contains(got, token) {
			t.Fatalf("github token survived redaction: %q", got)
		}
	}
}

func TestRedactsURLUserinfo(t *testing.T) {
	got := RedactText("https://someone:hunter2@example.com/repo.git")
	if strings.Contains(got, "hunter2") {
		t.Fatalf("URL password survived redaction: %q", got)
	}
	if !strings.Contains(got, "example.com/repo.git") {
		t.Fatalf("host should stay readable: %q", got)
	}
}

func TestRedactsSecretLookingEnvironmentValues(t *testing.T) {
	redacted := RedactEnv(map[string]string{
		"GITHUB_TOKEN":         "ghp_1234567890abcdefghijklmnopqrstuvwx",
		"OPENAI_API_KEY":       "sk-proj-1234567890abcdefghij",
		"TELEGRAM_BOT_TOKEN":   "123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw",
		"POCKETCLAW_WORKSPACE": "/storage/emulated/0/Download/pocketclaw",
	})
	for _, key := range []string{"GITHUB_TOKEN", "OPENAI_API_KEY", "TELEGRAM_BOT_TOKEN"} {
		if redacted[key] != redactedMarker {
			t.Fatalf("%s was not redacted: %q", key, redacted[key])
		}
	}
	if redacted["POCKETCLAW_WORKSPACE"] != "/storage/emulated/0/Download/pocketclaw" {
		t.Fatalf("a non-secret value must stay readable: %q", redacted["POCKETCLAW_WORKSPACE"])
	}
}

func TestRedactsSecretAssignmentsInFreeText(t *testing.T) {
	got := RedactText("running with GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwx set")
	if strings.Contains(got, "ghp_1234567890abcdefghijklmnopqrstuvwx") {
		t.Fatalf("assignment value survived redaction: %q", got)
	}
	if !strings.Contains(got, "GITHUB_TOKEN=") {
		t.Fatalf("the variable name should stay, only its value goes: %q", got)
	}
}

// Flag-directed redaction is scoped per tool on purpose. Applying curl's flags
// everywhere would blank the pattern in `grep -E` and the filename in `sort -u`,
// destroying exactly the diagnostics these logs exist to provide.
func TestFlagDirectedRedactionIsScopedToTheToolThatMeansIt(t *testing.T) {
	curl := RedactArgv("curl", []string{"-u", "alice:hunter2", "https://example.com"})
	if strings.Contains(strings.Join(curl, " "), "hunter2") {
		t.Fatalf("curl -u credentials must be redacted: %q", curl)
	}

	sorted := RedactArgv("sort", []string{"-u", "notes.txt"})
	if sorted[1] != "notes.txt" {
		t.Fatalf("`sort -u notes.txt` must keep its filename, got %q", sorted[1])
	}

	grepped := RedactArgv("grep", []string{"-E", "^error:.*timeout$", "app.log"})
	if grepped[1] != "^error:.*timeout$" {
		t.Fatalf("`grep -E` must keep its pattern, got %q", grepped[1])
	}
}

func TestRedactsFlagValueInTheSameElement(t *testing.T) {
	argv := RedactArgv("wget", []string{"--password=hunter2", "https://example.com"})
	if strings.Contains(argv[0], "hunter2") {
		t.Fatalf("--flag=value credential survived: %q", argv[0])
	}
	if !strings.HasPrefix(argv[0], "--password=") {
		t.Fatalf("the flag name should stay: %q", argv[0])
	}
}

func TestRedactFieldValueDropsSecretKeyedFields(t *testing.T) {
	if got := redactFieldValue("session_token", "anything at all"); got != redactedMarker {
		t.Fatalf("a secret-named field must be dropped whole, got %v", got)
	}
	if got := redactFieldValue("tool", "sha256sum"); got != "sha256sum" {
		t.Fatalf("an ordinary field must survive, got %v", got)
	}
}
