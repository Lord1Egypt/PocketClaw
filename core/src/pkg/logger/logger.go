package logger

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/term"
)

type LogLevel = zerolog.Level

const (
	DEBUG = zerolog.DebugLevel
	INFO  = zerolog.InfoLevel
	WARN  = zerolog.WarnLevel
	ERROR = zerolog.ErrorLevel
	FATAL = zerolog.FatalLevel

	Component = "component"
)

var (
	logLevelNames = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		FATAL: "FATAL",
	}

	currentLevel  = INFO
	logger        zerolog.Logger
	logFile       *rotatingFile
	once          sync.Once
	mu            sync.RWMutex
	writers       []io.Writer
	consoleWriter zerolog.ConsoleWriter
)

func init() {
	once.Do(func() {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)

		// Reduce the caller to the file's own name before anything renders it.
		//
		// Zerolog reports the path the compiler recorded. Under -trimpath that
		// is the module path — github.com/sipeed/picoclaw/web/backend/api/
		// gateway.go — and without it an absolute build path. Both are
		// developer-facing, and both reach PocketClaw's user-visible Logs
		// screen and its exported log files. Only the file name carries
		// meaning for a user; the package is already shown as the component
		// field. This is the earliest structured layer that sees the caller,
		// so every writer and every export inherits the short form and no
		// message text is ever rewritten.
		zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
			return ShortCallerLocation(file, line)
		}

		isTTY := term.IsTerminal(int(os.Stdout.Fd()))

		consoleWriter = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05", // TODO: make it configurable???

			// Custom formatter to handle multiline strings and JSON objects
			FormatFieldValue: formatFieldValue,
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				Component,
				zerolog.CallerFieldName,
				zerolog.MessageFieldName,
			},
			FieldsExclude: []string{Component},
			FormatPrepare: func(fields map[string]any) error {
				if isTTY {
					fields[Component] = fmt.Sprintf("\x1b[33m%v\x1b[0m", fields[Component])
				}
				return nil
			},
			NoColor: !isTTY,
		}

		writers = append(writers, consoleWriter)

		logger = zerolog.New(io.MultiWriter(writers...)).With().Timestamp().Caller().Logger()
	})
}

func formatFieldValue(i any) string {
	var s string

	switch val := i.(type) {
	case string:
		s = val
	case []byte:
		s = string(val)
	default:
		return fmt.Sprintf("%v", i)
	}

	if unquoted, err := strconv.Unquote(s); err == nil {
		s = unquoted
	}

	if strings.Contains(s, "\n") {
		return fmt.Sprintf("\n%s", s)
	}

	if strings.Contains(s, " ") {
		if (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
			(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) {
			return s
		}
		return fmt.Sprintf("%q", s)
	}

	return s
}

func SetLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
	zerolog.SetGlobalLevel(level)
}

func SetConsoleLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	logger = logger.Level(level)
}

func DisableConsole() {
	mu.Lock()
	defer mu.Unlock()
	writers[0] = io.Discard
	logger = logger.Output(io.MultiWriter(writers...))
}

func EnableConsole() {
	mu.Lock()
	defer mu.Unlock()
	writers[0] = consoleWriter
	logger = logger.Output(io.MultiWriter(writers...))
}

func GetLevel() LogLevel {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// ParseLevel converts a case-insensitive level name to a LogLevel.
// Returns the level and true if valid, or (INFO, false) if unrecognized.
func ParseLevel(s string) (LogLevel, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return DEBUG, true
	case "info":
		return INFO, true
	case "warn", "warning":
		return WARN, true
	case "error":
		return ERROR, true
	case "fatal":
		return FATAL, true
	default:
		return INFO, false
	}
}

// SetLevelFromString sets the log level from a string value.
// If the string is empty or not a recognized level name, the current level is kept.
func SetLevelFromString(s string) {
	if s == "" {
		return
	}
	if level, ok := ParseLevel(s); ok {
		SetLevel(level)
	}
}

// Log rotation bounds.
//
// The log had no rotation at all and was pure append, so a long-lived install
// accumulated an unbounded file — one observed at 18 MB, still carrying lines a
// much older build had written. These numbers are deliberately small: this is a
// phone, the value of a gateway log is almost entirely in its recent tail, and
// anything older is a liability rather than an asset.
const (
	// maxLogFileBytes is the size at which the active log is rotated.
	maxLogFileBytes = 2 << 20 // 2 MiB

	// maxLogRotations is how many previous files are kept beside it, as
	// gateway.log.1 … gateway.log.N. Older ones are deleted, bounding the whole
	// directory at roughly (maxLogRotations + 1) * maxLogFileBytes.
	maxLogRotations = 2
)

