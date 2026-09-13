package api

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// PC-DEF-038. The gateway is infrastructure: it must start with zero
// providers and zero models, because provider setup is reached THROUGH the
// running dashboard. Every case here failed before the readiness split --
// a fresh install logged "Skip auto-starting gateway: no default model
// configured" and manual start returned precondition_failed, so the only
// route to configuring a first provider was gated on already having one.
//
// The matrix is the owner-specified A-H. Cases that need a live process or a
// device are marked where they are proven at a different layer.

// freshInstall is the state the defect was reported in: no config file at all,
// therefore no providers and no models.
func freshInstall(t *testing.T) *Handler {
	t.Helper()
	return NewHandler(filepath.Join(t.TempDir(), "config.json"))
}

// configuredWithoutCredentials names a default model whose provider has no
// key, the shape of case G. Built through config.DefaultConfig so the fixture
// cannot drift away from the real schema.
func configuredWithoutCredentials(t *testing.T) *Handler {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].SetAPIKey("")
	cfg.ModelList[0].AuthMethod = ""
	if err := config.SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	return NewHandler(path)
}

// A. 0 providers, 0 models -> the gateway may start.
func TestA_GatewayStartsWithNoProvidersAndNoModels(t *testing.T) {
	h := freshInstall(t)

	ready, reason, err := h.gatewayInfrastructureReady()
	if err != nil {
		t.Fatalf("gatewayInfrastructureReady() error = %v", err)
	}
	if !ready {
		t.Fatalf("gatewayInfrastructureReady() = false (%q), want true: the gateway "+
			"must not require a provider or a model to start", reason)
	}
}

// B. The auto-start path uses infrastructure readiness, so a restart with an
// empty registry still brings the gateway up. Proven here at the decision
// layer that TryAutoStartGateway consults; the process lifecycle itself is
// device-verified.
func TestB_AutoStartDecisionIgnoresModelState(t *testing.T) {
	h := freshInstall(t)

	ready, _, err := h.gatewayInfrastructureReady()
	if err != nil || !ready {
		t.Fatalf("auto-start precondition = (%v, %v), want ready with no error", ready, err)
	}
}

// D and F. A chat request with no provider must be refused with an actionable
// reason -- and that refusal must be a property of chat, never of the gateway.
func TestDF_ChatIsRefusedButTheGatewayStaysStartable(t *testing.T) {
	h := freshInstall(t)

	chatOK, chatReason, err := h.chatReady()
	if err != nil {
		t.Fatalf("chatReady() error = %v", err)
	}
	if chatOK {
		t.Fatal("chatReady() = true with no model configured, want false")
	}
	if chatReason == "" {
		t.Fatal("chatReady() gave no reason; the user needs an actionable message")
	}

	infraOK, infraReason, err := h.gatewayInfrastructureReady()
	if err != nil {
		t.Fatalf("gatewayInfrastructureReady() error = %v", err)
	}
	if !infraOK {
		t.Fatalf("the gateway became unstartable because chat is unavailable (%q); "+
			"infrastructure health must not be derived from AI availability", infraReason)
	}
}

// G. Invalid or absent credentials are a provider failure. Infrastructure must
// not be marked dead for it.
func TestG_MissingCredentialsDoNotStopTheGateway(t *testing.T) {
	h := configuredWithoutCredentials(t)

	chatOK, chatReason, err := h.chatReady()
	if err != nil {
		t.Fatalf("chatReady() error = %v", err)
	}
	if chatOK {
		t.Fatal("chatReady() = true without credentials, want false")
	}
	if !strings.Contains(chatReason, "credentials") {
		t.Fatalf("chatReady() reason = %q, want it to name the credential problem", chatReason)
	}

	infraOK, _, err := h.gatewayInfrastructureReady()
	if err != nil {
		t.Fatalf("gatewayInfrastructureReady() error = %v", err)
	}
	if !infraOK {
		t.Fatal("a credential problem made the gateway unstartable; " +
			"provider health and gateway health must stay independent")
	}
}

// The status contract the UI renders: "Gateway: Running / AI provider: Not
// configured" has to be representable, so the two facts ship as two fields.
func TestStatusReportsGatewayAndChatReadinessIndependently(t *testing.T) {
	h := freshInstall(t)
	data := h.gatewayStatusData()

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}

	allowed, ok := decoded["gateway_start_allowed"].(bool)
	if !ok {
		t.Fatal("status has no gateway_start_allowed field")
	}
	if !allowed {
		t.Fatalf("gateway_start_allowed = false on a fresh install, reason = %v",
			decoded["gateway_start_reason"])
	}

	chatReady, ok := decoded["chat_ready"].(bool)
	if !ok {
		t.Fatal("status has no chat_ready field; the UI cannot distinguish " +
			"a running gateway from a configured provider without it")
	}
	if chatReady {
		t.Fatal("chat_ready = true with no model configured")
	}
	if decoded["chat_not_ready_reason"] == "" || decoded["chat_not_ready_reason"] == nil {
		t.Fatal("chat_not_ready_reason is absent; the user gets no actionable message")
	}
}
