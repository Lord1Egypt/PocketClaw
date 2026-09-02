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

const testSelfNumber = "+201012345678"

// newTestChannel builds a channel through the real constructor, so the identity
// normalization these tests depend on is the shipped one rather than a copy.
func newTestChannel(t *testing.T, selfNumber string, allowFrom ...string) (*WhatsAppNativeChannel, *bus.MessageBus) {
	t.Helper()
	messageBus := bus.NewMessageBus()
	bc := &config.Channel{
		Enabled:   true,
		Type:      config.ChannelWhatsAppNative,
		AllowFrom: config.FlexibleStringSlice(allowFrom),
	}
	created, err := NewWhatsAppNativeChannel(
		bc, "whatsapp_native", &config.WhatsAppSettings{UseNative: true},
		messageBus, t.TempDir(), selfNumber,
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

// newConfiguredChannel is the ordinary case: Self-Chat configured and its
// number on the allow-list, which is what the console's Pair button writes.
func newConfiguredChannel(t *testing.T) (*WhatsAppNativeChannel, *bus.MessageBus) {
	t.Helper()
	return newTestChannel(t, testSelfNumber, testSelfNumber)
}

func message(sender, chat, text string) *events.Message {
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

// selfMessage is the user writing in their own Self-Chat: same account on both
// sides of the routing metadata.
func selfMessage(text string) *events.Message {
	return message("201012345678@s.whatsapp.net", "201012345678@s.whatsapp.net", text)
}

func mustJID(s string) types.JID {
	jid, err := types.ParseJID(s)
	if err != nil {
		panic(err)
	}
	return jid
}

func expectNoInbound(t *testing.T, messageBus *bus.MessageBus) {
	t.Helper()
	select {
	case inbound := <-messageBus.InboundChan():
		t.Fatalf("a message reached the agent that should have been dropped: chat=%q", inbound.Context.ChatID)
	case <-time.After(150 * time.Millisecond):
	}
}

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

// --- Self-Chat is the whole world -----------------------------------------

func TestSelfChatMessageReachesTheAgent(t *testing.T) {
	ch, messageBus := newConfiguredChannel(t)

	ch.handleIncoming(selfMessage("PocketClaw WhatsApp Inbound Test"))

	expectInbound(t, messageBus, "PocketClaw WhatsApp Inbound Test")
}

func TestSelfChatUnicodeRoundTripsIntact(t *testing.T) {
	ch, messageBus := newConfiguredChannel(t)

	const text = "مرحبا يا PocketClaw 🦞"
	ch.handleIncoming(selfMessage(text))

	expectInbound(t, messageBus, text)
}

func TestSelfChatFromALinkedDeviceIsAccepted(t *testing.T) {
	// A message the user typed on their laptop arrives with a device suffix.
	// That suffix changes whenever they relink, so identity cannot key on it.
	ch, messageBus := newConfiguredChannel(t)

	ch.handleIncoming(message(
		"201012345678:12@s.whatsapp.net", "201012345678@s.whatsapp.net", "from my laptop",
	))

	expectInbound(t, messageBus, "from my laptop")
}

func TestAnotherContactIsDroppedBeforeTheBus(t *testing.T) {
	ch, messageBus := newConfiguredChannel(t)

	ch.handleIncoming(message(
		"447700900000@s.whatsapp.net", "447700900000@s.whatsapp.net", "hello there",
	))

	expectNoInbound(t, messageBus)
}

// TestTheUsersOwnMessageToAContactIsDropped is the case an allow-list alone
// would miss. The sender is the account owner, and the allow-list says the
// owner is allowed — but the conversation is with somebody else, and reading it
// is exactly what this feature promises not to do.
func TestTheUsersOwnMessageToAContactIsDropped(t *testing.T) {
	ch, messageBus := newConfiguredChannel(t)

	ch.handleIncoming(message(
		"201012345678@s.whatsapp.net", "447700900000@s.whatsapp.net", "note to a friend",
	))

	expectNoInbound(t, messageBus)
}

func TestAContactMessagingTheUserIsDropped(t *testing.T) {
	// Chat is the contact's JID even though the user is the recipient.
	ch, messageBus := newConfiguredChannel(t)

	ch.handleIncoming(message(
		"447700900000@s.whatsapp.net", "447700900000@s.whatsapp.net", "are you there",
	))

	expectNoInbound(t, messageBus)
}

func TestGroupsAreDroppedBeforeTheBus(t *testing.T) {
	ch, messageBus := newConfiguredChannel(t)

	ch.handleIncoming(message(
		"201012345678@s.whatsapp.net", "120363000000000000@g.us", "hello group",
	))

	expectNoInbound(t, messageBus)
}

func TestGroupsAreDroppedEvenWithAWildcardAllowList(t *testing.T) {
	ch, messageBus := newTestChannel(t, testSelfNumber, "*")

	ch.handleIncoming(message(
		"201012345678@s.whatsapp.net", "120363000000000000@g.us", "hello group",
	))

	expectNoInbound(t, messageBus)
}

func TestBroadcastAndNewsletterChatsAreDropped(t *testing.T) {
	ch, messageBus := newTestChannel(t, testSelfNumber, "*")

	for _, chat := range []string{
		"status@broadcast",
		"120363000000000000@newsletter",
		"120363000000000000@broadcast",
	} {
		ch.handleIncoming(message("201012345678@s.whatsapp.net", chat, "noise"))
	}

	expectNoInbound(t, messageBus)
}

// TestOtherChatTrafficCostsNothing is the API-waste gate. A linked device sees
// the whole account; if any of that reached the bus it would become an agent
// turn and a paid provider request. The bus is the gate before the agent, so
// zero publishes here is zero turns and zero provider calls.
func TestOtherChatTrafficCostsNothing(t *testing.T) {
	ch, messageBus := newTestChannel(t, testSelfNumber, "*")

	for i := 0; i < 100; i++ {
		ch.handleIncoming(message(
			"447700900000@s.whatsapp.net", "447700900000@s.whatsapp.net", "chatter"))
		ch.handleIncoming(message(
			"201012345678@s.whatsapp.net", "120363000000000000@g.us", "group chatter"))
		ch.handleIncoming(message(
			"201012345678@s.whatsapp.net", "447700900000@s.whatsapp.net", "outgoing chatter"))
	}

	expectNoInbound(t, messageBus)

	// And the one message that is ours still gets through afterwards, so the
	// boundary is a filter rather than a blanket mute.
	ch.handleIncoming(selfMessage("mine"))
	expectInbound(t, messageBus, "mine")
}

// --- Empty configuration denies -------------------------------------------

func TestNoSelfNumberDeniesEverything(t *testing.T) {
	ch, messageBus := newTestChannel(t, "", "*")

	ch.handleIncoming(selfMessage("hello"))

	expectNoInbound(t, messageBus)

	if _, err := ch.allowedTarget("201012345678@s.whatsapp.net"); err == nil {
		t.Error("outbound was permitted with no Self-Chat number configured")
	}
}

func TestEmptyAllowFromDeniesInbound(t *testing.T) {
	// BaseChannel treats an empty allow-list as allow-all, which is right for a
	// bot token and wrong for a personal account feeding an agent with shell
	// tools. Self-Chat routing alone must not be enough.
	ch, messageBus := newTestChannel(t, testSelfNumber)

	ch.handleIncoming(selfMessage("hello"))

	expectNoInbound(t, messageBus)
}

func TestEmptyAllowFromDeniesEvenTheOwnersOwnNumber(t *testing.T) {
	ch, _ := newTestChannel(t, testSelfNumber)

	if ch.IsAllowedSender(bus.SenderInfo{
		Platform:   "whatsapp",
		PlatformID: "201012345678",
	}) {
		t.Error("IsAllowedSender allowed a sender with no allow_from configured")
	}
}

func TestLookalikeNumbersAreRejected(t *testing.T) {
	ch, messageBus := newConfiguredChannel(t)

	for _, lookalike := range []string{
		"20101234567@s.whatsapp.net",   // one digit short
		"2010123456789@s.whatsapp.net", // one digit long
	} {
		ch.handleIncoming(message(lookalike, lookalike, "let me in"))
	}

	expectNoInbound(t, messageBus)
}

// --- Outbound boundary ----------------------------------------------------

func TestOutboundToSelfChatIsAllowed(t *testing.T) {
	ch, _ := newConfiguredChannel(t)

	for _, target := range []string{
		"201012345678@s.whatsapp.net",
		"201012345678",
		"+201012345678",
	} {
		if _, err := ch.allowedTarget(target); err != nil {
			t.Errorf("allowedTarget(%q) refused the Self-Chat: %v", target, err)
		}
	}
}

// TestOutboundElsewhereIsRefused is the boundary that a prompt cannot move.
// Even asked directly, the channel will not address anyone but the user.
func TestOutboundElsewhereIsRefused(t *testing.T) {
	ch, _ := newConfiguredChannel(t)

	for _, target := range []string{
		"447700900000@s.whatsapp.net",
		"447700900000",
		"120363000000000000@g.us",
		"status@broadcast",
		"120363000000000000@newsletter",
		"20101234567@s.whatsapp.net",
		"",
		"not-a-jid",
	} {
		if _, err := ch.allowedTarget(target); err == nil {
			t.Errorf("allowedTarget(%q) permitted a send outside Self-Chat", target)
		}
	}
}

// --- Identity helpers -----------------------------------------------------

func TestWhatsAppNumber(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"bare number", "201012345678", "201012345678"},
		{"typed with plus", "+201012345678", "201012345678"},
		{"plain jid", "201012345678@s.whatsapp.net", "201012345678"},
		{"jid with device", "201012345678:12@s.whatsapp.net", "201012345678"},
		{"empty", "", ""},
		{"whitespace", "   ", ""},
		// An @lid identity is an opaque id, not a phone number. Comparing it as
		// one could let an unrelated identity match an allowed number.
		{"lid identity", "123456789@lid", ""},
		{"group jid", "120363000000000000@g.us", ""},
		{"broadcast", "status@broadcast", ""},
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

func TestIsSelfChatRequiresBothHalves(t *testing.T) {
	ch, _ := newConfiguredChannel(t)
	const self = "201012345678@s.whatsapp.net"
	const other = "447700900000@s.whatsapp.net"

	cases := []struct {
		name         string
		sender, chat string
		want         bool
	}{
		{"self to self", self, self, true},
		{"self from a linked device", "201012345678:9@s.whatsapp.net", self, true},
		{"self to a contact", self, other, false},
		{"contact to contact", other, other, false},
		{"contact into our chat", other, self, false},
		{"group", self, "120363000000000000@g.us", false},
		{"broadcast", self, "status@broadcast", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := types.MessageInfo{MessageSource: types.MessageSource{
				Sender: mustJID(tc.sender), Chat: mustJID(tc.chat),
			}}
			if got := ch.isSelfChat(info); got != tc.want {
				t.Errorf("isSelfChat(sender=%s chat=%s) = %v, want %v", tc.sender, tc.chat, got, tc.want)
			}
		})
	}
}

// --- Lifecycle ------------------------------------------------------------

func TestLoggedOutStopsTheReconnectLoop(t *testing.T) {
	ch, _ := newConfiguredChannel(t)
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
	ch, _ := newConfiguredChannel(t)
	ch.runCtx = context.Background()

	ch.eventHandler(&events.Disconnected{})

	if ch.loggedOut.Load() {
		t.Error("a transient disconnect marked the session logged out")
	}
}

func TestHandleLoggedOutMarksTheChannelUnpaired(t *testing.T) {
	ch, _ := newConfiguredChannel(t)
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

// TestPairingCodeIsRequestedOnlyOnce guards the linking window: a second
// request would issue a new code and invalidate the one the user is part-way
// through typing.
func TestPairingCodeIsRequestedOnlyOnce(t *testing.T) {
	ch, _ := newConfiguredChannel(t)

	if !ch.pairRequested.CompareAndSwap(false, true) {
		t.Fatal("pairRequested started set")
	}
	// A nil client would panic if the guard let a second request through.
	if code := ch.requestPairingCode(nil); code != "" {
		t.Errorf("a second pairing request returned %q, want none", code)
	}
}

func TestPairingCodeIsNotRequestedWithoutASelfNumber(t *testing.T) {
	ch, _ := newTestChannel(t, "")

	if code := ch.requestPairingCode(nil); code != "" {
		t.Errorf("requestPairingCode returned %q with no Self-Chat number", code)
	}
}
