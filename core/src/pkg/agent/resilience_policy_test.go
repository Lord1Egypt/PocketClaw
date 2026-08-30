package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/common"
)

// Once any part of the answer is visible to the user, recovery must stop. A
// transparent failover would restart the response on another provider and
// duplicate the text already on screen, which is worse than the failure.
func TestVisibleStreamFailureIsNotRetried(t *testing.T) {
	visible := configuredStreamingVisibleError{err: errors.New("stream broke mid-answer")}
	if !isConfiguredStreamingVisibleError(visible) {
		t.Fatal("a post-publish stream failure must be recognised as visible")
	}

	// The same underlying error before anything was published is an ordinary
	// transport failure and stays eligible for recovery.
	if isConfiguredStreamingVisibleError(errors.New("stream broke mid-answer")) {
		t.Fatal("a bare transport error must not be mistaken for visible output")
	}
}

// Classification decides recovery, and these four will fail identically a
// second later. Retrying them is pure latency the user pays for nothing.
func TestTerminalClassesRefuseSameCandidateRetry(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"invalid credential", &common.HTTPError{StatusCode: 401, BodyPreview: "invalid api key"}},
		{"unpaid bill", &common.HTTPError{StatusCode: 402, BodyPreview: "payment required"}},
		{"exhausted quota", &common.HTTPError{StatusCode: 429, BodyPreview: "quota exceeded"}},
		{"malformed request", &common.HTTPError{StatusCode: 400, BodyPreview: "invalid request format"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			classified := providers.ClassifyError(tc.err, "p", "m")
			if classified == nil {
				t.Fatal("expected a classification")
			}
			if classified.AllowsSameCandidateRetry() {
				t.Fatalf("%s must not be retried on the same candidate", tc.name)
			}
		})
	}
}

// Transient conditions are the opposite case and must stay retriable, or the
// resilience work would have made things worse.
func TestTransientClassesAllowSameCandidateRetry(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"gateway timeout", &common.HTTPError{StatusCode: 504}},
		{"bad gateway", &common.HTTPError{StatusCode: 502}},
		{"overloaded", &common.HTTPError{StatusCode: 503}},
		{"throttled", &common.HTTPError{StatusCode: 429, BodyPreview: "rate limit"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			classified := providers.ClassifyError(tc.err, "p", "m")
			if classified == nil {
				t.Fatal("expected a classification")
			}
			if !classified.AllowsSameCandidateRetry() {
				t.Fatalf("%s is transient and may be retried", tc.name)
			}
		})
	}
}

// Backoff must never outlive a cancelled turn. A user who stops a turn should
// not wait out a provider's retry window first.
func TestBackoffStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	started := time.Now()
	err := sleepWithContext(ctx, 10*time.Second)
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("a cancelled backoff must report that it was cut short")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("cancellation must stop the wait immediately, took %s", elapsed)
	}
}

// A provider's Retry-After is honoured, but never without a ceiling: an
// unbounded sleep is indistinguishable from a hang to the user.
func TestRetryAfterIsCapped(t *testing.T) {
	if maxHonouredRetryAfter <= 0 {
		t.Fatal("there must be a ceiling on an honoured Retry-After")
	}
	if maxHonouredRetryAfter > time.Minute {
		t.Fatalf("a ceiling of %s is long enough to read as a hang", maxHonouredRetryAfter)
	}

	classified := providers.ClassifyError(
		&common.HTTPError{StatusCode: 429, BodyPreview: "slow down", RetryAfter: 10 * time.Minute},
		"p", "m",
	)
	if classified == nil || classified.RetryAfter != 10*time.Minute {
		t.Fatalf("the provider's instruction should be carried verbatim, got %+v", classified)
	}
	// The cap is applied where the wait happens, not where it is classified,
	// so the raw value stays available for cooldown.
}
