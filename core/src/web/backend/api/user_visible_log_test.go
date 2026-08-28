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

func TestLogBufferSuppressesOnlyRoutineEmptyTelegramPolling(t *testing.T) {
	token := "123456789:AAExampleSecretTokenValue"
	buffer := NewLogBuffer(20)
	for range 4 {
		buffer.Append(`DBG telego bot.go:247 > API call to: "https://api.telegram.org/bot` + token + `/getUpdates"`)
		buffer.Append("DBG telego bot.go:173 > API response getUpdates: Ok: true, Err: [<nil>], Result: []")
	}

	if lines, total, _ := buffer.LinesSince(0); total != 0 || len(lines) != 0 {
		t.Fatalf("routine empty polls grew stored history: total=%d lines=%#v", total, lines)
	}

	important := []string{
		`ERR telego bot.go:170 > Execution error getUpdates: request call: Post "https://api.telegram.org/bot` + token + `/getUpdates": context deadline exceeded`,
		`DBG telego bot.go:173 > API response getUpdates: Ok: true, Err: [<nil>], Result: [{"update_id":42}]`,
		`DBG telego bot.go:173 > API response getUpdates: Ok: false, Err: [429 "Too Many Requests"], Result: []`,
		`DBG telego bot.go:247 > API call to: "https://api.telegram.org/bot` + token + `/sendMessage"`,
		`DBG telego bot.go:247 > API call to: "https://api.telegram.org/bot` + token + `/editMessageText"`,
	}
	for _, line := range important {
		buffer.Append(line)
	}
	lines, total, _ := buffer.LinesSince(0)
	if total != len(important) || len(lines) != len(important) {
		t.Fatalf("important Telegram history = %#v, total=%d; want %d", lines, total, len(important))
	}
	joined := strings.Join(lines, "\n")
	for _, detail := range []string{"context deadline exceeded", "updates=1", "error_code=429", "sendMessage", "editMessageText"} {
		if !strings.Contains(joined, detail) {
			t.Fatalf("stored history lost %q: %q", detail, joined)
		}
	}
	for _, private := range []string{"update_id", "Result:"} {
		if strings.Contains(joined, private) {
			t.Fatalf("stored history retained Telegram payload %q: %q", private, joined)
		}
	}
	assertNoTelegramTokenFragment(t, joined, token)
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
			want:  `DBG telego bot.go:245 > Telegram API call: sendMessage`,
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

func TestLogBufferRedactsAgentInternalsAndTelegramPayloadsBeforeStorage(t *testing.T) {
	buffer := NewLogBuffer(20)
	raw := []string{
		"DBG agent context.go:989 > System prompt preview preview=\"# picoclaw 🦞 You are picoclaw, a helpful AI assistant.\"",
		"DBG agent pipeline_llm.go:147 > Full LLM request iteration=1 messages_json=\"actual conversation contents\" tools_json=\"full schema\"",
		"DBG agent pipeline_llm.go:565 > LLM response reasoning=\"private reasoning text\" model=test-model",
		"INF agent agent_message.go:159 > Routed message session_key=sk_v1_SUPERSECRETVALUE scope_key=sk_v1_SUPERSECRETVALUE route_main_session=sk_v1_SUPERSECRETVALUE route_channel=pico",
		"DBG events runtime_event_logger.go:180 > Runtime event chat_id=pico:1234 sender_id=pico-user inbound_chat_id=pico:1234 inbound_sender_id=pico-user",
		"DBG telego bot.go:173 > API response getUpdates: Ok: true, Err: [<nil>], Result: [{\"update_id\":42,\"message\":{\"from\":{\"id\":581234567,\"first_name\":\"Ada\",\"last_name\":\"Lovelace\",\"username\":\"private_user\",\"language_code\":\"ar\"},\"chat\":{\"id\":581234567},\"text\":\"مرحبا private body\"}}]",
		"DBG telego bot.go:173 > API response editMessageText: Ok: true, Err: [<nil>], Result: {\"chat\":{\"id\":581234567},\"text\":\"private assistant body\"}",
	}
	for _, line := range raw {
		buffer.Append(line)
	}

	lines, total, _ := buffer.LinesSince(0)
	if total != len(raw) {
		t.Fatalf("stored total = %d, want %d: %#v", total, len(raw), lines)
	}
	rendered := strings.Join(lines, "\n")
	for _, forbidden := range []string{
		"# picoclaw", "You are picoclaw", "actual conversation contents", "full schema",
		"private reasoning text", "sk_v1_SUPERSECRETVALUE", "pico:1234", "pico-user",
		"581234567", "Ada", "Lovelace", "private_user", "language_code", "مرحبا private body",
		"private assistant body", "update_id",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stored normal DEBUG contains private value %q: %q", forbidden, rendered)
		}
	}
	for _, required := range []string{
		"preview=<redacted>", "messages_json=<redacted>", "tools_json=<redacted>",
		"reasoning=<redacted>", "model=test-model", "session_key=<redacted>",
		"scope_key=<redacted>", "route_main_session=<redacted>",
		"route_channel=pocketclaw", "chat_id=<internal>", "sender_id=<internal>",
		"Telegram update received updates=1 type=message",
		"Telegram API completed operation=editMessageText ok=true",
	} {
		if !strings.Contains(rendered, required) {
			t.Fatalf("stored normal DEBUG lost safe metadata %q: %q", required, rendered)
		}
	}
}