// rotatingFile is the log sink: an append-only file that rotates itself once it
// crosses maxLogFileBytes.
//
// Checking the size when the file is opened is not enough. The gateway is a
// long-lived process, so a build that only rotated at startup would append past
// the threshold for as long as the process ran — which is precisely how the
// unbounded file this bound exists to prevent came about. The counter makes the
// bound hold within one process lifetime.
//
// The size is tracked rather than stat'd so there is no filesystem call in the
// path of a log line; it is seeded from the existing file so an append to a
// file inherited from a previous run is accounted for.
//
// The record that crosses the threshold is written to the *old* file, then the
// rotation happens. That keeps every record whole — a log line is never split
// across two files — and bounds the active file at threshold plus one record.
type rotatingFile struct {
	mu   sync.Mutex
	path string
	file *os.File
	size int64
}

func openRotatingFile(path string) (*rotatingFile, error) {
	// 0o700: on a private log directory nothing else has any business reading
	// these, and on a shared one the mode is advisory anyway.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	var size int64
	if info, statErr := file.Stat(); statErr == nil {
		size = info.Size()
	}
	rf := &rotatingFile{path: path, file: file, size: size}
	// A file inherited over the threshold rotates immediately rather than
	// waiting for one more full threshold's worth of writes.
	rf.mu.Lock()
	if rf.size >= maxLogFileBytes {
		rf.rotateLocked()
	}
	rf.mu.Unlock()
	return rf, nil
}

// Write appends to the active file and rotates when the threshold is crossed.
//
// The mutex is what makes concurrent writers safe: without it two goroutines
// could rotate at once and one would go on writing to a descriptor whose file
// has been renamed out from under it.
func (r *rotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		return len(p), nil
	}
	n, err := r.file.Write(p)
	r.size += int64(n)
	if r.size >= maxLogFileBytes {
		r.rotateLocked()
	}
	// The write itself is reported honestly; a rotation problem is not the
	// caller's to handle and must never surface as a failed log statement.
	return n, err
}

func (r *rotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

// rotateLocked shifts the generations and reopens an empty active file.
//
// Every failure path here is deliberately silent and non-fatal: this runs
// underneath the logger, so it cannot report a problem by logging one, and a
// gateway must not die because a log file could not be renamed. Whatever
// happens, the counter is reset — otherwise a persistent failure would attempt
// a rotation on every subsequent write.
func (r *rotatingFile) rotateLocked() {
	defer func() { r.size = 0 }()

	if r.file != nil {
		r.file.Close()
		r.file = nil
	}

	// Drop the oldest, then shift each generation down one.
	os.Remove(fmt.Sprintf("%s.%d", r.path, maxLogRotations))
	for i := maxLogRotations - 1; i >= 1; i-- {
		os.Rename(fmt.Sprintf("%s.%d", r.path, i), fmt.Sprintf("%s.%d", r.path, i+1))
	}
	if err := os.Rename(r.path, r.path+".1"); err != nil {
		// Truncating instead still keeps the bound, which is the property that
		// matters when the rename cannot be done.
		os.Truncate(r.path, 0)
	}

	file, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		// Logging stops rather than the process. The next EnableFileLogging
		// re-establishes it.
		return
	}
	r.file = file
	if info, statErr := file.Stat(); statErr == nil {
		r.size = info.Size()
	}
}

func EnableFileLogging(filePath string) error {
	mu.Lock()
	defer mu.Unlock()

	newFile, err := openRotatingFile(filePath)
	if err != nil {
		return err
	}

	// Close old file if exists
	if logFile != nil {
		logFile.Close()
	}

	logFile = newFile

	if len(writers) != 1 {
		return fmt.Errorf("failed to configure file logging: %w", err)
	}

	writers = append(writers, logFile)
	logger = logger.Output(io.MultiWriter(writers...))

	return nil
}

func DisableFileLogging() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	if len(writers) > 1 {
		writers = writers[:1]
		logger = logger.Output(io.MultiWriter(writers...))
	}
}

func ConfigureFromEnv() {
	if logFile := os.Getenv("PICOCLAW_LOG_FILE"); logFile != "" {
		if strings.HasPrefix(logFile, "~/") {
			if home := os.Getenv("HOME"); home != "" {
				logFile = filepath.Join(home, logFile[2:])
			}
		}

		if err := EnableFileLogging(logFile); err != nil {
			fmt.Fprintf(os.Stderr, "failed to enable file logging: %v\n", err)
		} else {
			DisableConsole()
		}
	}
}

const (
	locUnknown = "<unknown>"
)

