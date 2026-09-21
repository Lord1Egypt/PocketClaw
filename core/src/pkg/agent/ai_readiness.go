package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// AIConfigurationProblem explains why no usable AI provider could be built from
// this configuration, or returns nil when one could.
//
// It is called where the gateway decides to start in limited mode, so the
// placeholder provider can fail every request with a sentence the user can act
// on. It is deliberately *not* a precondition inside the message path:
// NewAgentLoop takes an injected provider, so an empty model_list does not imply
// there is nothing to send a request to, and checking the config there would
// refuse turns for a loop that has a perfectly good provider.
//
// A fresh install with a working Telegram bot and no model is the case this
// exists for. Before it, the placeholder failed with "no default model
// configured; gateway started in limited mode", which reached the chat window as
// "Error processing message: ..." -- implementation jargon, no instruction, and
// no hint that Telegram itself was fine.
//
// Deliberately narrow. It answers only what is decidable from configuration
// alone:
//
//   - is there any model at all
//   - is any of them enabled
//   - is one selected
//   - does the selected one still exist
//
// It does **not** decide whether a model's credential is usable. That question
// needs the OAuth credential store and the local-endpoint probe that
// web/backend/api's hasModelConfiguration owns, and a second implementation of
// it here is exactly the drift that had pkg/modelaccess reverted: an
// ambient-credential provider such as a local Ollama or an OAuth provider
// legitimately has no api_key, and a copy of that logic would start reporting
// those as unconfigured the moment the two fell out of step. A missing or
// rejected credential is already reported accurately at request time, by the
// provider's own 401 through AuthErrorMissingAPIKey.
func AIConfigurationProblem(cfg *config.Config) *UserFacingError {
	if cfg == nil {
		return newNoModelConfigured()
	}

	usable := 0
	enabled := 0
	for _, mc := range cfg.ModelList {
		if mc == nil || mc.IsVirtual() {
			continue
		}
		usable++
		if mc.Enabled {
			enabled++
		}
	}
	if usable == 0 {
		return newNoModelConfigured()
	}
	if enabled == 0 {
		return newNoModelEnabled()
	}

	// A per-agent primary model overrides the default, so a config that names
	// one is selected even with an empty agents.defaults.model_name.
	selected := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	if selected == "" {
		if agentModel := firstAgentPrimaryModel(cfg); agentModel != "" {
			selected = agentModel
		}
	}
	if selected == "" {
		return newNoModelSelected()
	}

	if _, err := cfg.GetModelConfig(selected); err != nil {
		// The name survived in the configuration but its model_list entry did
		// not -- a provider or model deleted without its references being
		// reconciled. Naming the model is what makes this fixable.
		return newSelectedModelGone(selected)
	}

	return nil
}

func firstAgentPrimaryModel(cfg *config.Config) string {
	for _, agent := range cfg.Agents.List {
		if agent.Model == nil {
			continue
		}
		if primary := strings.TrimSpace(agent.Model.Primary); primary != "" {
			return primary
		}
	}
	return ""
}
