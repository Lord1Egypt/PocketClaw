package openai_compat

import (
	"net/http"
	"net/url"
	"testing"
)

// The two halves of the logging contract, at the call site the owner named as the
// exemplar: the line has to be reconstructable, and it must not carry a secret.

func TestSafeEndpointDropsQueryAndFragment(t *testing.T) {
	cases := map[string]string{
		"https://opencode.ai/zen/go/v1/chat/completions": "https://opencode.ai/zen/go/v1/chat/completions",
		// A provider that puts the key in the query string.
		"https://generativelanguage.googleapis.com/v1/models?key=AIzaSECRETVALUE123456": "https://generativelanguage.googleapis.com/v1/models",
		// A signed URL carries its signature in the query.
		"https://host.invalid/v1/x?X-Amz-Signature=deadbeefSECRET&expires=1": "https://host.invalid/v1/x",
		// Credentials in userinfo must not survive either.
		"https://user:PASSWORDSECRET@host.invalid/v1/chat":         "https://host.invalid/v1/chat",
		"https://host.invalid/v1/chat#access_token=SECRETFRAGMENT": "https://host.invalid/v1/chat",
	}

	for raw, want := range cases {
		if got := safeEndpoint(raw); got != want {
			t.Errorf("safeEndpoint(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestSafeEndpointRefusesToEchoAnUnparseableValue(t *testing.T) {
	// A misconfigured api_base can be anything at all, so it is described rather
	// than repeated.
	if got := safeEndpoint("ht tp://%%%broken"); got != "<unparseable>" {
		t.Errorf("got %q, want <unparseable>", got)
	}
	if got := safeEndpoint("/relative/only"); got != "<relative>" {
		t.Errorf("got %q, want <relative>", got)
	}
}

// The header facts are booleans, so the line answers "was a credential attached"
// without being able to carry one.
func TestSessionHeaderPresenceIsDetectedWithoutItsValue(t *testing.T) {
	req := &http.Request{Header: http.Header{}, URL: &url.URL{}}
	if hasSessionHeader(req) {
		t.Error("no session header was set")
	}

	req.Header.Set("X-OpenCode-Session", "SECRETSESSIONVALUE")
	if !hasSessionHeader(req) {
		t.Error("the session header must be detected")
	}
}
