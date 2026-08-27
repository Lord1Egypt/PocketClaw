package api

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	oscPattern                       = regexp.MustCompile(`(?:\x1B\]|\x{009D})[\s\S]*?(?:\x07|\x1B\\|\x{009C})`)
	unterminatedOSCPattern           = regexp.MustCompile(`(?m)(?:\x1B\]|\x{009D})[^\n]*$`)
	stringControlPattern             = regexp.MustCompile(`(?:\x1B[P\^_X]|[\x{0090}\x{0098}\x{009E}\x{009F}])[\s\S]*?(?:\x1B\\|\x{009C})`)
	unterminatedStringControlPattern = regexp.MustCompile(`(?m)(?:\x1B[P\^_X]|[\x{0090}\x{0098}\x{009E}\x{009F}])[^\n]*$`)
	csiPattern                       = regexp.MustCompile(`(?:\x1B\[|\x{009B})[0-?]*[ -/]*[@-~]`)
	orphanedCSIPattern               = regexp.MustCompile(`\[(?:\?[0-9:;]+|[0-9][0-9:;]*)[ ]*[ABCDEFGHJKSTfmnsu]`)
	otherEscapePattern               = regexp.MustCompile(`\x1B[ -/]*[@-~]`)
	routinePicoWSPattern             = regexp.MustCompile(`(?:^| > )GET /pico/ws (?:101|2[0-9]{2})(?:\s|$)`)
	picoWSRequestPattern             = regexp.MustCompile(`((?:^| > )[A-Z]+) /pico/ws ([0-9]{3})(\s|$)`)
	legacyGatewayStartPattern        = regexp.MustCompile(`Starting gateway process \([^\r\n)]*\)`)
	picoLoggerComponentPattern       = regexp.MustCompile(`(?m)(^|[ \t])([A-Z]{3}) pico ([^ \t]+:[0-9]+)([ \t]+>)`)
	picoLoggerCallerPattern          = regexp.MustCompile(`(?m)(^|[ \t])([A-Z]{3}) ([^ \t]+) pico\.go:([0-9]+)([ \t]+>)`)
	telegramBotAPIURLPattern         = regexp.MustCompile(`(?i)https?://[^\s"']*/bot[^/\s"']+/(?:test/)?([A-Za-z][A-Za-z0-9_]*)`)
	telegramAPICallWrapperPattern    = regexp.MustCompile(`(?i)API call to: "Telegram API call: ([A-Za-z][A-Za-z0-9_]*)"`)
	authorizationCredentialPattern   = regexp.MustCompile(`(?i)(authorization[=:][ \t]*)(?:\[?(?:bearer|basic)[ \t]+)[A-Za-z0-9._~+/%:=-]+\]?`)
	legacyPIDFilePathPattern         = regexp.MustCompile(`(?:[^\s"']*[/\\])?\.picoclaw\.pid(?:\.tmp)?([\s"']|$)`)
	telegramSuccessfulNilError       = regexp.MustCompile(`((?:^| > )API response [A-Za-z][A-Za-z0-9_]*: Ok: true, Err:) \[<nil>\]`)
	routineTelegramGetUpdatesCall    = regexp.MustCompile(`(?:^| > )Telegram API call: getUpdates(?:, with data:.*)?$`)
	routineEmptyGetUpdatesResponse   = regexp.MustCompile(`(?:^| > )API response getUpdates: Ok: true, Err: none, Result: \[\](?:\s|$)`)
)

var exactCompatibilityMessages = map[string]string{
	"Starting Pico Protocol channel":                   "Starting PocketClaw realtime channel",
	"Pico Protocol channel started":                    "PocketClaw realtime channel started",
	"Stopping Pico Protocol channel":                   "Stopping PocketClaw realtime channel",
	"Pico Protocol channel stopped":                    "PocketClaw realtime channel stopped",
	"wrote pid file: <gateway PID file> success":       "Gateway PID file written successfully",
	"Failed to finalize streamed pico reasoning":       "Failed to finalize streamed realtime reasoning",
	"Pico reasoning publish skipped (timeout/cancel)":  "Realtime reasoning publish skipped (timeout/cancel)",
	"Failed to publish pico reasoning (best-effort)":   "Failed to publish realtime reasoning (best-effort)",
	"Failed to publish pico reasoning":                 "Failed to publish realtime reasoning",
	"Failed to publish pico interim assistant content": "Failed to publish realtime interim assistant content",
	"Failed to serialize pico tool calls":              "Failed to serialize realtime tool calls",
	"Failed to publish pico tool calls":                "Failed to publish realtime tool calls",
}

