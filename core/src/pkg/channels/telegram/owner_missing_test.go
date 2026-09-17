package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// Telegram configured with a valid bot credential but no owner identity.
//
// This is a distinct lifecycle state, not a working bot. A private sender must
// get deterministic setup guidance exactly once and nothing else may happen: no
// agent turn, no provider call, no tool, no session write, no config mutation
// and no command execution. A group is left unanswered.

func ownerMissingPrivateMessage(updateID, messageID int, userID int64, text string) *telego.Message {
	return &telego.Message{
		MessageID: messageID,
		From:      &telego.User{ID: userID, FirstName: "Sender"},
		Chat:      telego.Chat{ID: userID, Type: "private"},
		Text:      text,
	}
}

// requireNoInbound fails if the owner-missing path published anything. The bus
// is the only route to the agent, so an empty channel here is the proof that no
// turn, tool or provider call can follow.
func requireNoInbound(t *testing.T, messageBus *bus.MessageBus) {
	t.Helper()
	select {
	case msg := <-messageBus.InboundChan():
		t.Fatalf("owner-missing path must not publish inbound work: %+v", msg)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestOwnerMissingPrivateMessageGetsSetupGuidanceAndNoAgentTurn(t *testing.T) {
	stub := &pollingStub{}
	ch, messageBus := newPollingChannelWithOwners(t, stub, nil, true)
	require.True(t, ch.OwnerMissing(), "an empty AllowFrom must be the owner-missing state")

	err := ch.handleMessages(context.Background(), []*telego.Message{
		ownerMissingPrivateMessage(1, 10, 123456789, "hello"),
	})
	require.NoError(t, err)

	sent := stub.observedSentTexts()
	require.Len(t, sent, 1, "the setup guidance must be sent exactly once")
	require.Contains(t, sent[0], "setup is incomplete")
	require.Contains(t, sent[0], "123456789", "the sender's own id is the one fact they need")
	requireNoInbound(t, messageBus)
}

// /start is not special in this state: it must not execute the built-in start
// reply as if the sender were the authorized owner.
func TestOwnerMissingStartDoesNotRunTheAuthorizedStartCommand(t *testing.T) {
	stub := &pollingStub{}
	ch, messageBus := newPollingChannelWithOwners(t, stub, nil, true)

	err := ch.handleMessages(context.Background(), []*telego.Message{
		ownerMissingPrivateMessage(1, 11, 555000111, "/start"),
	})
	require.NoError(t, err)

	sent := stub.observedSentTexts()
	require.Len(t, sent, 1)
	require.NotContains(t, sent[0], "Hello! I am PocketClaw.")
	require.Contains(t, sent[0], "setup is incomplete")
	requireNoInbound(t, messageBus)
}

// A group or supergroup must not have setup guidance published into it.
func TestOwnerMissingGroupMessageIsLeftUnanswered(t *testing.T) {
	stub := &pollingStub{}
	ch, messageBus := newPollingChannelWithOwners(t, stub, nil, true)

	err := ch.handleMessages(context.Background(), []*telego.Message{{
		MessageID: 12,
		From:      &telego.User{ID: 123456789},
		Chat:      telego.Chat{ID: -100987654321, Type: "supergroup"},
		Text:      "hello",
	}})
	require.NoError(t, err)

	require.Empty(t, stub.observedSentTexts(),
		"no setup guidance may be emitted into a group")
	requireNoInbound(t, messageBus)
}

// Each sender is shown only their own numeric id.
func TestOwnerMissingGuidanceShowsOnlyTheSendersOwnId(t *testing.T) {
	stub := &pollingStub{}
	ch, _ := newPollingChannelWithOwners(t, stub, nil, true)

	require.NoError(t, ch.handleMessages(context.Background(),
		[]*telego.Message{ownerMissingPrivateMessage(1, 1, 111222333, "hi")}))
	require.NoError(t, ch.handleMessages(context.Background(),
		[]*telego.Message{ownerMissingPrivateMessage(2, 2, 444555666, "hi")}))

	sent := stub.observedSentTexts()
	require.Len(t, sent, 2)
	require.Contains(t, sent[0], "111222333")
	require.NotContains(t, sent[0], "444555666")
	require.Contains(t, sent[1], "444555666")
	require.NotContains(t, sent[1], "111222333")
}

// With an owner configured, a non-owner is still rejected silently: no
// guidance, no bus publish. The owner-missing state does not broaden access.
func TestOwnerConfiguredNonOwnerStillGetsNoGuidance(t *testing.T) {
	stub := &pollingStub{}
	ch, messageBus := newPollingChannel(t, stub)
	require.False(t, ch.OwnerMissing())

	err := ch.handleMessages(context.Background(), []*telego.Message{
		ownerMissingPrivateMessage(1, 1, 999888777, "hello"),
	})
	require.NoError(t, err)

	require.Empty(t, stub.observedSentTexts())
	requireNoInbound(t, messageBus)
}

// The constructor contract: zero owners is the setup state, one valid owner is
// normal, and several or invalid owners are still refused.
func TestNewTelegramChannelOwnerContracts(t *testing.T) {
	newChannel := func(t *testing.T, allowFrom config.FlexibleStringSlice) (*TelegramChannel, error) {
		t.Helper()
		messageBus := bus.NewMessageBus()
		t.Cleanup(messageBus.Close)
		return NewTelegramChannel(
			&config.Channel{Type: config.ChannelTelegram, Enabled: true, AllowFrom: allowFrom},
			&config.TelegramSettings{Token: *config.NewSecureString(testToken)},
			messageBus,
		)
	}

	ch, err := newChannel(t, config.FlexibleStringSlice{})
	require.NoError(t, err, "zero owners is the owner-missing setup state, not an error")
	require.True(t, ch.OwnerMissing())

	ch, err = newChannel(t, config.FlexibleStringSlice{"424242"})
	require.NoError(t, err)
	require.False(t, ch.OwnerMissing())

	for _, invalid := range []config.FlexibleStringSlice{
		{"*"},
		{"mutable_username"},
		{"424242", "515151"},
		{"0"},
		{"-5"},
		{""},
	} {
		if _, err := newChannel(t, invalid); err == nil {
			t.Fatalf("AllowFrom %v must be refused", invalid)
		}
	}
}
