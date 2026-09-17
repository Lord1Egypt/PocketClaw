package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
)

// PC-DEF-061. The first owner message after a managed pairing was not received.
//
// Long polling is at-least-once only while the client behaves: getUpdates hands
// over a batch and the next call, carrying the advanced offset, is what makes
// Telegram delete it. So the gap between the poller starting and the handler
// consuming is the one place an update that arrived can still be lost, and
// these tests pin the ordering that keeps that gap empty.

const pollingOwnerID = "777000777"

// pollingStub answers Telegram calls by method, and records what was asked.
//
// Concurrency-safe: the poller runs in its own goroutine from the moment Start
// is called, so an unguarded recorder here would race the test rather than test
// anything.
type pollingStub struct {
	mu sync.Mutex

	// getMeBlock, while non-nil, holds getMe until it is closed. This is the
	// four-second round trip from the physical device, made deterministic.
	getMeBlock chan struct{}

	// batches are handed out one per getUpdates call; afterwards it answers
	// empty, which is what a quiet long poll looks like.
	batches [][]telego.Update

	// failSend and failDelete make the corresponding Bot API call fail, so the
	// cleanup ordering can be exercised without a network.
	failSend     bool
	failDelete   bool
	unauthorized map[string]bool

	// webhookURL is what getWebhookInfo reports. Non-empty means the bot is
	// attached to another service through a webhook.
	webhookURL string
	// getUpdatesErrorCode makes getUpdates answer a Bot API error (409 for a
	// bot owned by another poller). Zero means a normal answer.
	getUpdatesErrorCode   int
	getUpdatesDescription string

	offsets    []int64
	getMeCalls int
	methods    []string
	deletes    []deleteCall
	sentTexts  []string
}

// deleteCall records one deleteMessage request.
type deleteCall struct {
	chatID    int64
	messageID int
}

func (s *pollingStub) Call(_ context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
	method := url[strings.LastIndex(url, "/")+1:]

	s.mu.Lock()
	s.methods = append(s.methods, method)
	block := s.getMeBlock
	s.mu.Unlock()

	switch method {
	case "getMe":
		s.mu.Lock()
		s.getMeCalls++
		s.mu.Unlock()
		if block != nil {
			<-block
		}
		if s.isUnauthorized(method) {
			return unauthorizedResponse()
		}
		return jsonResponse(&telego.User{ID: 42, Username: "pocketclaw_test_bot", IsBot: true})

	case "getWebhookInfo":
		s.mu.Lock()
		url := s.webhookURL
		s.mu.Unlock()
		return jsonResponse(&telego.WebhookInfo{URL: url})

	case "getUpdates":
		if s.isUnauthorized(method) {
			return unauthorizedResponse()
		}
		s.mu.Lock()
		conflictCode := s.getUpdatesErrorCode
		conflictDescription := s.getUpdatesDescription
		s.mu.Unlock()
		if conflictCode != 0 {
			return &ta.Response{
				Ok:    false,
				Error: &ta.Error{ErrorCode: conflictCode, Description: conflictDescription},
			}, nil
		}
		var params struct {
			Offset int64 `json:"offset"`
		}
		if data != nil && len(data.BodyRaw) > 0 {
			_ = json.Unmarshal(data.BodyRaw, &params)
		}
		s.mu.Lock()
		s.offsets = append(s.offsets, params.Offset)
		var batch []telego.Update
		if len(s.batches) > 0 {
			batch, s.batches = s.batches[0], s.batches[1:]
		}
		s.mu.Unlock()
		if batch == nil {
			batch = []telego.Update{}
		}
		return jsonResponse(batch)

	case "getMyCommands":
		if s.isUnauthorized(method) {
			return unauthorizedResponse()
		}
		return jsonResponse([]telego.BotCommand{})

	case "sendMessage":
		var params struct {
			Text string `json:"text"`
		}
		if data != nil && len(data.BodyRaw) > 0 {
			_ = json.Unmarshal(data.BodyRaw, &params)
		}
		s.mu.Lock()
		fail := s.failSend
		s.sentTexts = append(s.sentTexts, params.Text)
		s.mu.Unlock()
		if fail {
			return nil, errors.New("send failed")
		}
		return jsonResponse(&telego.Message{MessageID: 1})

	case "deleteMessage":
		var params struct {
			ChatID    int64 `json:"chat_id"`
			MessageID int   `json:"message_id"`
		}
		if data != nil && len(data.BodyRaw) > 0 {
			_ = json.Unmarshal(data.BodyRaw, &params)
		}
		s.mu.Lock()
		s.deletes = append(s.deletes, deleteCall{chatID: params.ChatID, messageID: params.MessageID})
		fail := s.failDelete
		s.mu.Unlock()
		if fail {
			return nil, errors.New("delete failed")
		}
		return jsonResponse(true)

	default:
		// setMyCommands and anything else the lifecycle touches.
		if s.isUnauthorized(method) {
			return unauthorizedResponse()
		}
		return jsonResponse(map[string]any{})
	}
}

