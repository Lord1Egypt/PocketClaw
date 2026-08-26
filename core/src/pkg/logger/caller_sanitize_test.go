package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Substrings that must never reach a normal user's Logs screen or an exported
// log file. The module path is the one that regressed: -trimpath removed the
// developer's absolute path and left the compiler recording the Go module path
// instead, which zerolog then reported as the caller.
var forbiddenInUserFacingLogs = []string{
	"github.com/sipeed",
	"picoclaw",
	"PicoClaw",
	"sipeed",
	"Sipeed",
	"/home/",
	".upstream",
}

func TestShortCallerLocationStripsModulePath(t *testing.T) {
	cases := []struct {
		name string
		file string
		line int
		want string
	}{
		{
			name: "the caller reported on device",
			file: "github.com/sipeed/picoclaw/web/backend/api/gateway.go",
			line: 298,
			want: "gateway.go:298",
		},
		{
			name: "absolute developer path, when built without -trimpath",
			file: "/home/lordegypt/PocketClaw-App/core/src/web/backend/api/gateway.go",
			line: 298,
			want: "gateway.go:298",
		},
		{
			name: "historical upstream reference checkout",
			file: "/home/lordegypt/PocketCLaw/.upstream/picoclaw-core-v0.3.1/pkg/gateway/run.go",
			line: 12,
			want: "run.go:12",
		},
		{
			name: "already short",
			file: "gateway.go",
			line: 1,
			want: "gateway.go:1",
		},
		{
			name: "windows separators",
			file: `C:\build\picoclaw\pkg\logger\logger.go`,
			line: 7,
			want: "logger.go:7",
		},
		{
			name: "degenerate input does not leak or panic",
			file: "",
			line: 0,
			want: "<unknown>:0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ShortCallerLocation(tc.file, tc.line)
			if got != tc.want {
				t.Fatalf("ShortCallerLocation(%q, %d) = %q, want %q", tc.file, tc.line, got, tc.want)
			}
			for _, banned := range forbiddenInUserFacingLogs {
				if strings.Contains(got, banned) {
					t.Fatalf("caller %q leaks %q", got, banned)
				}
			}
		})
	}
}

// The end-to-end assertion: a real log call through the real logger, read back
// from a real log file, must carry a caller with no module path in it. This is
// the property the device regression violated; a unit test of the helper alone
// would have passed even while the logger stayed misconfigured.
func TestLoggerEmitsSanitizedCaller(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "pocketclaw.log")

	if err := EnableFileLogging(logPath); err != nil {
		t.Fatalf("EnableFileLogging: %v", err)
	}
	t.Cleanup(DisableFileLogging)

	Warnf("removed stale pid file for PID %d", 12302)

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("logger wrote nothing to the log file")
	}

	var sawCaller bool
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not JSON: %q (%v)", line, err)
		}

		caller, _ := entry["caller"].(string)
		if caller == "" {
			t.Fatalf("log entry has no caller: %q", line)
		}
		sawCaller = true

		for _, banned := range forbiddenInUserFacingLogs {
			if strings.Contains(caller, banned) {
				t.Fatalf("caller %q leaks %q", caller, banned)
			}
		}
		if strings.Contains(caller, "/") {
			t.Fatalf("caller %q still carries a path separator; want bare file:line", caller)
		}
		if !strings.HasSuffix(strings.Split(caller, ":")[0], ".go") {
			t.Fatalf("caller %q is not a Go file name", caller)
		}
	}

	if !sawCaller {
		t.Fatal("no log entry carried a caller field")
	}
}
