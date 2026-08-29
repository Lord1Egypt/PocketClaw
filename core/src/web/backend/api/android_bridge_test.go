package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

type fakeNetworkModeController struct {
	public bool
	err    error
}

func (c *fakeNetworkModeController) ApplyPublicMode(public bool) error {
	if c.err != nil {
		return c.err
	}
	c.public = public
	return nil
}

func (c *fakeNetworkModeController) PublicMode() bool { return c.public }

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

func gatewayBridgeRequest(method, token string) *http.Request {
	req := httptest.NewRequest(method, androidGatewayBridgePath, nil)
	req.RemoteAddr = "127.0.0.1:48123"
	req.Header.Set("X-PocketClaw-Android-Bridge", token)
	return req
}

func TestAndroidGatewayBridgeExposesStatusOnlyToLoopbackCredential(t *testing.T) {
	resetGatewayTestState(t)
	gateway.mu.Lock()
	gateway.operationID = "gateway-physical-sync-1"
	gateway.mu.Unlock()
	handler := NewHandler(writeAndroidBridgeTestConfig(t, ""))
	mux := http.NewServeMux()
	handler.RegisterAndroidBridgeRoutes(mux, testAndroidBridgeToken)

	authorized := httptest.NewRecorder()
	mux.ServeHTTP(authorized, gatewayBridgeRequest(http.MethodGet, testAndroidBridgeToken))
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized status = %d, body=%s", authorized.Code, authorized.Body.String())
	}
	var status map[string]any
	if err := json.NewDecoder(authorized.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["gateway_status"] != "stopped" {
		t.Fatalf("gateway_status = %#v, want stopped", status["gateway_status"])
	}
	if status["operation_id"] != "gateway-physical-sync-1" {
		t.Fatalf("operation_id = %#v, want gateway-physical-sync-1", status["operation_id"])
	}

	unauthorized := httptest.NewRecorder()
	mux.ServeHTTP(unauthorized, gatewayBridgeRequest(http.MethodGet, "wrong-token"))
	if unauthorized.Code != http.StatusNotFound {
		t.Fatalf("unauthorized status = %d, want 404", unauthorized.Code)
	}

	nonLoopback := gatewayBridgeRequest(http.MethodGet, testAndroidBridgeToken)
	nonLoopback.RemoteAddr = "192.0.2.10:48123"
	nonLoopbackRecorder := httptest.NewRecorder()
	mux.ServeHTTP(nonLoopbackRecorder, nonLoopback)
	if nonLoopbackRecorder.Code != http.StatusNotFound {
		t.Fatalf("non-loopback status = %d, want 404", nonLoopbackRecorder.Code)
	}
}