func (s *pollingStub) isUnauthorized(method string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unauthorized[method]
}

func unauthorizedResponse() (*ta.Response, error) {
	return &ta.Response{
		Ok:    false,
		Error: &ta.Error{ErrorCode: 401, Description: "Unauthorized"},
	}, nil
}

func (s *pollingStub) observedDeletes() []deleteCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]deleteCall(nil), s.deletes...)
}

func (s *pollingStub) observedOffsets() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.offsets...)
}

func (s *pollingStub) calledGetMe() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getMeCalls
}

func (s *pollingStub) methodCount(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, observed := range s.methods {
		if observed == method {
			count++
		}
	}
	return count
}

func (s *pollingStub) observedSentTexts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.sentTexts...)
}

func jsonResponse(result any) (*ta.Response, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &ta.Response{Ok: true, Result: raw}, nil
}

// ownerMessage is a /start from the paired owner: six characters, as the device
// log recorded it.
func ownerMessage(updateID, messageID int) telego.Update {
	return telego.Update{
		UpdateID: updateID,
		Message: &telego.Message{
			MessageID: messageID,
			From:      &telego.User{ID: 777000777, FirstName: "Owner"},
			Chat:      telego.Chat{ID: 777000777, Type: "private"},
			Text:      "/start",
		},
	}
}

// newPollingChannel builds a channel whose only paired owner is pollingOwnerID,
// wired to a real bus so delivery is asserted end to end rather than at a seam.
func newPollingChannel(t *testing.T, stub *pollingStub) (*TelegramChannel, *bus.MessageBus) {
	t.Helper()
	return newPollingChannelWithOwners(t, stub,
		config.FlexibleStringSlice{pollingOwnerID}, false)
}

// newPollingChannelWithOwners is newPollingChannel with the owner list made
// explicit, so the owner-missing setup state can be built the way the factory
// builds it (empty AllowFrom, ownerMissing true).
func newPollingChannelWithOwners(
	t *testing.T,
	stub *pollingStub,
	allowFrom config.FlexibleStringSlice,
	ownerMissing bool,
) (*TelegramChannel, *bus.MessageBus) {
	t.Helper()

	// The intake caller is the production wiring: it records when a generation's
	// first getUpdates request is actually issued, which is what Start requires
	// before it reports Running.
	intakeRef := &telegramIntakeRef{}
	bot, err := telego.NewBot(testToken,
		telego.WithAPICaller(&telegramIntakeCaller{base: stub, ref: intakeRef}),
		telego.WithRequestConstructor(&stubConstructor{}),
		telego.WithDiscardLogger(),
	)
	require.NoError(t, err)

	messageBus := bus.NewMessageBus()
	t.Cleanup(messageBus.Close)

	base := channels.NewBaseChannel("telegram", nil, messageBus,
		allowFrom,
		channels.WithMaxMessageLength(4000),
	)
	ch := &TelegramChannel{
		BaseChannel:  base,
		bot:          bot,
		ownerMissing: ownerMissing,
		bc: &config.Channel{
			Type:      config.ChannelTelegram,
			Enabled:   true,
			AllowFrom: allowFrom,
		},
		tgCfg:        &config.TelegramSettings{},
		chatIDs:      make(map[string]int64),
		intakeRef:    intakeRef,
		startCleanup: make(map[int64]telegramStartCleanup),
		mediaGroups:  make(map[string]*telegramMediaGroup),
		progress:     channels.NewToolFeedbackAnimator(nil),
		// Command registration is not what is under test and would otherwise
		// make real calls on every start.
		registerFunc: func(context.Context, []commands.Definition) error { return nil },
	}
	return ch, messageBus
}

