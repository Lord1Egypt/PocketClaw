package agent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func formatProcessingError(err error) string {
	if err == nil {
		return ""
	}

	// A model chain that ran out of candidates has one line to say per
	// candidate. Its Error() concatenates every raw provider body, which in a
	// chat window is pages of JSON, billing URLs and provider internals for a
	// reader who needs to know which model failed and roughly why.
	var exhausted *providers.FallbackExhaustedError
	if errors.As(err, &exhausted) {
		return formatFallbackExhausted(exhausted)
	}

	if kind, ok := providers.ClassifyAuthError(err); ok {
		return fmt.Sprintf(
			"Error processing message: %s\n\nOriginal error:\n%s",
			authErrorFriendlyMessage(kind),
			err.Error(),
		)
	}

	return fmt.Sprintf("Error processing message: %v", err)
}

func authErrorFriendlyMessage(kind providers.AuthErrorKind) string {
	switch kind {
	case providers.AuthErrorInvalidAPIKey:
		return "Authentication failed: the API key appears to be invalid. Check the API key configured for this model or provider."
	case providers.AuthErrorMissingAPIKey:
		return "Authentication failed: no API key is configured for this model or provider. Add an API key in the model settings or config."
	case providers.AuthErrorExpiredToken:
		return "Authentication failed: the saved login or token appears to be expired. Re-authenticate the provider."
	default:
		return "Authentication failed: check the API key, token, OAuth login, or provider permissions for this model."
	}
}

// formatFallbackExhausted renders the candidate summary a user can act on.
// The full provider responses stay in the developer log, where the existing
// redaction rules apply to them.
func formatFallbackExhausted(err *providers.FallbackExhaustedError) string {
	var sb strings.Builder
	sb.WriteString("All configured models failed for this request.")
	for i, attempt := range err.Attempts {
		label := "Primary"
		if i > 0 {
			label = fmt.Sprintf("Fallback %d", i)
		}
		sb.WriteString(fmt.Sprintf("\n%s (%s): %s",
			label,
			candidateLabel(attempt),
			attemptFailureSummary(attempt),
		))
	}
	return sb.String()
}

// candidateLabel names the candidate without leaking a credential: a multi-key
// model_list entry can carry the key in its identity, so only the model id is
// shown.
func candidateLabel(attempt providers.FallbackAttempt) string {
	model := strings.TrimSpace(attempt.Model)
	if model == "" {
		model = "unknown model"
	}
	if provider := strings.TrimSpace(attempt.Provider); provider != "" {
		return provider + "/" + model
	}
	return model
}

func attemptFailureSummary(attempt providers.FallbackAttempt) string {
	if attempt.Skipped {
		return "skipped"
	}

	failErr := providers.ClassifyError(attempt.Error, attempt.Provider, attempt.Model)
	if failErr == nil {
		return "request failed"
	}

	summary := failoverReasonSummary(failErr.Reason)
	if failErr.Status > 0 {
		return fmt.Sprintf("%s (%d)", summary, failErr.Status)
	}
	return summary
}

func failoverReasonSummary(reason providers.FailoverReason) string {
	switch reason {
	case providers.FailoverAuth:
		return "authentication failed"
	case providers.FailoverRateLimit:
		return "rate limited"
	case providers.FailoverHardQuota:
		return "quota exhausted"
	case providers.FailoverBilling:
		return "billing problem"
	case providers.FailoverNetwork:
		return "network error"
	case providers.FailoverTimeout:
		return "timed out"
	case providers.FailoverFormat:
		return "request rejected"
	case providers.FailoverContextOverflow:
		return "context too long"
	case providers.FailoverOverloaded:
		return "provider overloaded"
	default:
		return "request failed"
	}
}
