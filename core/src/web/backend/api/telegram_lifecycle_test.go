package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// PC-DEF-062. Removing a configured bot from a client that is not the Android
// host.
//
// The point of these is that removal is authoritative and complete. A partial
// removal is worse than none: a disabled channel that still holds a token and
// an owner reads as connected to every surface that asks, and the owner would
// silently authorise whatever bot is paired next.

func telegramChannelOnDisk(t *testing.T, configPath string) *config.Channel {
	t.Helper()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	return cfg.Channels.Get(config.ChannelTelegram)
}

func telegramTokenOnDisk(t *testing.T, configPath string) string {
	t.Helper()
	channel := telegramChannelOnDisk(t, configPath)
	if channel == nil {
		return ""
	}
	decoded, err := channel.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded: %v", err)
	}
	settings, ok := decoded.(*config.TelegramSettings)
	if !ok || settings == nil {
		return ""
	}
	return strings.TrimSpace(settings.Token.String())
}

func TestTelegramDisconnectClearsTokenOwnerAndEnabledTogether(t *testing.T) {
	handler, mux, configPath := onboardingTestEnv(t)
	if _, _, err := handler.writeTelegramCredentials("123456789:live-token", 424242); err != nil {
		t.Fatalf("writeTelegramCredentials: %v", err)
	}
	if telegramTokenOnDisk(t, configPath) == "" {
		t.Fatal("the test did not manage to configure Telegram first")
	}

	recorder := onboardingRequest(t, mux, http.MethodDelete, "/api/telegram/configuration")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	if token := telegramTokenOnDisk(t, configPath); token != "" {
		t.Fatal("the bot token survived removal")
	}
	channel := telegramChannelOnDisk(t, configPath)
	if channel == nil {
		t.Fatal("the channel entry disappeared entirely; it should be present and empty")
	}
	if channel.Enabled {
		t.Fatal("the channel is still enabled after removal")
	}
	if len(channel.AllowFrom) != 0 {
		t.Fatalf("the owner allowlist survived removal: %v", channel.AllowFrom)
	}
}

// The whole reason removal lives in Core: configuration is only half of it.
func TestTelegramDisconnectReportsWhetherItReachedTheRuntime(t *testing.T) {
	handler, mux, _ := onboardingTestEnv(t)
	if _, _, err := handler.writeTelegramCredentials("123456789:live-token", 424242); err != nil {
		t.Fatalf("writeTelegramCredentials: %v", err)
	}

	recorder := onboardingRequest(t, mux, http.MethodDelete, "/api/telegram/configuration")
	var body struct {
		OK      bool `json:"ok"`
		Applied bool `json:"applied"`
		Pending bool `json:"pending"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.OK {
		t.Fatalf("body = %+v, want ok", body)
	}
	// Applied and pending are the two honest answers, and never both.
	if body.Applied && body.Pending {
		t.Fatal("removal reported as both applied and pending")
	}
}

// A second confirmation must not fail: the dialog can be reopened, and the
// removal is the same empty state either way.
func TestTelegramDisconnectIsIdempotent(t *testing.T) {
	handler, mux, configPath := onboardingTestEnv(t)
	if _, _, err := handler.writeTelegramCredentials("123456789:live-token", 424242); err != nil {
		t.Fatalf("writeTelegramCredentials: %v", err)
	}

	for attempt := range 2 {
		recorder := onboardingRequest(t, mux, http.MethodDelete, "/api/telegram/configuration")
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d: status = %d, body = %s",
				attempt, recorder.Code, recorder.Body.String())
		}
	}
	if token := telegramTokenOnDisk(t, configPath); token != "" {
		t.Fatal("the bot token survived a repeated removal")
	}
}

// Removal then re-pairing is what Replace bot does, and the owner contract has
// to hold on the second pairing exactly as on the first.
func TestTelegramCanBePairedAgainAfterRemoval(t *testing.T) {
	handler, mux, configPath := onboardingTestEnv(t)
	if _, _, err := handler.writeTelegramCredentials("111111111:first-token", 111); err != nil {
		t.Fatalf("first pairing: %v", err)
	}
	if recorder := onboardingRequest(
		t, mux, http.MethodDelete, "/api/telegram/configuration",
	); recorder.Code != http.StatusOK {
		t.Fatalf("removal status = %d", recorder.Code)
	}
	if _, _, err := handler.writeTelegramCredentials("222222222:second-token", 222); err != nil {
		t.Fatalf("second pairing: %v", err)
	}

	channel := telegramChannelOnDisk(t, configPath)
	if !channel.Enabled {
		t.Fatal("the replacement pairing did not enable the channel")
	}
	// Exactly one owner, and the new one -- a retained previous owner would be
	// an authorisation the user never granted.
	if len(channel.AllowFrom) != 1 || strings.TrimSpace(channel.AllowFrom[0]) != "222" {
		t.Fatalf("AllowFrom = %v, want exactly [222]", channel.AllowFrom)
	}
	if token := telegramTokenOnDisk(t, configPath); token != "222222222:second-token" {
		t.Fatalf("token on disk = %q, want the replacement", token)
	}
}

// Readiness and removal have to agree: after removal nothing may read as
// configured, or the UI would keep showing a connected bot.
func TestTelegramIsNotConfiguredAfterRemoval(t *testing.T) {
	handler, mux, _ := onboardingTestEnv(t)
	if _, _, err := handler.writeTelegramCredentials("123456789:live-token", 424242); err != nil {
		t.Fatalf("writeTelegramCredentials: %v", err)
	}
	if configured, err := handler.telegramIsConfigured(); err != nil || !configured {
		t.Fatalf("configured = %v, err = %v; want configured", configured, err)
	}

	onboardingRequest(t, mux, http.MethodDelete, "/api/telegram/configuration")

	configured, err := handler.telegramIsConfigured()
	if err != nil {
		t.Fatal(err)
	}
	if configured {
		t.Fatal("Telegram still reads as configured after removal")
	}
	state, _ := handler.telegramReadiness()
	if state != readinessNotConfigured {
		t.Fatalf("state = %q, want %q", state, readinessNotConfigured)
	}
}

// The removal response carries no credential and no owner id.
func TestTelegramDisconnectResponseCarriesNothingIdentifying(t *testing.T) {
	handler, mux, _ := onboardingTestEnv(t)
	if _, _, err := handler.writeTelegramCredentials("123456789:live-token", 424242); err != nil {
		t.Fatalf("writeTelegramCredentials: %v", err)
	}

	recorder := onboardingRequest(t, mux, http.MethodDelete, "/api/telegram/configuration")
	body := recorder.Body.String()
	for _, forbidden := range []string{"live-token", "123456789", "424242"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("removal response leaked %q: %s", forbidden, body)
		}
	}
}