// The bound is deliberately generous: these assert that delivery happens at
// all, never how fast. What proves intake is not waiting on getMe is that getMe
// stays blocked until after the assertion, which hangs the old ordering however
// long the bound is.
func waitForInbound(t *testing.T, messageBus *bus.MessageBus, within time.Duration) bus.InboundMessage {
	t.Helper()
	select {
	case msg := <-messageBus.InboundChan():
		return msg
	case <-time.After(within):
		t.Fatalf("no inbound message within %s", within)
		return bus.InboundMessage{}
	}
}

// The regression itself: the first poll's update must reach the bus without
// waiting for getMe.
//
// Before the fix, Username() -- a getMe on first use -- sat between the poller
// and the handler, so intake was held for as long as that call took. Four
// seconds on the device; held here until the test releases it, so a fix that
// reintroduces the dependency cannot pass by being fast.
func TestAuthenticationCompletesBeforePollingOwnsIntake(t *testing.T) {
	stub := &pollingStub{
		getMeBlock: make(chan struct{}),
		batches:    [][]telego.Update{{ownerMessage(9001, 1)}},
	}
	ch, messageBus := newPollingChannel(t, stub)

	startResult := make(chan error, 1)
	go func() { startResult <- ch.Start(context.Background()) }()
	require.Eventually(t, func() bool { return stub.calledGetMe() == 1 },
		time.Second, time.Millisecond)
	require.Zero(t, stub.methodCount("getUpdates"),
		"no unauthenticated poll may acknowledge an update")
	require.False(t, ch.IsRunning())
	close(stub.getMeBlock)
	require.NoError(t, <-startResult)
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	msg := waitForInbound(t, messageBus, 5*time.Second)
	require.Equal(t, "/start", msg.Content)
	require.Equal(t, pollingOwnerID, msg.Context.SenderID)

}

// Running must mean receiving. It is what the desktop pairing and the status
// snapshot both read, so it may not be true while nothing can consume an update.
func TestChannelIsNotRunningUntilIntakeConsumes(t *testing.T) {
	stub := &pollingStub{}
	ch, _ := newPollingChannel(t, stub)

	require.False(t, ch.IsRunning())
	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	require.True(t, ch.IsRunning())
	require.True(t, ch.bh.IsRunning(),
		"Running was reported before the update handler was consuming")
}

// The old fallback logged ready_unconfirmed and then set Running anyway. That
// let Android open the final t.me handoff against a receiver that had not
// attached. A missed scheduling bound must fail closed instead.
func TestChannelStartFailsClosedWhenHandlerReadinessIsUnconfirmed(t *testing.T) {
	stub := &pollingStub{}
	ch, _ := newPollingChannel(t, stub)
	ch.handlerReadyProbe = func(updateConsumer, time.Duration) bool { return false }

	err := ch.Start(context.Background())
	require.ErrorContains(t, err, "handler did not become ready")
	require.False(t, ch.IsRunning(),
		"unconfirmed intake must never authorize a managed handoff")
}

// A pending message sent before polling began is what the physical case was:
// the owner pressed Start in Telegram while PocketClaw was still being
// configured. Offset 0 is what asks Telegram for it, and it must not be
// discarded on arrival.
func TestUpdatePendingBeforePollingStartedIsProcessed(t *testing.T) {
	stub := &pollingStub{batches: [][]telego.Update{{ownerMessage(4200, 7)}}}
	ch, messageBus := newPollingChannel(t, stub)

	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	msg := waitForInbound(t, messageBus, 5*time.Second)
	require.Equal(t, "/start", msg.Content)

	offsets := stub.observedOffsets()
	require.NotEmpty(t, offsets)
	require.Zero(t, offsets[0],
		"the first poll must ask for everything Telegram still holds")
}

