package telegram

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// logTelegramLifecycle emits only a stage name and a wall-clock observation.
// No bot, owner, chat, message, token or path is carried by these events.
func logTelegramLifecycle(event string) {
	logger.DebugCF("telegram", "Telegram managed lifecycle advanced", map[string]any{
		"event":               event,
		"observed_at_unix_ms": time.Now().UnixMilli(),
	})
}

// telegramGeneration identifies one Telegram getUpdates owner.
//
// PC-DEF-061. A local, monotonic counter is the only safe way to tell one
// polling generation from the next without recording a token, bot id, owner id
// or chat id. It is process-local and carries no identity.
type telegramGeneration uint64

var telegramGenerationCounter atomic.Uint64

func nextTelegramGeneration() telegramGeneration {
	return telegramGeneration(telegramGenerationCounter.Add(1))
}

// logTelegramLifecycleGeneration is logTelegramLifecycle with the generation
// the event belongs to, so a timeline can prove which owner acted.
func logTelegramLifecycleGeneration(event string, generation uint64) {
	logger.DebugCF("telegram", "Telegram managed lifecycle advanced", map[string]any{
		"event":               event,
		"telegram_generation": generation,
		"observed_at_unix_ms": time.Now().UnixMilli(),
	})
}

// telegramIntake is the proof that a generation's getUpdates intake exists.
//
// Telego launches its long-polling goroutine and returns before that goroutine
// has issued its first HTTP getUpdates request. A started goroutine is not a
// poll: only the request itself is what makes Telegram hold updates for this
// process. This is closed when the generation's first getUpdates call is
// actually handed to the transport.
type telegramIntake struct {
	started chan struct{}
	once    sync.Once
}

func newTelegramIntake() *telegramIntake {
	return &telegramIntake{started: make(chan struct{})}
}

func (t *telegramIntake) markRequestStarted() {
	if t == nil {
		return
	}
	t.once.Do(func() { close(t.started) })
}

func (t *telegramIntake) established() bool {
	if t == nil {
		return false
	}
	select {
	case <-t.started:
		return true
	default:
		return false
	}
}

// telegramIntakeRef is the mutable slot a channel's API caller reads, so a new
// generation's intake can be installed before its poller starts.
type telegramIntakeRef struct {
	value atomic.Pointer[telegramIntake]
}

func (r *telegramIntakeRef) store(intake *telegramIntake) {
	if r != nil {
		r.value.Store(intake)
	}
}

func (r *telegramIntakeRef) load() *telegramIntake {
	if r == nil {
		return nil
	}
	return r.value.Load()
}

// telegramIntakeCaller wraps the real Bot API caller and records when a
// getUpdates request is issued. Every other method is delegated untouched.
type telegramIntakeCaller struct {
	base ta.Caller
	ref  *telegramIntakeRef
}

func (c *telegramIntakeCaller) Call(
	ctx context.Context,
	url string,
	data *ta.RequestData,
) (*ta.Response, error) {
	if strings.HasSuffix(url, "/getUpdates") {
		c.ref.load().markRequestStarted()
	}
	return c.base.Call(ctx, url, data)
}

// telegramIntakeWait bounds the wait for a generation's first getUpdates
// request. The request is issued one scheduling hop after the poller starts, so
// this is not a delay; it is the window in which a broken poller fails closed.
const telegramIntakeWait = 5 * time.Second

