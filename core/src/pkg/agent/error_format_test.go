package agent

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/common"
)

// forbiddenInUserFacingErrors are the shapes a provider response takes that
// must never reach a chat window, whichever path the failure came through.
var forbiddenInUserFacingErrors = []string{
	"Original error:",
	"Body:",
	`{"error"`,
	"https://",
	"request_id",
}

func assertUserFacingErrorIsSafe(t *testing.T, got string, raw string) {
	t.Helper()
	for _, forbidden := range forbiddenInUserFacingErrors {
		if strings.Contains(got, forbidden) {
			t.Fatalf("user-facing error exposed %q:\n%s", forbidden, got)
		}
	}
	if strings.Contains(got, raw) {
		t.Fatalf("user-facing error repeated the provider response verbatim:\n%s", got)
	}
}

// The reported defect: a single configured model failing with 401 put the
// provider's whole response into Telegram under an "Original error:" heading,
// including the balance message and the top-up link.
func TestDirectCreditsExhausted401IsConciseAndSafe(t *testing.T) {
	rawBody := `{"error":{"name":"CreditsError","message":"Insufficient balance. ` +
		`Add credits at https://opencode.ai/billing","request_id":"req_9c1"}}`
	err := fmt.Errorf("LLM call failed after retries: %w", &common.HTTPError{
		StatusCode:  401,
		BodyPreview: rawBody,
		ContentType: "application/json",
		APIBase:     "https://opencode.ai/v1",
	})

	got := formatProcessingError(err)
	assertUserFacingErrorIsSafe(t, got, rawBody)

	if !strings.Contains(got, "Authentication failed") {
		t.Fatalf("a 401 must still be named as an authentication failure: %q", got)
	}
	if !strings.Contains(got, "account balance") {
		t.Fatalf("a credits rejection must point at the balance: %q", got)
	}
	if strings.Contains(strings.ToLower(got), "api key appears to be invalid") {
		t.Fatalf("a credits rejection must not blame the API key: %q", got)
	}
}

// The same protection when the provider words a credit failure as a bad key.
func TestDirect401MentioningBalanceDoesNotBlameTheAPIKey(t *testing.T) {
	err := errors.New(
		`LLM call failed after retries: API request failed: Status: 401 ` +
			`Body: {"error":{"message":"invalid api key or insufficient credits"}}`,
	)

	got := formatProcessingError(err)
	if strings.Contains(strings.ToLower(got), "api key appears to be invalid") {
		t.Fatalf("a balance-mentioning 401 must not assert an invalid key: %q", got)
	}
	if !strings.Contains(got, "account balance") {
		t.Fatalf("the balance must be offered as a cause: %q", got)
	}
	assertUserFacingErrorIsSafe(t, got, "insufficient credits")
}

// An unambiguous bad key still gets the specific advice; the guard above must
// not have flattened every 401 into the generic message.
func TestDirectInvalidAPIKeyKeepsTheSpecificAdvice(t *testing.T) {
	err := errors.New(
		`LLM call failed after retries: API request failed: Status: 401 ` +
			`Body: {"error":{"message":"Incorrect API key provided"}}`,
	)

	got := formatProcessingError(err)
	if !strings.Contains(got, "API key appears to be invalid") {
		t.Fatalf("an unambiguous bad key must keep its specific hint: %q", got)
	}
	assertUserFacingErrorIsSafe(t, got, "Incorrect API key provided")
}

func TestDirectRateLimit429IsConciseAndSafe(t *testing.T) {
	rawBody := `{"error":{"message":"You exceeded your current quota, check ` +
		`plan and billing at https://platform.example.com/account/billing",` +
		`"code":"insufficient_quota","request_id":"req_abc123"}}`
	err := fmt.Errorf("LLM call failed after retries: %w", &common.HTTPError{
		StatusCode:  429,
		BodyPreview: rawBody,
		ContentType: "application/json",
		APIBase:     "https://platform.example.com/v1",
	})

	got := formatProcessingError(err)
	assertUserFacingErrorIsSafe(t, got, rawBody)
	if !strings.Contains(got, "429") {
		t.Fatalf("the status code is the one number worth keeping: %q", got)
	}
	lowered := strings.ToLower(got)
	if !strings.Contains(lowered, "rate limited") && !strings.Contains(lowered, "quota") {
		t.Fatalf("a 429 must be legible as a rate/quota problem: %q", got)
	}
}

func TestDirectBadRequest400IsConciseAndSafe(t *testing.T) {
	rawBody := `{"error":{"message":"invalid request payload: messages[3].content ` +
		`unsupported variant image_url","type":"invalid_request_error",` +
		`"docs":"https://provider.example.com/docs/errors"}}`
	err := fmt.Errorf("LLM call failed after retries: %w", &common.HTTPError{
		StatusCode:  400,
		BodyPreview: rawBody,
		ContentType: "application/json",
		APIBase:     "https://provider.example.com/v1",
	})

	got := formatProcessingError(err)
	assertUserFacingErrorIsSafe(t, got, rawBody)
	if !strings.Contains(got, "400") {
		t.Fatalf("the status code must survive: %q", got)
	}
}

