package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	ToolCall               = protocoltypes.ToolCall
	FunctionCall           = protocoltypes.FunctionCall
	LLMResponse            = protocoltypes.LLMResponse
	StreamChunk            = protocoltypes.StreamChunk
	UsageInfo              = protocoltypes.UsageInfo
	Message                = protocoltypes.Message
	ToolDefinition         = protocoltypes.ToolDefinition
	ToolFunctionDefinition = protocoltypes.ToolFunctionDefinition
	ExtraContent           = protocoltypes.ExtraContent
	GoogleExtra            = protocoltypes.GoogleExtra
	ContentBlock           = protocoltypes.ContentBlock
	CacheControl           = protocoltypes.CacheControl
	Attachment             = protocoltypes.Attachment
)

type LLMProvider interface {
	Chat(
		ctx context.Context,
		messages []Message,
		tools []ToolDefinition,
		model string,
		options map[string]any,
	) (*LLMResponse, error)
	GetDefaultModel() string
}

type StatefulProvider interface {
	LLMProvider
	Close()
}

// StreamingProvider is an optional interface for providers that support token streaming.
// onChunk receives the accumulated text so far (not individual deltas).
// The returned LLMResponse is the same complete response for compatibility with tool-call handling.
type StreamingProvider interface {
	ChatStream(
		ctx context.Context,
		messages []Message,
		tools []ToolDefinition,
		model string,
		options map[string]any,
		onChunk func(accumulated string),
	) (*LLMResponse, error)
}

type StreamingEventProvider interface {
	ChatStreamEvents(
		ctx context.Context,
		messages []Message,
		tools []ToolDefinition,
		model string,
		options map[string]any,
		onChunk func(StreamChunk),
	) (*LLMResponse, error)
}

// ThinkingCapable is an optional interface for providers that support
// extended thinking (e.g. Anthropic). Used by the agent loop to warn
// when thinking_level is configured but the active provider cannot use it.
type ThinkingCapable interface {
	SupportsThinking() bool
}

// NativeSearchCapable is an optional interface for providers that support
// built-in web search during LLM inference (e.g. OpenAI web_search_preview,
// xAI Grok search). When the active provider implements this interface and
// returns true, the agent loop can hide the client-side web_search tool to
// avoid duplicate search surfaces and use the provider's native search instead.
type NativeSearchCapable interface {
	SupportsNativeSearch() bool
}

// FailoverReason classifies why an LLM request failed for fallback decisions.
type FailoverReason string

const (
	FailoverAuth      FailoverReason = "auth"
	FailoverRateLimit FailoverReason = "rate_limit"
	// FailoverHardQuota is a rate-limit family failure the provider has
	// indicated is not going to clear on its own within this request's
	// lifetime: an exhausted plan, credit or period quota. Retrying the same
	// candidate is pointless; another candidate may still work.
	FailoverHardQuota       FailoverReason = "hard_quota"
	FailoverBilling         FailoverReason = "billing"
	FailoverNetwork         FailoverReason = "network"
	FailoverTimeout         FailoverReason = "timeout"
	FailoverFormat          FailoverReason = "format"
	FailoverContextOverflow FailoverReason = "context_overflow"
	FailoverOverloaded      FailoverReason = "overloaded"
	FailoverUnknown         FailoverReason = "unknown"
)

// FailoverError wraps an LLM provider error with classification metadata.
type FailoverError struct {
	Reason   FailoverReason
	Provider string
	Model    string
	Status   int
	Wrapped  error

	// RetryAfter is the provider's own instruction to wait, taken from a
	// Retry-After header or an equivalent field. Zero means the provider gave
	// none, which is not the same as "retry immediately".
	RetryAfter time.Duration
}

func (e *FailoverError) Error() string {
	return fmt.Sprintf("failover(%s): provider=%s model=%s status=%d: %v",
		e.Reason, e.Provider, e.Model, e.Status, e.Wrapped)
}

func (e *FailoverError) Unwrap() error {
	return e.Wrapped
}

// IsRetriable returns true if this error should trigger fallback to the next
// candidate. Non-retriable: format errors (bad request structure, image
// dimension/size) and context overflow, which has its own compact-and-retry
// path on the current candidate.
//
// Hard quota is retriable in this sense: the candidate is finished, but a
// different one may well succeed. Whether the *same* candidate may be retried
// is a separate question — see AllowsSameCandidateRetry.
func (e *FailoverError) IsRetriable() bool {
	return e.Reason != FailoverFormat && e.Reason != FailoverContextOverflow
}

// AllowsSameCandidateRetry reports whether retrying the identical
// provider/model could plausibly succeed.
//
// This is deliberately distinct from IsRetriable. Moving to another candidate
// and retrying the same one fail for different reasons: a bad API key, an
// exhausted quota or an unpaid bill will still be bad, exhausted and unpaid a
// few seconds later, so a same-candidate retry is pure latency. Transient
// conditions are the opposite.
func (e *FailoverError) AllowsSameCandidateRetry() bool {
	switch e.Reason {
	case FailoverAuth, FailoverBilling, FailoverHardQuota, FailoverFormat:
		return false
	default:
		return true
	}
}

// ModelConfig holds primary model and fallback list.
type ModelConfig struct {
	Primary   string
	Fallbacks []string
}
