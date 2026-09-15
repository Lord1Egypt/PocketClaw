package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

type fakeNetworkModeController struct {
	public         bool
	desiredPublic  bool
	err            error
	reconcileCalls int
}

func (c *fakeNetworkModeController) ApplyPublicMode(public bool) error {
	if c.err != nil {
		return c.err
	}
	c.public = public
	return nil
}

func (c *fakeNetworkModeController) PublicMode() bool { return c.public }

// PC-DEF-040. Mirrors the real runtime: a no-op unless the user asked for LAN and
// the listener is not already there.
func (c *fakeNetworkModeController) ReconcileAfterDashboardClaimed() {
	c.reconcileCalls++
	if c.err != nil {
		return
	}
	if !c.desiredPublic || c.public {
		return
	}
	c.public = true
}

const testAndroidBridgeToken = "process-local-test-bridge-token"

func writeAndroidBridgeTestConfig(t *testing.T, token string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Gateway.Port = 19999
	telegram := cfg.Channels.Get(config.ChannelTelegram)
	telegram.Enabled = token != ""
	decoded, err := telegram.GetDecoded()
	if err != nil {
		t.Fatal(err)
	}
	decoded.(*config.TelegramSettings).Token.Set(token)
	if err := config.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	return path
}

func telegramBridgeRequest(body string, token string) *http.Request {
	req := httptest.NewRequest(http.MethodPut, androidTelegramBridgePath, bytes.NewBufferString(body))
	req.RemoteAddr = "127.0.0.1:48123"
	req.Header.Set("X-PocketClaw-Android-Bridge", token)
	return req
}

func readTelegramBridgeConfig(t *testing.T, path string) (*config.Config, *config.Channel, *config.TelegramSettings) {
	t.Helper()
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	channel := cfg.Channels.Get(config.ChannelTelegram)
	decoded, err := channel.GetDecoded()
	if err != nil {
		t.Fatal(err)
	}
	return cfg, channel, decoded.(*config.TelegramSettings)
}

func TestAndroidTelegramBridgeWritesThroughAuthoritativeConfig(t *testing.T) {
	path := writeAndroidBridgeTestConfig(t, "")
	mux := http.NewServeMux()
	NewHandler(path).RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, telegramBridgeRequest(
		`{"token":"new-child-token","owner_user_id":24680}`,
		testAndroidBridgeToken,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	cfg, channel, settings := readTelegramBridgeConfig(t, path)
	if !channel.Enabled || channel.Type != config.ChannelTelegram {
		t.Fatalf("telegram channel = %#v, want enabled Telegram", channel)
	}
	if settings.Token.String() != "new-child-token" {
		t.Fatal("Telegram credential was not persisted through SaveConfig")
	}
	if len(channel.AllowFrom) != 1 || channel.AllowFrom[0] != "24680" {
		t.Fatalf("owner allow-list was not persisted: %#v", channel.AllowFrom)
	}
	if cfg.Gateway.Port != 19999 {
		t.Fatalf("unrelated config was changed: gateway port = %d", cfg.Gateway.Port)
	}
}

func TestAndroidTelegramBridgeFailurePreservesWorkingConfiguration(t *testing.T) {
	path := writeAndroidBridgeTestConfig(t, "existing-child-token")
	mux := http.NewServeMux()
	NewHandler(path).RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

	for name, req := range map[string]*http.Request{
		"wrong credential": telegramBridgeRequest(
			`{"token":"replacement-token","owner_user_id":123}`,
			"wrong-bridge-token",
		),
		"invalid payload": telegramBridgeRequest(
			`{"token":"","owner_user_id":123}`,
			testAndroidBridgeToken,
		),
		"missing owner": telegramBridgeRequest(
			`{"token":"replacement-token"}`,
			testAndroidBridgeToken,
		),
		"invalid owner": telegramBridgeRequest(
			`{"token":"replacement-token","owner_user_id":0}`,
			testAndroidBridgeToken,
		),
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code < 400 {
				t.Fatalf("status = %d, want rejection", rec.Code)
			}
			_, channel, settings := readTelegramBridgeConfig(t, path)
			if !channel.Enabled || settings.Token.String() != "existing-child-token" {
				t.Fatal("rejected replacement changed the existing working bot")
			}
		})
	}
}

func TestAndroidTelegramBridgeRejectsNonLoopbackCaller(t *testing.T) {
	path := writeAndroidBridgeTestConfig(t, "existing-child-token")
	mux := http.NewServeMux()
	NewHandler(path).RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)
	req := telegramBridgeRequest(`{"token":"replacement-token","owner_user_id":123}`, testAndroidBridgeToken)
	req.RemoteAddr = "192.0.2.10:48123"

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want hidden route", rec.Code)
	}
	_, _, settings := readTelegramBridgeConfig(t, path)
	if settings.Token.String() != "existing-child-token" {
		t.Fatal("non-loopback request changed Telegram credentials")
	}
}

