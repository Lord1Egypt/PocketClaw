package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// The owner's explicit first-run requirement: a fresh install whose Telegram bot
// is correctly connected, with no AI model configured, must answer an owner's
// message with what to fix. Silence reads as a broken Telegram integration and
// sends the owner to re-pair a bot that was never at fault.

func configuredModel(name string, enabled bool) *config.ModelConfig {
	mc := &config.ModelConfig{
		ModelName: name, Provider: "openai", Model: "gpt-4o", Enabled: enabled,
	}
	mc.SetAPIKey("sk-test")
	return mc
}

func TestFirstRunWithNoModelReportsAnActionableSetupError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = nil
	cfg.Agents.Defaults.ModelName = ""

	problem := AIConfigurationProblem(cfg)
	if problem == nil {
		t.Fatal("a config with no model must not be allowed to attempt a turn")
	}
	if problem.Code != CodeNoModelConfigured {
		t.Errorf("code = %q, want %q", problem.Code, CodeNoModelConfigured)
	}

	message := problem.UserMessage()
	// It must say PocketClaw is connected: the point is to separate a working
	// channel from an incomplete AI configuration.
	if !strings.Contains(message, "connected") {
		t.Errorf("message must state that PocketClaw is connected, got %q", message)
	}
	// And it must say what to do.
	if !strings.Contains(strings.ToLower(message), "open pocketclaw") {
		t.Errorf("message must tell the user what to do, got %q", message)
	}
	if !strings.Contains(message, CodeNoModelConfigured) {
		t.Errorf("message must carry the stable code, got %q", message)
	}
}

// The error must never claim Telegram is broken.
func TestNoModelErrorDoesNotBlameTheChannel(t *testing.T) {
	for _, problem := range []*UserFacingError{
		newNoModelConfigured(), newNoModelEnabled(), newNoModelSelected(),
		newSelectedModelGone("gone"),
	} {
		lower := strings.ToLower(problem.UserMessage())
		for _, forbidden := range []string{
			"telegram is not", "telegram failed", "telegram error",
			"channel failed", "not connected",
		} {
			if strings.Contains(lower, forbidden) {
				t.Errorf("%s must not blame the channel: %q", problem.Code, problem.UserMessage())
			}
		}
	}
}

func TestEveryModelDisabledIsReportedSeparatelyFromNoneConfigured(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{configuredModel("m", false)}
	cfg.Agents.Defaults.ModelName = "m"

	problem := AIConfigurationProblem(cfg)
	if problem == nil || problem.Code != CodeNoModelEnabled {
		t.Fatalf("problem = %v, want %s", problem, CodeNoModelEnabled)
	}
}

func TestNoDefaultSelectedIsReported(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{configuredModel("m", true)}
	cfg.Agents.Defaults.ModelName = ""
	cfg.Agents.List = nil

	problem := AIConfigurationProblem(cfg)
	if problem == nil || problem.Code != CodeNoModelSelected {
		t.Fatalf("problem = %v, want %s", problem, CodeNoModelSelected)
	}
}

// A per-agent primary model is a selection, even with an empty default.
func TestAnAgentPrimaryModelCountsAsASelection(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{configuredModel("m", true)}
	cfg.Agents.Defaults.ModelName = ""
	cfg.Agents.List = []config.AgentConfig{{
		ID: "main", Model: &config.AgentModelConfig{Primary: "m"},
	}}

	if problem := AIConfigurationProblem(cfg); problem != nil {
		t.Fatalf("problem = %v, want none: an agent primary model is a selection", problem)
	}
}

// The stale-reference case: a name that survived in the configuration whose
// model_list entry did not.
func TestASelectedModelThatNoLongerExistsIsNamed(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{configuredModel("survivor", true)}
	cfg.Agents.Defaults.ModelName = "deleted-by-provider-removal"

	problem := AIConfigurationProblem(cfg)
	if problem == nil || problem.Code != CodeSelectedModelGone {
		t.Fatalf("problem = %v, want %s", problem, CodeSelectedModelGone)
	}
	if !strings.Contains(problem.UserMessage(), "deleted-by-provider-removal") {
		t.Errorf("the message must name the missing model, got %q", problem.UserMessage())
	}
}

