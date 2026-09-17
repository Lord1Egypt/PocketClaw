package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
	"github.com/sipeed/picoclaw/pkg/status"
)

// PC-DEF-061. "Connected" has to mean Telegram is receiving.
//
// The physical failure was a user who was told Connected and whose first /start
// went unanswered, because completion was reported when the gateway had been
// restarted -- not when the channel was consuming. These pin the mapping from
// the gateway's own status snapshot to what the Dashboard is allowed to say,
// and in particular that no state short of ready is ever reported as ready.

// testPollingGeneration stands in for the local polling owner a real gateway
// channel reports. PC-DEF-061: a ready answer must name its generation.
var testPollingGeneration = uint64(7)

// fakeGatewayHealth serves a detail snapshot, and only to a bearer that matches.
func fakeGatewayHealth(t *testing.T, token string, channels []status.Channel) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		body := health.StatusResponse{}
		if r.URL.Query().Get("detail") == "1" {
			body.Detail = &status.Snapshot{Channels: channels}
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	return server
}

// readinessEnv configures Telegram, points the handler at a fake gateway, and
// makes the gateway look like it is running.
func readinessEnv(t *testing.T, channels []status.Channel) *Handler {
	t.Helper()
	handler, _, configPath := onboardingTestEnv(t)

	// A configured Telegram channel, written the way the pairing writes one.
	if _, _, err := handler.writeTelegramCredentials("123456789:test-token", 424242); err != nil {
		t.Fatalf("writeTelegramCredentials: %v", err)
	}

	const token = "gateway-bearer-token"
	server := fakeGatewayHealth(t, token, channels)
	host, port := splitHostPortForTest(t, server.URL)

	gateway.mu.Lock()
	previous := gateway.pidData
	gateway.pidData = &ppid.PidFileData{
		PID: 1, Host: host, Port: port, Token: token,
	}
	gateway.mu.Unlock()
	t.Cleanup(func() {
		gateway.mu.Lock()
		gateway.pidData = previous
		gateway.mu.Unlock()
	})

	// The gateway's liveness depends on a process this package owns, which a
	// test cannot produce; the fake health server is what stands in for it.
	previousProbe := gatewayRunningProbe
	gatewayRunningProbe = func(*Handler) bool { return true }
	t.Cleanup(func() { gatewayRunningProbe = previousProbe })

	_ = configPath
	return handler
}

// A bot owned by another service is a terminal conflict, never ready, and it
// names no polling generation that could authorize a handoff.
func TestTelegramReadinessReportsWebhookConflict(t *testing.T) {
	handler := readinessEnv(t, []status.Channel{{
		Name: "telegram", Configured: true, RuntimeFailure: "conflict:webhook_active",
	}})

	state, detail, generation := handler.telegramReadinessWithGeneration()
	if state != readinessConflict {
		t.Fatalf("state = %q, want %q", state, readinessConflict)
	}
	if detail != "webhook_active" {
		t.Fatalf("detail = %q, want webhook_active", detail)
	}
	if generation != 0 {
		t.Fatalf("generation = %d, want 0: a conflict authorizes nothing", generation)
	}
}

func TestTelegramReadinessReportsBotInUse(t *testing.T) {
	handler := readinessEnv(t, []status.Channel{{
		Name: "telegram", Configured: true, RuntimeFailure: "conflict:bot_in_use",
	}})

	state, detail, generation := handler.telegramReadinessWithGeneration()
	if state != readinessConflict || detail != "bot_in_use" || generation != 0 {
		t.Fatalf("state/detail/generation = %q/%q/%d, want telegram_conflict/bot_in_use/0",
			state, detail, generation)
	}
}

// 401 and 409 are different failures and must stay distinct all the way to
// readiness.
func TestTelegramReadinessKeeps401And409Distinct(t *testing.T) {
	auth := readinessEnv(t, []status.Channel{{
		Name: "telegram", Configured: true, RuntimeFailure: "authentication_failed",
	}})
	state, detail, _ := auth.telegramReadinessWithGeneration()
	if state != readinessAuthenticationFailed || detail != "invalid_credentials" {
		t.Fatalf("401 mapped to %q/%q, want authentication_failed/invalid_credentials", state, detail)
	}
}