// Exactly once: one update in, one message out, and the advanced offset means
// Telegram will not hand it back.
func TestFirstMessageIsProcessedExactlyOnce(t *testing.T) {
	stub := &pollingStub{batches: [][]telego.Update{{ownerMessage(5100, 3)}}}
	ch, messageBus := newPollingChannel(t, stub)

	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	require.Equal(t, "/start", waitForInbound(t, messageBus, 5*time.Second).Content)

	select {
	case extra := <-messageBus.InboundChan():
		t.Fatalf("the same update was processed twice: %q", extra.Content)
	case <-time.After(300 * time.Millisecond):
	}

	require.Eventually(t, func() bool {
		offsets := stub.observedOffsets()
		return len(offsets) >= 2 && offsets[len(offsets)-1] == 5101
	}, 5*time.Second, 10*time.Millisecond,
		"the poll after delivery must carry update_id+1")
}

// Bot replacement. A new bot must never inherit an offset from the one it
// replaced, or it would skip that bot's first updates -- exactly the reported
// symptom. PocketClaw persists no offset, and this is what holds that true.
func TestReplacedBotStartsFromAnUnsetOffset(t *testing.T) {
	first := &pollingStub{batches: [][]telego.Update{{ownerMessage(88000, 1)}}}
	oldChannel, oldBus := newPollingChannel(t, first)
	require.NoError(t, oldChannel.Start(context.Background()))
	require.Equal(t, "/start", waitForInbound(t, oldBus, 5*time.Second).Content)
	require.Eventually(t, func() bool { return len(first.observedOffsets()) >= 2 },
		5*time.Second, 10*time.Millisecond)
	require.NoError(t, oldChannel.Stop(context.Background()))

	// The replacement is a separate bot and a separate channel, as a re-pairing
	// produces.
	second := &pollingStub{batches: [][]telego.Update{{ownerMessage(1, 1)}}}
	newChannel, newBus := newPollingChannel(t, second)
	require.NoError(t, newChannel.Start(context.Background()))
	t.Cleanup(func() { _ = newChannel.Stop(context.Background()) })

	// update_id 1 is far below the previous bot's offset: it arrives only
	// because nothing was inherited.
	require.Equal(t, "/start", waitForInbound(t, newBus, 5*time.Second).Content)
	require.Zero(t, second.observedOffsets()[0])
}

// Executes the real built-in /start definition and delivers its response
// through the real Telegram Send path. This keeps the regression about the
// product contract rather than merely proving an update reached an internal
// channel.
func replyToFirstStart(t *testing.T, ch *TelegramChannel, messageBus *bus.MessageBus) {
	t.Helper()
	inbound := waitForInbound(t, messageBus, 5*time.Second)
	require.Equal(t, "/start", inbound.Content)

	var reply string
	result := commands.NewExecutor(
		commands.NewRegistry(commands.BuiltinDefinitions()),
		&commands.Runtime{},
	).Execute(context.Background(), commands.Request{
		Channel:  inbound.Channel,
		ChatID:   inbound.ChatID,
		SenderID: inbound.SenderID,
		Text:     inbound.Content,
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	require.Equal(t, commands.OutcomeHandled, result.Outcome)
	require.NoError(t, result.Err)
	require.Equal(t, "Hello! I am PocketClaw.", reply)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: ch.Name(),
		ChatID:  inbound.ChatID,
		Context: inbound.Context,
		Content: reply,
	})
	require.NoError(t, err)
}

func TestManagedFreshAndReplacementBotsReplyToFirstStartExactlyOnce(t *testing.T) {
	tests := []struct {
		name     string
		updateID int
	}{
		{name: "fresh bot", updateID: 1},
		{name: "replacement bot with independent offset", updateID: 99001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &pollingStub{
				batches: [][]telego.Update{{ownerMessage(tt.updateID, 1)}},
			}
			ch, messageBus := newPollingChannel(t, stub)
			require.NoError(t, ch.Start(context.Background()))
			t.Cleanup(func() { _ = ch.Stop(context.Background()) })

			replyToFirstStart(t, ch, messageBus)
			require.Eventually(t, func() bool {
				return stub.methodCount("sendMessage") == 1
			}, 5*time.Second, 10*time.Millisecond)

			select {
			case duplicate := <-messageBus.InboundChan():
				t.Fatalf("first update was delivered twice: %q", duplicate.Content)
			case <-time.After(300 * time.Millisecond):
			}
			require.Equal(t, 1, stub.methodCount("sendMessage"),
				"one /start must produce exactly one reply")
			require.Zero(t, stub.observedOffsets()[0],
				"every bot identity starts from its own unset offset")
		})
	}
}

