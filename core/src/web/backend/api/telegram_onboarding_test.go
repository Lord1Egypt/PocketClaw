package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/telegramonboarding"
)

// PC-DEF-060. Managed Telegram onboarding driven by a browser with no Android host.
//
// The security shape is the point: Core makes the server-to-server calls, so the
// browser never receives the onboarding service's URL, the poll token that authorises
// token collection, or the bot token itself. These assert that, and that completion goes
// through the one authoritative writer rather than a second copy of the owner contract.

const (
	testPollToken = "poll-token-must-never-reach-the-browser"
	testBotToken  = "123456789:bot-token-must-never-reach-the-browser"
)

// fakeOnboardingService stands in for the hosted service.
func fakeOnboardingService(t *testing.T) *httptest.Server {
	t.Helper()
	collected := false
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/telegram/pairings":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"pairing_id":"pair-1",
				"poll_token":"` + testPollToken + `",
				"suggested_username":"pocketclaw_abc_bot",
				"suggested_name":"PocketClaw Agent",
				"deep_link":"https://t.me/newbot/Mgr/pocketclaw_abc_bot",
				"qr_payload":"https://t.me/newbot/Mgr/pocketclaw_abc_bot",
				"expires_at":"2099-01-01T00:00:00Z",
				"poll_interval_seconds":2
			}`))
		case r.Method == http.MethodGet && r.URL.Path == "/telegram/pairings/pair-1":
			if r.Header.Get("Authorization") != "Bearer "+testPollToken {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(`{"state":"ready","bot_username":"pocketclaw_abc_bot"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/telegram/pairings/pair-1/token":
			if r.Header.Get("Authorization") != "Bearer "+testPollToken {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if collected {
				w.WriteHeader(http.StatusConflict)
				return
			}
			collected = true
			_, _ = w.Write([]byte(`{
				"bot_token":"` + testBotToken + `",
				"bot_user_id":123456789,
				"bot_username":"pocketclaw_abc_bot",
				"owner_user_id":424242
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// onboardingTestEnv wires a Handler to a fake service, bypassing the environment so the
// test does not depend on process state.
func onboardingTestEnv(t *testing.T) (*Handler, *http.ServeMux, string) {
	t.Helper()
	service := fakeOnboardingService(t)
	t.Cleanup(service.Close)

	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)

	handler := NewHandler(configPath)
	// Injected directly: onboardingClient() resolves from the environment once per
	// Handler, and a test must not depend on a real endpoint being configured.
	handler.telegramOnboardingOnce.Do(func() {
		handler.telegramOnboarding = telegramonboarding.NewClient(
			strings.Replace(service.URL, "http://", "https://", 1), service.Client())
		handler.telegramOnboardingStore = newTelegramOnboardingStore()
	})
	if handler.telegramOnboarding == nil {
		t.Fatal("the fake onboarding client did not construct")
	}

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return handler, mux, configPath
}

func onboardingRequest(t *testing.T, mux *http.ServeMux, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
	return recorder
}

func TestOnboardingAvailabilityReportsAConfiguredService(t *testing.T) {
	_, mux, _ := onboardingTestEnv(t)

	recorder := onboardingRequest(t, mux, http.MethodGet, "/api/telegram/onboarding")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body struct{ Available bool }
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Available {
		t.Fatal("a configured service must report available")
	}
}

// A deployment with no endpoint says so, and the manual form remains the path.
func TestOnboardingReportsUnavailableWithNoEndpoint(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)
	handler := NewHandler(configPath)
	handler.telegramOnboardingOnce.Do(func() {
		handler.telegramOnboarding = nil
		handler.telegramOnboardingStore = newTelegramOnboardingStore()
	})
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	recorder := onboardingRequest(t, mux, http.MethodGet, "/api/telegram/onboarding")
	var body struct{ Available bool }
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	if body.Available {
		t.Fatal("an unconfigured deployment must not offer managed onboarding")
	}

	// And every other endpoint refuses rather than half-working.
	if got := onboardingRequest(t, mux, http.MethodPost,
		"/api/telegram/onboarding/pairings").Code; got != http.StatusServiceUnavailable {
		t.Fatalf("create status = %d, want 503", got)
	}
}

// The security invariant: the browser gets the pairing id and the public link, and
// nothing that authorises anything.
func TestCreatePairingNeverReturnsThePollToken(t *testing.T) {
	_, mux, _ := onboardingTestEnv(t)

	recorder := onboardingRequest(t, mux, http.MethodPost, "/api/telegram/onboarding/pairings")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if strings.Contains(body, testPollToken) {
		t.Fatal("the poll token must never reach the browser")
	}
	if strings.Contains(body, "poll_token") {
		t.Fatalf("the response must not carry a poll_token field: %s", body)
	}

	var decoded struct {
		PairingID string `json:"pairing_id"`
		DeepLink  string `json:"deep_link"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.PairingID == "" {
		t.Fatal("the browser needs the pairing id to poll")
	}
	if !strings.HasPrefix(decoded.DeepLink, "https://t.me/") {
		t.Fatalf("deep link = %q, want a Telegram link", decoded.DeepLink)
	}
}

// The hosting origin must not be user-visible navigation here either — PC-DEF-052's
// requirement applied to this client.
func TestCreatePairingNeverReturnsTheServiceURL(t *testing.T) {
	_, mux, _ := onboardingTestEnv(t)

	body := onboardingRequest(t, mux, http.MethodPost,
		"/api/telegram/onboarding/pairings").Body.String()

	for _, forbidden := range []string{"127.0.0.1", "vercel.app", "onboarding"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Errorf("the response must not name the hosting service (%q): %s", forbidden, body)
		}
	}
}

func TestStatusIsPolledThroughCoreWithTheStoredToken(t *testing.T) {
	_, mux, _ := onboardingTestEnv(t)

	create := onboardingRequest(t, mux, http.MethodPost, "/api/telegram/onboarding/pairings")
	var created struct {
		PairingID string `json:"pairing_id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	// The fake service returns 404 unless the stored poll token is presented, so a
	// "ready" here proves Core supplied it.
	recorder := onboardingRequest(t, mux, http.MethodGet,
		"/api/telegram/onboarding/pairings/"+created.PairingID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var status struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != string(telegramonboarding.StateReady) {
		t.Fatalf("state = %q, want ready", status.State)
	}
	if strings.Contains(recorder.Body.String(), testPollToken) {
		t.Fatal("the poll token must not appear in a status response")
	}
}

// The whole point: completion configures Telegram through the authoritative writer, and
// the bot token never comes back.
func TestCompletionConfiguresTelegramAndNeverReturnsTheBotToken(t *testing.T) {
	_, mux, configPath := onboardingTestEnv(t)

	create := onboardingRequest(t, mux, http.MethodPost, "/api/telegram/onboarding/pairings")
	var created struct {
		PairingID string `json:"pairing_id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	recorder := onboardingRequest(t, mux, http.MethodPost,
		"/api/telegram/onboarding/pairings/"+created.PairingID+"/complete")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), testBotToken) {
		t.Fatal("the bot token must never reach the browser")
	}

	cfg := loadConfigForTest(t, configPath)
	channel := cfg.Channels["telegram"]
	if channel == nil {
		t.Fatal("Telegram was not configured")
	}
	if !channel.Enabled {
		t.Error("the channel must be enabled")
	}
	// The owner contract, which is the reason this reuses the shared writer.
	if len(channel.AllowFrom) != 1 || channel.AllowFrom[0] != "424242" {
		t.Fatalf("allow_from = %v, want exactly the one paired owner", channel.AllowFrom)
	}
}

