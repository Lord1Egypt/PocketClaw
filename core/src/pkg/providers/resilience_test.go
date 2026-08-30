package providers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

func TestPrimarySuccessUsesNoFallback(t *testing.T) {
	fake := newFakeProvider(scriptSuccess())
	candidates := testCandidates("primary", "backup")

	result, err := newTestChain().ExecuteCandidate(context.Background(), candidates, fake.run)
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if fake.callCount() != 1 {
		t.Fatalf("a healthy primary must be called once, got %d", fake.callCount())
	}
	if len(result.Attempts) != 0 {
		t.Fatalf("no failed attempts expected, got %d", len(result.Attempts))
	}
}

func TestTransientRateLimitFallsBackAndSucceeds(t *testing.T) {
	fake := newFakeProvider(scriptHTTP(429, "rate limit, slow down"), scriptSuccess())
	candidates := testCandidates("primary", "backup")

	result, err := newTestChain().ExecuteCandidate(context.Background(), candidates, fake.run)
	if err != nil {
		t.Fatalf("expected fallback to succeed: %v", err)
	}
	if result.Model != "backup" {
		t.Fatalf("expected the backup to answer, got %q", result.Model)
	}
	if fake.callsFor("provider0", "primary") != 1 {
		t.Fatalf("primary should be tried once, got %d", fake.callsFor("provider0", "primary"))
	}
}

// An exhausted quota must not produce a retry storm against the candidate that
// already said no.
func TestHardQuotaDoesNotRetrySameCandidate(t *testing.T) {
	fake := newFakeProvider(
		scriptHTTP(429, "You exceeded your current quota, please check your plan"),
		scriptSuccess(),
	)
	candidates := testCandidates("primary", "backup")
	chain := newTestChain()

	result, err := chain.ExecuteCandidate(context.Background(), candidates, fake.run)
	if err != nil {
		t.Fatalf("expected fallback to succeed: %v", err)
	}
	if result.Model != "backup" {
		t.Fatalf("expected the backup to answer, got %q", result.Model)
	}
	if calls := fake.callsFor("provider0", "primary"); calls != 1 {
		t.Fatalf("an exhausted quota must be asked exactly once, got %d", calls)
	}
	if chain.Cooldown().IsAvailable(candidates[0].StableKey()) {
		t.Fatal("a candidate that reported hard quota must enter cooldown")
	}
}

func TestServerErrorsFallBack(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		reason FailoverReason
	}{
		{"502 bad gateway", 502, FailoverNetwork},
		{"503 unavailable", 503, FailoverOverloaded},
		{"504 gateway timeout", 504, FailoverTimeout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeProvider(scriptHTTP(tc.status, "upstream problem"), scriptSuccess())
			candidates := testCandidates("primary", "backup")

			result, err := newTestChain().ExecuteCandidate(context.Background(), candidates, fake.run)
			if err != nil {
				t.Fatalf("expected fallback to succeed: %v", err)
			}
			if result.Model != "backup" {
				t.Fatalf("expected the backup to answer, got %q", result.Model)
			}
			if len(result.Attempts) != 1 || result.Attempts[0].Reason != tc.reason {
				t.Fatalf("expected one %s attempt, got %+v", tc.reason, result.Attempts)
			}
		})
	}
}

func TestTimeoutFallsBack(t *testing.T) {
	fake := newFakeProvider(scriptTimeout(), scriptSuccess())
	candidates := testCandidates("primary", "backup")

	result, err := newTestChain().ExecuteCandidate(context.Background(), candidates, fake.run)
	if err != nil {
		t.Fatalf("expected fallback to succeed: %v", err)
	}
	if result.Model != "backup" {
		t.Fatalf("expected the backup to answer, got %q", result.Model)
	}
}

