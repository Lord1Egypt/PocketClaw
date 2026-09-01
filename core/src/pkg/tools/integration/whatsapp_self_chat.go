package integrationtools

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/whatsapp/selfchat"
)

// WhatsAppSelfChatToolName is the tool the model calls.
const WhatsAppSelfChatToolName = "whatsapp_self_chat"

// MaxWhatsAppSelfChatMessageChars bounds the message body. WhatsApp itself
// accepts far more, but the text travels as an Intent extra, and an
// unbounded body would be a TransactionTooLargeException rather than a
// message. Long enough for anything a person reads on a phone.
const MaxWhatsAppSelfChatMessageChars = 4096

// selfChatOpener is the host call, replaced in tests.
type selfChatOpener func(ctx context.Context, number, message string) error

// WhatsAppSelfChatTool opens WhatsApp on the user's own chat with a message
// prepared in the compose box.
//
// It never sends. The Intent hands the text to WhatsApp and stops there; the
// Send button belongs to the user. Reporting otherwise to the model would make
// it claim a message was delivered when it is still sitting in a compose box.
type WhatsAppSelfChatTool struct {
	// selfNumber reads the configured number fresh on every call, so a number
	// changed or disconnected in the console takes effect without a restart.
	selfNumber func() string
	open       selfChatOpener
}

func NewWhatsAppSelfChatTool(selfNumber func() string) *WhatsAppSelfChatTool {
	return &WhatsAppSelfChatTool{selfNumber: selfNumber, open: selfchat.Open}
}

func (t *WhatsAppSelfChatTool) Name() string { return WhatsAppSelfChatToolName }

func (t *WhatsAppSelfChatTool) Description() string {
	return "Opens WhatsApp with the message prepared for the user's configured " +
		"self-chat. The user completes sending inside WhatsApp. This tool never " +
		"sends a message: a successful result means WhatsApp was opened with the " +
		"text waiting in the compose box, not that anything was delivered."
}

func (t *WhatsAppSelfChatTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type": "string",
				"description": "The text to place in the WhatsApp compose box. " +
					"The user sends it themselves.",
			},
		},
		"required": []string{"message"},
	}
}

func (t *WhatsAppSelfChatTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	message, _ := args["message"].(string)
	message = strings.TrimSpace(message)
	if message == "" {
		return ErrorResult("message is required")
	}
	chars := utf8.RuneCountInString(message)
	if chars > MaxWhatsAppSelfChatMessageChars {
		return ErrorResult(
			"message is too long for WhatsApp Self-Chat; keep it under " +
				strconv.Itoa(MaxWhatsAppSelfChatMessageChars) + " characters",
		)
	}

	number := strings.TrimSpace(t.selfNumber())
	if number == "" {
		return ErrorResult(
			"WhatsApp Self-Chat is not configured. Ask the user to set their own " +
				"WhatsApp number in PocketClaw under Channels → WhatsApp Self-Chat.",
		)
	}

	if err := t.open(ctx, number, message); err != nil {
		// The number and the body are both private; only the outcome is logged.
		logger.WarnCF("tool", "whatsapp_self_chat could not open WhatsApp", map[string]any{
			"tool":          WhatsAppSelfChatToolName,
			"message_chars": chars,
			"status":        "failed",
		})
		return ErrorResult(err.Error()).WithError(err)
	}

	logger.InfoCF("tool", "whatsapp_self_chat opened WhatsApp", map[string]any{
		"tool":          WhatsAppSelfChatToolName,
		"message_chars": chars,
		"status":        "opened",
	})

	return &ToolResult{
		ForLLM: "OPENED. WhatsApp is showing the user's self-chat with the message " +
			"prepared in the compose box. It has NOT been sent — the user sends it.",
	}
}
