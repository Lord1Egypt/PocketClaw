package telegram

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func TestStartCommandRegistration_DoesNotBlock(t *testing.T) {
	ch := &TelegramChannel{}
	started := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch.registerFunc = func(context.Context, []commands.Definition) error {
		started <- struct{}{}
		return errors.New("temporary failure")
	}

	ch.startCommandRegistration(ctx, []commands.Definition{{Name: "help"}})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("registration did not start asynchronously")
	}
}

func TestStartCommandRegistration_RetriesUntilSuccessThenStops(t *testing.T) {
	ch := &TelegramChannel{
		commandRegDelayFn: func(int) time.Duration { return 5 * time.Millisecond },
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var attempts atomic.Int32
	ch.registerFunc = func(context.Context, []commands.Definition) error {
		n := attempts.Add(1)
		if n < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	ch.startCommandRegistration(ctx, []commands.Definition{{Name: "help", Description: "Help"}})

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		if attempts.Load() >= 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if attempts.Load() < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", attempts.Load())
	}

	stable := attempts.Load()
	time.Sleep(30 * time.Millisecond)
	if attempts.Load() != stable {
		t.Fatalf("expected retries to stop after success, got %d -> %d", stable, attempts.Load())
	}
}

func TestStartCommandRegistration_StopsAfterCancel(t *testing.T) {
	ch := &TelegramChannel{
		commandRegDelayFn: func(int) time.Duration { return 5 * time.Millisecond },
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var attempts atomic.Int32
	ch.registerFunc = func(context.Context, []commands.Definition) error {
		attempts.Add(1)
		return errors.New("always fail")
	}

	ch.startCommandRegistration(ctx, []commands.Definition{{Name: "help", Description: "Help"}})

	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond) // allow in-flight attempt to settle
	stable := attempts.Load()
	time.Sleep(30 * time.Millisecond)
	if attempts.Load() != stable {
		t.Fatalf("expected retries to quiesce after cancel, got %d -> %d", stable, attempts.Load())
	}
}

// The complete registry, not a hardcoded handful.
//
// Start passes commands.BuiltinDefinitions() straight through, so what
// registration receives is the authoritative command set. A regression that
// narrowed the menu to /start would show up here as a shorter list.
func TestStartCommandRegistration_ReceivesTheCompleteCommandSet(t *testing.T) {
	ch := &TelegramChannel{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make(chan []commands.Definition, 1)
	ch.registerFunc = func(_ context.Context, defs []commands.Definition) error {
		received <- defs
		return nil
	}

	ch.startCommandRegistration(ctx, commands.BuiltinDefinitions())

	select {
	case defs := <-received:
		want := commands.BuiltinDefinitions()
		if len(defs) != len(want) {
			t.Fatalf("registered %d commands, want the whole set of %d", len(defs), len(want))
		}
		names := make(map[string]bool, len(defs))
		for _, def := range defs {
			names[def.Name] = true
		}
		for _, def := range want {
			if !names[def.Name] {
				t.Errorf("/%s was not passed to registration", def.Name)
			}
		}
		if !names["start"] {
			t.Error("/start must be registered: it is the command Telegram shows a new chat")
		}
	case <-time.After(time.Second):
		t.Fatal("registration was never attempted")
	}
}

// A reconnected or replaced managed bot is a new Telegram bot with an empty
// command menu, and the gateway restart that brings it up runs Start again.
// Registration must therefore be attempted again on the new channel rather than
// remembering that it already succeeded once.
func TestStartCommandRegistration_RegistersAgainAfterAReconnect(t *testing.T) {
	var attempts atomic.Int32
	register := func(context.Context, []commands.Definition) error {
		attempts.Add(1)
		return nil
	}
	defs := commands.BuiltinDefinitions()

	first := &TelegramChannel{registerFunc: register}
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	first.startCommandRegistration(firstCtx, defs)
	waitForAttempts(t, &attempts, 1)

	// What Stop does to the registration goroutine.
	if first.commandRegCancel != nil {
		first.commandRegCancel()
	}
	cancelFirst()

	second := &TelegramChannel{registerFunc: register}
	secondCtx, cancelSecond := context.WithCancel(context.Background())
	defer cancelSecond()
	second.startCommandRegistration(secondCtx, defs)

	waitForAttempts(t, &attempts, 2)
}

// An empty definition list means the menu is never touched. That is correct --
// there is nothing to publish -- but it is also the shape that would silently
// produce the owner's symptom, so it is pinned rather than left implicit.
func TestStartCommandRegistration_DoesNothingWithNoDefinitions(t *testing.T) {
	ch := &TelegramChannel{}
	var attempts atomic.Int32
	ch.registerFunc = func(context.Context, []commands.Definition) error {
		attempts.Add(1)
		return nil
	}

	ch.startCommandRegistration(context.Background(), nil)

	time.Sleep(30 * time.Millisecond)
	if got := attempts.Load(); got != 0 {
		t.Fatalf("attempts = %d, want 0 for an empty definition list", got)
	}
}

func waitForAttempts(t *testing.T, attempts *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if attempts.Load() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("attempts = %d, want at least %d", attempts.Load(), want)
}