func networkModeBridgeRequest(method, body, token string) *http.Request {
	req := httptest.NewRequest(method, androidNetworkModeBridgePath, bytes.NewBufferString(body))
	req.RemoteAddr = "127.0.0.1:48123"
	req.Header.Set("X-PocketClaw-Android-Bridge", token)
	return req
}

func readNetworkModeState(t *testing.T, mux *http.ServeMux) androidNetworkModeResponse {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, networkModeBridgeRequest(http.MethodGet, "", testAndroidBridgeToken))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status endpoint = %d", recorder.Code)
		}
		var state androidNetworkModeResponse
		if err := json.NewDecoder(recorder.Body).Decode(&state); err != nil {
			t.Fatal(err)
		}
		if state.Status != "applying" {
			return state
		}
		if time.Now().After(deadline) {
			t.Fatal("network mode change did not complete")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAndroidNetworkModeBridgeAppliesThroughLoopbackCredential(t *testing.T) {
	controller := &fakeNetworkModeController{}
	handler := NewHandler(writeAndroidBridgeTestConfig(t, ""))
	handler.SetLauncherNetworkModeController(controller)
	mux := http.NewServeMux()
	handler.RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, networkModeBridgeRequest(
		http.MethodPut,
		`{"public":true}`,
		testAndroidBridgeToken,
	))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("apply status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	state := readNetworkModeState(t, mux)
	if state.Status != "succeeded" || !state.Public || !controller.public {
		t.Fatalf("state = %#v, controller public=%t", state, controller.public)
	}
}

func TestAndroidNetworkModeBridgeReportsRollbackState(t *testing.T) {
	controller := &fakeNetworkModeController{err: errors.New("injected bind failure")}
	handler := NewHandler(writeAndroidBridgeTestConfig(t, ""))
	handler.SetLauncherNetworkModeController(controller)
	mux := http.NewServeMux()
	handler.RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, networkModeBridgeRequest(
		http.MethodPut,
		`{"public":true}`,
		testAndroidBridgeToken,
	))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("apply status = %d", recorder.Code)
	}
	state := readNetworkModeState(t, mux)
	if state.Status != "failed" || state.Public || state.Error == "" {
		t.Fatalf("rollback state = %#v", state)
	}
}

func TestAndroidNetworkModeBridgeHidesRouteFromUnauthorizedCaller(t *testing.T) {
	handler := NewHandler(writeAndroidBridgeTestConfig(t, ""))
	handler.SetLauncherNetworkModeController(&fakeNetworkModeController{})
	mux := http.NewServeMux()
	handler.RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, networkModeBridgeRequest(http.MethodPut, `{"public":true}`, "wrong-token"))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unauthorized status = %d, want 404", recorder.Code)
	}
}

func contextMemoryBridgeRequest(method, body, token string) *http.Request {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, androidContextMemoryBridgePath, nil)
	} else {
		req = httptest.NewRequest(
			method, androidContextMemoryBridgePath, bytes.NewBufferString(body),
		)
	}
	req.RemoteAddr = "127.0.0.1:48123"
	req.Header.Set("X-PocketClaw-Android-Bridge", token)
	return req
}

func decodeContextMemory(t *testing.T, rec *httptest.ResponseRecorder) androidContextMemoryResponse {
	t.Helper()
	var body androidContextMemoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not JSON: %v (%s)", err, rec.Body.String())
	}
	return body
}

// A config that never named the setting must read as the shipped default, which
// is what the agent applies too.
func TestContextMemoryBridgeReportsTheDefaultWhenUnset(t *testing.T) {
	path := writeAndroidBridgeTestConfig(t, "existing-token")
	mux := http.NewServeMux()
	NewHandler(path).RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, contextMemoryBridgeRequest(http.MethodGet, "", testAndroidBridgeToken))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	body := decodeContextMemory(t, rec)
	if body.RecentMessages != config.DefaultTelegramRecentContextMessages {
		t.Fatalf("recent_messages = %d, want the default %d",
			body.RecentMessages, config.DefaultTelegramRecentContextMessages)
	}
	if body.Min != minTelegramRecentContextMessages || body.Max != maxTelegramRecentContextMessages {
		t.Fatalf("range = %d..%d, want %d..%d",
			body.Min, body.Max, minTelegramRecentContextMessages, maxTelegramRecentContextMessages)
	}
}

