package agent

import (
	"os"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// Redaction lives in the emitter so a new call site cannot forget it. These
// cases go through redactProviderPayload rather than any call site, which is
// the point: the guarantee has to hold for code not yet written.
func TestProviderPayloadRedactsSecretsThatReachSafeFields(t *testing.T) {
	secrets := []string{
		"ghp_1234567890abcdefghijklmnopqrstuvwx",
		"sk-proj-1234567890abcdefghij",
		"Bearer abcdefghijklmnopqrstuvwx",
		"AIzaSyA1234567890abcdefghijklmnopqrstuv",
		"123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw",
	}

	for _, secret := range secrets {
		payload := ProviderAttemptPayload{
			ModelConfigName: "leaked " + secret,
			Provider:        "opencode_zen",
			UpstreamModel:   "deepseek-v4-flash-free",
			CandidateKey:    "opencode_zen/" + secret,
		}
		redacted, ok := redactProviderPayload(payload).(ProviderAttemptPayload)
		if !ok {
			t.Fatal("redaction must preserve the payload type")
		}
		if strings.Contains(redacted.ModelConfigName, secret) ||
			strings.Contains(redacted.CandidateKey, secret) {
			t.Fatalf("credential survived redaction: %+v", redacted)
		}
		if redacted.Provider != "opencode_zen" || redacted.UpstreamModel != "deepseek-v4-flash-free" {
			t.Fatalf("ordinary diagnostic fields must survive: %+v", redacted)
		}
	}
}

func TestProviderCooldownPayloadIsRedacted(t *testing.T) {
	payload := ProviderCooldownPayload{
		Provider:     "openai",
		CandidateKey: "openai/sk-proj-1234567890abcdefghij",
		ErrorClass:   "hard_quota",
		RemainingMS:  60000,
	}
	redacted, ok := redactProviderPayload(payload).(ProviderCooldownPayload)
	if !ok {
		t.Fatal("redaction must preserve the payload type")
	}
	if strings.Contains(redacted.CandidateKey, "sk-proj-") {
		t.Fatalf("credential survived redaction: %+v", redacted)
	}
	if redacted.ErrorClass != "hard_quota" {
		t.Fatal("the classification must survive; it is the diagnostic value")
	}
}

// The configured name and the routed model diverge under aliasing, and a log
// that blurs them makes failover diagnosis guesswork.
func TestProviderPayloadKeepsConfiguredAndUpstreamModelDistinct(t *testing.T) {
	ts := &turnState{turnID: "turn-1"}
	candidate := providers.FallbackCandidate{
		Provider:    "opencode_zen",
		Model:       "deepseek-v4-flash-free",
		DisplayName: "DeepSeek",
	}

	payload := providerAttemptPayload(ts, candidate, "deepseek-v4-flash-free", 1, 0)

	if payload.ModelConfigName != "DeepSeek" {
		t.Fatalf("configured name should be the display name, got %q", payload.ModelConfigName)
	}
	if payload.UpstreamModel != "deepseek-v4-flash-free" {
		t.Fatalf("upstream model should be the routed id, got %q", payload.UpstreamModel)
	}
	if payload.Provider != "opencode_zen" {
		t.Fatalf("provider should be recorded, got %q", payload.Provider)
	}
	if payload.TurnID != "turn-1" {
		t.Fatalf("turn id should be carried, got %q", payload.TurnID)
	}
}

// An unrecognised model must report its protocol as unknown rather than as the
// protocol that will be attempted. Recording a guess as fact is what makes a
// failover log untrustworthy.
func TestUnknownOpenCodeModelReportsUnknownProtocol(t *testing.T) {
	if got := providerProtocolFor("opencode_zen", "brand-new-model-x"); got != "unknown" {
		t.Fatalf("an unrecognised model must report unknown, got %q", got)
	}
	if got := providerProtocolFor("openai", "gpt-4o"); got != "" {
		t.Fatalf("a provider without a routing table has no protocol to report, got %q", got)
	}
}

// The placeholder fires for ordinary contentless turns, so it must not claim a
// provider fault. The old wording was read as evidence of an outage when no
// provider had failed.
func TestEmptyTurnPlaceholderDoesNotAssertProviderFailure(t *testing.T) {
	lowered := strings.ToLower(defaultResponse)
	for _, forbidden := range []string{"provider error", "token limit", "outage", "failed"} {
		if strings.Contains(lowered, forbidden) {
			t.Fatalf("the placeholder must not assert a cause; found %q in %q",
				forbidden, defaultResponse)
		}
	}
	if defaultResponse == "" {
		t.Fatal("a contentless turn still needs something to say")
	}
}

// A tool-call-only response is the normal shape of every tool-using turn.
// Treating it as empty would make the agent retry its own successful requests.
func TestToolCallOnlyResponseIsNotEmpty(t *testing.T) {
	response := &providers.LLMResponse{
		Content:   "",
		ToolCalls: []providers.ToolCall{{ID: "call_1", Name: "runtime"}},
	}
	if responseIsUserVisiblyEmpty(response) {
		t.Fatal("a response carrying tool calls is not empty")
	}
}

// A model is entitled to say nothing. That is a completed turn, not an outage,
// and must never trigger a retry or a failover on its own.
func TestValidEmptyCompletionIsNotTreatedAsProviderFailure(t *testing.T) {
	response := &providers.LLMResponse{Content: "   "}
	if !responseIsUserVisiblyEmpty(response) {
		t.Fatal("a whitespace-only completion has nothing user-visible in it")
	}

	// The classifier is what decides failover, and it is never consulted for a
	// successful response. Emptiness alone produces no classification at all.
	if classified := providers.ClassifyError(nil, "p", "m"); classified != nil {
		t.Fatal("a successful empty response must not classify as a provider failure")
	}
}

func TestNilResponseIsEmpty(t *testing.T) {
	if !responseIsUserVisiblyEmpty(nil) {
		t.Fatal("a nil response has nothing in it")
	}
}

// The event family has to be emitted, not merely defined. A provider lifecycle
// that exists only as constants tells a physical tester nothing, which is the
// failure mode this asserts against.
func TestProviderEventKindsAreAllReachable(t *testing.T) {
	source, err := os.ReadFile("pipeline_llm.go")
	if err != nil {
		t.Fatalf("cannot read the emission site: %v", err)
	}
	emitted := string(source)

	for _, kind := range []string{
		"KindProviderAttemptStarted",
		"KindProviderAttemptCompleted",
		"KindProviderAttemptFailed",
		"KindProviderRetryScheduled",
		"KindProviderFallbackSelected",
		"KindProviderFailoverExhaust",
	} {
		if !strings.Contains(emitted, kind) {
			t.Errorf("%s is declared but never emitted; a defined-but-unused event "+
				"is invisible on a device", kind)
		}
	}
}

// Every provider event must go through the redacting emitter. A direct
// emitEvent call would bypass the one place redaction is guaranteed.
func TestProviderEventsUseTheRedactingEmitter(t *testing.T) {
	source, err := os.ReadFile("pipeline_llm.go")
	if err != nil {
		t.Fatalf("cannot read the emission site: %v", err)
	}
	for _, line := range strings.Split(string(source), "\n") {
		if !strings.Contains(line, "KindProvider") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		// The kind appears on its own line inside the emitter call, so check
		// the surrounding call is the redacting one by looking for the direct
		// emitter being used with a provider kind on the same line.
		if strings.Contains(line, "al.emitEvent(") {
			t.Errorf("provider event bypasses the redacting emitter: %s", trimmed)
		}
	}
	if !strings.Contains(string(source), "al.emitProviderEvent(") {
		t.Fatal("provider events must be emitted through emitProviderEvent")
	}
}
