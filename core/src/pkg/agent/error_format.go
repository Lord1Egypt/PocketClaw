package agent

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// formatProcessingError renders a failed turn for the chat window.
//
// A provider's response *body* never reaches the user. It is written for an API
// client, not a person: raw JSON, billing links, request ids and account
// internals, none of which help someone decide what to do next, and some of
// which should not be repeated into a chat at all. Everything that is not the
// provider's own words still passes through — an unsupported-media explanation
// or a configuration error is guidance the user needs verbatim.
//
// PC-DEF-032 carved out one exception: the provider's own message field, taken
// alone, redacted and capped. "Model deepseek-v4.1-flash is not supported" is
// the difference between a user who can fix their configuration and one staring
// at a status code. See providerErrorDetail.
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

	// A single configured model fails through this path instead, and it used to
	// append the provider's whole response under an "Original error:" heading.
	if summary, ok := formatProviderFailure(err); ok {
		return summary
	}

	return fmt.Sprintf("Error processing message: %v", err)
}

// formatProviderFailure summarises a failure that is the provider's, reporting
// false for anything else.
//
// The distinction is what keeps this safe without making it useless: an error
// PocketClaw itself raised carries advice worth reading, while an error the
// provider returned carries a response body worth suppressing.
func formatProviderFailure(err error) (string, bool) {
	authKind, isAuth := providers.ClassifyAuthError(err)
	failErr := providers.ClassifyError(err, "", "")
	if !isAuth && failErr == nil {
		return "", false
	}

	if isAuth || (failErr != nil && failErr.Reason == providers.FailoverAuth) {
		return authErrorFriendlyMessage(resolveAuthErrorKind(authKind, isAuth, err)), true
	}

	summary := failureSummary(failErr.Reason, failErr.Status)
	sentence := fmt.Sprintf("The model could not complete this request: %s.", summary)
	if failErr.Status > 0 {
		sentence = fmt.Sprintf("The model could not complete this request: %s (%d).", summary, failErr.Status)
	}
	if detail := providerErrorDetail(err); detail != "" {
		// Attributed, so the user can tell the provider's words from ours and
		// knows which system to go and change.
		sentence += fmt.Sprintf(" The provider said: %s", detail)
	}
	return sentence, true
}

// accountBalancePattern recognises a rejection that is about money rather than
// credentials. Providers commonly return both as 401.
var accountBalancePattern = regexp.MustCompile(
	`(?i)\b(?:insufficient|inadequate|negative|zero|no|out\s+of|low)\s+(?:account\s+)?` +
		`(?:balance|credit|credits|funds)\b|\bcredits?\s*error\b|\bcredit\s+balance\b|` +
		`\bbalance\s+(?:is\s+)?(?:too\s+)?low\b|\btop[-\s]?up\b`,
)

// resolveAuthErrorKind picks the advice to give for an authentication failure.
//
// A 401 whose body says the account ran out of credit is not a bad key. Telling
// the user their API key is invalid sends them to replace a key that works,
// which is worse than saying nothing specific at all.
func resolveAuthErrorKind(
	kind providers.AuthErrorKind,
	classified bool,
	err error,
) providers.AuthErrorKind {
	if !classified || kind == "" {
		kind = providers.AuthErrorGeneric
	}
	if kind == providers.AuthErrorInvalidAPIKey && accountBalancePattern.MatchString(err.Error()) {
		return providers.AuthErrorGeneric
	}
	return kind
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
		// Deliberately names the balance: a credit-exhausted account and a bad
		// key both arrive as 401, and this branch is where the ambiguous ones
		// land.
		return "Authentication failed: check the API key, account balance, or provider permissions for this model."
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

	summary := failureSummary(failErr.Reason, failErr.Status)
	if failErr.Status > 0 {
		summary = fmt.Sprintf("%s (%d)", summary, failErr.Status)
	}
	// One candidate gets one line, so the provider's own words are appended to
	// that line rather than given a paragraph of their own.
	if detail := providerErrorDetail(attempt.Error); detail != "" {
		summary += " — " + detail
	}
	return summary
}

// failureSummary words a classified failure for a person.
//
// The classifier folds a bare 500 into the timeout bucket because timeout is
// the safest transient read for a retry decision. It is the wrong word to show
// a user, who did not experience a timeout and would go looking for one.
func failureSummary(reason providers.FailoverReason, status int) string {
	if status == 500 && reason == providers.FailoverTimeout {
		return "provider error"
	}
	return failoverReasonSummary(reason)
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
