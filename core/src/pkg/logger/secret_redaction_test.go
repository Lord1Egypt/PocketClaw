package logger

import "testing"

// Provider credentials reach a log the same way a Telegram token does: inside a
// string some upstream library built, not through a field this logger controls.
//
// Every value below is fabricated. None is, or resembles, a real credential.
func TestRedactsProviderCredentialForms(t *testing.T) {
	const fakeKey = "abcdefghijklmnopqrstuvwxyz012345"

	cases := []struct {
		name  string
		input string
		leak  string
	}{
		{
			"api-key header",
			`request failed: {"X-Api-Key": "` + fakeKey + `"}`,
			fakeKey,
		},
		{
			"api_key struct dump",
			"provider error: api_key=" + fakeKey + " rejected",
			fakeKey,
		},
		{
			"x-api-key with colon",
			"upstream sent x-api-key: " + fakeKey,
			fakeKey,
		},
		{
			"query parameter",
			"GET https://example.invalid/v1/models?key=" + fakeKey + " -> 401",
			fakeKey,
		},
		{
			"access_token query parameter",
			"redirect to https://example.invalid/cb?state=1&access_token=" + fakeKey,
			fakeKey,
		},
		{
			"OpenAI-style prefix",
			"401 from provider using sk-Aa0Bb1Cc2Dd3Ee4Ff5Gg6Hh7",
			"sk-Aa0Bb1Cc2Dd3Ee4Ff5Gg6Hh7",
		},
		{
			"Anthropic-style prefix",
			"auth error for sk-ant-Aa0Bb1Cc2Dd3Ee4Ff5",
			"sk-ant-Aa0Bb1Cc2Dd3Ee4Ff5",
		},
		{
			"Google-style prefix",
			"request denied: AIzaSyA0000000000000000000000000000",
			"AIzaSyA0000000000000000000000000000",
		},
		{
			"GitHub-style prefix",
			"push rejected for ghp_A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8",
			"ghp_A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8",
		},
		{
			"Slack-style prefix",
			"channel post failed with xoxb-1111111111-abcdefghij",
			"xoxb-1111111111-abcdefghij",
		},
		{
			"Authorization header still redacted",
			"Authorization: Bearer " + fakeKey,
			fakeKey,
		},
		{
			"Telegram bot URL still redacted",
			`API call to: "https://api.telegram.org/bot123456789:AAFfffffffffffffffffffffffffffff/getMe"`,
			"123456789:AAFfffffffffffffffffffffffffffff",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactSecrets(tc.input)
			if contains(got, tc.leak) {
				t.Fatalf("credential survived redaction:\n  in:  %s\n  out: %s", tc.input, got)
			}
			if !contains(got, "<redacted>") {
				t.Fatalf("nothing was redacted:\n  in:  %s\n  out: %s", tc.input, got)
			}
		})
	}
}

// A log that has destroyed its own diagnostic value is not a safer log. These
// are the strings a blunter rule — "redact any long token" — would eat.
func TestRedactionLeavesOrdinaryDiagnosticTextAlone(t *testing.T) {
	for _, input := range []string{
		"model gpt-4o-mini returned 429 after 3 retries",
		"session sk_v1_4a74954be48c6b76a61243556ecfb66d4b2eb5d27f56ff84f92a25da0096c578 resumed",
		"staged core fingerprint 05418871286f6150b9066a0be9da337531e3f98fb568f3b2142c9896da8f5b16",
		"GET /api/gateway/status 200 in 4ms",
		"channel telegram started, 2 allowed senders",
		"wrote pid file: /data/user/0/app/files/.picoclaw/.picoclaw.pid success",
		"tool run failed: exit status 1",
	} {
		if got := redactSecrets(input); got != input {
			t.Errorf("ordinary text was altered:\n  in:  %s\n  out: %s", input, got)
		}
	}
}

func contains(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
