//go:build whatsapp_native

package whatsapp

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// newTestChannel builds a channel through the real constructor, so the
// allow-list normalization these tests depend on is the shipped one rather than
// a copy of it.
func newTestChannel(t *testing.T, allowFrom ...string) (*WhatsAppNativeChannel, *bus.MessageBus) {
	t.Helper()
	messageBus := bus.NewMessageBus()
	bc := &config.Channel{
		Enabled:   true,
		Type:      config.ChannelWhatsAppNative,
		AllowFrom: config.FlexibleStringSlice(allowFrom),
	}
	created, err := NewWhatsAppNativeChannel(
		bc, "whatsapp_native", &config.WhatsAppSettings{UseNative: true},
		messageBus, t.TempDir(),
	)
	if err != nil {
		t.Fatalf("NewWhatsAppNativeChannel: %v", err)
	}
	ch, ok := created.(*WhatsAppNativeChannel)
	if !ok {
		t.Fatalf("constructor returned %T", created)
	}
	ch.runCtx = context.Background()
	return ch, messageBus
}

func directMessage(sender, chat, text string) *events.Message {
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Sender: mustJID(sender),
				Chat:   mustJID(chat),
			},
			ID:       "mid-test",
			PushName: "Owner",
		},
		Message: &waE2E.Message{Conversation: proto.String(text)},
	}
}

func mustJID(s string) types.JID {
	jid, err := types.ParseJID(s)
	if err != nil {
		panic(err)
	}
	return jid
}

// expectNoInbound fails if anything reaches the agent.
func expectNoInbound(t *testing.T, messageBus *bus.MessageBus) {
	t.Helper()
	select {
	case inbound := <-messageBus.InboundChan():
		t.Fatalf("message reached the agent but should have been denied: chat=%q", inbound.Context.ChatID)
	case <-time.After(150 * time.Millisecond):
	}
}

