package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func testRegistry() *callbackRegistry {
	return newCallbackRegistry()
}

func mintFor(t *testing.T, r *callbackRegistry, action, value, chatID, senderID string) string {
	t.Helper()
	data, err := r.mint(callbackEntry{
		action: action, value: value, chatID: chatID, senderID: senderID,
	})
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	return data
}

// H + I. callback_data is an opaque handle. A model name in it would tell
// Telegram, anyone the message is forwarded to, and any chat export what
// PocketClaw is configured with — and would let a crafted callback name a model
// that was never offered.
func TestCallbackDataCarriesNoModelOrConfiguration(t *testing.T) {
	r := testRegistry()
	const model = "gemini-2.5-flash"
	data := mintFor(t, r, "model.select", model, "chat-1", "user-1")

	for _, forbidden := range []string{
		model, "gemini", "flash", "sk-", "http://", "https://", "api", "token", "key",
	} {
		if strings.Contains(strings.ToLower(data), strings.ToLower(forbidden)) {
			t.Fatalf("callback_data leaked %q: %q", forbidden, data)
		}
	}

	// Telegram rejects callback_data above 64 bytes outright.
	if len(data) > 64 {
		t.Fatalf("callback_data is %d bytes, over Telegram's 64-byte limit", len(data))
	}

	// It must still resolve to the real action server-side.
	entry, ok := r.resolve(data)
	if !ok || entry.value != model {
		t.Fatalf("handle did not resolve to its action: %+v ok=%v", entry, ok)
	}
}

// Two mints never collide, so one picker's handle cannot stand for another's.
func TestCallbackHandlesAreUnique(t *testing.T) {
	r := testRegistry()
	seen := map[string]bool{}
	for range 200 {
		data := mintFor(t, r, "model.select", "m", "chat-1", "user-1")
		if seen[data] {
			t.Fatal("a callback handle repeated; handles must be unguessable and distinct")
		}
		seen[data] = true
	}
}

// J. An invented handle resolves to nothing.
func TestUnknownCallbackHandleIsRejected(t *testing.T) {
	r := testRegistry()
	for _, data := range []string{"pc:not-a-real-handle", "no-prefix", "", "pc:"} {
		if _, ok := r.resolve(data); ok {
			t.Fatalf("unknown handle %q resolved", data)
		}
	}
}

// K. A handle stops working once its TTL passes, so a forwarded or
// screenshotted picker stops being an action anyone can replay.
func TestExpiredCallbackHandleIsRejected(t *testing.T) {
	r := testRegistry()
	now := time.Now()
	r.now = func() time.Time { return now }

	data := mintFor(t, r, "model.select", "m", "chat-1", "user-1")
	if _, ok := r.resolve(data); !ok {
		t.Fatal("a fresh handle should resolve")
	}

	r.now = func() time.Time { return now.Add(callbackHandleTTL + time.Second) }
	if _, ok := r.resolve(data); ok {
		t.Fatal("an expired handle resolved")
	}
	// And it is dropped rather than left to accumulate.
	if _, ok := r.resolve(data); ok {
		t.Fatal("the expired handle survived resolution")
	}
}

// L. A handle is bound to the chat and person it was built for.
func TestCallbackHandleIsBoundToItsChatAndSender(t *testing.T) {
	r := testRegistry()
	data := mintFor(t, r, "model.select", "m", "chat-1", "user-1")
	entry, ok := r.resolve(data)
	if !ok {
		t.Fatal("handle should resolve")
	}

	if !entry.authorizes("chat-1", "user-1") {
		t.Fatal("the owner must be authorized")
	}
	for _, other := range []struct{ chat, sender string }{
		{"chat-2", "user-1"}, // forwarded into another chat
		{"chat-1", "user-2"}, // someone else tapping in the same group
		{"chat-2", "user-2"},
	} {
		if entry.authorizes(other.chat, other.sender) {
			t.Fatalf("chat=%q sender=%q was authorized for another's menu", other.chat, other.sender)
		}
	}
}

// U. Closing a picker invalidates its buttons, so a stale menu left in the chat
// cannot be tapped again.
func TestInvalidatingAMessageDropsItsHandles(t *testing.T) {
	r := testRegistry()
	mine := mintFor(t, r, "model.select", "m", "chat-1", "user-1")
	other := mintFor(t, r, "model.select", "m", "chat-1", "user-1")
	r.bindMessage(mine, "chat-1", "100")
	r.bindMessage(other, "chat-1", "200")

	r.invalidateMessage("chat-1", "100")

	if _, ok := r.resolve(mine); ok {
		t.Fatal("a closed picker's handle still resolves")
	}
	if _, ok := r.resolve(other); !ok {
		t.Fatal("closing one picker invalidated another's buttons")
	}
}

// The registry is bounded, so repeatedly opening pickers cannot grow it without
// limit.
func TestRegistryIsBounded(t *testing.T) {
	r := testRegistry()
	for range maxTrackedCallbacks + 50 {
		mintFor(t, r, "model.select", "m", "chat-1", "user-1")
	}
	r.mu.Lock()
	size := len(r.entries)
	r.mu.Unlock()
	if size > maxTrackedCallbacks {
		t.Fatalf("registry holds %d entries, above its %d bound", size, maxTrackedCallbacks)
	}
}

