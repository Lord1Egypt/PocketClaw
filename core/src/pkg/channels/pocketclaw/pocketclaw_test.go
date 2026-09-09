package pocketclaw

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

func newTestPicoChannel(t *testing.T) *PocketClawChannel {
	t.Helper()

	bc := &config.Channel{Type: config.ChannelPocketClaw, Enabled: true}
	cfg := &config.PocketClawSettings{}
	cfg.SetToken("test-token")
	ch, err := NewPocketClawChannel(bc, cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("NewPocketClawChannel: %v", err)
	}

	ch.ctx = context.Background()
	return ch
}

func TestHandleMessageSend_ForwardsMessageMetadata(t *testing.T) {
	msgBus := bus.NewMessageBus()
	bc := &config.Channel{Type: config.ChannelPocketClaw, Enabled: true}
	cfg := &config.PocketClawSettings{}
	cfg.SetToken("test-token")
	ch, err := NewPocketClawChannel(bc, cfg, msgBus)
	if err != nil {
		t.Fatalf("NewPocketClawChannel: %v", err)
	}
	ch.ctx = context.Background()

	ch.handleMessageSend(&pocketClawConn{id: "conn-1", sessionID: "sess-1"}, PocketClawMessage{
		Type:      TypeMessageSend,
		ID:        "msg-1",
		SessionID: "sess-1",
		Payload: map[string]any{
			PayloadKeyContent: "hello",
		},
	})

	select {
	case inbound := <-msgBus.InboundChan():
		if inbound.Content != "hello" {
			t.Fatalf("content = %q, want hello", inbound.Content)
		}
		if got := inbound.Context.Raw["session_id"]; got != "sess-1" {
			t.Fatalf("session_id raw = %q, want sess-1", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected inbound pico message")
	}
}

func TestHandleMessageSend_IgnoresClientForgedIdentityAndSession(t *testing.T) {
	msgBus := bus.NewMessageBus()
	bc := &config.Channel{
		Type:      config.ChannelPocketClaw,
		Enabled:   true,
		AllowFrom: config.FlexibleStringSlice{"*"},
	}
	cfg := &config.PocketClawSettings{}
	cfg.SetToken("test-token")
	ch, err := NewPocketClawChannel(bc, cfg, msgBus)
	if err != nil {
		t.Fatal(err)
	}
	ch.ctx = context.Background()

	ch.handleMessageSend(&pocketClawConn{id: "conn-1", sessionID: "server-session"}, PocketClawMessage{
		Type:      TypeMessageSend,
		ID:        "msg-1",
		SessionID: "client-forged-session",
		Payload: map[string]any{
			PayloadKeyContent: "hello",
			"sender_id":       OwnerPrincipal,
			"chat_id":         "forged-chat",
			"owner_id":        OwnerPrincipal,
			"principal":       OwnerPrincipal,
			"user_id":         OwnerPrincipal,
			"allow_from":      []string{"*"},
		},
	})

	select {
	case inbound := <-msgBus.InboundChan():
		if inbound.SenderID != OwnerPrincipal || inbound.Context.SenderID != OwnerPrincipal {
			t.Fatalf("effective sender = %q/%q, want server owner", inbound.SenderID, inbound.Context.SenderID)
		}
		if inbound.ChatID != "pocketclaw:server-session" || inbound.Context.ChatID != "pocketclaw:server-session" {
			t.Fatalf("effective chat = %q/%q, want connection-bound session", inbound.ChatID, inbound.Context.ChatID)
		}
		if got := inbound.Context.Raw["session_id"]; got != "server-session" {
			t.Fatalf("session metadata = %q, want server-session", got)
		}
	case <-time.After(time.Second):
		t.Fatal("authorized server-derived owner did not reach the message bus")
	}
}

func TestNewPicoChannelEnforcesOnlyServerOwnerPrincipal(t *testing.T) {
	bc := &config.Channel{
		Type:      config.ChannelPocketClaw,
		Enabled:   true,
		AllowFrom: config.FlexibleStringSlice{"*"},
	}
	cfg := &config.PocketClawSettings{}
	cfg.SetToken("test-token")
	ch, err := NewPocketClawChannel(bc, cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	if !ch.IsAllowedSender(bus.SenderInfo{Platform: "pocketclaw", PlatformID: OwnerPrincipal}) {
		t.Fatal("server-derived owner principal was rejected")
	}
	if ch.IsAllowedSender(bus.SenderInfo{Platform: "pocketclaw", PlatformID: "attacker"}) {
		t.Fatal("stale wildcard config bypassed owner-only authorization")
	}
}

func TestFinalizeTrackedToolFeedbackMessage_StopsTrackingBeforeEdit(t *testing.T) {
	ch := &PocketClawChannel{
		progress: channels.NewToolFeedbackAnimator(nil),
	}
	ch.RecordToolFeedbackMessage("pocketclaw:chat-1", "msg-1", "🔧 `read_file`")

	msgIDs, handled := ch.finalizeTrackedToolFeedbackMessage(
		context.Background(),
		"pocketclaw:chat-1",
		"final reply",
		func(_ context.Context, chatID, messageID string, payload map[string]any, contextUsage *bus.ContextUsage) error {
			if _, ok := ch.currentToolFeedbackMessage(chatID); ok {
				t.Fatal("expected tracked tool feedback to be stopped before edit")
			}
			if chatID != "pocketclaw:chat-1" || messageID != "msg-1" {
				t.Fatalf("unexpected edit args: %s %s", chatID, messageID)
			}
			if got := payload[PayloadKeyContent]; got != "final reply" {
				t.Fatalf("unexpected content payload: %#v", got)
			}
			if contextUsage != nil {
				t.Fatalf("unexpected context usage: %+v", contextUsage)
			}
			return nil
		},
		nil,
		nil,
	)
	if !handled {
		t.Fatal("expected finalizeTrackedToolFeedbackMessage to handle tracked message")
	}
	if len(msgIDs) != 1 || msgIDs[0] != "msg-1" {
		t.Fatalf("finalizeTrackedToolFeedbackMessage() ids = %v, want [msg-1]", msgIDs)
	}
}

func TestDismissTrackedToolFeedbackMessage_DeletesProgressMessage(t *testing.T) {
	ch := &PocketClawChannel{
		progress: channels.NewToolFeedbackAnimator(nil),
	}
	ch.RecordToolFeedbackMessage("pocketclaw:chat-1", "msg-1", "🔧 `read_file`")

	var deleted struct {
		chatID    string
		messageID string
	}
	ch.deleteMessageFn = func(_ context.Context, chatID string, messageID string) error {
		deleted.chatID = chatID
		deleted.messageID = messageID
		return nil
	}

	ch.DismissToolFeedbackMessage(context.Background(), "pocketclaw:chat-1")

	if deleted.chatID != "pocketclaw:chat-1" || deleted.messageID != "msg-1" {
		t.Fatalf("unexpected delete target: %+v", deleted)
	}
	if _, ok := ch.currentToolFeedbackMessage("pocketclaw:chat-1"); ok {
		t.Fatal("expected tracked tool feedback to be cleared after dismissal")
	}
}

func TestSend_ThoughtMessageDoesNotFinalizeTrackedToolFeedback(t *testing.T) {
	ch := newTestPicoChannel(t)

	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	ch.RecordToolFeedbackMessage("pocketclaw:sess-1", "msg-progress", "🔧 `read_file`\nReading config")

	if _, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "pocketclaw:sess-1",
		Content: "thinking trace",
		Context: bus.InboundContext{
			Channel: "pocketclaw",
			ChatID:  "pocketclaw:sess-1",
			Raw: map[string]string{
				"message_kind":      MessageKindThought,
				PayloadKeyModelName: "gpt-5.4-mini",
			},
		},
	}); err != nil {
		t.Fatalf("Send(thought) error = %v", err)
	}

	select {
	case msg := <-received:
		if msg.Type != TypeMessageCreate {
			t.Fatalf("thought message type = %q, want %q", msg.Type, TypeMessageCreate)
		}
		payload := msg.Payload
		if got := payload[PayloadKeyContent]; got != "thinking trace" {
			t.Fatalf("thought content = %#v, want %q", got, "thinking trace")
		}
		if got := payload[PayloadKeyKind]; got != MessageKindThought {
			t.Fatalf("thought kind = %#v, want %q", got, MessageKindThought)
		}
		if got := payload[PayloadKeyModelName]; got != "gpt-5.4-mini" {
			t.Fatalf("thought model_name = %#v, want %q", got, "gpt-5.4-mini")
		}
		if got := payload["message_id"]; got == "msg-progress" || got == nil || got == "" {
			t.Fatalf("thought message_id = %#v, want new non-progress id", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected thought message to be delivered")
	}

	if msgID, ok := ch.currentToolFeedbackMessage("pocketclaw:sess-1"); !ok || msgID != "msg-progress" {
		t.Fatalf("tracked tool feedback = (%q, %v), want (msg-progress, true)", msgID, ok)
	}

	if _, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "pocketclaw:sess-1",
		Content: "final reply",
		Context: bus.InboundContext{
			Channel: "pocketclaw",
			ChatID:  "pocketclaw:sess-1",
			Raw: map[string]string{
				PayloadKeyModelName: "gpt-5.4",
			},
		},
		ContextUsage: &bus.ContextUsage{
			UsedTokens:       321,
			TotalTokens:      4096,
			CompressAtTokens: 3072,
			UsedPercent:      8,
		},
	}); err != nil {
		t.Fatalf("Send(final) error = %v", err)
	}

	select {
	case msg := <-received:
		if msg.Type != TypeMessageUpdate {
			t.Fatalf("final message type = %q, want %q", msg.Type, TypeMessageUpdate)
		}
		payload := msg.Payload
		if got := payload["message_id"]; got != "msg-progress" {
			t.Fatalf("final message_id = %#v, want %q", got, "msg-progress")
		}
		if got := payload[PayloadKeyContent]; got != "final reply" {
			t.Fatalf("final content = %#v, want %q", got, "final reply")
		}
		if got := payload[PayloadKeyModelName]; got != "gpt-5.4" {
			t.Fatalf("final model_name = %#v, want %q", got, "gpt-5.4")
		}
		rawUsage, ok := payload["context_usage"].(map[string]any)
		if !ok {
			t.Fatalf("final context_usage = %#v, want map payload", payload["context_usage"])
		}
		if got, ok := rawUsage["used_tokens"].(float64); !ok || got != 321 {
			t.Fatalf("used_tokens = %#v, want 321", rawUsage["used_tokens"])
		}
		if got, ok := rawUsage["total_tokens"].(float64); !ok || got != 4096 {
			t.Fatalf("total_tokens = %#v, want 4096", rawUsage["total_tokens"])
		}
	case <-time.After(time.Second):
		t.Fatal("expected final reply to finalize tracked tool feedback")
	}

	if _, ok := ch.currentToolFeedbackMessage("pocketclaw:sess-1"); ok {
		t.Fatal("expected tracked tool feedback to be cleared after final reply")
	}
}

func TestSend_ToolCallsMessageIncludesModelName(t *testing.T) {
	ch := newTestPicoChannel(t)

	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	if _, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "pocketclaw:sess-1",
		Content: "",
		Context: bus.InboundContext{
			Channel: "pocketclaw",
			ChatID:  "pocketclaw:sess-1",
			Raw: map[string]string{
				"message_kind":      MessageKindToolCalls,
				PayloadKeyModelName: "gpt-5.4",
				PayloadKeyToolCalls: `[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"README.md\"}"}}]`,
			},
		},
	}); err != nil {
		t.Fatalf("Send(tool_calls) error = %v", err)
	}

	select {
	case msg := <-received:
		if msg.Type != TypeMessageCreate {
			t.Fatalf("tool_calls message type = %q, want %q", msg.Type, TypeMessageCreate)
		}
		payload := msg.Payload
		if got := payload[PayloadKeyKind]; got != MessageKindToolCalls {
			t.Fatalf("tool_calls kind = %#v, want %q", got, MessageKindToolCalls)
		}
		if got := payload[PayloadKeyModelName]; got != "gpt-5.4" {
			t.Fatalf("tool_calls model_name = %#v, want %q", got, "gpt-5.4")
		}
		if _, ok := payload[PayloadKeyToolCalls].([]any); !ok {
			t.Fatalf("tool_calls payload = %#v, want parsed array", payload[PayloadKeyToolCalls])
		}
	case <-time.After(time.Second):
		t.Fatal("expected tool_calls message to be delivered")
	}
}

