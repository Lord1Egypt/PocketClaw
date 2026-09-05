package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/common"
)

func TestFormatProcessingError_InvalidAPIKey(t *testing.T) {
	err := errors.New(
		`LLM call failed after retries: API request failed: Status: 401 Body: {"error":{"message":"Incorrect API key provided"}}`,
	)

	got := formatProcessingError(err)
	if !strings.Contains(got, "API key appears to be invalid") {
		t.Fatalf("formatted error missing friendly API key hint: %q", got)
	}
	if !strings.Contains(got, "Original error:") {
		t.Fatalf("formatted error missing original error label: %q", got)
	}
	if !strings.Contains(got, err.Error()) {
		t.Fatalf("formatted error missing original error: %q", got)
	}
}

func TestFormatProcessingError_GenericAuthHTTPError(t *testing.T) {
	err := &common.HTTPError{
		StatusCode:  401,
		BodyPreview: `{"error":"unauthorized"}`,
		ContentType: "application/json",
		APIBase:     "https://api.example.com",
	}

	got := formatProcessingError(err)
	if !strings.Contains(got, "check the API key, token, OAuth login, or provider permissions") {
		t.Fatalf("formatted error missing generic auth hint: %q", got)
	}
	if !strings.Contains(got, "Original error:") {
		t.Fatalf("formatted error missing original error: %q", got)
	}
}

func TestFormatProcessingError_NonAuth(t *testing.T) {
	err := errors.New("connection reset by peer")
	got := formatProcessingError(err)
	want := "Error processing message: connection reset by peer"
	if got != want {
		t.Fatalf("formatted error = %q, want %q", got, want)
	}
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