// writeOwnerlessTelegramConfig persists a valid token with no owner: the state
// the desktop manual save (PATCH /api/config) can produce, and the reason Core
// has to defend it.
func writeOwnerlessTelegramConfig(t *testing.T, handler *Handler) {
	t.Helper()
	cfg, channel, settings, err := handler.loadTelegramConfigForUpdate()
	if err != nil {
		t.Fatalf("loadTelegramConfigForUpdate: %v", err)
	}
	settings.Token.Set("123456789:test-token")
	channel.Enabled = true
	channel.Type = config.ChannelTelegram
	channel.AllowFrom = config.FlexibleStringSlice{}
	if err := config.SaveConfig(handler.configPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
}

// A valid token with no owner is an incomplete setup, never ready, and it must
// not name a polling generation that could authorize a handoff.
func TestTelegramReadinessReportsOwnerMissingAsSetupRequired(t *testing.T) {
	handler, _, _ := onboardingTestEnv(t)
	writeOwnerlessTelegramConfig(t, handler)

	if !handler.telegramOwnerMissing() {
		t.Fatal("an enabled token with no owner must report owner_missing")
	}
	state, detail, generation := handler.telegramReadinessWithGeneration()
	if state != readinessSetupRequired {
		t.Fatalf("state = %q, want %q", state, readinessSetupRequired)
	}
	if detail != "owner_missing" {
		t.Fatalf("detail = %q, want owner_missing", detail)
	}
	if generation != 0 {
		t.Fatalf("generation = %d, want 0: an incomplete setup authorizes nothing", generation)
	}
}

func TestTelegramReadinessEndpointReportsSetupRequired(t *testing.T) {
	handler, mux, _ := onboardingTestEnv(t)
	writeOwnerlessTelegramConfig(t, handler)

	recorder := onboardingRequest(t, mux, http.MethodGet, "/api/telegram/readiness")
	var body struct {
		State  string `json:"state"`
		Ready  bool   `json:"ready"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.State != string(readinessSetupRequired) {
		t.Fatalf("state = %q, want setup_required", body.State)
	}
	if body.Ready {
		t.Fatal("an incomplete setup must never report ready")
	}
	if body.Detail != "owner_missing" {
		t.Fatalf("detail = %q, want owner_missing", body.Detail)
	}
}

func TestTelegramOwnerMissingIsFalseWhenAnOwnerIsConfigured(t *testing.T) {
	handler, _, _ := onboardingTestEnv(t)
	if _, _, err := handler.writeTelegramCredentials("123456789:test-token", 424242); err != nil {
		t.Fatalf("writeTelegramCredentials: %v", err)
	}
	if handler.telegramOwnerMissing() {
		t.Fatal("a configured owner must not report owner_missing")
	}
}

// The same authoritative state is derived from the persisted configuration, so
// the desktop manual save and the mobile/managed writer converge on one
// contract rather than each deciding for itself.
func TestTelegramOwnerStateIsDerivedFromConfigurationRegardlessOfWriter(t *testing.T) {
	handler, _, _ := onboardingTestEnv(t)

	writeOwnerlessTelegramConfig(t, handler)
	if state, _ := handler.telegramReadiness(); state != readinessSetupRequired {
		t.Fatalf("ownerless config: state = %q, want setup_required", state)
	}

	// The mobile/managed writer always names exactly one owner.
	if _, _, err := handler.writeTelegramCredentials("123456789:test-token", 424242); err != nil {
		t.Fatalf("writeTelegramCredentials: %v", err)
	}
	if handler.telegramOwnerMissing() {
		t.Fatal("the managed writer must leave no owner-missing state")
	}
}

func TestTelegramReadinessIsNotReadyWhileTheChannelIsStarting(t *testing.T) {
	handler := readinessEnv(t, []status.Channel{
		{Name: "telegram", Configured: true, Started: true, Running: false, PollingGeneration: &testPollingGeneration},
	})

	state, _ := handler.telegramReadiness()
	if state != readinessChannelStarting {
		t.Fatalf("state = %q, want %q", state, readinessChannelStarting)
	}
}

func TestTelegramReadinessWaitsForTheCommandMenu(t *testing.T) {
	notYet := false
	handler := readinessEnv(t, []status.Channel{
		{
			Name: "telegram", Configured: true, Started: true, Running: true, PollingGeneration: &testPollingGeneration,
			CommandsRegistered: &notYet,
		},
	})

	state, _ := handler.telegramReadiness()
	if state != readinessRegisteringCommands {
		t.Fatalf("state = %q, want %q", state, readinessRegisteringCommands)
	}
}

func TestTelegramReadinessIsReadyOnlyWhenBothAreTrue(t *testing.T) {
	registered := true
	handler := readinessEnv(t, []status.Channel{
		{
			Name: "telegram", Configured: true, Started: true, Running: true, PollingGeneration: &testPollingGeneration,
			CommandsRegistered: &registered,
		},
	})

	state, _ := handler.telegramReadiness()
	if state != readinessReady {
		t.Fatalf("state = %q, want %q", state, readinessReady)
	}
}

func TestTelegramReadinessRejectsThePreviousGenerationWhileConfigApplies(t *testing.T) {
	registered := true
	handler := readinessEnv(t, []status.Channel{
		{
			Name: "telegram", Configured: true, Started: true, Running: true, PollingGeneration: &testPollingGeneration,
			CommandsRegistered: &registered,
		},
	})
	finishPendingConfigApply()
	takePendingConfigApply()
	t.Cleanup(func() {
		finishPendingConfigApply()
		takePendingConfigApply()
		setPendingConfigApplyError("")
	})

	markConfigApplyPending("telegram_configured")
	state, detail := handler.telegramReadiness()
	if state != readinessGatewayStarting || detail != "configuration_applying" {
		t.Fatalf("pending state = %q (%q), want gateway_starting (configuration_applying)", state, detail)
	}

	if reason := claimPendingConfigApply(); reason != "telegram_configured" {
		t.Fatalf("claimed reason = %q", reason)
	}
	state, detail = handler.telegramReadiness()
	if state != readinessGatewayStarting || detail != "configuration_applying" {
		t.Fatalf("claimed state = %q (%q), want no readiness gap", state, detail)
	}

	finishPendingConfigApply()
	state, _ = handler.telegramReadiness()
	if state != readinessReady {
		t.Fatalf("applied state = %q, want %q", state, readinessReady)
	}
}

// PC-DEF-061. Running without a named polling owner is exactly the state that
// let a handoff open against a receiver that had not established intake. It
// must not be reported ready.
func TestTelegramReadinessRejectsARunningChannelWithNoPollingGeneration(t *testing.T) {
	registered := true
	handler := readinessEnv(t, []status.Channel{
		{
			Name: "telegram", Configured: true, Started: true, Running: true,
			CommandsRegistered: &registered,
		},
	})

	state, detail := handler.telegramReadiness()
	if state != readinessChannelStarting || detail != "generation_unconfirmed" {
		t.Fatalf("state = %q (%q), want channel_starting (generation_unconfirmed)", state, detail)
	}
}

func TestTelegramReadinessReportsAuthenticationFailureAsTerminal(t *testing.T) {
	registered := true
	handler := readinessEnv(t, []status.Channel{
		{
			Name: "telegram", Configured: true, Started: true, Running: true,
			CommandsRegistered: &registered, PollingGeneration: &testPollingGeneration,
			RuntimeFailure: "authentication_failed",
		},
	})

	state, detail, generation := handler.telegramReadinessWithGeneration()
	if state != readinessAuthenticationFailed || detail != "invalid_credentials" {
		t.Fatalf("state = %q (%q), want authentication_failed (invalid_credentials)", state, detail)
	}
	if generation != 0 {
		t.Fatalf("failed generation %d must not authorize a handoff", generation)
	}
}

// The ready answer names the generation it authorized, so a client can require
// the same owner across the handoff.
func TestTelegramReadinessReportsThePollingGeneration(t *testing.T) {
	registered := true
	handler := readinessEnv(t, []status.Channel{
		{
			Name: "telegram", Configured: true, Started: true, Running: true,
			CommandsRegistered: &registered, PollingGeneration: &testPollingGeneration,
		},
	})
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	recorder := onboardingRequest(t, mux, http.MethodGet, "/api/telegram/readiness")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		State      string `json:"state"`
		Ready      bool   `json:"ready"`
		Generation uint64 `json:"generation"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Ready || body.Generation != testPollingGeneration {
		t.Fatalf("body = %+v, want ready generation %d", body, testPollingGeneration)
	}
}

// A channel that publishes no menu reports nothing, which must not be read as
// "not yet" -- a gate that waited on it would never finish.
func TestTelegramReadinessDoesNotWaitOnAChannelWithNoMenu(t *testing.T) {
	handler := readinessEnv(t, []status.Channel{
		{Name: "telegram", Configured: true, Started: true, Running: true, PollingGeneration: &testPollingGeneration},
	})

	state, _ := handler.telegramReadiness()
	if state != readinessReady {
		t.Fatalf("state = %q, want %q", state, readinessReady)
	}
}

// The gateway is up but has not built the channel yet.
func TestTelegramReadinessReportsGatewayStartingWithoutTheChannel(t *testing.T) {
	handler := readinessEnv(t, []status.Channel{
		{Name: "pocketclaw", Configured: true, Started: true, Running: true},
	})

	state, _ := handler.telegramReadiness()
	if state != readinessGatewayStarting {
		t.Fatalf("state = %q, want %q", state, readinessGatewayStarting)
	}
}

// No token on disk is a distinct answer from "starting": nothing is coming.
func TestTelegramReadinessReportsNotConfigured(t *testing.T) {
	handler, _, _ := onboardingTestEnv(t)

	state, _ := handler.telegramReadiness()
	if state != readinessNotConfigured {
		t.Fatalf("state = %q, want %q", state, readinessNotConfigured)
	}
}

// The endpoint never reports ready without the state agreeing.
func TestTelegramReadinessEndpointReportsTheState(t *testing.T) {
	registered := true
	handler := readinessEnv(t, []status.Channel{
		{
			Name: "telegram", Configured: true, Started: true, Running: true, PollingGeneration: &testPollingGeneration,
			CommandsRegistered: &registered,
		},
	})
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	recorder := onboardingRequest(t, mux, http.MethodGet, "/api/telegram/readiness")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		State string `json:"state"`
		Ready bool   `json:"ready"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.State != string(readinessReady) || !body.Ready {
		t.Fatalf("body = %+v, want ready", body)
	}
	// Nothing identifying belongs in a readiness answer.
	for _, forbidden := range []string{"424242", "123456789", "token"} {
		if bytesContainsFold(recorder.Body.Bytes(), forbidden) {
			t.Fatalf("readiness leaked %q: %s", forbidden, recorder.Body.String())
		}
	}
}

func splitHostPortForTest(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	trimmed := rawURL
	for _, prefix := range []string{"http://", "https://"} {
		if len(trimmed) > len(prefix) && trimmed[:len(prefix)] == prefix {
			trimmed = trimmed[len(prefix):]
		}
	}
	for i := len(trimmed) - 1; i >= 0; i-- {
		if trimmed[i] == ':' {
			port, err := strconv.Atoi(trimmed[i+1:])
			if err != nil {
				t.Fatalf("port in %q: %v", rawURL, err)
			}
			return trimmed[:i], port
		}
	}
	t.Fatalf("no port in %q", rawURL)
	return "", 0
}

func bytesContainsFold(haystack []byte, needle string) bool {
	return len(needle) > 0 && indexFold(string(haystack), needle) >= 0
}

func indexFold(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if equalFoldASCII(haystack[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 'a' - 'A'
		}
		if 'A' <= y && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

var _ = config.ChannelTelegram
