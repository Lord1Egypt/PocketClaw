package telegram

import (
	"context"
	"encoding/json"
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

	offsets    []int64
	getMeCalls int
	methods    []string
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
		return jsonResponse(&telego.User{ID: 42, Username: "pocketclaw_test_bot", IsBot: true})

	case "getUpdates":
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
		return jsonResponse([]telego.BotCommand{})

	default:
		// setMyCommands, sendMessage and anything else the lifecycle touches.
		return jsonResponse(map[string]any{})
	}
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

	bot, err := telego.NewBot(testToken,
		telego.WithAPICaller(stub),
		telego.WithRequestConstructor(&stubConstructor{}),
		telego.WithDiscardLogger(),
	)
	require.NoError(t, err)

	messageBus := bus.NewMessageBus()
	t.Cleanup(messageBus.Close)

	base := channels.NewBaseChannel("telegram", nil, messageBus,
		config.FlexibleStringSlice{pollingOwnerID},
		channels.WithMaxMessageLength(4000),
	)
	ch := &TelegramChannel{
		BaseChannel: base,
		bot:         bot,
		bc: &config.Channel{
			Type:      config.ChannelTelegram,
			Enabled:   true,
			AllowFrom: config.FlexibleStringSlice{pollingOwnerID},
		},
		tgCfg:       &config.TelegramSettings{},
		chatIDs:     make(map[string]int64),
		mediaGroups: make(map[string]*telegramMediaGroup),
		progress:    channels.NewToolFeedbackAnimator(nil),
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
func TestFirstPollUpdateIsDeliveredWhileGetMeIsStillBlocked(t *testing.T) {
	stub := &pollingStub{
		getMeBlock: make(chan struct{}),
		batches:    [][]telego.Update{{ownerMessage(9001, 1)}},
	}
	ch, messageBus := newPollingChannel(t, stub)

	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	msg := waitForInbound(t, messageBus, 5*time.Second)
	require.Equal(t, "/start", msg.Content)
	require.Equal(t, pollingOwnerID, msg.Context.SenderID)

	// Only now is the identity call allowed to finish, proving intake never
	// depended on it.
	close(stub.getMeBlock)
}

// Running must mean receiving. It is what the desktop pairing and the status
// snapshot both read, so it may not be true while nothing can consume an update.
func TestChannelIsNotRunningUntilIntakeConsumes(t *testing.T) {
	stub := &pollingStub{getMeBlock: make(chan struct{})}
	ch, _ := newPollingChannel(t, stub)
	defer close(stub.getMeBlock)

	require.False(t, ch.IsRunning())
	require.NoError(t, ch.Start(context.Background()))
	t.Cleanup(func() { _ = ch.Stop(context.Background()) })

	require.True(t, ch.IsRunning())
	require.True(t, ch.bh.IsRunning(),
		"Running was reported before the update handler was consuming")
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
