// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"regexp"
	"strings"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// secretFieldPattern names payload fields whose value must never be persisted,
// whatever it happens to contain.
var secretFieldPattern = regexp.MustCompile(
	`(?i)(token|secret|password|passwd|credential|api[_-]?key|authorization|cookie|session)`,
)

// secretValuePattern catches credential material that reached a field which
// does not look secret by name.
var secretValuePattern = regexp.MustCompile(
	`(?i)(bearer\s+[A-Za-z0-9._~+/=-]{8,}|gh[pousr]_[A-Za-z0-9]{16,}|github_pat_[A-Za-z0-9_]{20,}|sk-[A-Za-z0-9_-]{16,}|AIza[A-Za-z0-9_-]{30,}|\b\d{6,12}:[A-Za-z0-9_-]{30,})`,
)

const providerRedactedMarker = "<redacted>"

// emitProviderEvent publishes a provider lifecycle event.
//
// Redaction happens here rather than at the call sites, so a new call site
// cannot forget it. The payloads are already built from safe fields only; this
// is the backstop that makes that a property of the system rather than of
// whoever writes the next emitter.
func (al *AgentLoop) emitProviderEvent(kind runtimeevents.Kind, meta HookMeta, payload any) {
	al.emitEvent(kind, meta, redactProviderPayload(payload))
}

// redactProviderPayload scrubs the string fields of a provider payload.
func redactProviderPayload(payload any) any {
	switch typed := payload.(type) {
	case ProviderAttemptPayload:
		typed.ModelConfigName = redactProviderText("model_config_name", typed.ModelConfigName)
		typed.Provider = redactProviderText("provider", typed.Provider)
		typed.UpstreamModel = redactProviderText("upstream_model", typed.UpstreamModel)
		typed.Protocol = redactProviderText("protocol", typed.Protocol)
		typed.CandidateKey = redactProviderText("candidate_key", typed.CandidateKey)
		typed.ErrorClass = redactProviderText("error_class", typed.ErrorClass)
		return typed
	case ProviderCooldownPayload:
		typed.Provider = redactProviderText("provider", typed.Provider)
		typed.UpstreamModel = redactProviderText("upstream_model", typed.UpstreamModel)
		typed.CandidateKey = redactProviderText("candidate_key", typed.CandidateKey)
		typed.ErrorClass = redactProviderText("error_class", typed.ErrorClass)
		return typed
	default:
		return payload
	}
}

func redactProviderText(field, value string) string {
	if value == "" {
		return value
	}
	if secretFieldPattern.MatchString(field) {
		return providerRedactedMarker
	}
	return secretValuePattern.ReplaceAllString(value, providerRedactedMarker)
}

// providerAttemptPayload builds a safe attempt payload from the turn and the
// candidate actually used.
//
// The configured name and the upstream model are carried separately on purpose.
// Routing aliases make them diverge — a config entry called "DeepSeek" may be
// served as "deepseek-v4-flash-free" over chat_completions — and a log that
// blurs the two turns failover diagnosis into guesswork.
func providerAttemptPayload(
	ts *turnState,
	candidate providers.FallbackCandidate,
	upstreamModel string,
	attempt, fallbackIndex int,
) ProviderAttemptPayload {
	configName := strings.TrimSpace(candidate.DisplayName)
	if configName == "" {
		configName = upstreamModel
	}
	payload := ProviderAttemptPayload{
		ModelConfigName: configName,
		Provider:        candidate.Provider,
		UpstreamModel:   upstreamModel,
		Protocol:        providerProtocolFor(candidate.Provider, upstreamModel),
		Attempt:         attempt,
		FallbackIndex:   fallbackIndex,
		CandidateKey:    candidate.StableKey(),
	}
	if ts != nil {
		payload.TurnID = ts.turnID
		payload.Iteration = ts.currentIteration()
	}
	return payload
}

// providerProtocolFor reports the request protocol where the provider exposes
// one.
//
// An empty result means "not applicable". A model the routing table does not
// recognise reports "unknown" rather than the protocol that will be attempted,
// because recording a guess as fact is exactly what makes a failover log
// untrustworthy.
func providerProtocolFor(provider, model string) string {
	if !providers.IsOpenCodeProvider(provider) {
		return ""
	}
	protocol, known := providers.ClassifyOpenCodeModel(model)
	if !known {
		return "unknown"
	}
	return string(protocol)
}