// Reconnecting the same bot is a fresh start too, and must not need the owner
// to send anything twice.
func TestRestartedChannelReceivesTheNextMessage(t *testing.T) {
	stub := &pollingStub{batches: [][]telego.Update{{ownerMessage(6001, 1)}}}
	ch, messageBus := newPollingChannel(t, stub)

	require.NoError(t, ch.Start(context.Background()))
	require.Equal(t, "/start", waitForInbound(t, messageBus, 5*time.Second).Content)
	require.NoError(t, ch.Stop(context.Background()))
	require.False(t, ch.IsRunning())

	stub.mu.Lock()
	stub.batches = [][]telego.Update{{ownerMessage(6002, 2)}}
	stub.mu.Unlock()

	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })
	require.Equal(t, "/start", waitForInbound(t, messageBus, 5*time.Second).Content)
}

// The owner contract is unchanged by any of this: one paired owner, and nobody
// else reaches the agent.
func TestPollingStillRejectsANonOwner(t *testing.T) {
	stranger := ownerMessage(7001, 1)
	stranger.Message.From = &telego.User{ID: 123456, FirstName: "Stranger"}
	stranger.Message.Chat = telego.Chat{ID: 123456, Type: "private"}

	stub := &pollingStub{batches: [][]telego.Update{{stranger}}}
	ch, messageBus := newPollingChannel(t, stub)

	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	select {
	case msg := <-messageBus.InboundChan():
		t.Fatalf("a message from an unpaired sender reached the agent: %q", msg.Content)
	case <-time.After(500 * time.Millisecond):
	}
}

// getMe still happens -- which bot this is has to stay in the log -- just not
// where it can hold up intake.
func TestBotIdentityIsStillResolved(t *testing.T) {
	stub := &pollingStub{}
	ch, _ := newPollingChannel(t, stub)

	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	require.Eventually(t, func() bool { return stub.calledGetMe() > 0 },
		5*time.Second, 10*time.Millisecond)
}

// PC-DEF-061. A started poller goroutine is not an established poll. Telego
// returns from UpdatesViaLongPolling before its first getUpdates request is
// issued, so a generation whose intake never becomes real must fail closed
// rather than be reported ready with nothing receiving.
func TestChannelStartFailsClosedWhenIntakeIsUnconfirmed(t *testing.T) {
	stub := &pollingStub{}
	ch, _ := newPollingChannel(t, stub)
	ch.intakeProbe = func(*telegramIntake, time.Duration) bool { return false }

	err := ch.Start(context.Background())
	require.ErrorContains(t, err, "intake did not become usable")
	require.False(t, ch.IsRunning(),
		"unconfirmed intake must never authorize a managed handoff")
	require.Zero(t, ch.PollingGeneration(),
		"a refused start must not leave a generation to be inherited")
}

func TestGetMeUnauthorizedFailsBeforePollingStarts(t *testing.T) {
	stub := &pollingStub{unauthorized: map[string]bool{"getMe": true}}
	ch, _ := newPollingChannel(t, stub)

	err := ch.Start(context.Background())
	require.ErrorIs(t, err, errTelegramAuthentication)
	require.False(t, ch.IsRunning())
	require.Equal(t, "authentication_failed", ch.RuntimeFailure())
	require.Zero(t, stub.methodCount("getUpdates"))
	require.Eventually(t, func() bool { return ch.PollingGeneration() == 0 },
		time.Second, time.Millisecond)
}

func TestGetUpdatesUnauthorizedRetiresGenerationWithoutRetry(t *testing.T) {
	stub := &pollingStub{unauthorized: map[string]bool{"getUpdates": true}}
	ch, _ := newPollingChannel(t, stub)

	err := ch.Start(context.Background())
	require.ErrorIs(t, err, errTelegramAuthentication)
	require.False(t, ch.IsRunning())
	require.Equal(t, "authentication_failed", ch.RuntimeFailure())
	require.Eventually(t, func() bool { return ch.PollingGeneration() == 0 },
		time.Second, time.Millisecond)
	require.Equal(t, 1, stub.methodCount("getUpdates"),
		"invalid credentials must not enter Telego's retry loop")
}