// expectInbound fails unless exactly the expected text reaches the agent.
func expectInbound(t *testing.T, messageBus *bus.MessageBus, want string) {
	t.Helper()
	select {
	case inbound := <-messageBus.InboundChan():
		if inbound.Content != want {
			t.Fatalf("content = %q, want %q", inbound.Content, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the message to reach the agent")
	}
}

// TestEmptyAllowFromDeniesInbound is the central security property of this
// channel. BaseChannel treats an empty allow-list as allow-all, which is the
// right default for a bot token nobody else holds. It is the wrong default
// here: this transport is linked to the user's personal account and feeds an
// agent holding shell and Python tools, so an unconfigured channel must accept
// nobody.
func TestEmptyAllowFromDeniesInbound(t *testing.T) {
	ch, messageBus := newTestChannel(t)

	ch.handleIncoming(directMessage(
		"201012345678@s.whatsapp.net", "201012345678@s.whatsapp.net", "hello",
	))

	expectNoInbound(t, messageBus)
}

func TestEmptyAllowFromDeniesEvenTheOwnersOwnNumber(t *testing.T) {
	// "Deny by default" has to mean nobody, not "nobody except an identity we
	// guessed". Seeding the list is the console's job, not the transport's.
	ch, messageBus := newTestChannel(t)

	if ch.IsAllowedSender(bus.SenderInfo{
		Platform:   "whatsapp",
		PlatformID: "201012345678@s.whatsapp.net",
	}) {
		t.Error("IsAllowedSender allowed a sender with no allow_from configured")
	}
	expectNoInbound(t, messageBus)
}

func TestConfiguredOwnNumberIsAccepted(t *testing.T) {
	ch, messageBus := newTestChannel(t, "+201012345678")

	ch.handleIncoming(directMessage(
		"201012345678@s.whatsapp.net", "201012345678@s.whatsapp.net", "PocketClaw WhatsApp Inbound Test",
	))

	expectInbound(t, messageBus, "PocketClaw WhatsApp Inbound Test")
}

func TestConfiguredNumberMatchesASenderCarryingADeviceSuffix(t *testing.T) {
	// A message sent from a linked device arrives as "<number>:<device>@...".
	// The device changes whenever the user relinks, so matching it would make
	// the allow-list silently stop working.
	ch, messageBus := newTestChannel(t, "+201012345678")

	ch.handleIncoming(directMessage(
		"201012345678:12@s.whatsapp.net", "201012345678@s.whatsapp.net", "from my laptop",
	))

	expectInbound(t, messageBus, "from my laptop")
}

func TestUnauthorizedSenderIsDenied(t *testing.T) {
	ch, messageBus := newTestChannel(t, "+201012345678")

	ch.handleIncoming(directMessage(
		"447700900000@s.whatsapp.net", "447700900000@s.whatsapp.net", "let me in",
	))

	expectNoInbound(t, messageBus)
}

func TestASenderWhoseNumberIsAPrefixOfTheAllowedOneIsDenied(t *testing.T) {
	ch, _ := newTestChannel(t, "+201012345678")

	if ch.IsAllowedSender(bus.SenderInfo{
		Platform:   "whatsapp",
		PlatformID: "20101234567@s.whatsapp.net",
	}) {
		t.Error("a shorter number matched the allowed number")
	}
	if ch.IsAllowedSender(bus.SenderInfo{
		Platform:   "whatsapp",
		PlatformID: "2010123456789@s.whatsapp.net",
	}) {
		t.Error("a longer number matched the allowed number")
	}
}

func TestGroupMessagesAreIgnored(t *testing.T) {
	// Phase B is direct/self chat only. Group membership is not an allow-list:
	// anyone who can add the account to a group could otherwise reach the agent.
	ch, messageBus := newTestChannel(t, "+201012345678")

	ch.handleIncoming(directMessage(
		"201012345678@s.whatsapp.net", "120363000000000000@g.us", "hello group",
	))

	expectNoInbound(t, messageBus)
}

func TestGroupMessagesAreIgnoredEvenFromAnAllowedSender(t *testing.T) {
	ch, messageBus := newTestChannel(t, "+201012345678", "*")

	ch.handleIncoming(directMessage(
		"201012345678@s.whatsapp.net", "120363000000000000@g.us", "hello group",
	))

	expectNoInbound(t, messageBus)
}

func TestDirectMessagesAreLabelledDirect(t *testing.T) {
	ch, messageBus := newTestChannel(t, "+201012345678")

	ch.handleIncoming(directMessage(
		"201012345678@s.whatsapp.net", "201012345678@s.whatsapp.net", "hi",
	))

	select {
	case inbound := <-messageBus.InboundChan():
		if inbound.Context.ChatType != "direct" {
			t.Errorf("ChatType = %q, want %q", inbound.Context.ChatType, "direct")
		}
		if inbound.Context.Raw["peer_kind"] != "direct" {
			t.Errorf("peer_kind = %q, want %q", inbound.Context.Raw["peer_kind"], "direct")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the message")
	}
}

func TestWhatsAppNumber(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"bare number", "201012345678", "201012345678"},
		{"typed with plus", "+201012345678", "201012345678"},
		{"plain jid", "201012345678@s.whatsapp.net", "201012345678"},
		{"jid with device", "201012345678:12@s.whatsapp.net", "201012345678"},
		{"empty", "", ""},
		{"whitespace", "   ", ""},
		// A @lid identity is an opaque id, not a phone number. Comparing it as
		// one could let an unrelated identity match an allowed number.
		{"lid identity", "123456789@lid", ""},
		{"group jid", "120363000000000000@g.us", ""},
		{"username", "alice", ""},
		{"mixed", "20abc12345678", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := whatsAppNumber(tc.input); got != tc.want {
				t.Errorf("whatsAppNumber(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestLoggedOutStopsTheReconnectLoop(t *testing.T) {
	// A revoked session cannot be reconnected. Without this the loop retries it
	// every five minutes for as long as the gateway runs.
	ch, _ := newTestChannel(t, "+201012345678")
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch.runCtx = runCtx
	ch.loggedOut.Store(true)

	done := make(chan struct{})
	go func() {
		ch.reconnectWithBackoff()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("reconnect loop kept running after a logout")
	}
}

func TestTransientDisconnectDoesNotMarkTheSessionLoggedOut(t *testing.T) {
	// Deleting session state on a network drop would make every tunnel change
	// cost a re-pair.
	ch, _ := newTestChannel(t, "+201012345678")
	ch.runCtx = context.Background()

	ch.eventHandler(&events.Disconnected{})

	if ch.loggedOut.Load() {
		t.Error("a transient disconnect marked the session logged out")
	}
}

func TestHandleLoggedOutMarksTheChannelUnpaired(t *testing.T) {
	ch, _ := newTestChannel(t, "+201012345678")
	ch.runCtx = context.Background()
	ch.SetRunning(true)

	ch.handleLoggedOut(&events.LoggedOut{})

	if !ch.loggedOut.Load() {
		t.Error("loggedOut was not set")
	}
	if ch.IsRunning() {
		t.Error("channel still reports running after a logout")
	}
}
