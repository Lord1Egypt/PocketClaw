package providers

import (
	"context"
	"errors"
	"io"
	"net"
	"regexp"
	"strings"
	"syscall"

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

// Common patterns in Go HTTP error messages
var httpStatusPatterns = []*regexp.Regexp{
	regexp.MustCompile(`status[:\s]+(\d{3})`),
	regexp.MustCompile(`http[/\s]+\d*\.?\d*\s+(\d{3})`),
	regexp.MustCompile(`\b([3-5]\d{2})\b`),
}

// errorPattern defines a single pattern (string or regex) for error classification.
type errorPattern struct {
	substring string
	regex     *regexp.Regexp
}

func substr(s string) errorPattern { return errorPattern{substring: s} }
func rxp(r string) errorPattern    { return errorPattern{regex: regexp.MustCompile("(?i)" + r)} }

// Error patterns organized by FailoverReason, matching OpenClaw production (~40 patterns).
var (
	rateLimitPatterns = []errorPattern{
		rxp(`rate[_ ]limit`),
		substr("too many requests"),
		substr("429"),
		substr("exceeded your current quota"),
		rxp(`exceeded.*quota`),
		rxp(`resource has been exhausted`),
		rxp(`resource.*exhausted`),
		substr("resource_exhausted"),
		substr("quota exceeded"),
		substr("usage limit"),
	}

	overloadedPatterns = []errorPattern{
		rxp(`overloaded_error`),
		rxp(`"type"\s*:\s*"overloaded_error"`),
		substr("overloaded"),
	}

	timeoutPatterns = []errorPattern{
		substr("timeout"),
		substr("timed out"),
		substr("deadline exceeded"),
		substr("context deadline exceeded"),
	}

	networkPatterns = []errorPattern{
		substr("connection reset"),
		substr("reset by peer"),
		substr("connection refused"),
		substr("connection aborted"),
		substr("broken pipe"),
		substr("use of closed network connection"),
		substr("network is unreachable"),
		substr("host is unreachable"),
		substr("no such host"),
		substr("temporary failure in name resolution"),
		substr("server misbehaving"),
		substr("read tcp"),
		substr("write tcp"),
		substr("dial tcp"),
		substr("tls:"),
		substr("x509:"),
		substr("certificate"),
		substr("handshake"),
		substr("unexpected eof"),
		substr("read: eof"),
		substr("write: eof"),
	}

	billingPatterns = []errorPattern{
		rxp(`\b402\b`),
		substr("payment required"),
		substr("insufficient credits"),
		substr("credit balance"),
		substr("plans & billing"),
		substr("insufficient balance"),
	}

	authPatterns = []errorPattern{
		rxp(`\b(?:invalid|incorrect|malformed|wrong)[-_\s]+(?:api[-_\s]*)?key\b`),
		rxp(`\b(?:api[-_\s]*)?key[-_\s]+(?:is[-_\s]+)?(?:invalid|incorrect|malformed|wrong)\b`),
		rxp(`invalid[_ ]?api[_ ]?key`),
		substr("incorrect api key"),
		substr("invalid token"),
		substr("authentication"),
		substr("re-authenticate"),
		substr("oauth token refresh failed"),
		substr("unauthorized"),
		substr("forbidden"),
		substr("access denied"),
		substr("expired"),
		substr("token has expired"),
		rxp(`\b401\b`),
		rxp(`\b403\b`),
		substr("no credentials found"),
		substr("no api key found"),
	}

	formatPatterns = []errorPattern{
		substr("string should match pattern"),
		substr("tool_use.id"),
		substr("tool_use_id"),
		substr("messages.1.content.1.tool_use.id"),
		substr("invalid request format"),
		// Zhipu API error code 1210: parameter error (e.g., image format incompatible)
		substr("error code: 1210"),
		substr("error code 1210"),
		substr("zhipu api error code: 1210"),
	}
	contextOverflowPatterns = []errorPattern{
		rxp(`context[_ ]?length[_ ]?exceeded`),
		rxp(`context[_ ]?window[_ ]?exceeded`),
		substr("maximum context length"),
		substr("token limit"),
		substr("too many tokens"),
		substr("prompt is too long"),
		substr("request too large"),
		// Payload size rather than token count: the request body itself was
		// refused. Both are cured the same way — send less.
		substr("request entity too large"),
		substr("payload too large"),
		// OpenAI rejects a single oversized message field this way.
		rxp(`string too long.*maximum length`),
		// Volcengine/Doubao, Gemini and DashScope word the same condition
		// without any of the phrases above.
		rxp(`exceeds? max(imum)? message tokens`),
		rxp(`input token count.*exceeds`),
		substr("range of input length"),
	}

	imageDimensionPatterns = []errorPattern{
		rxp(`image dimensions exceed max`),
	}

	imageSizePatterns = []errorPattern{
		rxp(`image exceeds.*mb`),
	}

	// hardQuotaPatterns mark a rate-limit family failure the provider has said
	// will not clear on its own: an exhausted plan, credit or period quota.
	// These are deliberately narrow. A plain "rate limit exceeded" is transient
	// and must not land here, because treating it as hard quota would put a
	// healthy provider into a long cooldown.
	hardQuotaPatterns = []errorPattern{
		substr("exceeded your current quota"),
		substr("quota exceeded"),
		rxp(`quota.*exhausted`),
		rxp(`exhausted.*quota`),
		substr("insufficient_quota"),
		substr("billing hard limit"),
		substr("monthly limit"),
		substr("daily limit exceeded"),
		substr("usage limit reached"),
		substr("out of credits"),
		substr("no credits remaining"),
		substr("plan limit reached"),
	}

	// Server-side status codes, split by what they actually indicate. Collapsing
	// all of them into "timeout" loses the distinction between a gateway that
	// could not reach upstream, an upstream that is deliberately shedding load,
	// and one that genuinely ran out of time.
	upstreamNetworkStatusCodes = map[int]bool{
		502: true, // bad gateway: the edge could not reach upstream
	}
	overloadedStatusCodes = map[int]bool{
		503: true, // service unavailable: upstream is shedding load
		529: true, // used by several providers for "overloaded"
		//nolint:gomnd // Cloudflare origin-error family, all load/reachability
		521: true, 522: true, 523: true,
	}
	upstreamTimeoutStatusCodes = map[int]bool{
		500: true, // opaque server error; timeout is the safest transient read
		504: true, // gateway timeout
		524: true, // Cloudflare: origin did not respond in time
	}
)

// ClassifyError classifies an error into a FailoverError with reason.
// Returns nil if the error is not classifiable (unknown errors should not trigger fallback).
func ClassifyError(err error, provider, model string) *FailoverError {
	if err == nil {
		return nil
	}

	// Context cancellation: user abort, never fallback.
	if err == context.Canceled {
		return nil
	}

	// Context deadline exceeded: treat as timeout, always fallback.
	if err == context.DeadlineExceeded {
		return &FailoverError{
			Reason:   FailoverTimeout,
			Provider: provider,
			Model:    model,
			Wrapped:  err,
		}
	}

	msg := strings.ToLower(err.Error())

	// Concrete transport errors should continue the fallback chain even when
	// providers do not expose a structured HTTP status.
	if reason := classifyByErrorType(err); reason != "" {
		return &FailoverError{
			Reason:   reason,
			Provider: provider,
			Model:    model,
			Wrapped:  err,
		}
	}

	// Image dimension/size errors: non-retriable, non-fallback.
	if IsImageDimensionError(msg) || IsImageSizeError(msg) {
		return &FailoverError{
			Reason:   FailoverFormat,
			Provider: provider,
			Model:    model,
			Wrapped:  err,
		}
	}

	// Try HTTP status code extraction first.
	var httpErr *common.HTTPError
	if errors.As(err, &httpErr) && httpErr != nil {
		if reason := classifyByStatus(httpErr.StatusCode); reason != "" {
			return &FailoverError{
				Reason:     refineRateLimitReason(reason, msg),
				Provider:   provider,
				Model:      model,
				Status:     httpErr.StatusCode,
				Wrapped:    err,
				RetryAfter: httpErr.RetryAfter,
			}
		}
	}
	if status := extractHTTPStatus(msg); status > 0 {
		if reason := classifyByStatus(status); reason != "" {
			return &FailoverError{
				Reason:   refineRateLimitReason(reason, msg),
				Provider: provider,
				Model:    model,
				Status:   status,
				Wrapped:  err,
			}
		}
	}

	// Message pattern matching (priority order from OpenClaw).
	if reason := classifyByMessage(msg); reason != "" {
		return &FailoverError{
			Reason:   reason,
			Provider: provider,
			Model:    model,
			Wrapped:  err,
		}
	}

	return nil
}

// classifyByErrorType maps concrete transport-layer error types to a retryable
// fallback reason before message heuristics are applied.
func classifyByErrorType(err error) FailoverReason {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return FailoverNetwork
	}

	for _, transportErr := range []error{
		syscall.ECONNRESET,
		syscall.ECONNABORTED,
		syscall.ECONNREFUSED,
		syscall.ETIMEDOUT,
		syscall.EHOSTUNREACH,
		syscall.ENETUNREACH,
		syscall.EPIPE,
	} {
		if errors.Is(err, transportErr) {
			if transportErr == syscall.ETIMEDOUT {
				return FailoverTimeout
			}
			return FailoverNetwork
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return FailoverTimeout
		}
		return FailoverNetwork
	}

	return ""
}