func TestCommandMenuUnauthorizedIsTerminal(t *testing.T) {
	for _, method := range []string{"getMyCommands", "setMyCommands"} {
		t.Run(method, func(t *testing.T) {
			stub := &pollingStub{unauthorized: map[string]bool{method: true}}
			ch, _ := newPollingChannel(t, stub)
			ch.registerFunc = nil

			require.NoError(t, ch.Start(context.Background()))
			require.Eventually(t, func() bool {
				return ch.RuntimeFailure() == "authentication_failed"
			}, time.Second, time.Millisecond)
			require.Eventually(t, func() bool {
				return !ch.IsRunning() && ch.PollingGeneration() == 0
			}, time.Second, time.Millisecond)
			require.Equal(t, 1, stub.methodCount(method),
				"an authentication failure must not be retried")
			if method == "getMyCommands" {
				require.Zero(t, stub.methodCount("setMyCommands"),
					"a rejected read must not fall through to a write")
			}
		})
	}
}

// The active generation is named, and retired to zero once its poller has
// confirmed exit, so a successor cannot inherit a stale generation's success.
func TestChannelNamesItsPollingGenerationAndRetiresIt(t *testing.T) {
	stub := &pollingStub{}
	ch, _ := newPollingChannel(t, stub)

	require.Zero(t, ch.PollingGeneration())
	require.NoError(t, ch.Start(context.Background()))
	first := ch.PollingGeneration()
	require.NotZero(t, first)

	require.NoError(t, ch.Stop(context.Background()))
	require.Zero(t, ch.PollingGeneration())

	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })
	require.NotZero(t, ch.PollingGeneration())
	require.NotEqual(t, first, ch.PollingGeneration(),
		"a restarted channel is a new polling generation")
}

// The intake caller is what proves a request exists, so it must record
// getUpdates and only getUpdates.
func TestIntakeCallerSeparatesRequestFromUsableIntake(t *testing.T) {
	intake := newTelegramIntake()
	ref := &telegramIntakeRef{}
	ref.store(intake)
	base := &recordingCaller{}
	caller := &telegramIntakeCaller{base: base, ref: ref}

	_, _ = caller.Call(context.Background(), "https://api.telegram.org/bot1:AA/getMe", nil)
	require.False(t, channelClosed(intake.started), "getMe is not polling intake")
	require.True(t, channelClosed(intake.auth), "successful getMe confirms authentication")

	_, _ = caller.Call(context.Background(), "https://api.telegram.org/bot1:AA/getUpdates", nil)
	require.True(t, channelClosed(intake.started), "the getUpdates request was issued")
	require.True(t, channelClosed(intake.usable), "Telegram accepted getUpdates")

	// A second request does not reopen anything and does not panic.
	_, _ = caller.Call(context.Background(), "https://api.telegram.org/bot1:AA/getUpdates", nil)
	require.True(t, channelClosed(intake.usable))
	require.Equal(t, 3, base.calls)
}

func TestWaitForIntakeUsable(t *testing.T) {
	t.Run("request initiation alone is insufficient", func(t *testing.T) {
		intake := newTelegramIntake()
		intake.markRequestStarted()
		require.Error(t, waitForIntakeUsable(intake, time.Millisecond))
	})

	t.Run("returns only after Telegram accepts getUpdates", func(t *testing.T) {
		intake := newTelegramIntake()
		intake.markUsable()
		require.NoError(t, waitForIntakeUsable(intake, time.Second))
	})

	t.Run("401 is terminal", func(t *testing.T) {
		intake := newTelegramIntake()
		intake.markUnauthorized()
		require.ErrorIs(t, waitForIntakeUsable(intake, time.Second), errTelegramAuthentication)
	})
}

func channelClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

type recordingCaller struct {
	calls  int
	bodies [][]byte
}