// normalizeUserVisibleLog applies the plain-text contract shared by every
// PocketClaw diagnostic surface. It removes terminal protocols without
// removing printable Unicode, and handles only known compatibility leaks.
func normalizeUserVisibleLog(input string) string {
	if input == "" {
		return input
	}

	// Drop malformed source bytes rather than introducing U+FFFD. Valid UTF-8,
	// including a literal replacement character supplied by the source, stays
	// untouched.
	if !utf8.ValidString(input) {
		input = strings.ToValidUTF8(input, "")
	}

	text := strings.ReplaceAll(input, "\r\n", "\n")
	text = oscPattern.ReplaceAllString(text, "")
	text = unterminatedOSCPattern.ReplaceAllString(text, "")
	text = stringControlPattern.ReplaceAllString(text, "")
	text = unterminatedStringControlPattern.ReplaceAllString(text, "")
	text = csiPattern.ReplaceAllString(text, "")
	text = orphanedCSIPattern.ReplaceAllString(text, "")
	text = otherEscapePattern.ReplaceAllString(text, "")

	var output strings.Builder
	currentLine := make([]rune, 0, len(text))
	flushLine := func(newline bool) {
		output.WriteString(string(currentLine))
		currentLine = currentLine[:0]
		if newline {
			output.WriteByte('\n')
		}
	}

	for _, r := range text {
		switch {
		case r == '\n':
			flushLine(true)
		case r == '\r':
			currentLine = currentLine[:0]
		case r == '\b':
			if len(currentLine) > 0 {
				currentLine = currentLine[:len(currentLine)-1]
			}
		case r == '\t':
			currentLine = append(currentLine, r)
		case r < 0x20 || (r >= 0x7f && r <= 0x9f):
			continue
		default:
			currentLine = append(currentLine, r)
		}
	}
	flushLine(false)

	result := legacyGatewayStartPattern.ReplaceAllString(output.String(), "Starting gateway process")
	result = picoLoggerComponentPattern.ReplaceAllString(result, "${1}${2} realtime ${3}${4}")
	result = picoLoggerCallerPattern.ReplaceAllString(result, "${1}${2} ${3} realtime.go:${4}${5}")
	result = telegramBotAPIURLPattern.ReplaceAllString(result, "Telegram API call: $1")
	result = telegramAPICallWrapperPattern.ReplaceAllString(result, "Telegram API call: $1")
	result = authorizationCredentialPattern.ReplaceAllString(result, "${1}<redacted>")
	result = legacyPIDFilePathPattern.ReplaceAllString(result, "<gateway PID file>$1")
	result = telegramSuccessfulNilError.ReplaceAllString(result, "${1} none")
	result = normalizePicoStructuredFields(result)
	result = normalizeExactCompatibilityMessages(result)
	if routineTelegramGetUpdatesCall.MatchString(result) || routineEmptyGetUpdatesResponse.MatchString(result) {
		return ""
	}
	if routinePicoWSPattern.MatchString(result) {
		return ""
	}
	return picoWSRequestPattern.ReplaceAllString(result, "$1 /internal realtime connection $2$3")
}

func normalizePicoStructuredFields(input string) string {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		internalChannel := hasExactLogToken(line, "channel=pico")
		line = replaceExactLogToken(line, "channel=pico", "channel=pocketclaw")
		line = replaceExactLogToken(line, "type=pico", "type=pocketclaw")
		if internalChannel {
			line = replaceExactLogToken(line, "path=/pico/", "path=<internal>")
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func normalizeExactCompatibilityMessages(input string) string {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		messageStart := 0
		if separator := strings.Index(line, " > "); separator >= 0 {
			messageStart = separator + len(" > ")
		}
		messageAndFields := line[messageStart:]
		for message, replacement := range exactCompatibilityMessages {
			if messageAndFields == message {
				lines[i] = line[:messageStart] + replacement
				break
			}
			if strings.HasPrefix(messageAndFields, message+" ") {
				lines[i] = line[:messageStart] + replacement + messageAndFields[len(message):]
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func hasExactLogToken(input, token string) bool {
	return replaceExactLogToken(input, token, token+"\x00") != input
}

func replaceExactLogToken(input, token, replacement string) string {
	var output strings.Builder
	searchFrom := 0
	for searchFrom < len(input) {
		relative := strings.Index(input[searchFrom:], token)
		if relative < 0 {
			break
		}
		start := searchFrom + relative
		end := start + len(token)
		beforeOK := start == 0 || isLogTokenSpace(input[start-1])
		afterOK := end == len(input) || isLogTokenSpace(input[end])
		if beforeOK && afterOK {
			output.WriteString(input[searchFrom:start])
			output.WriteString(replacement)
			searchFrom = end
			continue
		}
		output.WriteString(input[searchFrom : start+1])
		searchFrom = start + 1
	}
	if searchFrom == 0 {
		return input
	}
	output.WriteString(input[searchFrom:])
	return output.String()
}

func isLogTokenSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}