// classifyByStatus maps HTTP status codes to FailoverReason.
//
// The 5xx family is split rather than collapsed: 502 is a reachability problem,
// 503/529 mean the upstream is deliberately shedding load, and 504 means it ran
// out of time. They deserve different cooldown and retry treatment, and a log
// that says "timeout" for all three is actively misleading during diagnosis.
func classifyByStatus(status int) FailoverReason {
	switch {
	case status == 401 || status == 403:
		return FailoverAuth
	case status == 402:
		return FailoverBilling
	case status == 408:
		return FailoverTimeout
	case status == 429:
		return FailoverRateLimit
	case status == 400:
		return FailoverFormat
	case upstreamNetworkStatusCodes[status]:
		return FailoverNetwork
	case overloadedStatusCodes[status]:
		return FailoverOverloaded
	case upstreamTimeoutStatusCodes[status]:
		return FailoverTimeout
	}
	return ""
}

// classifyByMessage matches error messages against patterns.
// Priority order matters (from OpenClaw classifyFailoverReason).
func classifyByMessage(msg string) FailoverReason {
	// Hard quota is checked before the general rate-limit patterns, which it
	// overlaps by design: "exceeded your current quota" matches both, and the
	// more specific reading is the useful one.
	if matchesAny(msg, hardQuotaPatterns) {
		return FailoverHardQuota
	}
	if matchesAny(msg, rateLimitPatterns) {
		return FailoverRateLimit
	}
	if matchesAny(msg, overloadedPatterns) {
		return FailoverOverloaded
	}
	if matchesAny(msg, billingPatterns) {
		return FailoverBilling
	}
	if matchesAny(msg, timeoutPatterns) {
		return FailoverTimeout
	}
	if matchesAny(msg, networkPatterns) {
		return FailoverNetwork
	}
	if matchesAny(msg, authPatterns) {
		return FailoverAuth
	}
	if matchesAny(msg, formatPatterns) {
		return FailoverFormat
	}
	if matchesAny(msg, contextOverflowPatterns) {
		return FailoverContextOverflow
	}
	return ""
}