func TestAFullyConfiguredSetupIsAllowedToRun(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{configuredModel("m", true)}
	cfg.Agents.Defaults.ModelName = "m"

	if problem := AIConfigurationProblem(cfg); problem != nil {
		t.Fatalf("problem = %v, want none", problem)
	}
}

// A model with no api_key must still be allowed to run. An ambient-credential
// provider (a local Ollama, an OAuth provider) legitimately has none, and
// deciding that here would be a second, drifting copy of
// web/backend/api's hasModelConfiguration. A real credential problem is reported
// at request time by the provider's own 401.
func TestAModelWithNoAPIKeyIsNotBlockedByThePrecondition(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "local", Provider: "ollama", Model: "llama3", Enabled: true,
	}}
	cfg.Agents.Defaults.ModelName = "local"

	if problem := AIConfigurationProblem(cfg); problem != nil {
		t.Fatalf("problem = %v, want none: credential usability is not this check's job", problem)
	}
}

// Virtual entries are multi-key expansions, not configuration the user made.
func TestANilConfigIsTreatedAsUnconfiguredRatherThanPanicking(t *testing.T) {
	problem := AIConfigurationProblem(nil)
	if problem == nil || problem.Code != CodeNoModelConfigured {
		t.Fatalf("problem = %v, want %s", problem, CodeNoModelConfigured)
	}
}

// The formatter must prefer the user-facing wording over its generic branch.
func TestFormatProcessingErrorRendersAUserFacingError(t *testing.T) {
	got := formatProcessingError(newNoModelConfigured())

	if strings.Contains(got, "Error processing message") {
		t.Fatalf("a user-facing error must not fall through to the generic branch: %q", got)
	}
	if !strings.Contains(got, CodeNoModelConfigured) {
		t.Fatalf("rendered message lost its code: %q", got)
	}
}

// Codes are a support and localisation handle: renaming one breaks both.
func TestUserFacingCodesAreStableAndDistinct(t *testing.T) {
	codes := map[string]bool{}
	for _, code := range []string{
		CodeNoModelConfigured, CodeNoModelSelected,
		CodeSelectedModelGone, CodeNoModelEnabled,
	} {
		if !strings.HasPrefix(code, "PC-E-") {
			t.Errorf("code %q must use the PC-E- prefix", code)
		}
		if codes[code] {
			t.Errorf("code %q is reused", code)
		}
		codes[code] = true
	}
	if len(codes) != 4 {
		t.Fatalf("expected 4 distinct codes, got %d", len(codes))
	}
}

// Nothing in a precondition message may carry a secret, a path or a stack trace.
func TestUserFacingMessagesCarryNoSensitiveShapes(t *testing.T) {
	for _, problem := range []*UserFacingError{
		newNoModelConfigured(), newNoModelEnabled(), newNoModelSelected(),
		newSelectedModelGone("m"),
	} {
		message := problem.UserMessage()
		for _, forbidden := range []string{
			"sk-", "Bearer ", "/home/", "/data/", "goroutine ", ".go:", "0x",
		} {
			if strings.Contains(message, forbidden) {
				t.Errorf("%s leaks %q: %s", problem.Code, forbidden, message)
			}
		}
	}
}

// The developer log keeps the cause; the user message never shows it.
func TestUserFacingErrorKeepsTheCauseOutOfTheUserMessage(t *testing.T) {
	wrapped := &UserFacingError{
		Code:    "PC-E-AI-001",
		Message: "Safe advice.",
		cause:   errSensitiveForTest,
	}

	if strings.Contains(wrapped.UserMessage(), "sk-live-leak") {
		t.Fatal("UserMessage must not render the wrapped cause")
	}
	if !strings.Contains(wrapped.Error(), "sk-live-leak") {
		t.Fatal("Error must keep the cause for the developer log")
	}
}

var errSensitiveForTest = errSensitive{}

type errSensitive struct{}

func (errSensitive) Error() string { return "upstream said sk-live-leak" }
