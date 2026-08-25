// Command server runs the PocketClaw Telegram onboarding service.
//
// It issues short-lived pairing sessions, watches the PocketClaw manager bot
// for managed-bot creation, and delivers each resulting child bot token once to
// the PocketClaw install that started the pairing.
//
// It stores nothing on disk and holds nothing about the user beyond what one
// pairing needs. See README.md.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/api"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/naming"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/onboarding"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/pairing"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/ratelimit"
	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/telegram"
)

type config struct {
	managerToken     string
	managerUsername  string
	listenAddr       string
	pairingTTL       time.Duration
	rateBurst        int
	ratePerSecond    float64
	trustProxyHeader bool
}

func loadConfig() (config, error) {
	cfg := config{
		managerToken:    strings.TrimSpace(os.Getenv("TELEGRAM_MANAGER_BOT_TOKEN")),
		managerUsername: strings.TrimPrefix(strings.TrimSpace(os.Getenv("TELEGRAM_MANAGER_BOT_USERNAME")), "@"),
		listenAddr:      envOr("LISTEN_ADDR", ":8080"),
		pairingTTL:      10 * time.Minute,
		rateBurst:       10,
		ratePerSecond:   0.2,
	}

	if cfg.managerToken == "" {
		return config{}, errors.New("TELEGRAM_MANAGER_BOT_TOKEN is not set; see .env.example")
	}
	if cfg.managerUsername == "" {
		return config{}, errors.New("TELEGRAM_MANAGER_BOT_USERNAME is not set; see .env.example")
	}
	if err := naming.ValidateUsername(cfg.managerUsername); err != nil {
		return config{}, fmt.Errorf("TELEGRAM_MANAGER_BOT_USERNAME is invalid: %w", err)
	}

	if raw := os.Getenv("PAIRING_TTL_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return config{}, fmt.Errorf("PAIRING_TTL_SECONDS must be a positive integer, got %q", raw)
		}
		cfg.pairingTTL = time.Duration(seconds) * time.Second
	}
	if raw := os.Getenv("RATE_LIMIT_BURST"); raw != "" {
		burst, err := strconv.Atoi(raw)
		if err != nil || burst <= 0 {
			return config{}, fmt.Errorf("RATE_LIMIT_BURST must be a positive integer, got %q", raw)
		}
		cfg.rateBurst = burst
	}
	if raw := os.Getenv("RATE_LIMIT_PER_SECOND"); raw != "" {
		rate, err := strconv.ParseFloat(raw, 64)
		if err != nil || rate <= 0 {
			return config{}, fmt.Errorf("RATE_LIMIT_PER_SECOND must be a positive number, got %q", raw)
		}
		cfg.ratePerSecond = rate
	}
	cfg.trustProxyHeader = os.Getenv("TRUST_PROXY_HEADER") == "true"

	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := loadConfig()
	if err != nil {
		log.Error("configuration error", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot := telegram.New(cfg.managerToken, nil)
	store := pairing.NewStore(cfg.pairingTTL)
	service := onboarding.New(store, bot, cfg.managerUsername, log)

	// Fail closed at startup rather than issuing links that can never resolve.
	verifyCtx, cancelVerify := context.WithTimeout(ctx, 30*time.Second)
	me, err := service.VerifyManager(verifyCtx)
	cancelVerify()
	if err != nil {
		log.Error("manager bot is not usable", "error", err)
		os.Exit(1)
	}
	if !strings.EqualFold(me.Username, cfg.managerUsername) {
		log.Error("configured manager username does not match the bot behind the token",
			"configured", cfg.managerUsername, "actual", me.Username)
		os.Exit(1)
	}
	log.Info("manager bot verified", "username", me.Username, "id", me.ID, "can_manage_bots", me.CanManageBots)

	limiter := ratelimit.New(cfg.rateBurst, cfg.ratePerSecond)
	server := &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           api.New(service, store, limiter, cfg.trustProxyHeader, log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := service.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("telegram update loop stopped", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Info("listening", "addr", cfg.listenAddr, "pairing_ttl", cfg.pairingTTL.String())
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("http server stopped", "error", err)
		os.Exit(1)
	}
	log.Info("shutdown complete")
}
