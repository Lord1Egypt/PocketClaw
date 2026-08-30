package providers

import (
	"context"
	"errors"
	"fmt"
	"github.com/sipeed/picoclaw/pkg/providers/common"
	"io"
	"net"
	"net/url"
	"syscall"
	"testing"
)

type stubNetError struct {
	msg     string
	timeout bool
}

func (e stubNetError) Error() string   { return e.msg }
func (e stubNetError) Timeout() bool   { return e.timeout }
func (e stubNetError) Temporary() bool { return false }

func TestClassifyError_Nil(t *testing.T) {
	result := ClassifyError(nil, "openai", "gpt-4")
	if result != nil {
		t.Errorf("expected nil for nil error, got %+v", result)
	}
}

func TestClassifyError_ContextCanceled(t *testing.T) {
	result := ClassifyError(context.Canceled, "openai", "gpt-4")
	if result != nil {
		t.Errorf("expected nil for context.Canceled (user abort), got %+v", result)
	}
}

func TestClassifyError_ContextDeadlineExceeded(t *testing.T) {
	result := ClassifyError(context.DeadlineExceeded, "openai", "gpt-4")
	if result == nil {
		t.Fatal("expected non-nil for deadline exceeded")
	}
	if result.Reason != FailoverTimeout {
		t.Errorf("reason = %q, want timeout", result.Reason)
	}
}

func TestClassifyError_StatusCodes(t *testing.T) {
	tests := []struct {
		status int
		reason FailoverReason
	}{
		{401, FailoverAuth},
		{403, FailoverAuth},
		{402, FailoverBilling},
		{408, FailoverTimeout},
		{429, FailoverRateLimit},
		{400, FailoverFormat},
		// The 5xx family is split by what each status actually indicates
		// rather than collapsed into "timeout": 502 is a reachability
		// problem, 503/521/522/523/529 mean the upstream is shedding load,
		// and 500/504/524 are best read as it running out of time.
		{500, FailoverTimeout},
		{502, FailoverNetwork},
		{503, FailoverOverloaded},
		{521, FailoverOverloaded},
		{522, FailoverOverloaded},
		{523, FailoverOverloaded},
		{524, FailoverTimeout},
		{529, FailoverOverloaded},
	}

	for _, tt := range tests {
		err := fmt.Errorf("API error: status: %d something went wrong", tt.status)
		result := ClassifyError(err, "test", "model")
		if result == nil {
			t.Errorf("status %d: expected non-nil", tt.status)
			continue
		}
		if result.Reason != tt.reason {
			t.Errorf("status %d: reason = %q, want %q", tt.status, result.Reason, tt.reason)
		}
	}
}

func TestClassifyError_RateLimitPatterns(t *testing.T) {
	patterns := []string{
		"rate limit exceeded",
		"rate_limit reached",
		"too many requests",
		"resource has been exhausted",
		"resource_exhausted",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverRateLimit {
			t.Errorf("pattern %q: reason = %q, want rate_limit", msg, result.Reason)
		}
	}
}

// A quota the provider says is exhausted will still be exhausted after any
// retry this turn could wait out, so it is separated from transient rate
// limiting. Getting this wrong in either direction is costly: treating a
// transient limit as hard quota puts a healthy provider into a long cooldown,
// and the reverse produces a retry storm against a provider that has already
// said no.
func TestClassifyError_HardQuotaPatterns(t *testing.T) {
	patterns := []string{
		"exceeded your current quota",
		"quota exceeded",
		"usage limit reached",
		"insufficient_quota",
		"out of credits",
		"plan limit reached",
	}

	for _, msg := range patterns {
		result := ClassifyError(errors.New(msg), "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverHardQuota {
			t.Errorf("pattern %q: reason = %q, want hard_quota", msg, result.Reason)
		}
		if result.AllowsSameCandidateRetry() {
			t.Errorf("pattern %q: hard quota must not permit a same-candidate retry", msg)
		}
		if !result.IsRetriable() {
			t.Errorf("pattern %q: hard quota must still allow moving to another candidate", msg)
		}
	}
}

// A plain rate limit must stay transient. This is the guard against the
// hard-quota patterns being widened until they swallow ordinary throttling.
func TestClassifyError_PlainRateLimitIsNotHardQuota(t *testing.T) {
	for _, msg := range []string{"rate limit exceeded", "too many requests", "429 slow down"} {
		result := ClassifyError(errors.New(msg), "openai", "gpt-4")
		if result == nil {
			t.Fatalf("pattern %q: expected non-nil", msg)
		}
		if result.Reason == FailoverHardQuota {
			t.Errorf("pattern %q: transient throttling must not be read as hard quota", msg)
		}
		if !result.AllowsSameCandidateRetry() {
			t.Errorf("pattern %q: transient throttling may be retried", msg)
		}
	}
}