func (c *recordingCaller) Call(_ context.Context, _ string, data *ta.RequestData) (*ta.Response, error) {
	c.calls++
	if data != nil {
		c.bodies = append(c.bodies, append([]byte(nil), data.BodyRaw...))
	}
	return jsonResponse(map[string]any{})
}

func TestOnlyFirstGetUpdatesIsShortenedForReadiness(t *testing.T) {
	intake := newTelegramIntake()
	ref := &telegramIntakeRef{}
	ref.store(intake)
	base := &recordingCaller{}
	caller := &telegramIntakeCaller{base: base, ref: ref}
	request := &ta.RequestData{ContentType: "application/json", BodyRaw: []byte(`{"timeout":30}`)}

	_, _ = caller.Call(context.Background(), "https://api.telegram.org/bot1:AA/getUpdates", request)
	_, _ = caller.Call(context.Background(), "https://api.telegram.org/bot1:AA/getUpdates", request)
	require.Len(t, base.bodies, 2)

	var first, second struct {
		Timeout int `json:"timeout"`
	}
	require.NoError(t, json.Unmarshal(base.bodies[0], &first))
	require.NoError(t, json.Unmarshal(base.bodies[1], &second))
	require.Zero(t, first.Timeout)
	require.Equal(t, 30, second.Timeout,
		"steady-state polling must keep its long hold instead of hammering Telegram")
}

type fakeConsumer struct {
	mu      sync.Mutex
	running bool
}

func (f *fakeConsumer) IsRunning() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

func (f *fakeConsumer) start() {
	f.mu.Lock()
	f.running = true
	f.mu.Unlock()
}

func TestWaitForHandlerConsuming(t *testing.T) {
	t.Run("returns once the consumer reports running", func(t *testing.T) {
		consumer := &fakeConsumer{}
		go func() {
			time.Sleep(10 * time.Millisecond)
			consumer.start()
		}()
		require.True(t, waitForHandlerConsuming(consumer, 5*time.Second))
	})

	t.Run("gives up rather than blocking a start forever", func(t *testing.T) {
		require.False(t, waitForHandlerConsuming(&fakeConsumer{}, 20*time.Millisecond))
	})
}

// PC-DEF-061 UX cleanup. Telegram's native /start is removed from the private
// bot chat after PocketClaw has received it and successfully sent the built-in
// reply. It is cosmetic and best-effort; it never runs before the reply, never
// targets anything but the exact inbound message, and never fails the turn.

// A. Successful /start: the reply is sent, then deleteMessage removes exactly
// the inbound message the reply belongs to.
func TestStartCleanupDeletesTheExactInboundMessageAfterReply(t *testing.T) {
	const inboundMessageID = 77
	stub := &pollingStub{batches: [][]telego.Update{{ownerMessage(9001, inboundMessageID)}}}
	ch, messageBus := newPollingChannel(t, stub)
	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	replyToFirstStart(t, ch, messageBus)

	deletes := stub.observedDeletes()
	require.Len(t, deletes, 1, "the /start message must be deleted exactly once")
	require.Equal(t, inboundMessageID, deletes[0].messageID,
		"deletion must target the inbound /start message, not a later one")
	require.Equal(t, int64(777000777), deletes[0].chatID)
	require.Equal(t, 1, stub.methodCount("sendMessage"))
}

// B. Reply failure: no reply was delivered, so nothing may be deleted.
func TestStartCleanupDoesNotDeleteWhenTheReplyFails(t *testing.T) {
	stub := &pollingStub{
		batches:  [][]telego.Update{{ownerMessage(9002, 88)}},
		failSend: true,
	}
	ch, messageBus := newPollingChannel(t, stub)
	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	inbound := waitForInbound(t, messageBus, 5*time.Second)
	require.Equal(t, "/start", inbound.Content)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: ch.Name(),
		ChatID:  inbound.ChatID,
		Context: inbound.Context,
		Content: commands.StartReplyText,
	})
	require.Error(t, err)
	require.Empty(t, stub.observedDeletes(),
		"a /start must never be deleted when its reply did not land")
}

