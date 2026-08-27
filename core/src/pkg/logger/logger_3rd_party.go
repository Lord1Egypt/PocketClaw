// this file is for compatible with 3rd party loggers, should not be called in PicoClaw project

package logger

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	// Telegram puts the complete bot credential in every Bot API URL. Match
	// both the normal colon and its percent-encoded form without retaining the
	// public-ID prefix or any secret prefix/suffix.
	telegramBotURLTokenPattern = regexp.MustCompile(`(?i)\bbot\d{6,}(?::|%3A)[A-Za-z0-9_-]{10,}`)
	telegramBareTokenPattern   = regexp.MustCompile(`(?i)\b\d{6,}(?::|%3A)[A-Za-z0-9_-]{10,}\b`)
	authorizationPattern       = regexp.MustCompile(`(?i)(authorization[=:][ \t]*)(?:\[?(?:bearer|basic)[ \t]+)[A-Za-z0-9._~+/%:=-]+\]?`)
	telegramAPICallPattern     = regexp.MustCompile(`(?i)^API call to: "https?://[^"\s]*/bot<redacted>/(?:test/)?([A-Za-z][A-Za-z0-9_]*)"`)
	telegramSafeAPICallPattern = regexp.MustCompile(`(?i)^Telegram API call: ([A-Za-z][A-Za-z0-9_]*)(?:, with data:.*)?$`)
	telegramAPIOperation       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	telegramErrorCodePattern   = regexp.MustCompile(`Err: \[([0-9]+)(?:\s|\])`)
)

// redactSecrets runs before logMessage and therefore before any console/file
// writer. Third-party libraries can keep useful method, status, and error text,
// but no credential fragment reaches stdout or a downstream log store.
func redactSecrets(s string) string {
	s = authorizationPattern.ReplaceAllString(s, "${1}<redacted>")
	s = telegramBotURLTokenPattern.ReplaceAllString(s, "bot<redacted>")
	return telegramBareTokenPattern.ReplaceAllString(s, "<redacted>")
}

// prepareThirdPartyLog applies policy before logMessage reaches any writer.
// Telego request parameters and response bodies contain credentials, chat/user
// identifiers, names, and message text. Normal logs retain only the operation
// and result metadata. getUpdates request lines are omitted because the paired
// response carries the useful outcome; failures still emit their separate
// Execution error line.
func prepareThirdPartyLog(_ LogLevel, component, input string) (string, bool) {
	if component != "telego" {
		return redactSecrets(input), true
	}
	return NormalizeTelegramUserVisibleMessage(input)
}

// NormalizeTelegramUserVisibleMessage is the shared Core contract for Telego
// request/response text. It runs before Telego output reaches stdout and is
// also reused by the backend as a defense for older/raw input.
func NormalizeTelegramUserVisibleMessage(input string) (string, bool) {
	message := redactSecrets(input)
	if normalized, keep, matched := normalizeTelegramAPIResponse(message); matched {
		return normalized, keep
	}
	match := telegramAPICallPattern.FindStringSubmatch(message)
	if len(match) != 2 {
		match = telegramSafeAPICallPattern.FindStringSubmatch(message)
	}
	if len(match) == 2 {
		operation := match[1]
		if strings.EqualFold(operation, "getUpdates") {
			return "", false
		}
		return "Telegram API call: " + operation, true
	}
	return message, true
}

func normalizeTelegramAPIResponse(message string) (normalized string, keep, matched bool) {
	const prefix = "API response "
	if !strings.HasPrefix(message, prefix) {
		return "", false, false
	}

	remainder := strings.TrimPrefix(message, prefix)
	operationEnd := strings.Index(remainder, ": Ok: ")
	if operationEnd <= 0 {
		return "Telegram API response details=<redacted>", true, true
	}
	operation := remainder[:operationEnd]
	if !telegramAPIOperation.MatchString(operation) {
		return "Telegram API response details=<redacted>", true, true
	}
	details := remainder[operationEnd+len(": Ok: "):]
	okEnd := strings.Index(details, ", Err: [")
	if okEnd < 0 {
		return fmt.Sprintf("Telegram API response operation=%s details=<redacted>", operation), true, true
	}
	okText := details[:okEnd]

	switch okText {
	case "false":
		fields := fmt.Sprintf("Telegram API failed operation=%s ok=false", operation)
		if code := telegramErrorCodePattern.FindStringSubmatch(message); len(code) == 2 {
			fields += " error_code=" + code[1]
		}
		return fields, true, true
	case "true":
		if !strings.EqualFold(operation, "getUpdates") {
			return fmt.Sprintf("Telegram API completed operation=%s ok=true", operation), true, true
		}
	default:
		return fmt.Sprintf("Telegram API response operation=%s details=<redacted>", operation), true, true
	}

	const resultMarker = "], Result: "
	resultStart := strings.Index(details, resultMarker)
	if resultStart < 0 {
		return "Telegram API response operation=getUpdates ok=true result=malformed", true, true
	}
	rawResult := details[resultStart+len(resultMarker):]
	var updates []json.RawMessage
	if err := json.Unmarshal([]byte(rawResult), &updates); err != nil {
		return "Telegram API response operation=getUpdates ok=true result=malformed", true, true
	}
	if len(updates) == 0 {
		return "", false, true
	}

	types := telegramUpdateTypes(updates)
	if len(types) == 1 {
		return fmt.Sprintf("Telegram update received updates=%d type=%s", len(updates), types[0]), true, true
	}
	return fmt.Sprintf("Telegram update received updates=%d types=%s", len(updates), strings.Join(types, ",")), true, true
}

