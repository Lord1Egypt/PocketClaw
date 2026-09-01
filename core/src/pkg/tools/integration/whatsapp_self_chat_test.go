package integrationtools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/logger"
)

func newTestSelfChatTool(number string) (*WhatsAppSelfChatTool, *struct {
	calls   int
	number  string
	message string
}) {
	seen := &struct {
		calls   int
		number  string
		message string
	}{}
	tool := NewWhatsAppSelfChatTool(func() string { return number })
	tool.open = func(_ context.Context, n, m string) error {
		seen.calls++
		seen.number = n
		seen.message = m
		return nil
	}
	return tool, seen
}

func TestWhatsAppSelfChatReportsOpenedNotSent(t *testing.T) {
	tool, seen := newTestSelfChatTool("+201012345678")

	result := tool.Execute(context.Background(), map[string]any{"message": "PocketClaw Agent Test"})

	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.ForLLM)
	}
	if !strings.HasPrefix(result.ForLLM, "OPENED") {
		t.Errorf("ForLLM = %q, want it to start with OPENED", result.ForLLM)
	}
	if strings.Contains(strings.ToLower(result.ForLLM), " sent.") {
		t.Errorf("ForLLM claims the message was sent: %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "NOT been sent") {
		t.Errorf("ForLLM = %q, want it to say the message was not sent", result.ForLLM)
	}
	if seen.calls != 1 || seen.number != "+201012345678" || seen.message != "PocketClaw Agent Test" {
		t.Errorf("host call = %+v", *seen)
	}
}

// The tool description is the only thing the model reads before calling, so it
// has to carry the "opened, not sent" contract itself.
func TestWhatsAppSelfChatDescriptionStatesTheUserSends(t *testing.T) {
	tool, _ := newTestSelfChatTool("+201012345678")
	description := tool.Description()
	for _, want := range []string{"prepared", "user completes sending inside WhatsApp", "never sends"} {
		if !strings.Contains(strings.ToLower(description), strings.ToLower(want)) {
			t.Errorf("description does not mention %q: %s", want, description)
		}
	}
}

func TestWhatsAppSelfChatRequiresConfiguredNumber(t *testing.T) {
	tool, seen := newTestSelfChatTool("")

	result := tool.Execute(context.Background(), map[string]any{"message": "hello"})

	if !result.IsError {
		t.Fatal("expected an error when no self number is configured")
	}
	if !strings.Contains(result.ForLLM, "not configured") {
		t.Errorf("ForLLM = %q, want it to say Self-Chat is not configured", result.ForLLM)
	}
	if seen.calls != 0 {
		t.Error("the host was asked to open WhatsApp without a configured number")
	}
}

func TestWhatsAppSelfChatRequiresNonEmptyMessage(t *testing.T) {
	tool, seen := newTestSelfChatTool("+201012345678")

	for _, args := range []map[string]any{
		{},
		{"message": ""},
		{"message": "   \n\t "},
		{"message": 42},
	} {
		result := tool.Execute(context.Background(), args)
		if !result.IsError {
			t.Errorf("args %v produced a success result", args)
		}
	}
	if seen.calls != 0 {
		t.Error("the host was asked to open WhatsApp with an empty message")
	}
}

func TestWhatsAppSelfChatEnforcesMaxLength(t *testing.T) {
	tool, seen := newTestSelfChatTool("+201012345678")

	atLimit := strings.Repeat("م", MaxWhatsAppSelfChatMessageChars)
	if result := tool.Execute(context.Background(), map[string]any{"message": atLimit}); result.IsError {
		t.Fatalf("a message at the limit was rejected: %s", result.ForLLM)
	}

	overLimit := strings.Repeat("م", MaxWhatsAppSelfChatMessageChars+1)
	result := tool.Execute(context.Background(), map[string]any{"message": overLimit})
	if !result.IsError {
		t.Fatal("a message over the limit was accepted")
	}
	if seen.calls != 1 {
		t.Errorf("host calls = %d, want only the at-limit message to reach the host", seen.calls)
	}
}

func TestWhatsAppSelfChatSchemaTakesOnlyAMessage(t *testing.T) {
	tool, _ := newTestSelfChatTool("+201012345678")

	params := tool.Parameters()
	properties, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties = %#v", params["properties"])
	}
	if len(properties) != 1 {
		t.Errorf("schema exposes %d properties, want only message", len(properties))
	}
	if _, ok := properties["message"]; !ok {
		t.Error("schema has no message property")
	}
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "message" {
		t.Errorf("required = %#v, want [message]", params["required"])
	}
	if tool.Name() != "whatsapp_self_chat" {
		t.Errorf("Name() = %q", tool.Name())
	}
}

// The message body and the user's number are private. Only the outcome and a
// character count may reach the logs, which PocketClaw exports on request.
func TestWhatsAppSelfChatLogsNeitherNumberNorMessage(t *testing.T) {
	const number = "+201012345678"
	const secret = "meet me at the usual place at nine"

	logPath := filepath.Join(t.TempDir(), "tool.log")
	if err := logger.EnableFileLogging(logPath); err != nil {
		t.Fatalf("EnableFileLogging: %v", err)
	}
	defer logger.DisableFileLogging()
	logger.SetLevel(logger.DEBUG)

	tool, _ := newTestSelfChatTool(number)
	if result := tool.Execute(context.Background(), map[string]any{"message": secret}); result.IsError {
		t.Fatalf("unexpected error result: %s", result.ForLLM)
	}

	failing := NewWhatsAppSelfChatTool(func() string { return number })
	failing.open = func(context.Context, string, string) error { return errors.New("not_installed") }
	failing.Execute(context.Background(), map[string]any{"message": secret})

	written, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	logged := string(written)
	if strings.Contains(logged, secret) {
		t.Error("the message body reached the log")
	}
	if strings.Contains(logged, number) || strings.Contains(logged, "201012345678") {
		t.Error("the WhatsApp number reached the log")
	}
	if !strings.Contains(logged, "message_chars") {
		t.Errorf("expected a message_chars field in the log, got: %s", logged)
	}
}