// A bare 500 is classified as a timeout so the chain retries it, which is the
// right failover decision and the wrong word to show someone who did not
// experience a timeout.
func TestDirectServerError500IsNotDescribedAsATimeout(t *testing.T) {
	err := fmt.Errorf("LLM call failed after retries: %w", &common.HTTPError{
		StatusCode:  500,
		BodyPreview: `{"error":"internal"}`,
	})

	got := formatProcessingError(err)
	if strings.Contains(strings.ToLower(got), "timed out") {
		t.Fatalf("a 500 must not be reported as a timeout: %q", got)
	}
	if !strings.Contains(got, "500") {
		t.Fatalf("the status code must survive: %q", got)
	}
	assertUserFacingErrorIsSafe(t, got, `{"error":"internal"}`)
}

// A genuine gateway timeout keeps the accurate word.
func TestDirectGatewayTimeout504StaysATimeout(t *testing.T) {
	err := fmt.Errorf("LLM call failed after retries: %w", &common.HTTPError{
		StatusCode:  504,
		BodyPreview: "gateway timeout",
	})

	got := formatProcessingError(err)
	if !strings.Contains(strings.ToLower(got), "timed out") {
		t.Fatalf("a 504 is a timeout and should say so: %q", got)
	}
}

// Errors PocketClaw raises itself are advice, not a provider response, and must
// still reach the user intact.
func TestNonProviderErrorsStillReachTheUserIntact(t *testing.T) {
	err := visionUnsupportedModelError("gpt-4o-mini", false)
	got := formatProcessingError(err)
	if !strings.Contains(got, "does not support image input") {
		t.Fatalf("actionable guidance was suppressed: %q", got)
	}
	if !strings.Contains(got, "gpt-4o-mini") {
		t.Fatalf("the model name is part of the advice: %q", got)
	}
}

// A transport failure is the provider's to report, and its wording carries
// nothing the summary does not.
func TestFormatProcessingError_NonAuth(t *testing.T) {
	err := errors.New("connection reset by peer")
	got := formatProcessingError(err)
	if !strings.Contains(strings.ToLower(got), "network error") {
		t.Fatalf("a transport failure must be summarised: %q", got)
	}
	assertUserFacingErrorIsSafe(t, got, "connection reset by peer")
}

// A chain that ran out of candidates used to reach the user as the concatenated
// raw bodies of every provider response — pages of JSON, billing URLs and
// provider internals in a chat window.
func TestFallbackExhaustedErrorIsConciseAndCarriesNoProviderInternals(t *testing.T) {
	rawBillingBody := `{"error":{"message":"You exceeded your current quota, ` +
		`please check your plan and billing details at ` +
		`https://platform.example.com/account/billing","type":"insufficient_quota",` +
		`"param":null,"code":"insufficient_quota","request_id":"req_abc123"}}`

	err := &providers.FallbackExhaustedError{
		Attempts: []providers.FallbackAttempt{
			{Provider: "opencodego", Model: "deepseek-vision", Error: errors.New("HTTP 400: invalid request payload")},
			{Provider: "gemini", Model: "gemini-2.5", Error: errors.New("HTTP 429: " + rawBillingBody)},
			{Provider: "deepseek", Model: "deepseek-chat", Error: errors.New("HTTP 401: unauthorized")},
			{Provider: "openai", Model: "gpt-4o", Skipped: true},
		},
	}

	got := formatProcessingError(err)

	for _, forbidden := range []string{
		"https://platform.example.com", "insufficient_quota", "req_abc123", `{"error"`,
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("user-facing error exposed provider internals %q:\n%s", forbidden, got)
		}
	}

	// One line per candidate plus the heading, and nothing longer.
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 1+len(err.Attempts) {
		t.Fatalf("expected one line per candidate, got %d lines:\n%s", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "All configured models failed") {
		t.Fatalf("missing summary heading: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "Primary ") {
		t.Fatalf("the first candidate must be named as the primary: %q", lines[1])
	}
	for i, want := range []string{"deepseek-vision", "gemini-2.5", "deepseek-chat", "gpt-4o"} {
		if !strings.Contains(lines[i+1], want) {
			t.Fatalf("line %d does not name its candidate %q: %q", i+1, want, lines[i+1])
		}
	}
	if !strings.Contains(lines[4], "skipped") {
		t.Fatalf("a skipped candidate must say so: %q", lines[4])
	}
}

// The classified reason is the part a user can act on, so it must survive.
func TestFallbackExhaustedErrorKeepsTheActionableReason(t *testing.T) {
	err := &providers.FallbackExhaustedError{
		Attempts: []providers.FallbackAttempt{
			{Provider: "p", Model: "m", Error: errors.New("HTTP 401: invalid api key")},
		},
	}
	got := formatProcessingError(err)
	if !strings.Contains(strings.ToLower(got), "authentication failed") {
		t.Fatalf("an auth failure must be legible to the user: %s", got)
	}
	if !strings.Contains(got, "401") {
		t.Fatalf("the status code is the one number worth keeping: %s", got)
	}
}
