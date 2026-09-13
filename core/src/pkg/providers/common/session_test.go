package common

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// PC-DEF-032. The conversation identity a gateway routes on has to be stable
// for a conversation and must not be the conversation's own key.

var uuidShape = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestStableSessionIDIsUUIDShaped(t *testing.T) {
	got := StableSessionID("telegram:123456789")
	if !uuidShape.MatchString(got) {
		t.Fatalf("StableSessionID() = %q, which is not the UUID shape the gateway accepts", got)
	}
}

// The whole point: one conversation, one value, however many requests it makes.
// This covers the owner's items 2, 3 and 4 at the level they are actually
// decided — turns, tool continuations and retries all re-read the same scope
// from the same turn options, so identity is a property of the scope alone.
func TestStableSessionIDIsStableForOneScope(t *testing.T) {
	const scope = "telegram:123456789"

	first := StableSessionID(scope)
	for i := 0; i < 100; i++ {
		if got := StableSessionID(scope); got != first {
			t.Fatalf("call %d returned %q, want %q: a per-request value defeats "+
				"the header's purpose", i, got, first)
		}
	}
}

func TestStableSessionIDDiffersPerConversation(t *testing.T) {
	seen := map[string]string{}
	for _, scope := range []string{
		"telegram:111", "telegram:222", "web:session-a", "web:session-b", "agent:main",
	} {
		id := StableSessionID(scope)
		if previous, clash := seen[id]; clash {
			t.Fatalf("scopes %q and %q share session id %q", previous, scope, id)
		}
		seen[id] = scope
	}
}

// A session key carries the channel and the chat or user id. On Telegram that
// is the owner's own account, and it must not reach a third party.
func TestStableSessionIDDoesNotCarryTheScope(t *testing.T) {
	const chatID = "123456789"
	id := StableSessionID("telegram:" + chatID)

	if strings.Contains(id, chatID) {
		t.Fatalf("the chat id survived into the session id: %q", id)
	}
	if strings.Contains(strings.ToLower(id), "telegram") {
		t.Fatalf("the channel name survived into the session id: %q", id)
	}
}

// Item 6. The salt is random and the scope is a session key; a credential is
// neither, and must not become one by accident.
func TestStableSessionIDNeverContainsACredential(t *testing.T) {
	const apiKey = "sk-proj-AbCdEf0123456789AbCdEf0123456789"

	// Even when a caller passes something that carries a key, the output is a
	// fixed-width digest with no fragment of the input in it.
	for _, scope := range []string{apiKey, "telegram:1|" + apiKey, ""} {
		id := StableSessionID(scope)
		if strings.Contains(id, "sk-") || strings.Contains(id, "AbCdEf0123456789") {
			t.Fatalf("credential material appeared in the session id: %q", id)
		}
		if !uuidShape.MatchString(id) {
			t.Fatalf("StableSessionID(%q) = %q, not UUID-shaped", scope, id)
		}
	}
}

// Summarisation and context compaction call a provider outside any one
// conversation. They still need a header, because "no header" is the 400.
func TestStableSessionIDHasAProcessFallback(t *testing.T) {
	blank := StableSessionID("")
	if blank == "" {
		t.Fatal("an empty scope produced no session id, which would omit the header")
	}
	if blank != StableSessionID("   ") {
		t.Fatal("whitespace and empty are not the same absence")
	}
	if blank == StableSessionID("telegram:1") {
		t.Fatal("the fallback collides with a real conversation")
	}
}

func TestSessionIDFromOptions(t *testing.T) {
	want := StableSessionID("web:abc")
	if got := SessionIDFromOptions(map[string]any{SessionOptionKey: "web:abc"}); got != want {
		t.Fatalf("SessionIDFromOptions() = %q, want %q", got, want)
	}
	// A missing or wrongly typed scope falls back rather than failing the turn.
	if got := SessionIDFromOptions(map[string]any{}); got != StableSessionID("") {
		t.Fatalf("a missing scope did not fall back: %q", got)
	}
	if got := SessionIDFromOptions(map[string]any{SessionOptionKey: 42}); got != StableSessionID("") {
		t.Fatalf("a non-string scope did not fall back: %q", got)
	}
}

// Item 7, at the helper: a provider with no session header configured sends
// none. The opt-in is what keeps this off every unrelated service.
func TestApplySessionHeaderIsOptIn(t *testing.T) {
	options := map[string]any{SessionOptionKey: "telegram:1"}

	req := httptest.NewRequest(http.MethodPost, "https://example.test/v1/chat/completions", nil)
	ApplySessionHeader(req, "", options)
	if len(req.Header) != 0 {
		t.Fatalf("a provider with no session header sent one: %v", req.Header)
	}

	ApplySessionHeader(req, "x-opencode-session", options)
	if got := req.Header.Get("x-opencode-session"); got != StableSessionID("telegram:1") {
		t.Fatalf("x-opencode-session = %q, want the scope's session id", got)
	}
}