// With an alternative available, a Retry-After must not become a wait: the
// candidate is put into cooldown and the next one is tried at once.
func TestRetryAfterWithFallbackDoesNotWait(t *testing.T) {
	fake := newFakeProvider(
		scriptHTTPRetryAfter(429, "slow down", 90*time.Second),
		scriptSuccess(),
	)
	candidates := testCandidates("primary", "backup")
	chain := newTestChain()

	started := time.Now()
	result, err := chain.ExecuteCandidate(context.Background(), candidates, fake.run)
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("expected fallback to succeed: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("the chain waited %s instead of moving to the backup", elapsed)
	}
	if result.Model != "backup" {
		t.Fatalf("expected the backup to answer, got %q", result.Model)
	}
	remaining := chain.Cooldown().CooldownRemaining(candidates[0].StableKey())
	if remaining < 60*time.Second {
		t.Fatalf("Retry-After should extend the cooldown, got %s", remaining)
	}
}

func TestAuthFailureDoesNotRetrySameCandidate(t *testing.T) {
	fake := newFakeProvider(scriptHTTP(401, "invalid api key"), scriptSuccess())
	candidates := testCandidates("primary", "backup")

	result, err := newTestChain().ExecuteCandidate(context.Background(), candidates, fake.run)
	if err != nil {
		t.Fatalf("expected fallback to succeed: %v", err)
	}
	if calls := fake.callsFor("provider0", "primary"); calls != 1 {
		t.Fatalf("a bad credential must be tried once, got %d", calls)
	}
	if result.Model != "backup" {
		t.Fatalf("expected the backup to answer, got %q", result.Model)
	}

	classified := ClassifyError(&common.HTTPError{StatusCode: 401}, "p", "m")
	if classified.AllowsSameCandidateRetry() {
		t.Fatal("an invalid credential will not become valid on retry")
	}
}

func TestBillingFailureDoesNotRetrySameCandidate(t *testing.T) {
	classified := ClassifyError(errors.New("payment required: insufficient credits"), "p", "m")
	if classified == nil || classified.Reason != FailoverBilling {
		t.Fatalf("expected billing classification, got %+v", classified)
	}
	if classified.AllowsSameCandidateRetry() {
		t.Fatal("an unpaid bill will not be paid by retrying")
	}
}

func TestFormatErrorIsTerminal(t *testing.T) {
	fake := newFakeProvider(scriptHTTP(400, "invalid request format"), scriptSuccess())
	candidates := testCandidates("primary", "backup")

	_, err := newTestChain().ExecuteCandidate(context.Background(), candidates, fake.run)
	if err == nil {
		t.Fatal("a malformed request must not fall back; the next provider would reject it too")
	}
	if fake.callCount() != 1 {
		t.Fatalf("expected exactly one attempt, got %d", fake.callCount())
	}
}

// A candidate that cannot serve a tool-calling turn is skipped before it is
// called, not after it fails.
func TestToolUnsupportedCandidateIsSkipped(t *testing.T) {
	fake := newFakeProvider(scriptSuccess())
	candidates := []FallbackCandidate{
		{Provider: "p0", Model: "text-embedding-3-large"},
		{Provider: "p1", Model: "gpt-4o"},
	}

	result, err := newTestChain().WithToolTurn(true).
		ExecuteCandidate(context.Background(), candidates, fake.run)
	if err != nil {
		t.Fatalf("expected the tool-capable candidate to answer: %v", err)
	}
	if result.Model != "gpt-4o" {
		t.Fatalf("expected gpt-4o to answer, got %q", result.Model)
	}
	if fake.callsFor("p0", "text-embedding-3-large") != 0 {
		t.Fatal("an embedding model must never be called for a tool-calling turn")
	}
	if len(result.Attempts) != 1 || !result.Attempts[0].Skipped {
		t.Fatalf("the unusable candidate should be recorded as skipped, got %+v", result.Attempts)
	}
}

// Unknown must never be treated as unsupported, or failover would be disabled
// for every model PocketClaw does not happen to enumerate.
func TestUnknownCapabilityIsNotTreatedAsUnsupported(t *testing.T) {
	if got := SupportsToolCalls("brand-new-model-x"); got != CapabilityUnknown {
		t.Fatalf("an unrecognised model must be unknown, got %q", got)
	}
	usable, _ := CandidateUsableForToolTurn(
		FallbackCandidate{Provider: "p", Model: "brand-new-model-x"}, true,
	)
	if !usable {
		t.Fatal("an unknown model must still be attempted")
	}

	fake := newFakeProvider(scriptSuccess())
	candidates := []FallbackCandidate{{Provider: "p0", Model: "brand-new-model-x"}}
	if _, err := newTestChain().WithToolTurn(true).
		ExecuteCandidate(context.Background(), candidates, fake.run); err != nil {
		t.Fatalf("unknown capability must not block execution: %v", err)
	}
	if fake.callCount() != 1 {
		t.Fatalf("expected the unknown candidate to be tried, got %d calls", fake.callCount())
	}
}

