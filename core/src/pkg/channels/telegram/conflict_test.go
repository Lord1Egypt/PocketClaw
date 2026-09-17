package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// PC-DEF-069. A bot that already belongs to another service.
//
// getMe succeeding proves only that the token is valid. It says nothing about
// whether PocketClaw can own the update stream, so two conflict classes are
// handled terminally and without touching the other service:
//
//   - an active webhook, found non-destructively with getWebhookInfo;
//   - another long poller, which Telegram answers with HTTP 409.
//
// Neither is retried, neither leaves a polling generation behind, and neither
// ever deletes or replaces a webhook.

// A configured webhook is refused before any poll is attempted, and the webhook
// is never deleted or replaced.
func TestWebhookConflictIsDetectedNonDestructively(t *testing.T) {
	stub := &pollingStub{webhookURL: "https://other-service.invalid/telegram/hook"}
	ch, _ := newPollingChannel(t, stub)

	err := ch.Start(context.Background())
	require.ErrorIs(t, err, errTelegramWebhookActive)
	require.Equal(t, "conflict:webhook_active", ch.RuntimeFailure())
	require.False(t, ch.IsRunning())
	require.False(t, ch.CommandsRegistered())
	require.Zero(t, ch.PollingGeneration())

	require.Equal(t, 1, stub.methodCount("getWebhookInfo"))
	require.Zero(t, stub.methodCount("getUpdates"),
		"an active webhook must be refused before any poll is attempted")
	require.Zero(t, stub.methodCount("deleteWebhook"),
		"PocketClaw must never delete another service's webhook")
	require.Zero(t, stub.methodCount("setWebhook"),
		"PocketClaw must never replace another service's webhook")
}

// A 409 on getUpdates is terminal: one ownership attempt, no retry loop, the
// generation retired, and Running never reported.
func TestGetUpdatesConflictRetiresTheGenerationWithoutRetry(t *testing.T) {
	stub := &pollingStub{
		getUpdatesErrorCode:   409,
		getUpdatesDescription: "Conflict: terminated by other getUpdates request",
	}
	ch, messageBus := newPollingChannel(t, stub)

	err := ch.Start(context.Background())
	require.ErrorIs(t, err, errTelegramConflict)
	require.Equal(t, "conflict:bot_in_use", ch.RuntimeFailure())
	require.False(t, ch.IsRunning())
	require.False(t, ch.CommandsRegistered())
	require.Eventually(t, func() bool { return ch.PollingGeneration() == 0 },
		time.Second, time.Millisecond)

	// A second ownership attempt would mean the retry loop is fighting the
	// external poller. Give it time to happen if it were going to.
	time.Sleep(300 * time.Millisecond)
	require.Equal(t, 1, stub.methodCount("getUpdates"),
		"a 409 must not enter Telego's eight-second retry loop")
	require.Zero(t, stub.methodCount("deleteWebhook"))
	requireNoInbound(t, messageBus)
}

// 409 and 401 are different failures with different user actions. A conflict
// must never be classified as an invalid credential.
func TestConflictIsNotAuthenticationFailure(t *testing.T) {
	stub := &pollingStub{
		getUpdatesErrorCode:   409,
		getUpdatesDescription: "Conflict: terminated by other getUpdates request",
	}
	ch, _ := newPollingChannel(t, stub)

	_ = ch.Start(context.Background())
	require.Equal(t, "conflict:bot_in_use", ch.RuntimeFailure())
	require.NotEqual(t, "authentication_failed", ch.RuntimeFailure())
}

// A 409 whose description names a webhook is the webhook subtype even when the
// proactive getWebhookInfo was unreadable or raced: the status code is the
// reliable signal, the description only refines it.
func TestGetUpdatesWebhookConflictClassifiesAsWebhookActive(t *testing.T) {
	stub := &pollingStub{
		getUpdatesErrorCode:   409,
		getUpdatesDescription: "Conflict: can't use getUpdates method while webhook is active",
	}
	ch, _ := newPollingChannel(t, stub)

	_ = ch.Start(context.Background())
	require.Equal(t, "conflict:webhook_active", ch.RuntimeFailure())
}

// A webhook that appears after the proactive check is still caught. The check
// reads empty, then the poll answers 409 with a webhook description.
func TestWebhookAppearingAfterThePreflightIsStillCaught(t *testing.T) {
	stub := &pollingStub{
		webhookURL:            "",
		getUpdatesErrorCode:   409,
		getUpdatesDescription: "Conflict: can't use getUpdates method while webhook is active",
	}
	ch, _ := newPollingChannel(t, stub)

	_ = ch.Start(context.Background())
	require.Equal(t, "conflict:webhook_active", ch.RuntimeFailure())
	require.Zero(t, stub.methodCount("deleteWebhook"))
}
