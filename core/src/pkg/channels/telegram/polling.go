package telegram

import (
	"context"
	"runtime"
	"time"

	"github.com/mymmrac/telego"

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
			if first {
				logTelegramLifecycle("first_update_received")
			}
			logger.DebugCF("telegram", "Telegram polling delivered an update", map[string]any{
				"event":        "polling.update_delivered",
				"update_id":    update.UpdateID,
				"next_offset":  update.UpdateID + 1,
				"first_update": first,
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
