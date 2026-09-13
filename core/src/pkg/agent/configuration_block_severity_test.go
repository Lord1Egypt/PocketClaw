package agent

import (
	"errors"
	"strings"
	"testing"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
)

// PC-DEF-057 follow-up. An expected configuration block must not be recorded as a
// runtime failure.
//
// The physical log showed PC-E-AI-004 -- "every configured AI model is disabled"
// -- arriving as `ERR agent > LLM call failed`, severity=error. Telegram and the
// gateway were healthy; the product was waiting on the owner. The user-facing
// reply is deliberately unchanged.

func TestConfigurationBlockIsWarningSeverity(t *testing.T) {
	severity := runtimeSeverityForAgentEvent(
		runtimeevents.KindAgentError,
		ErrorPayload{
			Stage:          "llm",
			Message:        "no model",
			Classification: ClassificationConfigurationBlocked,
		},
	)

	if severity != runtimeevents.SeverityWarn {
		t.Fatalf("severity = %v, want warn: a healthy runtime waiting on the owner "+
			"must not look like an outage", severity)
	}
}

// An ordinary failure keeps error severity. Fixing the classification must not
// downgrade real faults.
func TestAnUnclassifiedErrorKeepsErrorSeverity(t *testing.T) {
	severity := runtimeSeverityForAgentEvent(
		runtimeevents.KindAgentError,
		ErrorPayload{Stage: "llm", Message: "provider returned 500"},
	)

	if severity != runtimeevents.SeverityError {
		t.Fatalf("severity = %v, want error", severity)
	}
}

// Every existing caller constructs ErrorPayload without a classification, so the
// zero value has to mean "ordinary failure".
func TestTheZeroValueClassificationIsAnOrdinaryFailure(t *testing.T) {
	if (ErrorPayload{}).Classification == ClassificationConfigurationBlocked {
		t.Fatal("an unset classification must not read as a configuration block")
	}
	if runtimeSeverityForAgentEvent(
		runtimeevents.KindAgentError, ErrorPayload{},
	) != runtimeevents.SeverityError {
		t.Fatal("an unclassified error payload must stay error severity")
	}
}

// Other error-severity kinds are untouched.
func TestOtherErrorKindsAreUnaffected(t *testing.T) {
	for _, kind := range []runtimeevents.Kind{
		runtimeevents.KindAgentSubTurnOrphan,
		runtimeevents.KindProviderFailoverExhaust,
	} {
		if got := runtimeSeverityForAgentEvent(kind, nil); got != runtimeevents.SeverityError {
			t.Errorf("%v severity = %v, want error", kind, got)
		}
	}
}

// The user-facing half must be byte-identical to before: the owner asked for the
// log semantics to change, not the reply.
func TestTheUserFacingReplyIsUnchangedByTheReclassification(t *testing.T) {
	problem := newNoModelEnabled()

	if problem.Code != CodeNoModelEnabled {
		t.Fatalf("code = %q, want %q", problem.Code, CodeNoModelEnabled)
	}
	rendered := formatProcessingError(problem)
	for _, expected := range []string{
		"PocketClaw is connected",
		"every configured AI model is disabled",
		"Open PocketClaw, enable a model, then send this again.",
		CodeNoModelEnabled,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("the user-facing reply lost %q:\n%s", expected, rendered)
		}
	}
}

// The turn still ends, and the error still carries its user-facing identity after
// the pipeline returns it -- that is what formatProcessingError depends on.
func TestAUserFacingErrorSurvivesBeingReturnedFromTheTurn(t *testing.T) {
	returned := error(newNoModelEnabled())

	problem, ok := AsUserFacingError(returned)
	if !ok {
		t.Fatal("the user-facing identity must survive, or the chat window gets " +
			"the generic branch")
	}
	if problem.Code != CodeNoModelEnabled {
		t.Fatalf("code = %q", problem.Code)
	}

	// Also through a wrap, since other call sites wrap.
	wrapped := errors.Join(errors.New("context"), returned)
	if _, ok := AsUserFacingError(wrapped); !ok {
		t.Fatal("the identity must survive a wrap")
	}
}