func telegramUpdateTypes(updates []json.RawMessage) []string {
	typeSet := make(map[string]struct{})
	for _, raw := range updates {
		var update map[string]json.RawMessage
		if err := json.Unmarshal(raw, &update); err != nil {
			typeSet["unknown"] = struct{}{}
			continue
		}
		found := false
		for _, updateType := range []string{
			"message", "edited_message", "channel_post", "edited_channel_post",
			"business_connection", "business_message", "edited_business_message",
			"deleted_business_messages", "message_reaction", "message_reaction_count",
			"inline_query", "chosen_inline_result", "callback_query", "shipping_query",
			"pre_checkout_query", "purchased_paid_media", "poll", "poll_answer",
			"my_chat_member", "chat_member", "chat_join_request", "chat_boost",
			"removed_chat_boost",
		} {
			if value, ok := update[updateType]; ok && len(value) > 0 && string(value) != "null" {
				typeSet[updateType] = struct{}{}
				found = true
				break
			}
		}
		if !found {
			typeSet["unknown"] = struct{}{}
		}
	}

	types := make([]string, 0, len(typeSet))
	for updateType := range typeSet {
		types = append(types, updateType)
	}
	sort.Strings(types)
	return types
}

func (b *Logger) write(level LogLevel, message string) {
	message, keep := prepareThirdPartyLog(level, b.component, message)
	if !keep {
		return
	}
	logMessage(level, b.component, message, nil)
}

// Logger implements common Logger interface
type Logger struct {
	component string
	levels    map[int]LogLevel
}

// Debug logs debug messages
func (b *Logger) Debug(v ...any) {
	b.write(DEBUG, fmt.Sprint(v...))
}

// Info logs info messages
func (b *Logger) Info(v ...any) {
	b.write(INFO, fmt.Sprint(v...))
}

// Warn logs warning messages
func (b *Logger) Warn(v ...any) {
	b.write(WARN, fmt.Sprint(v...))
}

// Error logs error messages
func (b *Logger) Error(v ...any) {
	b.write(ERROR, fmt.Sprint(v...))
}

// Debugf logs formatted debug messages
func (b *Logger) Debugf(format string, v ...any) {
	b.write(DEBUG, fmt.Sprintf(format, v...))
}

// Infof logs formatted info messages
func (b *Logger) Infof(format string, v ...any) {
	b.write(INFO, fmt.Sprintf(format, v...))
}

// Warnf logs formatted warning messages
func (b *Logger) Warnf(format string, v ...any) {
	b.write(WARN, fmt.Sprintf(format, v...))
}

// Warningf logs formatted warning messages
func (b *Logger) Warningf(format string, v ...any) {
	b.write(WARN, fmt.Sprintf(format, v...))
}

// Errorf logs formatted error messages
func (b *Logger) Errorf(format string, v ...any) {
	b.write(ERROR, fmt.Sprintf(format, v...))
}

// Fatalf logs formatted fatal messages and exits
func (b *Logger) Fatalf(format string, v ...any) {
	b.write(FATAL, fmt.Sprintf(format, v...))
}

// Log logs a message at a given level with caller information
// the func name must be this because 3rd party loggers expect this
// msgL: message level (DEBUG, INFO, WARN, ERROR, FATAL)
// caller: unused parameter reserved for compatibility
// format: format string
// a: format arguments
//
//nolint:goprintffuncname
func (b *Logger) Log(msgL, caller int, format string, a ...any) {
	level := LogLevel(msgL)
	if b.levels != nil {
		if lvl, ok := b.levels[msgL]; ok {
			level = lvl
		}
	}
	b.write(level, fmt.Sprintf(format, a...))
}

// Sync flushes log buffer (no-op for this implementation)
func (b *Logger) Sync() error {
	return nil
}

// WithLevels sets log levels mapping for this logger
func (b *Logger) WithLevels(levels map[int]LogLevel) *Logger {
	b.levels = levels
	return b
}

// NewLogger creates a new logger instance with optional component name
func NewLogger(component string) *Logger {
	return &Logger{component: component}
}