func TestNormalizeUserVisibleLogMapsOnlyKnownCompatibilityDisplays(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"channel and type": {
			input: "DBG channels manager.go:1101 > Attempting to initialize channel channel=pico type=pico",
			want:  "DBG channels manager.go:1101 > Attempting to initialize channel channel=pocketclaw type=pocketclaw",
		},
		"internal route": {
			input: "INF channels manager.go:1291 > Webhook handler registered channel=pico path=/pico/",
			want:  "INF channels manager.go:1291 > Webhook handler registered channel=pocketclaw path=<internal>",
		},
		"channel startup": {
			input: "INF channels manager.go:1347 > Starting channel channel=pico",
			want:  "INF channels manager.go:1347 > Starting channel channel=pocketclaw",
		},
		"security warning": {
			input: "WRN channels base.go:131 > SECURITY: Channel allows EVERYONE (allow_from is empty) channel=pico hint=\"Set allow_from to your ID, or use '*' to explicitly acknowledge open access.\"",
			want:  "WRN channels base.go:131 > SECURITY: Channel allows EVERYONE (allow_from is empty) channel=pocketclaw hint=\"Set allow_from to your ID, or use '*' to explicitly acknowledge open access.\"",
		},
		"protocol starting": {
			input: "INF pico pico.go:248 > Starting Pico Protocol channel",
			want:  "INF realtime realtime.go:248 > Starting PocketClaw realtime channel",
		},
		"protocol started": {
			input: "INF pico pico.go:251 > Pico Protocol channel started",
			want:  "INF realtime realtime.go:251 > PocketClaw realtime channel started",
		},
		"protocol stopping": {
			input: "INF pico pico.go:257 > Stopping Pico Protocol channel",
			want:  "INF realtime realtime.go:257 > Stopping PocketClaw realtime channel",
		},
		"protocol stopped": {
			input: "INF pico pico.go:272 > Pico Protocol channel stopped",
			want:  "INF realtime realtime.go:272 > PocketClaw realtime channel stopped",
		},
		"pid file": {
			input: "DBG pid pidfile.go:112 > wrote pid file: /data/user/0/app/files/.picoclaw/.picoclaw.pid success",
			want:  "DBG pid pidfile.go:112 > Gateway PID file written successfully",
		},
		"realtime failure with structured fields": {
			input: "WRN agent agent_outbound.go:205 > Failed to publish pico reasoning channel=pico error=timeout",
			want:  "WRN agent agent_outbound.go:205 > Failed to publish realtime reasoning channel=pocketclaw error=timeout",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := normalizeUserVisibleLog(tt.input); got != tt.want {
				t.Fatalf("normalizeUserVisibleLog() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeUserVisibleLogDoesNotRewritePicoSubstrings(t *testing.T) {
	input := "INF picometer picophone.go:42 > compatibility pico_client.go topic=pico-test filename=my-pico-notes.txt archive=.picoclaw.pid.backup"
	if got := normalizeUserVisibleLog(input); got != input {
		t.Fatalf("normalizeUserVisibleLog() = %q, want unchanged %q", got, input)
	}
}

func TestRepresentativeStartupHasNoUnintendedLegacyBrand(t *testing.T) {
	token := "123456789:AAExampleSecretTokenValue"
	raw := []string{
		"INF gateway gateway.go:1033 > Starting gateway process",
		"DBG pid pidfile.go:112 > wrote pid file: /data/user/0/app/files/.picoclaw/.picoclaw.pid success",
		"INF tools loader.go:80 > Tools loaded count=17",
		"INF agent agent.go:90 > Agent initialized",
		"INF telegram telegram.go:120 > Telegram channel initialized username=@PocketClawBot",
		"DBG channels manager.go:1101 > Attempting to initialize channel channel=pico type=pico",
		"WRN channels base.go:131 > SECURITY: Channel allows EVERYONE (allow_from is empty) channel=pico hint=\"Set allow_from to your ID, or use '*' to explicitly acknowledge open access.\"",
		"INF channels manager.go:1291 > Webhook handler registered channel=pico path=/pico/",
		"INF channels manager.go:1347 > Starting channel channel=pico",
		"INF pico pico.go:248 > Starting Pico Protocol channel",
		`DBG telego bot.go:247 > API call to: "https://api.telegram.org/bot` + token + `/getUpdates"`,
	}

	buffer := NewLogBuffer(len(raw))
	for _, line := range raw {
		buffer.Append(line)
	}
	lines, total, _ := buffer.LinesSince(0)
	if total != len(raw)-1 {
		t.Fatalf("stored total = %d, want %d after routine poll suppression", total, len(raw)-1)
	}
	rendered := strings.Join(lines, "\n")
	for _, forbidden := range []string{"pico", "picoclaw", "sipeed"} {
		if strings.Contains(strings.ToLower(rendered), forbidden) {
			t.Fatalf("representative user-visible startup contains %q: %q", forbidden, rendered)
		}
	}
	for _, required := range []string{
		"Gateway PID file written successfully",
		"channel=pocketclaw type=pocketclaw",
		"channel=pocketclaw path=<internal>",
		"SECURITY: Channel allows EVERYONE",
		"Starting PocketClaw realtime channel",
	} {
		if !strings.Contains(rendered, required) {
			t.Fatalf("representative user-visible startup lost %q: %q", required, rendered)
		}
	}
	if strings.Contains(rendered, "Telegram API call: getUpdates") {
		t.Fatalf("representative startup retained routine Telegram poll: %q", rendered)
	}
}
