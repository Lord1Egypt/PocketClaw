// Package onboarding ties Telegram's managed-bot updates to the pairing
// sessions that asked for them.
package onboarding

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/deeplink"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/naming"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/pairing"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/telegram"
)

// BotAPI is the slice of the Telegram client this package needs. It is an
// interface so tests can drive the whole flow without a network.
type BotAPI interface {
	GetMe(ctx context.Context) (telegram.User, error)
	GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]telegram.Update, error)
	GetManagedBotToken(ctx context.Context, userID int64) (string, error)
}

// Service creates pairings and resolves them from Telegram updates.
type Service struct {
	store           *pairing.Store
	bot             BotAPI
	managerUsername string
	log             *slog.Logger

	// createAttempts bounds username regeneration when a suggestion collides
	// with another live pairing.
	createAttempts int
}

// New returns a Service. managerUsername is the public @username of the
// PocketClaw manager bot, which appears in every deep link.
func New(store *pairing.Store, bot BotAPI, managerUsername string, log *slog.Logger) *Service {
	return &Service{
		store:           store,
		bot:             bot,
		managerUsername: managerUsername,
		log:             log,
		createAttempts:  5,
	}
}

// ManagerUsername reports the configured manager bot username.
func (s *Service) ManagerUsername() string { return s.managerUsername }

// CreatePairing issues a new pairing session and returns it along with the
// poll token, which is produced exactly once here.
func (s *Service) CreatePairing() (pairing.Snapshot, string, error) {
	var lastErr error
	for attempt := 0; attempt < s.createAttempts; attempt++ {
		username, err := naming.GenerateUsername()
		if err != nil {
			return pairing.Snapshot{}, "", err
		}
		link, err := deeplink.NewBot(s.managerUsername, username, naming.DefaultDisplayName)
		if err != nil {
			return pairing.Snapshot{}, "", err
		}
		snap, token, err := s.store.Create(username, naming.DefaultDisplayName, link)
		if errors.Is(err, pairing.ErrUsernameInUse) {
			// Astronomically unlikely; retrying is cheaper than failing.
			lastErr = err
			continue
		}
		if err != nil {
			return pairing.Snapshot{}, "", err
		}
		return snap, token, nil
	}
	return pairing.Snapshot{}, "", fmt.Errorf("could not allocate a free bot username: %w", lastErr)
}

// VerifyManager checks that the configured manager bot exists and that
// Telegram has granted it bot-management rights.
func (s *Service) VerifyManager(ctx context.Context) (telegram.User, error) {
	me, err := s.bot.GetMe(ctx)
	if err != nil {
		return telegram.User{}, err
	}
	if !me.CanManageBots {
		return me, fmt.Errorf("manager bot @%s does not have bot management enabled; "+
			"turn on Bot Management Mode in the BotFather mini app", me.Username)
	}
	return me, nil
}

// HandleUpdate resolves one Telegram update against the live pairings. An
// update that matches nothing is ignored: it is normal for a manager bot to
// see bots created outside any pairing this instance issued.
func (s *Service) HandleUpdate(ctx context.Context, update telegram.Update) {
	if update.ManagedBot == nil {
		return
	}
	created := update.ManagedBot
	snap, ok := s.store.MatchUsername(created.Bot.Username)
	if !ok {
		s.log.Info("ignoring a managed-bot update with no live pairing",
			"bot_username", created.Bot.Username)
		return
	}

	if err := s.store.MarkCreated(snap.ID, created.User.ID, created.Bot.ID, created.Bot.Username); err != nil {
		s.log.Error("could not mark pairing created", "pairing_id", snap.ID, "error", err)
		return
	}
	s.log.Info("managed bot created",
		"pairing_id", snap.ID, "bot_username", created.Bot.Username, "bot_id", created.Bot.ID)

	token, err := s.bot.GetManagedBotToken(ctx, created.Bot.ID)
	if err != nil {
		// err is already redacted by the telegram client. Never log the token.
		s.log.Error("could not retrieve the managed bot token",
			"pairing_id", snap.ID, "bot_id", created.Bot.ID, "error", err)
		_ = s.store.MarkFailed(snap.ID, pairing.ReasonTokenRetrievalFailed)
		return
	}
	if err := s.store.MarkReady(snap.ID, token); err != nil {
		s.log.Error("could not store the managed bot token", "pairing_id", snap.ID, "error", err)
		_ = s.store.MarkFailed(snap.ID, pairing.ReasonTokenRetrievalFailed)
		return
	}
	s.log.Info("pairing ready for delivery", "pairing_id", snap.ID)
}

// Run long-polls Telegram until ctx is cancelled.
//
// Long-polling rather than a webhook: it needs no inbound reachability for
// Telegram, no webhook secret, and no TLS coupling. The cost is that only one
// instance may poll a given bot token at a time; see README.md, "Scaling".
func (s *Service) Run(ctx context.Context) error {
	const pollTimeoutSeconds = 30
	var offset int64
	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		updates, err := s.bot.GetUpdates(ctx, offset, pollTimeoutSeconds)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			s.log.Error("getUpdates failed", "error", err, "retry_in", backoff)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			s.HandleUpdate(ctx, update)
		}
	}
}