// The keyboard renders labels for the reader and handles for the wire.
func TestKeyboardRendersLabelsAndOpaqueHandles(t *testing.T) {
	channel := &TelegramChannel{callbacks: testRegistry()}
	menu := &bus.InteractiveMenu{Rows: []bus.MenuRow{
		{Buttons: []bus.MenuButton{{Label: "✓ gemini-2.5-flash", Action: "model.select", Value: "gemini-2.5-flash", Current: true}}},
		{Buttons: []bus.MenuButton{{Label: "✕ Cancel", Action: "menu.cancel"}}},
	}}

	rows, handles := channel.menuToKeyboard(menu, "chat-1", "user-1")
	if len(rows) != 2 || len(handles) != 2 {
		t.Fatalf("expected two rows and two handles, got %d/%d", len(rows), len(handles))
	}
	if rows[0][0].Label != "✓ gemini-2.5-flash" {
		t.Fatalf("the label the reader sees was altered: %q", rows[0][0].Label)
	}
	if strings.Contains(rows[0][0].Data, "gemini") {
		t.Fatalf("the wire carries the model name: %q", rows[0][0].Data)
	}
}

// Opening a second picker makes the first one a dead card. Its handles stop
// working, and the caller is told which message to retire so the conversation
// does not accumulate cards still saying "Choose a model:".
func TestOpeningANewPickerRetiresTheOldOne(t *testing.T) {
	r := testRegistry()

	first := mintFor(t, r, "model.select", "Alpha", "chat-1", "user-1")
	r.bindMessage(first, "chat-1", "100")
	if previous := r.replaceActivePicker("chat-1", "100"); previous != "" {
		t.Fatalf("the first picker displaced %q; there was nothing before it", previous)
	}
	if _, ok := r.resolve(first); !ok {
		t.Fatal("the live picker's buttons should work")
	}

	second := mintFor(t, r, "model.select", "Beta", "chat-1", "user-1")
	r.bindMessage(second, "chat-1", "200")
	previous := r.replaceActivePicker("chat-1", "200")

	if previous != "100" {
		t.Fatalf("the displaced picker was reported as %q, want 100 — it would be "+
			"left in the chat as a dead card", previous)
	}
	if _, ok := r.resolve(first); ok {
		t.Fatal("the old picker is still actionable after being replaced")
	}
	if _, ok := r.resolve(second); !ok {
		t.Fatal("replacing invalidated the new picker's own buttons")
	}
	if live := r.activePicker("chat-1"); live != "200" {
		t.Fatalf("live picker = %q, want 200", live)
	}
}

// One chat's picker does not disturb another's.
func TestPickerReplacementIsPerChat(t *testing.T) {
	r := testRegistry()

	other := mintFor(t, r, "model.select", "Alpha", "chat-2", "user-2")
	r.bindMessage(other, "chat-2", "900")
	r.replaceActivePicker("chat-2", "900")

	mine := mintFor(t, r, "model.select", "Alpha", "chat-1", "user-1")
	r.bindMessage(mine, "chat-1", "100")
	r.replaceActivePicker("chat-1", "100")
	r.replaceActivePicker("chat-1", "200")

	if _, ok := r.resolve(other); !ok {
		t.Fatal("replacing a picker in one chat killed another chat's picker")
	}
	if live := r.activePicker("chat-2"); live != "900" {
		t.Fatalf("chat-2 live picker = %q, want 900", live)
	}
}

// Re-registering the same message is not a replacement, so nothing is retired.
func TestReRegisteringTheSamePickerRetiresNothing(t *testing.T) {
	r := testRegistry()
	handle := mintFor(t, r, "model.select", "Alpha", "chat-1", "user-1")
	r.bindMessage(handle, "chat-1", "100")

	r.replaceActivePicker("chat-1", "100")
	if previous := r.replaceActivePicker("chat-1", "100"); previous != "" {
		t.Fatalf("the same picker reported itself as displaced: %q", previous)
	}
	if _, ok := r.resolve(handle); !ok {
		t.Fatal("re-registering invalidated the picker's own buttons")
	}
}

// A cancelled picker stops being the chat's live one, so the next picker has
// nothing to retire.
func TestCancellingClearsTheLivePicker(t *testing.T) {
	r := testRegistry()
	handle := mintFor(t, r, "model.select", "Alpha", "chat-1", "user-1")
	r.bindMessage(handle, "chat-1", "100")
	r.replaceActivePicker("chat-1", "100")

	r.invalidateMessage("chat-1", "100")
	r.clearActivePicker("chat-1", "100")

	if live := r.activePicker("chat-1"); live != "" {
		t.Fatalf("a cancelled picker is still the live one: %q", live)
	}
	if previous := r.replaceActivePicker("chat-1", "200"); previous != "" {
		t.Fatalf("the next picker tried to retire a closed card: %q", previous)
	}
}