// waitForIntakeEstablished reports whether the generation's getUpdates intake
// became real within the bound. A false answer is fatal for the start: without
// a poll there is nothing to hand a first message to.
func waitForIntakeEstablished(intake *telegramIntake, within time.Duration) bool {
	if intake == nil {
		return false
	}
	deadline := time.Now().Add(within)
	for !intake.established() {
		if time.Now().After(deadline) {
			return false
		}
		// Yield rather than sleep: the request is issued by a goroutine in this
		// process and needs a scheduling slot, not time to pass.
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	return true
}

// Long-polling intake, and the ordering it depends on.
//
// PC-DEF-061. Telegram long polling is at-least-once only while the client
// behaves: getUpdates returns a batch and the *next* getUpdates call, which
// carries the advanced offset, is what tells Telegram to delete that batch
// forever. Telego's poller loops immediately, so a batch is confirmed as gone
// before anything in this process has necessarily looked at it.
//
// That makes the gap between the poller starting and the handler consuming the
// only place PocketClaw can lose an update it has already been given. Start
// used to put a getMe round trip inside that gap -- four seconds on the
// physical device -- and reported the channel Running while it was open. The
// rules this file exists to keep:
//
//   - nothing that can block goes between the poller and the handler;
//   - Running is not reported until the handler is consuming;
//   - an update that is dropped is said out loud, because the silent version of
//     this is indistinguishable from Telegram never having sent it.

// pollingStopWait bounds how long Stop waits for the poller to release the
// library's long-polling lock. Reached only if the poller is wedged; the normal
// case is one scheduling hop after the context is cancelled.
const pollingStopWait = 5 * time.Second

// handlerConsumingWait bounds the wait for the handler goroutine to be
// scheduled. It is a yield loop over the handler's own state, not a delay: the
// wait ends the moment the handler reports it is consuming, which in practice
// is the next scheduling point.
const handlerConsumingWait = 200 * time.Millisecond

// updateConsumer is the part of telego's bot handler this file needs.
type updateConsumer interface {
	IsRunning() bool
}

// waitForHandlerConsuming reports whether the handler became the consumer of
// the update channel within the bound.
//
// A false answer is not fatal -- the poller's channel is buffered, so updates
// wait rather than vanish -- but it means Running would be claiming more than
// is known, so the caller says so instead of asserting it.
func waitForHandlerConsuming(consumer updateConsumer, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for !consumer.IsRunning() {
		if time.Now().After(deadline) {
			return false
		}
		// Yield rather than sleep: the handler is a goroutine in this process
		// and needs a scheduling slot, not time to pass.
		runtime.Gosched()
	}
	return true
}

// observeUpdates forwards the poller's updates to the handler and records what
// passed through.
//
// Deliberately unbuffered. This exists to make delivery observable, not to add
// a second place an update can sit unprocessed -- a buffer here would recreate
// the very window the ordering fix closes.
//
// update_id and the offset it confirms are the only way to tell "Telegram never
// sent it" apart from "PocketClaw was handed it and lost it", which is exactly
// the question the first-message investigation could not answer from the logs.
// Neither is a user identifier under the logging contract; chat and sender ids
// are, and none is logged here.
func (c *TelegramChannel) observeUpdates(in <-chan telego.Update) <-chan telego.Update {
	out := make(chan telego.Update)
	// Closed when the poller's channel closes, which is the only observable
	// signal that Telego has released its long-polling lock. Stop waits on it,
	// so a channel that has been stopped can be started again -- Telego refuses
	// a second UpdatesViaLongPolling on the same bot until the first has
	// finished unwinding, and Stop used to return before that happened, which
	// turned a restart into "long polling already running" and left Telegram
	// down.
	done := make(chan struct{})
	c.pollingDone = done

	go func() {
		defer close(out)
		defer close(done)

		first := true
		for update := range in {
			generation := c.generation.Load()
			if first {
				logTelegramLifecycleGeneration("first_update_received", generation)
			}
			logger.DebugCF("telegram", "Telegram polling delivered an update", map[string]any{
				"event":               "polling.update_delivered",
				"update_id":           update.UpdateID,
				"next_offset":         update.UpdateID + 1,
				"first_update":        first,
				"telegram_generation": generation,
			})
			first = false

			select {
			case out <- update:
			case <-c.ctx.Done():
				// Telegram has already been told this batch was received, so
				// this update is gone for good. Nothing can recover it; the
				// least this can do is not lose it silently.
				logger.WarnCF("telegram",
					"Telegram update dropped during shutdown and cannot be redelivered",
					map[string]any{
						"event":     "polling.update_dropped",
						"update_id": update.UpdateID,
					})
				return
			}
		}
	}()

	return out
}

// awaitPollingStopped waits for the poller to unwind after the context is
// cancelled.
//
// Bounded, and a expiry is reported rather than hidden: the next Start would
// fail with Telego's "already running" error, and a silent wait would make that
// look like a Telegram problem.
func (c *TelegramChannel) awaitPollingStopped(ctx context.Context) {
	done := c.pollingDone
	if done == nil {
		return
	}
	timer := time.NewTimer(pollingStopWait)
	defer timer.Stop()
	select {
	case <-done:
	case <-ctx.Done():
	case <-timer.C:
		logger.WarnCF("telegram", "Telegram polling did not stop within the wait",
			map[string]any{"event": "polling.stop_timeout"})
	}
}
