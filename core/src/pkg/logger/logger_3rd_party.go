// this file is for compatible with 3rd party loggers, should not be called in PicoClaw project

package logger

import (
	"fmt"
	"regexp"
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
)

const emptyTelegramGetUpdatesResponse = "API response getUpdates: Ok: true, Err: [<nil>], Result: []"

// redactSecrets runs before logMessage and therefore before any console/file
// writer. Third-party libraries can keep useful method, status, and error text,
// but no credential fragment reaches stdout or a downstream log store.
func redactSecrets(s string) string {
	s = authorizationPattern.ReplaceAllString(s, "${1}<redacted>")
	s = telegramBotURLTokenPattern.ReplaceAllString(s, "bot<redacted>")
	return telegramBareTokenPattern.ReplaceAllString(s, "<redacted>")
}

// prepareThirdPartyLog applies policy before logMessage reaches any writer.
// Telego emits a request line before it knows the result, so getUpdates request
// lines are always omitted; failures still emit their separate Execution error
// line, while non-empty and unsuccessful API responses remain visible.
func prepareThirdPartyLog(level LogLevel, component, input string) (string, bool) {
	message := redactSecrets(input)
	if level != DEBUG || component != "telego" {
		return message, true
	}
	if message == emptyTelegramGetUpdatesResponse {
		return "", false
	}
	match := telegramAPICallPattern.FindStringSubmatch(message)
	if len(match) == 2 && strings.EqualFold(match[1], "getUpdates") {
		return "", false
	}
	return message, true
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