// C. Delete failure: the reply already stands, the turn still succeeds, and the
// failure is not retried.
func TestStartCleanupDeleteFailureDoesNotBreakTheReply(t *testing.T) {
	stub := &pollingStub{
		batches:    [][]telego.Update{{ownerMessage(9003, 99)}},
		failDelete: true,
	}
	ch, messageBus := newPollingChannel(t, stub)
	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	// replyToFirstStart requires no error, so a failed cosmetic delete cannot
	// have failed the turn.
	replyToFirstStart(t, ch, messageBus)
	require.Len(t, stub.observedDeletes(), 1)
	require.Equal(t, 1, stub.methodCount("sendMessage"))

	time.Sleep(200 * time.Millisecond)
	require.Len(t, stub.observedDeletes(), 1, "a failed delete must not be retried")
	require.Equal(t, 1, stub.methodCount("sendMessage"), "the reply must not be duplicated")
}

// D. Normal messages are never deleted, even if a reply is sent for them.
func TestStartCleanupNeverDeletesNormalMessages(t *testing.T) {
	msg := ownerMessage(9004, 11)
	msg.Message.Text = "hi"
	stub := &pollingStub{batches: [][]telego.Update{{msg}}}
	ch, messageBus := newPollingChannel(t, stub)
	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	inbound := waitForInbound(t, messageBus, 5*time.Second)
	require.Equal(t, "hi", inbound.Content)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: ch.Name(),
		ChatID:  inbound.ChatID,
		Context: inbound.Context,
		Content: commands.StartReplyText,
	})
	require.NoError(t, err)
	require.Empty(t, stub.observedDeletes())
}

// E. Only the built-in /start is eligible. Other commands, including one whose
// name merely begins with "start", are never deleted.
func TestStartCleanupNeverDeletesOtherCommands(t *testing.T) {
	for i, command := range []string{"/help", "/clear", "/startle", "/stop"} {
		msg := ownerMessage(9100+i, 20+i)
		msg.Message.Text = command
		stub := &pollingStub{batches: [][]telego.Update{{msg}}}
		ch, messageBus := newPollingChannel(t, stub)
		require.NoError(t, ch.Start(context.Background()))

		inbound := waitForInbound(t, messageBus, 5*time.Second)
		require.Equal(t, command, inbound.Content)

		_, err := ch.Send(context.Background(), bus.OutboundMessage{
			Channel: ch.Name(),
			ChatID:  inbound.ChatID,
			Context: inbound.Context,
			Content: commands.StartReplyText,
		})
		require.NoError(t, err)
		require.Empty(t, stub.observedDeletes(), "%s must never be deleted", command)

		require.NoError(t, ch.Stop(context.Background()))
	}
}

// F. Private chats only. A /start in a group is not deleted.
func TestStartCleanupIsPrivateChatOnly(t *testing.T) {
	msg := ownerMessage(9006, 13)
	msg.Message.Chat = telego.Chat{ID: -100200, Type: "group"}
	stub := &pollingStub{batches: [][]telego.Update{{msg}}}
	ch, messageBus := newPollingChannel(t, stub)
	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	// Whether or not the group trigger lets it through, no cleanup is recorded.
	select {
	case <-messageBus.InboundChan():
	case <-time.After(300 * time.Millisecond):
	}

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: ch.Name(),
		ChatID:  "-100200",
		Content: commands.StartReplyText,
	})
	require.NoError(t, err)
	require.Empty(t, stub.observedDeletes(),
		"a group /start must not be deleted")
}

// G. Generation safety: deletion belongs to the generation that received the
// /start. A receipt from a superseded generation is never cleaned up.
func TestStartCleanupRequiresTheSameActiveGeneration(t *testing.T) {
	stub := &pollingStub{batches: [][]telego.Update{{ownerMessage(9007, 14)}}}
	ch, messageBus := newPollingChannel(t, stub)
	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	inbound := waitForInbound(t, messageBus, 5*time.Second)
	require.Equal(t, "/start", inbound.Content)

	// A new generation now owns intake; the receipt belongs to the old one.
	ch.generation.Store(ch.generation.Load() + 1)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: ch.Name(),
		ChatID:  inbound.ChatID,
		Context: inbound.Context,
		Content: commands.StartReplyText,
	})
	require.NoError(t, err)
	require.Empty(t, stub.observedDeletes(),
		"a superseded generation's receipt must not be deleted by its successor")
}