func TestSendPlaceholder_EmitsNormalMessageWithoutKind(t *testing.T) {
	ch := newTestPicoChannel(t)
	ch.bc.Placeholder.Enabled = true

	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	msgID, err := ch.SendPlaceholder(context.Background(), "pocketclaw:sess-1")
	if err != nil {
		t.Fatalf("SendPlaceholder() error = %v", err)
	}
	if msgID == "" {
		t.Fatal("expected placeholder message id")
	}

	select {
	case msg := <-received:
		if msg.Type != TypeMessageCreate {
			t.Fatalf("placeholder message type = %q, want %q", msg.Type, TypeMessageCreate)
		}
		payload := msg.Payload
		if got := payload["message_id"]; got != msgID {
			t.Fatalf("placeholder message_id = %#v, want %q", got, msgID)
		}
		if got := payload[PayloadKeyContent]; got != "Thinking..." {
			t.Fatalf("placeholder content = %#v, want %q", got, "Thinking...")
		}
		if got := payload[PayloadKeyPlaceholder]; got != true {
			t.Fatalf("placeholder marker = %#v, want true", got)
		}
		if got, ok := payload[PayloadKeyKind]; ok {
			t.Fatalf("placeholder kind = %#v, want absent", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected placeholder message to be delivered")
	}
}

func TestBeginStream_CreatesAndUpdatesSameMessage(t *testing.T) {
	ch := newTestPicoChannel(t)
	ch.config.Streaming = config.StreamingConfig{
		Enabled:         true,
		ThrottleSeconds: 1,
		MinGrowthChars:  1,
	}
	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	streamer, err := ch.BeginStream(context.Background(), "pocketclaw:sess-1")
	if err != nil {
		t.Fatalf("BeginStream() error = %v", err)
	}
	if setter, ok := streamer.(interface{ SetModelName(modelName string) }); ok {
		setter.SetModelName("gpt-5.4")
	}
	if err := streamer.Update(context.Background(), "hello"); err != nil {
		t.Fatalf("Update(first) error = %v", err)
	}
	first := mustReceivePicoMessage(t, received)
	if first.Type != TypeMessageCreate {
		t.Fatalf("first type = %q, want %q", first.Type, TypeMessageCreate)
	}
	msgID, _ := first.Payload["message_id"].(string)
	if msgID == "" {
		t.Fatalf("first message_id = %#v, want non-empty", first.Payload["message_id"])
	}
	if got := first.Payload[PayloadKeyContent]; got != "hello" {
		t.Fatalf("first content = %#v, want hello", got)
	}
	if got := first.Payload[PayloadKeyModelName]; got != "gpt-5.4" {
		t.Fatalf("first model_name = %#v, want %q", got, "gpt-5.4")
	}

	rawStreamer := streamer.(*pocketClawStreamer)
	rawStreamer.mu.Lock()
	rawStreamer.lastAt = time.Now().Add(-2 * time.Second)
	rawStreamer.mu.Unlock()
	secondContent := "hello world with enough growth to pass the default streaming threshold"
	if err := streamer.Update(context.Background(), secondContent); err != nil {
		t.Fatalf("Update(second) error = %v", err)
	}
	second := mustReceivePicoMessage(t, received)
	if second.Type != TypeMessageUpdate {
		t.Fatalf("second type = %q, want %q", second.Type, TypeMessageUpdate)
	}
	if got := second.Payload["message_id"]; got != msgID {
		t.Fatalf("second message_id = %#v, want %q", got, msgID)
	}
	if got := second.Payload[PayloadKeyContent]; got != secondContent {
		t.Fatalf("second content = %#v, want %q", got, secondContent)
	}
	if got := second.Payload[PayloadKeyModelName]; got != "gpt-5.4" {
		t.Fatalf("second model_name = %#v, want %q", got, "gpt-5.4")
	}
}

func TestBeginStream_DefaultStreamingShowsSmallIncrements(t *testing.T) {
	ch := newTestPicoChannel(t)
	ch.config.Streaming = config.StreamingConfig{Enabled: true}
	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	streamer, err := ch.BeginStream(context.Background(), "pocketclaw:sess-1")
	if err != nil {
		t.Fatalf("BeginStream() error = %v", err)
	}
	if err := streamer.Update(context.Background(), "h"); err != nil {
		t.Fatalf("Update(first) error = %v", err)
	}
	first := mustReceivePicoMessage(t, received)
	if first.Type != TypeMessageCreate {
		t.Fatalf("first type = %q, want %q", first.Type, TypeMessageCreate)
	}
	msgID, _ := first.Payload["message_id"].(string)
	if msgID == "" {
		t.Fatalf("first message_id = %#v, want non-empty", first.Payload["message_id"])
	}

	if err := streamer.Update(context.Background(), "he"); err != nil {
		t.Fatalf("Update(second) error = %v", err)
	}
	second := mustReceivePicoMessage(t, received)
	if second.Type != TypeMessageUpdate {
		t.Fatalf("second type = %q, want %q", second.Type, TypeMessageUpdate)
	}
	if got := second.Payload["message_id"]; got != msgID {
		t.Fatalf("second message_id = %#v, want %q", got, msgID)
	}
	if got := second.Payload[PayloadKeyContent]; got != "he" {
		t.Fatalf("second content = %#v, want he", got)
	}
}

func TestBeginStream_StreamsReasoningAsThoughtUpdates(t *testing.T) {
	ch := newTestPicoChannel(t)
	ch.config.Streaming = config.StreamingConfig{Enabled: true}
	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	streamer, err := ch.BeginStream(context.Background(), "pocketclaw:sess-1")
	if err != nil {
		t.Fatalf("BeginStream() error = %v", err)
	}
	reasoningStreamer, ok := streamer.(bus.ReasoningStreamer)
	if !ok {
		t.Fatal("pico stream should support reasoning updates")
	}
	if setter, ok := streamer.(interface{ SetModelName(modelName string) }); ok {
		setter.SetModelName("gpt-5.4-mini")
	}
	if err := reasoningStreamer.UpdateReasoning(context.Background(), "thinking"); err != nil {
		t.Fatalf("UpdateReasoning(first) error = %v", err)
	}
	first := mustReceivePicoMessage(t, received)
	if first.Type != TypeMessageCreate {
		t.Fatalf("first type = %q, want %q", first.Type, TypeMessageCreate)
	}
	msgID, _ := first.Payload["message_id"].(string)
	if msgID == "" {
		t.Fatalf("first message_id = %#v, want non-empty", first.Payload["message_id"])
	}
	if got := first.Payload[PayloadKeyKind]; got != MessageKindThought {
		t.Fatalf("first kind = %#v, want %q", got, MessageKindThought)
	}
	if got := first.Payload[PayloadKeyContent]; got != "thinking" {
		t.Fatalf("first content = %#v, want thinking", got)
	}
	if got := first.Payload[PayloadKeyModelName]; got != "gpt-5.4-mini" {
		t.Fatalf("first model_name = %#v, want %q", got, "gpt-5.4-mini")
	}

	if err := reasoningStreamer.UpdateReasoning(context.Background(), "thinking more"); err != nil {
		t.Fatalf("UpdateReasoning(second) error = %v", err)
	}
	second := mustReceivePicoMessage(t, received)
	if second.Type != TypeMessageUpdate {
		t.Fatalf("second type = %q, want %q", second.Type, TypeMessageUpdate)
	}
	if got := second.Payload["message_id"]; got != msgID {
		t.Fatalf("second message_id = %#v, want %q", got, msgID)
	}
	if got := second.Payload[PayloadKeyKind]; got != MessageKindThought {
		t.Fatalf("second kind = %#v, want %q", got, MessageKindThought)
	}
	if got := second.Payload[PayloadKeyContent]; got != "thinking more" {
		t.Fatalf("second content = %#v, want thinking more", got)
	}
	if got := second.Payload[PayloadKeyModelName]; got != "gpt-5.4-mini" {
		t.Fatalf("second model_name = %#v, want %q", got, "gpt-5.4-mini")
	}
}

func TestBeginStream_ThrottlesIntermediateUpdatesAndFinalFlushes(t *testing.T) {
	ch := newTestPicoChannel(t)
	ch.config.Streaming = config.StreamingConfig{
		Enabled:         true,
		ThrottleSeconds: 60,
		MinGrowthChars:  100,
	}
	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	streamer, err := ch.BeginStream(context.Background(), "pocketclaw:sess-1")
	if err != nil {
		t.Fatalf("BeginStream() error = %v", err)
	}
	if err := streamer.Update(context.Background(), "first"); err != nil {
		t.Fatalf("Update(first) error = %v", err)
	}
	if err := streamer.Update(context.Background(), "first plus short growth"); err != nil {
		t.Fatalf("Update(throttled) error = %v", err)
	}
	if err := streamer.Update(context.Background(), "first"+strings.Repeat("x", 120)); err != nil {
		t.Fatalf("Update(enough growth too soon) error = %v", err)
	}

	first := mustReceivePicoMessage(t, received)
	if first.Type != TypeMessageCreate {
		t.Fatalf("first type = %q, want %q", first.Type, TypeMessageCreate)
	}
	msgID, _ := first.Payload["message_id"].(string)
	assertNoPicoMessage(t, received)

	rawStreamer := streamer.(*pocketClawStreamer)
	rawStreamer.mu.Lock()
	rawStreamer.lastAt = time.Now().Add(-61 * time.Second)
	rawStreamer.mu.Unlock()
	if err := streamer.Update(context.Background(), "first plus small growth"); err != nil {
		t.Fatalf("Update(enough time too little growth) error = %v", err)
	}
	assertNoPicoMessage(t, received)

	if err := streamer.Finalize(context.Background(), "first plus final text"); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	final := mustReceivePicoMessage(t, received)
	if final.Type != TypeMessageUpdate {
		t.Fatalf("final type = %q, want %q", final.Type, TypeMessageUpdate)
	}
	if got := final.Payload["message_id"]; got != msgID {
		t.Fatalf("final message_id = %#v, want %q", got, msgID)
	}
	if got := final.Payload[PayloadKeyContent]; got != "first plus final text" {
		t.Fatalf("final content = %#v, want final text", got)
	}
	assertNoPicoMessage(t, received)
}

func TestBeginStream_FinalizeIncludesContextUsage(t *testing.T) {
	ch := newTestPicoChannel(t)
	ch.config.Streaming = config.StreamingConfig{
		Enabled:         true,
		ThrottleSeconds: 0,
		MinGrowthChars:  0,
	}
	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	streamer, err := ch.BeginStream(context.Background(), "pocketclaw:sess-1")
	if err != nil {
		t.Fatalf("BeginStream() error = %v", err)
	}
	if setter, ok := streamer.(interface{ SetModelName(modelName string) }); ok {
		setter.SetModelName("gpt-5.4")
	}
	if err := streamer.Update(context.Background(), "partial"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	first := mustReceivePicoMessage(t, received)
	msgID, _ := first.Payload["message_id"].(string)

	contextStreamer, ok := streamer.(interface {
		FinalizeWithContext(ctx context.Context, content string, usage *bus.ContextUsage) error
	})
	if !ok {
		t.Fatal("streamer should support FinalizeWithContext")
	}
	if err := contextStreamer.FinalizeWithContext(context.Background(), "final", &bus.ContextUsage{
		UsedTokens:        10,
		TotalTokens:       100,
		HistoryTokens:     5,
		CompressAtTokens:  80,
		SummarizeAtTokens: 60,
		UsedPercent:       10,
	}); err != nil {
		t.Fatalf("FinalizeWithContext() error = %v", err)
	}

	final := mustReceivePicoMessage(t, received)
	if final.Type != TypeMessageUpdate {
		t.Fatalf("final type = %q, want %q", final.Type, TypeMessageUpdate)
	}
	if got := final.Payload["message_id"]; got != msgID {
		t.Fatalf("final message_id = %#v, want %q", got, msgID)
	}
	if got := final.Payload[PayloadKeyModelName]; got != "gpt-5.4" {
		t.Fatalf("final model_name = %#v, want %q", got, "gpt-5.4")
	}
	rawUsage, ok := final.Payload["context_usage"].(map[string]any)
	if !ok {
		t.Fatalf("final context_usage = %#v, want map", final.Payload["context_usage"])
	}
	if got := rawUsage["used_tokens"]; got != float64(10) {
		t.Fatalf("used_tokens = %#v, want 10", got)
	}
	if got := rawUsage["history_tokens"]; got != float64(5) {
		t.Fatalf("history_tokens = %#v, want 5", got)
	}
	if got := rawUsage["summarize_at_tokens"]; got != float64(60) {
		t.Fatalf("summarize_at_tokens = %#v, want 60", got)
	}
}

func TestCreateAndAddConnection_RespectsMaxConnectionsConcurrently(t *testing.T) {
	ch := newTestPicoChannel(t)

	const (
		maxConns   = 5
		goroutines = 64
		sessionID  = "session-a"
	)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errCount := 0

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			pc, err := ch.createAndAddConnection(nil, sessionID, maxConns)
			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				successCount++
				if pc == nil {
					t.Errorf("pc is nil on success")
				}
				return
			}
			if !errors.Is(err, channels.ErrTemporary) {
				t.Errorf("unexpected error: %v", err)
				return
			}
			errCount++
		}()
	}
	wg.Wait()

	if successCount > maxConns {
		t.Fatalf("successCount=%d > maxConns=%d", successCount, maxConns)
	}
	if successCount+errCount != goroutines {
		t.Fatalf("success=%d err=%d total=%d want=%d", successCount, errCount, successCount+errCount, goroutines)
	}
	if got := ch.currentConnCount(); got != maxConns {
		t.Fatalf("currentConnCount=%d want=%d", got, maxConns)
	}
}

