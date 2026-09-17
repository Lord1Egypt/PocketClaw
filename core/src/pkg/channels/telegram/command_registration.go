package telegram

import (
	"context"
	"errors"
	"math/rand"
	"slices"
	"time"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/logger"
)

var commandRegistrationBackoff = []time.Duration{
	5 * time.Second,
	15 * time.Second,
	60 * time.Second,
	5 * time.Minute,
	10 * time.Minute,
}

func commandRegistrationDelay(attempt int) time.Duration {
	if len(commandRegistrationBackoff) == 0 {
		return 0
	}
	base := commandRegistrationBackoff[min(attempt, len(commandRegistrationBackoff)-1)]
	// Full jitter in [0.5, 1.0) to avoid synchronized retries across instances.
	return time.Duration(float64(base) * (0.5 + rand.Float64()*0.5))
}

// RegisterCommands publishes the command menu on Telegram.
//
// What it logs is what it actually did, not what it was asked to do. An earlier
// version reported success with the number of *definitions* it received, so
// "commands registered count=14" was printed whether fourteen commands reached
// Telegram, none did because every definition had been filtered out, or the call
// was skipped because Telegram already agreed. That made the log useless as
// evidence either way, which matters here because the command menu cannot be
// inspected from inside the process -- only Telegram knows what it is showing.
func (c *TelegramChannel) RegisterCommands(ctx context.Context, defs []commands.Definition) error {
	botCommands := make([]telego.BotCommand, 0, len(defs))
	var dropped []string
	for _, def := range defs {
		if def.Name == "" || def.Description == "" {
			// Telegram requires both, so such an entry cannot be published. It
			// is named rather than skipped in silence: a command missing from
			// the menu because its description was lost is indistinguishable
			// from one that was never defined.
			dropped = append(dropped, def.Name)
			continue
		}
		botCommands = append(botCommands, telego.BotCommand{
			Command:     def.Name,
			Description: def.Description,
		})
	}
	if len(dropped) > 0 {
		logger.WarnCF("telegram", "Commands cannot be published and are missing from the menu",
			map[string]any{"commands": dropped, "defined": len(defs)})
	}

	current, err := c.bot.GetMyCommands(ctx, &telego.GetMyCommandsParams{})
	if err != nil {
		if errors.Is(err, errTelegramAuthentication) {
			return err
		}
		// If we can't read current commands, fall through to set them.
		logger.WarnCF("telegram", "Failed to get current commands, will set unconditionally",
			map[string]any{"error": err.Error()})
	} else if slices.Equal(current, botCommands) {
		logger.InfoCF("telegram", "Telegram command menu already current", map[string]any{
			"registered": len(current),
			"bot":        c.botUsername(),
		})
		return nil
	}

	if err = c.bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: botCommands,
	}); err != nil {
		return err
	}

	logger.InfoCF("telegram", "Telegram command menu set", map[string]any{
		"sent":    len(botCommands),
		"defined": len(defs),
		"bot":     c.botUsername(),
	})
	return nil
}

// botUsername identifies which bot the menu was published to, so a replaced or
// reconnected managed bot can be told apart from the one before it in the log.
func (c *TelegramChannel) botUsername() string {
	c.botIdentityMu.RLock()
	defer c.botIdentityMu.RUnlock()
	return c.botIdentity
}

// CommandsRegistered reports whether the menu reached Telegram.
//
// PC-DEF-061. This is what lets "Connected" mean ready for the owner's first
// message rather than merely configured. It latches on success and is never
// cleared while the channel lives: a menu Telegram has accepted stays accepted,
// and a retry after success is not attempted.
func (c *TelegramChannel) CommandsRegistered() bool {
	return c.commandsRegistered.Load()
}

func (c *TelegramChannel) startCommandRegistration(ctx context.Context, defs []commands.Definition) {
	if len(defs) == 0 {
		// Nothing to publish, so nothing can be waited on. Reported as done
		// rather than pending, or a readiness gate would never finish.
		c.commandsRegistered.Store(true)
		return
	}

	register := c.registerFunc
	if register == nil {
		register = c.RegisterCommands
	}
	delayFn := c.commandRegDelayFn
	if delayFn == nil {
		delayFn = commandRegistrationDelay
	}

	regCtx, cancel := context.WithCancel(ctx)
	c.commandRegCancel = cancel

	// Registration runs asynchronously so Telegram message intake is never blocked
	// by temporary upstream API failures. Retry stops on success or channel shutdown.
	go func() {
		attempt := 0
		timer := time.NewTimer(0)
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		defer timer.Stop()
		for {
			err := register(regCtx, defs)
			if err == nil {
				c.commandsRegistered.Store(true)
				// Deliberately carries no count: RegisterCommands reports what
				// it actually published, and a second number here that came from
				// the definition list would contradict it.
				logger.InfoCF("telegram", "Telegram command registration completed",
					map[string]any{"event": "commands.registration_completed"})
				return
			}
			if errors.Is(err, errTelegramAuthentication) {
				// Invalid credentials cannot heal on a timer. The API caller has
				// already retired this generation; do not create a second retry
				// loop here.
				return
			}

			delay := delayFn(attempt)
			logger.WarnCF("telegram", "Telegram command registration failed; will retry", map[string]any{
				"error":       err.Error(),
				"retry_after": delay.String(),
			})
			attempt++

			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(delay)

			select {
			case <-regCtx.Done():
				return
			case <-timer.C:
			}
		}
	}()
}
