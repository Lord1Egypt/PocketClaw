package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

type userVisibleLogCase struct {
	Name     string `json:"name"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

func loadUserVisibleLogContract(t *testing.T) []userVisibleLogCase {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", "test", "fixtures", "user_visible_log_contract.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	var cases []userVisibleLogCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("Unmarshal contract: %v", err)
	}
	return cases
}

func TestNormalizeUserVisibleLogContract(t *testing.T) {
	for _, tt := range loadUserVisibleLogContract(t) {
		t.Run(tt.Name, func(t *testing.T) {
			got := normalizeUserVisibleLog(tt.Input)
			if got != tt.Expected {
				t.Fatalf("normalizeUserVisibleLog() = %q, want %q", got, tt.Expected)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("normalizeUserVisibleLog() returned invalid UTF-8: %q", got)
			}
			if !strings.Contains(tt.Input, "�") && strings.Contains(got, "�") {
				t.Fatalf("normalizeUserVisibleLog() introduced U+FFFD: %q", got)
			}
		})
	}
}

func TestLogBufferStoresOnlyNormalizedUserVisibleLines(t *testing.T) {
	buffer := NewLogBuffer(20)
	want := make([]string, 0)
	for _, tt := range loadUserVisibleLogContract(t) {
		buffer.Append(tt.Input)
		if tt.Expected != "" {
			want = append(want, tt.Expected)
		}
	}

	got, total, _ := buffer.LinesSince(0)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("LinesSince() = %#v, want %#v", got, want)
	}
	if total != len(want) {
		t.Fatalf("total = %d, want %d; suppressed lines must not consume history", total, len(want))
	}
}

func TestNormalizeUserVisibleLogDropsMalformedBytesWithoutReplacement(t *testing.T) {
	got := normalizeUserVisibleLog("valid µs \xff عربي")
	if got != "valid µs  عربي" {
		t.Fatalf("normalizeUserVisibleLog() = %q", got)
	}
	if strings.Contains(got, "�") {
		t.Fatalf("normalizeUserVisibleLog() introduced U+FFFD: %q", got)
	}
}

func TestNormalizeUserVisibleLogMapsOnlyExactPicoLoggerIdentity(t *testing.T) {
	got := normalizeUserVisibleLog("INF pico pico.go:1013 > WebSocket client connected")
	want := "INF realtime realtime.go:1013 > WebSocket client connected"
	if got != want {
		t.Fatalf("normalizeUserVisibleLog() = %q, want %q", got, want)
	}

	for input, want := range map[string]string{
		"INF pico picophone.go:1013 > exact component":        "INF realtime picophone.go:1013 > exact component",
		"INF picometer pico.go:1013 > exact caller":           "INF picometer realtime.go:1013 > exact caller",
		"INF picometer picophone.go:1013 > substrings":        "INF picometer picophone.go:1013 > substrings",
		"INF pico_client pico_client.go:1013 > compatibility": "INF pico_client pico_client.go:1013 > compatibility",
	} {
		if got := normalizeUserVisibleLog(input); got != want {
			t.Errorf("exact identity mapping: got %q, want %q", got, want)
		}
	}
}

func TestNormalizeUserVisibleLogRedactsTelegramBotAPICredentials(t *testing.T) {
	token := "123456789:AAExampleSecretTokenValue"
	tests := map[string]struct {
		input string
		want  string
	}{
		"getMe": {
			input: `DBG telego bot.go:247 > API call to: "https://api.telegram.org/bot` + token + `/getMe"`,
			want:  `DBG telego bot.go:247 > Telegram API call: getMe`,
		},
		"getUpdates timeout": {
			input: `ERR telego bot.go:170 > Execution error getUpdates: request call: Post "https://api.telegram.org/bot` + token + `/getUpdates": context deadline exceeded`,
			want:  `ERR telego bot.go:170 > Execution error getUpdates: request call: Post "Telegram API call: getUpdates": context deadline exceeded`,
		},
		"sendMessage with data": {
			input: `DBG telego bot.go:245 > API call to: "https://api.telegram.org/bot` + token + `/sendMessage", with data: {"chat_id":42}`,
			want:  `DBG telego bot.go:245 > Telegram API call: sendMessage, with data: {"chat_id":42}`,
		},
		"old partial editMessageText mask": {
			input: `DBG telego bot.go:247 > API call to: "https://api.telegram.org/bot123456789:AAEx****alue/editMessageText"`,
			want:  `DBG telego bot.go:247 > Telegram API call: editMessageText`,
		},
		"arbitrary method failure": {
			input: `ERR telego bot.go:170 > POST "https://telegram.example/v1/bot` + token + `/answerCustomQuery" 503 53.616µs`,
			want:  `ERR telego bot.go:170 > POST "Telegram API call: answerCustomQuery" 503 53.616µs`,
		},
		"authorization": {
			input: `ERR telego bot.go:170 > Authorization: Bearer ` + token + ` failed`,
			want:  `ERR telego bot.go:170 > Authorization: <redacted> failed`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := normalizeUserVisibleLog(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeUserVisibleLog() = %q, want %q", got, tt.want)
			}
			assertNoTelegramTokenFragment(t, got, token)
		})
	}
}

func TestLogBufferNeverStoresTelegramBotCredentialFragments(t *testing.T) {
	token := "123456789:AAExampleSecretTokenValue"
	buffer := NewLogBuffer(20)
	buffer.Append(`DBG telego bot.go:247 > API call to: "https://api.telegram.org/bot` + token + `/getMe"`)
	buffer.Append(`ERR telego bot.go:170 > Post "https://api.telegram.org/bot` + token + `/getUpdates": context deadline exceeded`)

	lines, total, _ := buffer.LinesSince(0)
	if total != 2 || len(lines) != 2 {
		t.Fatalf("stored history = %#v, total %d; want two useful events", lines, total)
	}
	joined := strings.Join(lines, "\n")
	assertNoTelegramTokenFragment(t, joined, token)
	for _, operation := range []string{"getMe", "getUpdates", "context deadline exceeded"} {
		if !strings.Contains(joined, operation) {
			t.Fatalf("stored history lost useful detail %q: %q", operation, joined)
		}
	}
}

func assertNoTelegramTokenFragment(t *testing.T, value, token string) {
	t.Helper()
	for _, forbidden := range []string{token, "123456789:", "AAExample", "TokenValue", "AAEx****alue"} {
		if strings.Contains(value, forbidden) {
			t.Fatalf("credential fragment %q remains in %q", forbidden, value)
		}
	}
}