func TestRemoveConnection_CleansBothIndexes(t *testing.T) {
	ch := newTestPicoChannel(t)

	pc, err := ch.createAndAddConnection(nil, "session-cleanup", 10)
	if err != nil {
		t.Fatalf("createAndAddConnection: %v", err)
	}

	removed := ch.removeConnection(pc.id)
	if removed == nil {
		t.Fatal("removeConnection returned nil")
	}

	ch.connsMu.RLock()
	defer ch.connsMu.RUnlock()

	if _, ok := ch.connections[pc.id]; ok {
		t.Fatalf("connID %s still exists in connections", pc.id)
	}
	if _, ok := ch.sessionConnections[pc.sessionID]; ok {
		t.Fatalf("session %s still exists in sessionConnections", pc.sessionID)
	}
	if got := len(ch.connections); got != 0 {
		t.Fatalf("len(connections)=%d want=0", got)
	}
}

func TestBroadcastToSession_TargetsOnlyRequestedSession(t *testing.T) {
	ch := newTestPicoChannel(t)

	target := &pocketClawConn{id: "target", sessionID: "s-target"}
	target.closed.Store(true)
	ch.addConnForTest(target)

	other := &pocketClawConn{id: "other", sessionID: "s-other"}
	ch.addConnForTest(other)

	err := ch.broadcastToSession("pocketclaw:s-target", newMessage(TypeMessageCreate, map[string]any{"content": "hello"}))
	if err == nil {
		t.Fatal("expected send failure due to closed target connection")
	}
	if !errors.Is(err, channels.ErrSendFailed) {
		t.Fatalf("expected ErrSendFailed, got %v", err)
	}
}