// extractHTTPStatus extracts an HTTP status code from an error message.
// Looks for patterns like "status: 429", "status 429", "http/1.1 429", "http 429", or standalone "429".
func extractHTTPStatus(msg string) int {
	for _, p := range httpStatusPatterns {
		if m := p.FindStringSubmatch(msg); len(m) > 1 {
			return parseDigits(m[1])
		}
	}
	return 0
}

// IsContextOverflowMessage reports whether an error message says the request
// was too large for the model: too many tokens, or too many bytes. It looks at
// the wording only; a caller must still rule out statuses that mean something
// else, because a 400 is classified as a format error before its body is read.
func IsContextOverflowMessage(msg string) bool {
	return matchesAny(strings.ToLower(msg), contextOverflowPatterns)
}

// IsImageDimensionError returns true if the message indicates an image dimension error.
func IsImageDimensionError(msg string) bool {
	return matchesAny(msg, imageDimensionPatterns)
}

// IsImageSizeError returns true if the message indicates an image file size error.
func IsImageSizeError(msg string) bool {
	return matchesAny(msg, imageSizePatterns)
}

// matchesAny checks if msg matches any of the patterns.
func matchesAny(msg string, patterns []errorPattern) bool {
	for _, p := range patterns {
		if p.regex != nil {
			if p.regex.MatchString(msg) {
				return true
			}
		} else if p.substring != "" {
			if strings.Contains(msg, p.substring) {
				return true
			}
		}
	}
	return false
}

// parseDigits converts a string of digits to an int.
func parseDigits(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// refineRateLimitReason upgrades a status-derived rate limit to hard quota when
// the body says so.
//
// A 429 alone cannot tell the two apart, and the difference matters: a
// transient rate limit clears in seconds, while an exhausted quota will still
// be exhausted after any retry this turn could wait out. Only the body
// distinguishes them, so the status classification is refined rather than
// trusted on its own.
func refineRateLimitReason(reason FailoverReason, msg string) FailoverReason {
	if reason != FailoverRateLimit {
		return reason
	}
	if matchesAny(msg, hardQuotaPatterns) {
		return FailoverHardQuota
	}
	return reason
}