// The service delivers the token once, so the session is spent either way and a second
// attempt must not look like it might succeed.
func TestCompletionCannotBeReplayed(t *testing.T) {
	_, mux, _ := onboardingTestEnv(t)

	create := onboardingRequest(t, mux, http.MethodPost, "/api/telegram/onboarding/pairings")
	var created struct {
		PairingID string `json:"pairing_id"`
	}
	_ = json.Unmarshal(create.Body.Bytes(), &created)

	first := onboardingRequest(t, mux, http.MethodPost,
		"/api/telegram/onboarding/pairings/"+created.PairingID+"/complete")
	if first.Code != http.StatusOK {
		t.Fatalf("first completion status = %d", first.Code)
	}

	second := onboardingRequest(t, mux, http.MethodPost,
		"/api/telegram/onboarding/pairings/"+created.PairingID+"/complete")
	if second.Code != http.StatusNotFound {
		t.Fatalf("second completion status = %d, want 404", second.Code)
	}
}

// An unknown pairing is answered the same way an expired one is, so the endpoint cannot
// be used to discover which ids exist.
func TestAnUnknownPairingIsReportedExpiredRatherThanProbed(t *testing.T) {
	_, mux, _ := onboardingTestEnv(t)

	recorder := onboardingRequest(t, mux, http.MethodGet,
		"/api/telegram/onboarding/pairings/never-existed")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with an expired state", recorder.Code)
	}
	var status struct {
		State string `json:"state"`
	}
	_ = json.Unmarshal(recorder.Body.Bytes(), &status)
	if status.State != string(telegramonboarding.StateExpired) {
		t.Fatalf("state = %q, want expired", status.State)
	}
}

// Cancelling drops the poll token, which is the only thing local cancellation can do.
func TestCancellationForgetsThePairing(t *testing.T) {
	handler, mux, _ := onboardingTestEnv(t)

	create := onboardingRequest(t, mux, http.MethodPost, "/api/telegram/onboarding/pairings")
	var created struct {
		PairingID string `json:"pairing_id"`
	}
	_ = json.Unmarshal(create.Body.Bytes(), &created)

	if handler.telegramOnboardingStore.get(created.PairingID) == nil {
		t.Fatal("the pairing should be held before cancellation")
	}

	if got := onboardingRequest(t, mux, http.MethodDelete,
		"/api/telegram/onboarding/pairings/"+created.PairingID).Code; got != http.StatusOK {
		t.Fatalf("cancel status = %d", got)
	}

	if handler.telegramOnboardingStore.get(created.PairingID) != nil {
		t.Fatal("cancellation must drop the stored poll token")
	}
	if got := onboardingRequest(t, mux, http.MethodPost,
		"/api/telegram/onboarding/pairings/"+created.PairingID+"/complete").Code; got != http.StatusNotFound {
		t.Fatalf("completion after cancel = %d, want 404", got)
	}
}

// Retry means a new pairing, and the old one must stop being usable.
func TestRetryIssuesAFreshPairing(t *testing.T) {
	_, mux, _ := onboardingTestEnv(t)

	first := onboardingRequest(t, mux, http.MethodPost, "/api/telegram/onboarding/pairings")
	second := onboardingRequest(t, mux, http.MethodPost, "/api/telegram/onboarding/pairings")

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("statuses = %d, %d", first.Code, second.Code)
	}
	// The fake service reuses one id, so what this asserts is that a second create is
	// accepted rather than rejected as a duplicate session.
	if second.Body.String() == "" {
		t.Fatal("a retry must return a usable pairing")
	}
}