func TestSendMedia_ResolvesMediaBeforeDelivery(t *testing.T) {
	ch := newTestPicoChannel(t)
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	localPath := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(localPath, []byte("attachment body"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	ref, err := store.Store(localPath, media.MediaMeta{
		Filename:    "report.txt",
		ContentType: "text/plain",
	}, "test-scope")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	closedConn := &pocketClawConn{id: "closed", sessionID: "sess-1"}
	closedConn.closed.Store(true)
	ch.addConnForTest(closedConn)

	_, err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "pocketclaw:sess-1",
		Parts: []bus.MediaPart{{
			Ref:         ref,
			Type:        "file",
			Filename:    "report.txt",
			ContentType: "text/plain",
		}},
	})
	if !errors.Is(err, channels.ErrSendFailed) {
		t.Fatalf("SendMedia() error = %v, want ErrSendFailed", err)
	}
}

func TestSendMedia_DismissesTrackedToolFeedbackMessage(t *testing.T) {
	ch := newTestPicoChannel(t)
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	localPath := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(localPath, []byte("attachment body"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	ref, err := store.Store(localPath, media.MediaMeta{
		Filename:    "report.txt",
		ContentType: "text/plain",
	}, "test-scope")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	ch.RecordToolFeedbackMessage("pocketclaw:sess-1", "msg-progress", "🔧 `read_file`")

	var deleted struct {
		chatID    string
		messageID string
	}
	ch.deleteMessageFn = func(_ context.Context, chatID string, messageID string) error {
		deleted.chatID = chatID
		deleted.messageID = messageID
		return nil
	}

	_, err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "pocketclaw:sess-1",
		Parts: []bus.MediaPart{{
			Ref:         ref,
			Type:        "file",
			Filename:    "report.txt",
			ContentType: "text/plain",
		}},
	})
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}

	select {
	case msg := <-received:
		if msg.Type != TypeMessageCreate {
			t.Fatalf("message type = %q, want %q", msg.Type, TypeMessageCreate)
		}
	case <-time.After(time.Second):
		t.Fatal("expected media message to be delivered")
	}

	if deleted.chatID != "pocketclaw:sess-1" || deleted.messageID != "msg-progress" {
		t.Fatalf("unexpected delete target: %+v", deleted)
	}
	if _, ok := ch.currentToolFeedbackMessage("pocketclaw:sess-1"); ok {
		t.Fatal("expected tracked tool feedback to be cleared after media delivery")
	}
}