// A 429 alone cannot tell the two apart; only the body can.
func TestClassifyError_429BodyDistinguishesQuotaFromThrottling(t *testing.T) {
	throttled := ClassifyError(
		&common.HTTPError{StatusCode: 429, BodyPreview: "rate limit, please slow down"},
		"openai", "gpt-4",
	)
	if throttled == nil || throttled.Reason != FailoverRateLimit {
		t.Fatalf("429 without quota wording should stay rate_limit, got %+v", throttled)
	}

	exhausted := ClassifyError(
		&common.HTTPError{StatusCode: 429, BodyPreview: "You exceeded your current quota"},
		"openai", "gpt-4",
	)
	if exhausted == nil || exhausted.Reason != FailoverHardQuota {
		t.Fatalf("429 with quota wording should be hard_quota, got %+v", exhausted)
	}
}

func TestClassifyError_OverloadedPatterns(t *testing.T) {
	patterns := []string{
		"overloaded_error",
		`{"type": "overloaded_error"}`,
		"server is overloaded",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "anthropic", "claude")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		// Overloaded is its own reason rather than rate_limit: the provider
		// is shedding load, not throttling this caller, and the two deserve
		// different cooldown treatment.
		if result.Reason != FailoverOverloaded {
			t.Errorf("pattern %q: reason = %q, want overloaded", msg, result.Reason)
		}
	}
}

func TestClassifyError_BillingPatterns(t *testing.T) {
	patterns := []string{
		"payment required",
		"insufficient credits",
		"credit balance too low",
		"plans & billing page",
		"insufficient balance",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverBilling {
			t.Errorf("pattern %q: reason = %q, want billing", msg, result.Reason)
		}
	}
}

func TestClassifyError_TimeoutPatterns(t *testing.T) {
	patterns := []string{
		"request timeout",
		"connection timed out",
		"deadline exceeded",
		"context deadline exceeded",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverTimeout {
			t.Errorf("pattern %q: reason = %q, want timeout", msg, result.Reason)
		}
	}
}

func TestClassifyError_NetworkPatterns(t *testing.T) {
	patterns := []string{
		`failed to send request: Post "https://example.com": tls: bad record MAC`,
		"read tcp 10.20.0.1:61279->172.65.90.20:443: read: connection reset by peer",
		"failed to send request: dial tcp 203.0.113.10:443: connect: connection refused",
		"tls handshake failure",
		"x509: certificate has expired or is not yet valid",
		"read tcp 127.0.0.1:443: read: unexpected EOF",
		"lookup api.example.com: no such host",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverNetwork {
			t.Errorf("pattern %q: reason = %q, want network", msg, result.Reason)
		}
	}
}

func TestClassifyError_NetworkTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "wrapped EOF",
			err: &url.Error{
				Op:  "Post",
				URL: "https://example.com",
				Err: io.EOF,
			},
		},
		{
			name: "dns error",
			err: &net.DNSError{
				Err:  "no such host",
				Name: "api.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err, "openai", "gpt-4")
			if result == nil {
				t.Fatal("expected non-nil")
			}
			if result.Reason != FailoverNetwork {
				t.Fatalf("reason = %q, want network", result.Reason)
			}
		})
	}
}

func TestClassifyError_TimeoutNetworkTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "wrapped syscall timeout",
			err:  fmt.Errorf("dial tcp: %w", syscall.ETIMEDOUT),
		},
		{
			name: "net error timeout",
			err: &url.Error{
				Op:  "Post",
				URL: "https://example.com",
				Err: stubNetError{msg: "i/o timeout", timeout: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err, "openai", "gpt-4")
			if result == nil {
				t.Fatal("expected non-nil")
			}
			if result.Reason != FailoverTimeout {
				t.Fatalf("reason = %q, want timeout", result.Reason)
			}
		})
	}
}

func TestClassifyError_TimeoutPatternsWinOverNetworkContext(t *testing.T) {
	patterns := []string{
		`failed to send request: Post "https://example.com": dial tcp 203.0.113.10:443: i/o timeout`,
		`read tcp 10.20.0.1:61279->172.65.90.20:443: i/o timeout`,
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverTimeout {
			t.Errorf("pattern %q: reason = %q, want timeout", msg, result.Reason)
		}
	}
}

func TestClassifyError_NetworkPatternsWinOverAuthExpired(t *testing.T) {
	err := errors.New(
		`Post "https://example.com": tls: failed to verify certificate: x509: certificate has expired or is not yet valid`,
	)
	result := ClassifyError(err, "openai", "gpt-4")
	if result == nil {
		t.Fatal("expected non-nil")
	}
	if result.Reason != FailoverNetwork {
		t.Fatalf("reason = %q, want network", result.Reason)
	}
}

func TestClassifyError_AuthPatterns(t *testing.T) {
	patterns := []string{
		"invalid api key",
		"invalid_api_key",
		"incorrect api key",
		"invalid token",
		"authentication failed",
		"re-authenticate",
		"oauth token refresh failed",
		"unauthorized access",
		"forbidden",
		"access denied",
		"expired",
		"token has expired",
		"no credentials found",
		"no api key found",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverAuth {
			t.Errorf("pattern %q: reason = %q, want auth", msg, result.Reason)
		}
	}
}

