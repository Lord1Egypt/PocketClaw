package logger

import (
	"strings"
	"testing"
)

const syntheticTelegramToken = "123456789:AAExampleSecretTokenValue"

func TestRedactSecretsRemovesCompleteTelegramCredential(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"getMe": {
			input: `API call to: "https://api.telegram.org/bot` + syntheticTelegramToken + `/getMe"`,
			want:  `API call to: "https://api.telegram.org/bot<redacted>/getMe"`,
		},
		"getUpdates timeout": {
			input: `Execution error getUpdates: Post "https://api.telegram.org/bot` + syntheticTelegramToken + `/getUpdates": context deadline exceeded`,
			want:  `Execution error getUpdates: Post "https://api.telegram.org/bot<redacted>/getUpdates": context deadline exceeded`,
		},
		"sendMessage status": {
			input: `POST https://api.telegram.org/bot` + syntheticTelegramToken + `/sendMessage 200 53.616µs`,
			want:  `POST https://api.telegram.org/bot<redacted>/sendMessage 200 53.616µs`,
		},
		"editMessageText failure": {
			input: `POST https://api.telegram.org/bot` + syntheticTelegramToken + `/editMessageText 500`,
			want:  `POST https://api.telegram.org/bot<redacted>/editMessageText 500`,
		},
		"arbitrary method and custom server": {
			input: `https://telegram.example.test/v1/bot` + syntheticTelegramToken + `/answerCustomQuery`,
			want:  `https://telegram.example.test/v1/bot<redacted>/answerCustomQuery`,
		},
		"percent encoded colon": {
			input: `https://api.telegram.org/bot123456789%3AAAExampleSecretTokenValue/getMe`,
			want:  `https://api.telegram.org/bot<redacted>/getMe`,
		},
		"bare credential": {
			input: `telegram token=` + syntheticTelegramToken,
			want:  `telegram token=<redacted>`,
		},
		"bearer authorization": {
			input: `Authorization: Bearer ` + syntheticTelegramToken,
			want:  `Authorization: <redacted>`,
		},
		"basic authorization": {
			input: `authorization=Basic dXNlcjpwYXNzd29yZA==`,
			want:  `authorization=<redacted>`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := redactSecrets(tt.input)
			if got != tt.want {
				t.Fatalf("redactSecrets() = %q, want %q", got, tt.want)
			}
			assertNoTelegramCredentialFragment(t, got)
		})
	}
}

func TestRedactSecretsPreservesPublicBotMetadata(t *testing.T) {
	t.Parallel()

	input := "Telegram bot @PocketClawBot has public bot ID 123456789"
	if got := redactSecrets(input); got != input {
		t.Fatalf("redactSecrets() = %q, want unchanged public metadata", got)
	}
}

func TestPrepareThirdPartyLogSuppressesOnlyRoutineEmptyGetUpdates(t *testing.T) {
	t.Parallel()

	request := `API call to: "https://api.telegram.org/bot` + syntheticTelegramToken + `/getUpdates", with data: {"timeout":30}`
	emptyResponse := "API response getUpdates: Ok: true, Err: [<nil>], Result: []"

	var history []string
	for range 4 {
		for _, raw := range []string{request, emptyResponse} {
			if message, keep := prepareThirdPartyLog(DEBUG, "telego", raw); keep {
				history = append(history, message)
			}
		}
	}
	if len(history) != 0 {
		t.Fatalf("routine empty polling grew history: %#v", history)
	}

	kept := map[string]string{
		"failure":            `Execution error getUpdates: Post "https://api.telegram.org/bot` + syntheticTelegramToken + `/getUpdates": context deadline exceeded`,
		"non-empty response": `API response getUpdates: Ok: true, Err: [<nil>], Result: [{"update_id":42}]`,
		"api error response": `API response getUpdates: Ok: false, Err: [429 "Too Many Requests"], Result: []`,
		"sendMessage":        `API call to: "https://api.telegram.org/bot` + syntheticTelegramToken + `/sendMessage"`,
		"editMessageText":    `API call to: "https://api.telegram.org/bot` + syntheticTelegramToken + `/editMessageText"`,
	}
	for name, raw := range kept {
		t.Run(name, func(t *testing.T) {
			message, keep := prepareThirdPartyLog(DEBUG, "telego", raw)
			if !keep {
				t.Fatalf("important Telegram diagnostic was suppressed: %q", raw)
			}
			if want := redactSecrets(raw); message != want {
				t.Fatalf("prepareThirdPartyLog() = %q, want %q", message, want)
			}
			assertNoTelegramCredentialFragment(t, message)
		})
	}
}

func TestPrepareThirdPartyLogDoesNotFilterOtherComponentsOrLevels(t *testing.T) {
	t.Parallel()

	input := "API response getUpdates: Ok: true, Err: [<nil>], Result: []"
	for _, tc := range []struct {
		level     LogLevel
		component string
	}{
		{level: INFO, component: "telego"},
		{level: DEBUG, component: "telegram"},
	} {
		if got, keep := prepareThirdPartyLog(tc.level, tc.component, input); !keep || got != input {
			t.Fatalf("prepareThirdPartyLog(%v, %q) = %q, %v; want unchanged", tc.level, tc.component, got, keep)
		}
	}
}

func assertNoTelegramCredentialFragment(t *testing.T, value string) {
	t.Helper()

	for _, forbidden := range []string{
		syntheticTelegramToken,
		"123456789:",
		"AAExample",
		"TokenValue",
		"AAEx****alue",
	} {
		if strings.Contains(value, forbidden) {
			t.Fatalf("credential fragment %q remains in %q", forbidden, value)
		}
	}
}
