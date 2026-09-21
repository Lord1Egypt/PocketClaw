package gateway

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
)

// The owner's explicit first-run requirement.
//
// Fresh install, Telegram configured and connected before any AI model. When the
// owner sends a message the bot must answer with what to fix, and must not read
// as a broken Telegram integration.
//
// This is the choke point for it: every request on a gateway that started in
// limited mode fails here, and that failure is the whole of the product's reply.

func limitedModeConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.ModelList = nil
	cfg.Agents.Defaults.ModelName = ""
	return cfg
}

func TestLimitedModeStartupProducesAUserFacingProblem(t *testing.T) {
	provider, modelID, err := createStartupProvider(limitedModeConfig(), true)
	if err != nil {
		t.Fatalf("createStartupProvider() error = %v", err)
	}
	if modelID != "" {
		t.Errorf("modelID = %q, want empty in limited mode", modelID)
	}

	blocked, ok := provider.(*startupBlockedProvider)
	if !ok {
		t.Fatalf("provider type = %T, want *startupBlockedProvider", provider)
	}
	if blocked.problem == nil {
		t.Fatal("a blocked provider must carry the reason it is blocked")
	}
	if blocked.problem.Code != agent.CodeNoModelConfigured {
		t.Errorf("code = %q, want %q", blocked.problem.Code, agent.CodeNoModelConfigured)
	}
}

// What the user actually receives.
func TestLimitedModeChatFailsWithActionableUserFacingText(t *testing.T) {
	provider, _, err := createStartupProvider(limitedModeConfig(), true)
	if err != nil {
		t.Fatalf("createStartupProvider() error = %v", err)
	}

	_, chatErr := provider.Chat(context.Background(), nil, nil, "", nil)
	if chatErr == nil {
		t.Fatal("a blocked provider must never answer a request")
	}

	problem, ok := agent.AsUserFacingError(chatErr)
	if !ok {
		t.Fatalf("Chat error must be a user-facing error so the chat window "+
			"renders advice instead of %q", chatErr)
	}

	message := problem.UserMessage()
	if !strings.Contains(message, "connected") {
		t.Errorf("must separate a working channel from an incomplete AI setup: %q", message)
	}
	if !strings.Contains(strings.ToLower(message), "open pocketclaw") {
		t.Errorf("must say what to do: %q", message)
	}
	// The old wording. It named an internal mode and told the user nothing.
	if strings.Contains(strings.ToLower(message), "limited mode") {
		t.Errorf("must not expose the internal mode name: %q", message)
	}
	if strings.Contains(message, "Error processing message") {
		t.Errorf("must not reach the generic internal branch: %q", message)
	}
}

// A configuration that has models but none selected needs a different action
// from one that has no models at all.
func TestLimitedModeDistinguishesNothingSelectedFromNothingConfigured(t *testing.T) {
	cfg := config.DefaultConfig()
	configured := &config.ModelConfig{
		ModelName: "m", Provider: "openai", Model: "gpt-4o", Enabled: true,
	}
	configured.SetAPIKey("sk-test")
	cfg.ModelList = []*config.ModelConfig{configured}
	cfg.Agents.Defaults.ModelName = ""
	cfg.Agents.List = nil

	provider, _, err := createStartupProvider(cfg, true)
	if err != nil {
		t.Fatalf("createStartupProvider() error = %v", err)
	}
	blocked := provider.(*startupBlockedProvider)
	if blocked.problem.Code != agent.CodeNoModelSelected {
		t.Fatalf("code = %q, want %q", blocked.problem.Code, agent.CodeNoModelSelected)
	}
}

// A blocked provider with no recorded reason must still refuse, not answer.
func TestABlockedProviderWithNoReasonStillRefuses(t *testing.T) {
	provider := &startupBlockedProvider{}

	if _, err := provider.Chat(context.Background(), nil, nil, "", nil); err == nil {
		t.Fatal("a blocked provider must fail closed")
	}
	if provider.GetDefaultModel() != "" {
		t.Error("a blocked provider has no default model")
	}
}

// Limited mode is opt-in. Without it a missing model is a startup failure, which
// is the correct behaviour for a CLI run that was told to expect a provider.
func TestWithoutAllowEmptyStartupAMissingModelIsAStartupError(t *testing.T) {
	if _, _, err := createStartupProvider(limitedModeConfig(), false); err == nil {
		t.Fatal("createStartupProvider() must fail when limited mode is not allowed")
	}
}
