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
	orphanedCSIPattern               = regexp.MustCompile(`\[(?:\??[0-9:;<=>]+)[0-9:;<=>? ]*[ABCDEFGHJKSTfmnsu]`)
	otherEscapePattern               = regexp.MustCompile(`\x1B[ -/]*[@-~]`)
	routinePicoWSPattern             = regexp.MustCompile(`(?:^| > )GET /pico/ws (?:101|2[0-9]{2})(?:\s|$)`)
	picoWSRequestPattern             = regexp.MustCompile(`((?:^| > )[A-Z]+) /pico/ws ([0-9]{3})(\s|$)`)
	legacyGatewayStartPattern        = regexp.MustCompile(`Starting gateway process \([^\r\n)]*\)`)
	picoLoggerComponentPattern       = regexp.MustCompile(`(?m)(^|[ \t])([A-Z]{3}) pico ([^ \t]+:[0-9]+)([ \t]+>)`)
	picoLoggerCallerPattern          = regexp.MustCompile(`(?m)(^|[ \t])([A-Z]{3}) ([^ \t]+) pico\.go:([0-9]+)([ \t]+>)`)
	telegramBotAPIURLPattern         = regexp.MustCompile(`(?i)https?://[^\s"']*/bot[^/\s"']+/(?:test/)?([A-Za-z][A-Za-z0-9_]*)`)
	telegramAPICallWrapperPattern    = regexp.MustCompile(`(?i)API call to: "Telegram API call: ([A-Za-z][A-Za-z0-9_]*)"`)
	authorizationCredentialPattern   = regexp.MustCompile(`(?i)(authorization[=:][ \t]*)(?:\[?(?:bearer|basic)[ \t]+)[A-Za-z0-9._~+/%:=-]+\]?`)
)

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
	if routinePicoWSPattern.MatchString(result) {
		return ""
	}
	return picoWSRequestPattern.ReplaceAllString(result, "$1 /internal realtime connection $2$3")
}