// ShortCallerLocation renders a compiler-recorded caller as "file.go:line".
//
// It deliberately touches only logger source metadata. Paths that appear
// inside a log *message* are the message's own content and are left alone.
func ShortCallerLocation(file string, line int) string {
	// Normalise both separators explicitly rather than via filepath.ToSlash,
	// which only rewrites the *host* separator: a Windows-recorded caller
	// would pass straight through a Linux build of this test and, worse,
	// through a Linux-built binary reading a Windows-style path.
	normalized := strings.ReplaceAll(strings.TrimSpace(file), `\`, "/")
	name := path.Base(normalized)
	if name == "" || name == "." || name == "/" {
		return fmt.Sprintf("%s:%d", locUnknown, line)
	}
	return fmt.Sprintf("%s:%d", name, line)
}

func getPackageNameFromFile(filePath string) string {
	dir := filepath.Dir(filePath)
	importPath := filepath.ToSlash(dir)

	parts := strings.Split(importPath, "/")
	if len(parts) == 0 {
		return locUnknown
	}

	pkg := parts[len(parts)-1]
	if pkg == "." {
		return "<main>"
	}

	return pkg
}

func getCallerSkip() (int, string) {
	for i := 2; i < 15; i++ {
		pc, file, _, ok := runtime.Caller(i)
		if !ok {
			continue
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		// bypass common loggers
		if strings.HasSuffix(file, "/logger.go") ||
			strings.HasSuffix(file, "/logger_3rd_party.go") ||
			strings.HasSuffix(file, "/log.go") {
			continue
		}

		funcName := fn.Name()
		if strings.HasPrefix(funcName, "runtime.") {
			continue
		}

		return i - 1, getPackageNameFromFile(file)
	}

	return 3, locUnknown
}

//nolint:zerologlint
func getEvent(logger zerolog.Logger, level LogLevel) *zerolog.Event {
	switch level {
	case zerolog.DebugLevel:
		return logger.Debug()
	case zerolog.InfoLevel:
		return logger.Info()
	case zerolog.WarnLevel:
		return logger.Warn()
	case zerolog.ErrorLevel:
		return logger.Error()
	case zerolog.FatalLevel:
		return logger.Fatal()
	default:
		return logger.Info()
	}
}

func logMessage(level LogLevel, component string, message string, fields map[string]any) {
	if level < currentLevel {
		return
	}

	skip, pkg := getCallerSkip()

	event := getEvent(logger, level)

	if component == "" {
		component = pkg
	}

	event.Str(Component, component)

	appendFields(event, sanitizeFieldsForLog(fields))

	event.CallerSkipFrame(skip).Msg(message)
}

var (
	// These exact structured fields carry routing or personal identifiers. The
	// runtime maps passed by callers are never mutated; only the copy handed to
	// log writers is normalized.
	internalLogIDFields = map[string]struct{}{
		"chat_id":           {},
		"inbound_chat_id":   {},
		"target_chat_id":    {},
		"sender_id":         {},
		"inbound_sender_id": {},
		"user_id":           {},
		"session_id":        {},
		"connection_id":     {},
		"conn_id":           {},
		"runtime_id":        {},
	}
	sensitiveLogKeyFields = map[string]struct{}{
		"session_key":        {},
		"scope_key":          {},
		"route_main_session": {},
	}
	rawContentLogFields = map[string]struct{}{
		"arguments":     {},
		"args":          {},
		"content":       {},
		"messages_json": {},
		"payload":       {},
		"preview":       {},
		"prompt":        {},
		"reasoning":     {},
		"response":      {},
		"text":          {},
		"tools_json":    {},
	}
	channelDisplayFields = map[string]struct{}{
		"channel":         {},
		"inbound_channel": {},
		"route_channel":   {},
		"scope_channel":   {},
		"target_channel":  {},
	}
)

// The realtime channel's identity, spelled here rather than imported.
//
// pkg/config imports this package, so the constants cannot come from there.
// These are the only channel values this file matches on, and they are matched
// exactly — unrelated strings that merely contain them are left alone.
const (
	realtimeChannelName = "pocketclaw"

	// legacyRealtimeChannelName is LEGACY READ-ONLY MIGRATION: the value a
	// build before the channel migration wrote into its log records.
	legacyRealtimeChannelName = "pico"

	// internalRealtimeRoutePrefix is the channel's HTTP route.
	//
	// Duplicated from config.RealtimeRoutePrefix, which cannot be imported here
	// because pkg/config imports this package. A test in pkg/config pins the two
	// together, because a route that moves without this constant stops being
	// redacted and nothing else fails.
	internalRealtimeRoutePrefix = "/pocketclaw/"

	// legacyInternalRealtimeRoutePrefix is the pre-migration route. A log record
	// written by an older build still carries it, and it must still redact.
	legacyInternalRealtimeRoutePrefix = "/pico/"
)

// sanitizeFieldsForLog enforces the normal log privacy contract before any
// console or file writer sees structured values. It deliberately matches
// exact field names and exact compatibility values; runtime routing data is
// not changed and unrelated strings containing "pico" are left untouched.
func sanitizeFieldsForLog(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return fields
	}

	// Capture this relationship before channel display names are normalized.
	// Otherwise the downstream user-visible sanitizer can no longer tell that
	// the internal route belongs to the realtime channel.
	//
	// Both channel spellings are accepted. The route is still /pico/ — it moves
	// with the rest of the HTTP surface in a later phase — while the channel is
	// already canonical, and a log line from a build on either side of that
	// split must still have its path redacted.
	internalRoute := (fields["path"] == internalRealtimeRoutePrefix ||
		fields["path"] == legacyInternalRealtimeRoutePrefix) &&
		(fields["channel"] == realtimeChannelName ||
			fields["channel"] == legacyRealtimeChannelName)

	safe := make(map[string]any, len(fields))
	for key, value := range fields {
		if _, omit := rawContentLogFields[key]; omit {
			continue
		}
		if _, redact := sensitiveLogKeyFields[key]; redact {
			safe[key] = "<redacted>"
			continue
		}
		if _, internal := internalLogIDFields[key]; internal {
			safe[key] = "<internal>"
			continue
		}

		switch typed := value.(type) {
		case string:
			if internalRoute && key == "path" {
				typed = "<internal>"
			}
			// A legacy channel value can still reach here from a record
			// written before the channel migration; the display name has been
			// pocketclaw all along.
			if _, channelField := channelDisplayFields[key]; channelField &&
				typed == legacyRealtimeChannelName {
				typed = realtimeChannelName
			}
			safe[key] = sanitizeLogValue(key, typed, 0)
		case error:
			safe[key] = redactSecrets(typed.Error())
		default:
			// Everything that is not a plain string: maps, slices, structs, and
			// numbers. This branch used to hand the value to the encoder
			// unexamined, so a secret nested one level down was written verbatim.
			safe[key] = sanitizeLogValue(key, value, 0)
		}
	}
	return safe
}

func appendFields(event *zerolog.Event, fields map[string]any) {
	for k, v := range fields {
		// Type switch to avoid double JSON serialization of strings
		switch val := v.(type) {
		case error:
			event.Str(k, val.Error())
		case string:
			event.Str(k, val)
		case int:
			event.Int(k, val)
		case int64:
			event.Int64(k, val)
		case float64:
			event.Float64(k, val)
		case bool:
			event.Bool(k, val)
		default:
			event.Interface(k, v) // Fallback for struct, slice and maps
		}
	}
}

func Debug(message string) {
	logMessage(DEBUG, "", message, nil)
}

func DebugC(component string, message string) {
	logMessage(DEBUG, component, message, nil)
}

func Debugf(message string, ss ...any) {
	logMessage(DEBUG, "", fmt.Sprintf(message, ss...), nil)
}

func DebugF(message string, fields map[string]any) {
	logMessage(DEBUG, "", message, fields)
}

func DebugCF(component string, message string, fields map[string]any) {
	logMessage(DEBUG, component, message, fields)
}

func Info(message string) {
	logMessage(INFO, "", message, nil)
}

func InfoC(component string, message string) {
	logMessage(INFO, component, message, nil)
}

func InfoF(message string, fields map[string]any) {
	logMessage(INFO, "", message, fields)
}

func Infof(message string, ss ...any) {
	logMessage(INFO, "", fmt.Sprintf(message, ss...), nil)
}

func InfoCF(component string, message string, fields map[string]any) {
	logMessage(INFO, component, message, fields)
}

func Warn(message string) {
	logMessage(WARN, "", message, nil)
}

func WarnC(component string, message string) {
	logMessage(WARN, component, message, nil)
}

func WarnF(message string, fields map[string]any) {
	logMessage(WARN, "", message, fields)
}

func WarnCF(component string, message string, fields map[string]any) {
	logMessage(WARN, component, message, fields)
}

func Warnf(message string, ss ...any) {
	logMessage(WARN, "", fmt.Sprintf(message, ss...), nil)
}

func Error(message string) {
	logMessage(ERROR, "", message, nil)
}

func ErrorC(component string, message string) {
	logMessage(ERROR, component, message, nil)
}

func Errorf(message string, ss ...any) {
	logMessage(ERROR, "", fmt.Sprintf(message, ss...), nil)
}

func ErrorF(message string, fields map[string]any) {
	logMessage(ERROR, "", message, fields)
}

func ErrorCF(component string, message string, fields map[string]any) {
	logMessage(ERROR, component, message, fields)
}

func Fatal(message string) {
	logMessage(FATAL, "", message, nil)
}

func FatalC(component string, message string) {
	logMessage(FATAL, component, message, nil)
}

func Fatalf(message string, ss ...any) {
	logMessage(FATAL, "", fmt.Sprintf(message, ss...), nil)
}

func FatalF(message string, fields map[string]any) {
	logMessage(FATAL, "", message, fields)
}

func FatalCF(component string, message string, fields map[string]any) {
	logMessage(FATAL, component, message, fields)
}
