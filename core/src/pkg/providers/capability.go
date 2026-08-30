package providers

import "strings"

// Capability is a tri-state answer about what a model can do.
//
// The third state carries real weight: "we do not know" is not "no". PocketClaw
// routes to providers whose model lists it does not enumerate, and refusing to
// use a candidate merely because it is unrecognised would disable failover for
// exactly the configurations that need it most.
type Capability string

const (
	CapabilitySupported   Capability = "supported"
	CapabilityUnsupported Capability = "unsupported"
	CapabilityUnknown     Capability = "unknown"
)

// nonConversationalModelMarkers identify model families that are not chat
// models at all, and therefore cannot serve a tool-calling turn.
//
// This list is deliberately tiny and confined to model classes whose purpose is
// unambiguous from their identifier. It is not an attempt to enumerate which
// chat models support tool calling: that varies by provider, by version and by
// endpoint, and guessing it wrong would either skip a working candidate or let
// a doomed one through while claiming it was checked.
var nonConversationalModelMarkers = []string{
	"embedding",
	"-embed",
	"text-embed",
	"whisper",
	"-tts",
	"tts-",
	"dall-e",
	"moderation",
	"rerank",
}

// SupportsToolCalls reports whether a model can take part in a tool-calling
// turn.
//
// Only definitive negatives are returned. Everything else is Unknown, which
// callers must treat as "may be attempted", not as a failure.
func SupportsToolCalls(model string) Capability {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return CapabilityUnknown
	}
	for _, marker := range nonConversationalModelMarkers {
		if strings.Contains(normalized, marker) {
			return CapabilityUnsupported
		}
	}
	return CapabilityUnknown
}

// CandidateUsableForToolTurn reports whether a fallback candidate may be used
// while a tool-calling turn is in flight.
//
// A candidate is rejected only when it is explicitly known to be unusable. This
// is the whole capability gate: a small, conservative filter rather than a
// capability framework, because the data to support a larger one does not
// exist and inventing it would be worse than not having it.
func CandidateUsableForToolTurn(candidate FallbackCandidate, turnHasTools bool) (bool, string) {
	if !turnHasTools {
		return true, ""
	}
	if SupportsToolCalls(candidate.Model) == CapabilityUnsupported {
		return false, "model is not a conversational model and cannot serve a tool-calling turn"
	}
	return true, ""
}
