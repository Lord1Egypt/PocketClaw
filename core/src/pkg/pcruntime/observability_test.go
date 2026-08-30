package pcruntime

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// captureRuntimeLog routes the logger to a file sink for the duration of the
// test and returns the runtime events it recorded. It exercises the real
// logging path, including the logger's own sanitisation, rather than a stub.
func captureRuntimeLog(t *testing.T, body func()) []map[string]any {
	t.Helper()

	sink := filepath.Join(t.TempDir(), "runtime.log")
	previousLevel := logger.GetLevel()
	logger.SetLevel(logger.DEBUG)
	logger.DisableConsole()
	if err := logger.EnableFileLogging(sink); err != nil {
		t.Fatalf("cannot capture logs: %v", err)
	}
	t.Cleanup(func() {
		logger.DisableFileLogging()
		logger.EnableConsole()
		logger.SetLevel(previousLevel)
	})

	body()
	logger.DisableFileLogging()

	var events []map[string]any
	for _, record := range decodeJSONLines(t, sink) {
		if record["component"] == logComponent {
			events = append(events, record)
		}
	}
	return events
}

func eventNames(events []map[string]any) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		if name, ok := event["event"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

func requireEvents(t *testing.T, events []map[string]any, wanted ...string) {
	t.Helper()
	seen := make(map[string]struct{}, len(events))
	for _, name := range eventNames(events) {
		seen[name] = struct{}{}
	}
	for _, want := range wanted {
		if _, ok := seen[want]; !ok {
			t.Fatalf("missing lifecycle event %q; recorded: %v", want, eventNames(events))
		}
	}
}

func TestSuccessfulExecutionEmitsACompleteLifecycle(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("ok", TimeoutQuick, 4096)))
	writeScript(t, binDir, "ok", "echo done\n")

	events := captureRuntimeLog(t, func() {
		if _, err := manager.Execute(context.Background(), ExecRequest{Tool: "ok"}); err != nil {
			t.Fatalf("execute was rejected: %v", err)
		}
	})

	requireEvents(t, events,
		EventExecQueued, EventResolveStarted, EventResolveCompleted,
		EventExecStarted, EventExecStdout, EventExecStderr, EventExecCompleted,
	)
}

func TestFailedResolutionEmitsACompleteLifecycle(t *testing.T) {
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("dig", TimeoutQuick, 4096)))

	events := captureRuntimeLog(t, func() {
		if _, err := manager.Execute(context.Background(), ExecRequest{Tool: "dig"}); err != nil {
			t.Fatalf("execute returned an error instead of a result: %v", err)
		}
	})

	requireEvents(t, events, EventExecQueued, EventResolveStarted, EventResolveFailed, EventExecFailed)
}

func TestTimeoutEmitsACompleteLifecycle(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("slow", TimeoutQuick, 4096)))
	writeScript(t, binDir, "slow", "sleep 30\n")

	events := captureRuntimeLog(t, func() {
		if _, err := manager.Execute(context.Background(), ExecRequest{Tool: "slow", TimeoutMS: 300}); err != nil {
			t.Fatalf("execute was rejected: %v", err)
		}
	})

	requireEvents(t, events,
		EventExecQueued, EventExecStarted, EventCleanupStarted, EventExecTimeout,
	)
}

func TestCancellationEmitsACompleteLifecycle(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("slow", TimeoutExtended, 4096)))
	writeScript(t, binDir, "slow", "sleep 30\n")

	events := captureRuntimeLog(t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()
		if _, err := manager.Execute(ctx, ExecRequest{Tool: "slow"}); err != nil {
			t.Fatalf("execute was rejected: %v", err)
		}
	})

	requireEvents(t, events,
		EventExecQueued, EventExecStarted, EventCleanupStarted, EventExecCancelled,
	)
}

// Without an operation id the lifecycle of concurrent executions cannot be
// separated in the log, which is the whole point of emitting one.
func TestEveryExecutionEventCarriesAnOperationID(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("ok", TimeoutQuick, 4096)))
	writeScript(t, binDir, "ok", "echo done\n")

	events := captureRuntimeLog(t, func() {
		if _, err := manager.Execute(context.Background(), ExecRequest{Tool: "ok"}); err != nil {
			t.Fatalf("execute was rejected: %v", err)
		}
	})

	for _, event := range events {
		name, _ := event["event"].(string)
		if !strings.HasPrefix(name, "runtime.exec.") {
			continue
		}
		id, ok := event["operation_id"].(string)
		if !ok || id == "" {
			t.Fatalf("%s carries no operation_id", name)
		}
	}
}

// Nothing a tool prints reaches the log. stdout is precisely where a fetched
// credential shows up — `gh auth token` prints one — so only its size is logged.
func TestToolOutputIsNeverPersistedToLogs(t *testing.T) {
	const secret = "ghp_1234567890abcdefghijklmnopqrstuvwx"
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("leak", TimeoutQuick, 4096)))
	writeScript(t, binDir, "leak", "echo "+secret+"\necho "+secret+" >&2\n")

	var result *ExecResult
	events := captureRuntimeLog(t, func() {
		var err error
		result, err = manager.Execute(context.Background(), ExecRequest{Tool: "leak"})
		if err != nil {
			t.Fatalf("execute was rejected: %v", err)
		}
	})

	// The caller still receives the real output; only persistence is redacted.
	if !strings.Contains(result.Stdout, secret) {
		t.Fatal("the caller must receive the tool's real output")
	}
	for _, event := range events {
		for key, value := range event {
			if text, ok := value.(string); ok && strings.Contains(text, secret) {
				t.Fatalf("a token reached the log in field %q: %q", key, text)
			}
		}
	}
}

// A credential passed as an argument must not survive into the log, but the
// rest of the command line must, or the log cannot be used for diagnosis.
func TestArgumentCredentialsAreRedactedInLogs(t *testing.T) {
	const secret = "sk-live-abcdefghijklmnopqrst"
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("curl", TimeoutQuick, 4096)))
	writeScript(t, binDir, "curl", "exit 0\n")

	events := captureRuntimeLog(t, func() {
		_, err := manager.Execute(context.Background(), ExecRequest{
			Tool: "curl",
			Args: []string{"-H", "Authorization: Bearer " + secret, "https://example.com/api"},
		})
		if err != nil {
			t.Fatalf("execute was rejected: %v", err)
		}
	})

	sawCommandLine := false
	for _, event := range events {
		raw, ok := event["argv_redacted"]
		if !ok {
			continue
		}
		text := strings.ToLower(strings.TrimSpace(strings.Join(asStrings(raw), " ")))
		if strings.Contains(text, strings.ToLower(secret)) {
			t.Fatalf("the credential reached the log: %q", text)
		}
		if strings.Contains(text, "https://example.com/api") {
			sawCommandLine = true
		}
	}
	if !sawCommandLine {
		t.Fatal("redaction removed the whole command line; the log has no diagnostic value left")
	}
}

func asStrings(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	case []string:
		return typed
	case string:
		return []string{typed}
	default:
		return nil
	}
}