func TestMalformedAndZeroByteResponsesClassify(t *testing.T) {
	malformed := ClassifyError(errors.New("failed to parse JSON response: unexpected EOF"), "p", "m")
	if malformed == nil || malformed.Reason != FailoverNetwork {
		t.Fatalf("a truncated body is a transport failure, got %+v", malformed)
	}

	zeroByte := ClassifyError(errors.New("read: EOF"), "p", "m")
	if zeroByte == nil || zeroByte.Reason != FailoverNetwork {
		t.Fatalf("a zero-byte body is a transport failure, got %+v", zeroByte)
	}
}

func TestCancellationDuringChainStopsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fake := newFakeProvider(scriptSuccess())
	candidates := testCandidates("primary", "backup")

	_, err := newTestChain().ExecuteCandidate(ctx, candidates, fake.run)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("user cancellation must win over provider recovery, got %v", err)
	}
	if fake.callCount() != 0 {
		t.Fatalf("no provider call should be made after cancellation, got %d", fake.callCount())
	}
}

func TestCooldownSkipsUnhealthyCandidateAndSuccessClearsIt(t *testing.T) {
	candidates := testCandidates("primary", "backup")
	chain := newTestChain()
	chain.Cooldown().MarkFailure(candidates[0].StableKey(), FailoverTimeout)

	fake := newFakeProvider(scriptSuccess())
	result, err := chain.ExecuteCandidate(context.Background(), candidates, fake.run)
	if err != nil {
		t.Fatalf("expected the healthy candidate to answer: %v", err)
	}
	if fake.callsFor("provider0", "primary") != 0 {
		t.Fatal("a candidate in cooldown must be skipped, not called")
	}
	if result.Model != "backup" {
		t.Fatalf("expected the backup to answer, got %q", result.Model)
	}

	chain.Cooldown().MarkSuccess(candidates[0].StableKey())
	if !chain.Cooldown().IsAvailable(candidates[0].StableKey()) {
		t.Fatal("a successful request must clear the cooldown")
	}
}

func TestFailoverExhaustedReportsEveryAttempt(t *testing.T) {
	fake := newFakeProvider(
		scriptHTTP(503, "unavailable"),
		scriptHTTP(503, "unavailable"),
	)
	candidates := testCandidates("primary", "backup")

	_, err := newTestChain().ExecuteCandidate(context.Background(), candidates, fake.run)
	if err == nil {
		t.Fatal("expected an aggregate failure once all candidates fail")
	}
	if !strings.Contains(err.Error(), "primary") || !strings.Contains(err.Error(), "backup") {
		t.Fatalf("the exhausted error should name every candidate tried: %v", err)
	}
}

func TestParseRetryAfterAcceptsBothFormsAndRefusesNonsense(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

	if got := common.ParseRetryAfter("120", now); got != 2*time.Minute {
		t.Fatalf("delta-seconds: got %s", got)
	}
	// http.TimeFormat is what servers actually send: RFC1123 with "GMT", which
	// is the only zone spelling http.ParseTime accepts.
	future := now.Add(90 * time.Second).Format(http.TimeFormat)
	if got := common.ParseRetryAfter(future, now); got < 80*time.Second {
		t.Fatalf("http-date: got %s", got)
	}
	// A stale or malformed header must never become an unbounded wait.
	past := now.Add(-time.Hour).Format(http.TimeFormat)
	if got := common.ParseRetryAfter(past, now); got != 0 {
		t.Fatalf("a past date must yield no delay, got %s", got)
	}
	for _, bad := range []string{"", "soon", "-5", "NaN"} {
		if got := common.ParseRetryAfter(bad, now); got != 0 {
			t.Fatalf("malformed %q must yield no delay, got %s", bad, got)
		}
	}
}