// Each preset and a custom value round-trip exactly.
func TestContextMemoryBridgeRoundTripsEveryAcceptedValue(t *testing.T) {
	for _, want := range []int{10, 15, 20, 25, 17, 5, 50} {
		path := writeAndroidBridgeTestConfig(t, "existing-token")
		mux := http.NewServeMux()
		NewHandler(path).RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, contextMemoryBridgeRequest(
			http.MethodPut,
			`{"recent_messages":`+strconv.Itoa(want)+`}`,
			testAndroidBridgeToken,
		))
		if rec.Code != http.StatusOK {
			t.Fatalf("%d: status = %d, body = %s", want, rec.Code, rec.Body.String())
		}
		if got := decodeContextMemory(t, rec).RecentMessages; got != want {
			t.Fatalf("response said %d, want %d", got, want)
		}

		cfg, err := config.LoadConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.Agents.Defaults.TelegramRecentContextMessages; got != want {
			t.Fatalf("stored %d, want %d", got, want)
		}

		readBack := httptest.NewRecorder()
		mux.ServeHTTP(readBack, contextMemoryBridgeRequest(http.MethodGet, "", testAndroidBridgeToken))
		if got := decodeContextMemory(t, readBack).RecentMessages; got != want {
			t.Fatalf("read back %d, want %d", got, want)
		}
	}
}

// Core is the authority on the range, so an out-of-range or malformed value is
// rejected and the stored configuration is left exactly as it was.
func TestContextMemoryBridgeRejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"zero", `{"recent_messages":0}`},
		{"negative", `{"recent_messages":-5}`},
		{"below minimum", `{"recent_messages":4}`},
		{"above maximum", `{"recent_messages":51}`},
		{"absurd", `{"recent_messages":100000}`},
		{"non numeric", `{"recent_messages":"twenty"}`},
		{"unknown field", `{"recent":20}`},
		{"malformed", `{`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeAndroidBridgeTestConfig(t, "existing-token")
			mux := http.NewServeMux()
			NewHandler(path).RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

			// Establish a known good value first.
			seed := httptest.NewRecorder()
			mux.ServeHTTP(seed, contextMemoryBridgeRequest(
				http.MethodPut, `{"recent_messages":20}`, testAndroidBridgeToken,
			))
			if seed.Code != http.StatusOK {
				t.Fatalf("seed failed: %s", seed.Body.String())
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, contextMemoryBridgeRequest(
				http.MethodPut, tc.body, testAndroidBridgeToken,
			))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 for %s", rec.Code, tc.body)
			}

			cfg, err := config.LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.Agents.Defaults.TelegramRecentContextMessages; got != 20 {
				t.Fatalf("a rejected write changed the stored value to %d", got)
			}
		})
	}
}

// Changing this number must change only this number.
func TestContextMemoryBridgeLeavesEverythingElseAlone(t *testing.T) {
	path := writeAndroidBridgeTestConfig(t, "existing-token")
	before, beforeChannel, beforeSettings := readTelegramBridgeConfig(t, path)

	mux := http.NewServeMux()
	NewHandler(path).RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, contextMemoryBridgeRequest(
		http.MethodPut, `{"recent_messages":25}`, testAndroidBridgeToken,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	after, afterChannel, afterSettings := readTelegramBridgeConfig(t, path)
	if afterSettings.Token.String() != beforeSettings.Token.String() {
		t.Fatal("the Telegram credential changed")
	}
	if afterChannel.Enabled != beforeChannel.Enabled {
		t.Fatal("the Telegram channel enabled state changed")
	}
	if after.Gateway.Port != before.Gateway.Port {
		t.Fatalf("gateway port changed from %d to %d", before.Gateway.Port, after.Gateway.Port)
	}
	if after.Agents.Defaults.SummarizeMessageThreshold != before.Agents.Defaults.SummarizeMessageThreshold {
		t.Fatal("the summarization threshold changed")
	}
	if after.Agents.Defaults.MaxToolIterations != before.Agents.Defaults.MaxToolIterations {
		t.Fatal("the tool iteration limit changed")
	}
	if len(after.Channels) != len(before.Channels) {
		t.Fatalf("channel count changed from %d to %d", len(before.Channels), len(after.Channels))
	}
}

// The route is invisible without the loopback credential, like its neighbours.
func TestContextMemoryBridgeHidesRouteFromUnauthorizedCaller(t *testing.T) {
	path := writeAndroidBridgeTestConfig(t, "existing-token")
	mux := http.NewServeMux()
	NewHandler(path).RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		rec := httptest.NewRecorder()
		body := ""
		if method == http.MethodPut {
			body = `{"recent_messages":20}`
		}
		mux.ServeHTTP(rec, contextMemoryBridgeRequest(method, body, "wrong-token"))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", method, rec.Code)
		}
	}
}

// A stored value outside the accepted range reads as the default rather than
// being reported back as if Core would honour it.
func TestContextMemoryBridgeNormalizesAnOutOfRangeStoredValue(t *testing.T) {
	path := writeAndroidBridgeTestConfig(t, "existing-token")
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Agents.Defaults.TelegramRecentContextMessages = 9999
	if err := config.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	NewHandler(path).RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, contextMemoryBridgeRequest(http.MethodGet, "", testAndroidBridgeToken))

	if got := decodeContextMemory(t, rec).RecentMessages; got != config.DefaultTelegramRecentContextMessages {
		t.Fatalf("reported %d for an out-of-range stored value, want the default %d",
			got, config.DefaultTelegramRecentContextMessages)
	}
}
