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
