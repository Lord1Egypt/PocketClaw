package tools

import (
	"strings"
	"testing"
)

func TestToolCallLogMessageOmitsArguments(t *testing.T) {
	t.Parallel()

	got := toolCallLogMessage("sendMessage")
	if got != "Tool call: sendMessage" {
		t.Fatalf("toolCallLogMessage() = %q", got)
	}
	for _, forbidden := range []string{"private user content", `{"text":`, "SUPERSECRETVALUE"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("tool log exposed argument content %q in %q", forbidden, got)
		}
	}
}