func TestClassifyError_FormatPatterns(t *testing.T) {
	patterns := []string{
		"string should match pattern",
		"tool_use.id is required",
		"invalid tool_use_id",
		"messages.1.content.1.tool_use.id must be valid",
		"invalid request format",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "anthropic", "claude")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverFormat {
			t.Errorf("pattern %q: reason = %q, want format", msg, result.Reason)
		}
	}
}

func TestClassifyError_ImageDimensionError(t *testing.T) {
	err := errors.New("image dimensions exceed max allowed 2048x2048")
	result := ClassifyError(err, "openai", "gpt-4o")
	if result == nil {
		t.Fatal("expected non-nil for image dimension error")
	}
	if result.Reason != FailoverFormat {
		t.Errorf("reason = %q, want format", result.Reason)
	}
	if result.IsRetriable() {
		t.Error("image dimension error should not be retriable")
	}
}

func TestClassifyError_ContextOverflowPatterns(t *testing.T) {
	patterns := []string{
		"context_length_exceeded",
		"context_window_exceeded",
		"maximum context length",
		"token limit",
		"too many tokens",
		"prompt is too long",
		"request too large",
	}

	for _, msg := range patterns {
		err := errors.New(msg)
		result := ClassifyError(err, "openai", "gpt-4")
		if result == nil {
			t.Errorf("pattern %q: expected non-nil", msg)
			continue
		}
		if result.Reason != FailoverContextOverflow {
			t.Errorf("pattern %q: reason = %q, want context_overflow", msg, result.Reason)
		}
	}
}

func TestClassifyError_ImageSizeError(t *testing.T) {
	err := errors.New("image exceeds 20 mb limit")
	result := ClassifyError(err, "openai", "gpt-4o")
	if result == nil {
		t.Fatal("expected non-nil for image size error")
	}
	if result.Reason != FailoverFormat {
		t.Errorf("reason = %q, want format", result.Reason)
	}
}

func TestClassifyError_UnknownError(t *testing.T) {
	err := errors.New("some completely random error")
	result := ClassifyError(err, "openai", "gpt-4")
	if result != nil {
		t.Errorf("expected nil for unknown error, got %+v", result)
	}
}

func TestClassifyError_ProviderModelPropagation(t *testing.T) {
	err := errors.New("rate limit exceeded")
	result := ClassifyError(err, "my-provider", "my-model")
	if result == nil {
		t.Fatal("expected non-nil")
	}
	if result.Provider != "my-provider" {
		t.Errorf("provider = %q, want my-provider", result.Provider)
	}
	if result.Model != "my-model" {
		t.Errorf("model = %q, want my-model", result.Model)
	}
}

func TestFailoverError_IsRetriable(t *testing.T) {
	tests := []struct {
		reason    FailoverReason
		retriable bool
	}{
		{FailoverAuth, true},
		{FailoverRateLimit, true},
		{FailoverBilling, true},
		{FailoverNetwork, true},
		{FailoverTimeout, true},
		{FailoverOverloaded, true},
		{FailoverFormat, false},
		{FailoverContextOverflow, false},
		{FailoverUnknown, true},
	}

	for _, tt := range tests {
		fe := &FailoverError{Reason: tt.reason}
		if fe.IsRetriable() != tt.retriable {
			t.Errorf("IsRetriable(%q) = %v, want %v", tt.reason, fe.IsRetriable(), tt.retriable)
		}
	}
}

func TestFailoverError_ErrorString(t *testing.T) {
	fe := &FailoverError{
		Reason:   FailoverRateLimit,
		Provider: "openai",
		Model:    "gpt-4",
		Status:   429,
		Wrapped:  errors.New("too many requests"),
	}
	s := fe.Error()
	if s == "" {
		t.Error("expected non-empty error string")
	}
}

func TestFailoverError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	fe := &FailoverError{Reason: FailoverTimeout, Wrapped: inner}
	if fe.Unwrap() != inner {
		t.Error("Unwrap should return wrapped error")
	}
}

func TestExtractHTTPStatus(t *testing.T) {
	tests := []struct {
		msg  string
		want int
	}{
		{"status: 429 rate limited", 429},
		{"status 401 unauthorized", 401},
		{"http/1.1 502 bad gateway", 502},
		{"error 429", 429},
		{"no status code here", 0},
		{"random number 12345", 0},
	}

	for _, tt := range tests {
		got := extractHTTPStatus(tt.msg)
		if got != tt.want {
			t.Errorf("extractHTTPStatus(%q) = %d, want %d", tt.msg, got, tt.want)
		}
	}
}

func TestIsImageDimensionError(t *testing.T) {
	if !IsImageDimensionError("image dimensions exceed max 4096x4096") {
		t.Error("should match image dimensions exceed max")
	}
	if IsImageDimensionError("normal error message") {
		t.Error("should not match normal error")
	}
}

func TestIsImageSizeError(t *testing.T) {
	if !IsImageSizeError("image exceeds 20 mb") {
		t.Error("should match image exceeds mb")
	}
	if IsImageSizeError("normal error message") {
		t.Error("should not match normal error")
	}
}