func TestSendMedia_IncludesCaptionAndAttachmentsInSinglePayload(t *testing.T) {
	ch := newTestPicoChannel(t)
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	clientConn, received, cleanup := newTestPicoWebSocket(t)
	defer cleanup()
	ch.addConnForTest(&pocketClawConn{id: "conn-1", conn: clientConn, sessionID: "sess-1"})

	localPath := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(localPath, []byte("png-body"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	ref, err := store.Store(localPath, media.MediaMeta{
		Filename:    "photo.png",
		ContentType: "image/png",
	}, "test-scope")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	_, err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "pocketclaw:sess-1",
		Parts: []bus.MediaPart{{
			Ref:         ref,
			Type:        "image",
			Filename:    "photo.png",
			ContentType: "image/png",
			Caption:     "recipe translation",
		}},
	})
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}

	select {
	case msg := <-received:
		if msg.Type != TypeMessageCreate {
			t.Fatalf("message type = %q, want %q", msg.Type, TypeMessageCreate)
		}
		payload := msg.Payload
		if got := payload[PayloadKeyContent]; got != "recipe translation" {
			t.Fatalf("content = %#v, want %q", got, "recipe translation")
		}
		rawAttachments, ok := payload["attachments"].([]any)
		if !ok || len(rawAttachments) != 1 {
			t.Fatalf("attachments = %#v, want 1 attachment", payload["attachments"])
		}
		attachment, ok := rawAttachments[0].(map[string]any)
		if !ok {
			t.Fatalf("attachment = %#v, want map", rawAttachments[0])
		}
		if got := attachment["type"]; got != "image" {
			t.Fatalf("attachment type = %#v, want image", got)
		}
		if got := attachment["filename"]; got != "photo.png" {
			t.Fatalf("attachment filename = %#v, want photo.png", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected media payload to be delivered")
	}
}

func TestPicoDownloadURLForRef(t *testing.T) {
	got, err := pocketClawDownloadURLForRef("media://attachment-1")
	if err != nil {
		t.Fatalf("pocketClawDownloadURLForRef() error = %v", err)
	}
	if got != "/pico/media/attachment-1" {
		t.Fatalf("pocketClawDownloadURLForRef() = %q, want %q", got, "/pico/media/attachment-1")
	}
}

func TestHandleMediaDownload_ServesStoredFile(t *testing.T) {
	ch := newTestPicoChannel(t)
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	if err := ch.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	localPath := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(localPath, []byte("downloadable"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	ref, err := store.Store(localPath, media.MediaMeta{
		Filename:    "report.txt",
		ContentType: "text/plain",
	}, "test-scope")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	refID := strings.TrimPrefix(ref, "media://")
	req := httptest.NewRequest("GET", "/pico/media/"+refID, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != "downloadable" {
		t.Fatalf("body = %q, want %q", body, "downloadable")
	}
	if got := rec.Header().Get("Content-Type"); got != "text/plain" {
		t.Fatalf("Content-Type = %q, want %q", got, "text/plain")
	}
}

func mustReceivePicoMessage(t *testing.T, received <-chan PocketClawMessage) PocketClawMessage {
	t.Helper()
	select {
	case msg := <-received:
		return msg
	case <-time.After(time.Second):
		t.Fatal("expected pico message")
	}
	return PocketClawMessage{}
}

func assertNoPicoMessage(t *testing.T, received <-chan PocketClawMessage) {
	t.Helper()
	select {
	case msg := <-received:
		t.Fatalf("unexpected pico message: %+v", msg)
	case <-time.After(150 * time.Millisecond):
	}
}

func (c *PocketClawChannel) addConnForTest(pc *pocketClawConn) {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()
	if c.connections == nil {
		c.connections = make(map[string]*pocketClawConn)
	}
	if c.sessionConnections == nil {
		c.sessionConnections = make(map[string]map[string]*pocketClawConn)
	}
	if _, exists := c.connections[pc.id]; exists {
		panic(fmt.Sprintf("duplicate conn id in test: %s", pc.id))
	}
	c.connections[pc.id] = pc
	bySession, ok := c.sessionConnections[pc.sessionID]
	if !ok {
		bySession = make(map[string]*pocketClawConn)
		c.sessionConnections[pc.sessionID] = bySession
	}
	bySession[pc.id] = pc
}

func newTestPicoWebSocket(t *testing.T) (*websocket.Conn, <-chan PocketClawMessage, func()) {
	t.Helper()

	received := make(chan PocketClawMessage, 4)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Upgrade() error = %v", err)
			return
		}
		defer conn.Close()
		for {
			var msg PocketClawMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			received <- msg
		}
	}))

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		server.Close()
		t.Fatalf("Dial() error = %v", err)
	}

	cleanup := func() {
		clientConn.Close()
		server.Close()
	}
	defer resp.Body.Close()
	return clientConn, received, cleanup
}
