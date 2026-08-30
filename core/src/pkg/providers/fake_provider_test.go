package providers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

// scriptedFailure describes one deterministic outcome for a fake provider call.
//
// Automated tests must never depend on a real provider being unhealthy, so
// every failure mode the resilience code claims to handle is reproduced here
// exactly, in order, with no timing dependence.
type scriptedFailure struct {
	err error
	// ok, when true, ends the script successfully for this call.
	ok bool
}

func scriptSuccess() scriptedFailure { return scriptedFailure{ok: true} }

func scriptHTTP(status int, body string) scriptedFailure {
	return scriptedFailure{err: &common.HTTPError{StatusCode: status, BodyPreview: body}}
}

func scriptHTTPRetryAfter(status int, body string, retryAfter time.Duration) scriptedFailure {
	return scriptedFailure{err: &common.HTTPError{
		StatusCode:  status,
		BodyPreview: body,
		RetryAfter:  retryAfter,
	}}
}

func scriptError(msg string) scriptedFailure {
	return scriptedFailure{err: errors.New(msg)}
}

func scriptTimeout() scriptedFailure {
	return scriptedFailure{err: context.DeadlineExceeded}
}

// fakeProvider replays a fixed script and counts how often it was called.
type fakeProvider struct {
	mu     sync.Mutex
	script []scriptedFailure
	calls  int
	// perCandidateCalls records calls keyed by provider/model so a test can
	// assert that a specific candidate was or was not retried.
	perCandidateCalls map[string]int
}

func newFakeProvider(script ...scriptedFailure) *fakeProvider {
	return &fakeProvider{script: script, perCandidateCalls: make(map[string]int)}
}

// run is the func the fallback chain drives.
func (f *fakeProvider) run(_ context.Context, candidate FallbackCandidate) (*LLMResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := ModelKey(candidate.Provider, candidate.Model)
	f.perCandidateCalls[key]++
	index := f.calls
	f.calls++

	if index >= len(f.script) {
		return &LLMResponse{Content: "ok"}, nil
	}
	step := f.script[index]
	if step.ok {
		return &LLMResponse{Content: "ok"}, nil
	}
	return nil, step.err
}

func (f *fakeProvider) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeProvider) callsFor(provider, model string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.perCandidateCalls[ModelKey(provider, model)]
}

func testCandidates(specs ...string) []FallbackCandidate {
	candidates := make([]FallbackCandidate, 0, len(specs))
	for i, spec := range specs {
		candidates = append(candidates, FallbackCandidate{
			Provider:    fmt.Sprintf("provider%d", i),
			Model:       spec,
			DisplayName: spec,
		})
	}
	return candidates
}

func newTestChain() *FallbackChain {
	return NewFallbackChain(NewCooldownTracker(), nil)
}
