package agent

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode"

	"github.com/sipeed/picoclaw/pkg/pcruntime"
	"github.com/sipeed/picoclaw/pkg/providers/common"
)

// The one sentence a provider is allowed to say to the user.
//
// PC-DEF-032. Collapsing every 4xx into "request rejected (400)" threw away the
// only part of the response that tells someone what to do next: OpenCode Zen
// answers a Go-only model with `Model deepseek-v4.1-flash is not supported`,
// and the user saw a bare status code instead. That is not a secret, it is the
// answer.
//
// The rule this file replaces was "a provider's own response text never reaches
// the user", and the reason for it still holds for the *body*: raw JSON,
// request ids, billing URLs and account internals are written for an API client
// and belong in the developer log. So the narrowing is deliberately small —
// exactly one known message field, redacted, single-line and length-capped.
// Nothing else from the response is repeated anywhere.

// providerDetailMaxRunes bounds what is quoted back. A provider message that
// runs longer than this is prose, a stack trace or an echoed request, none of
// which a chat window should carry.
const providerDetailMaxRunes = 200

// messageKeys are the fields providers put a human explanation in, in the order
// they are preferred. Nested `error` objects are unwrapped one level, which
// covers OpenAI, Anthropic, Gemini and the OpenAI-compatible gateways.
var messageKeys = []string{"message", "detail", "error_description", "msg", "reason"}

// highEntropyTokenPattern catches credential-shaped material that
// pcruntime.RedactText's per-vendor patterns do not know about.
//
// Mixed case plus a digit across 20+ credential-alphabet characters is what an
// opaque key looks like and what a model id does not: `deepseek-v4.1-flash`,
// `claude-sonnet-4-5` and `gpt-5.1-codex-max` are lowercase and carry
// separators this class excludes, so they survive intact.
var highEntropyTokenPattern = regexp.MustCompile(`[A-Za-z0-9_\-+/=]{20,}`)

// urlPattern removes links. A documentation link is not worth the risk that a
// provider embedded a signed URL or an account identifier in one.
var urlPattern = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://\S+`)

// providerErrorDetail returns the provider's own explanation, safe to show.
//
// It returns "" whenever there is nothing trustworthy to say: a non-HTTP
// failure, an HTML error page (which is a proxy or api_base problem, not a
// provider message), a body that is not JSON, or a message that is empty once
// redacted. An empty result means the caller reports the classification alone,
// exactly as before.
func providerErrorDetail(err error) string {
	var httpErr *common.HTTPError
	if !errors.As(err, &httpErr) || httpErr == nil {
		return ""
	}
	// An HTML body is a gateway, captive portal or wrong-api_base page. It has
	// no provider message in it and its text is not worth quoting.
	if httpErr.IsHTML {
		return ""
	}
	return sanitizeProviderDetail(extractProviderMessage(httpErr.BodyPreview))
}

// extractProviderMessage pulls the human-readable field out of a JSON body.
//
// Only JSON is read. A body that does not parse is passed through to nothing:
// echoing arbitrary bytes is the behaviour this file exists to avoid.
func extractProviderMessage(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	var decoded map[string]any
	if json.Unmarshal([]byte(body), &decoded) != nil {
		return ""
	}

	// `error` is either the message itself or the object carrying it.
	switch node := decoded["error"].(type) {
	case string:
		if strings.TrimSpace(node) != "" {
			return node
		}
	case map[string]any:
		if msg := firstStringField(node); msg != "" {
			return msg
		}
	}

	return firstStringField(decoded)
}

func firstStringField(node map[string]any) string {
	for _, key := range messageKeys {
		if value, ok := node[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// sanitizeProviderDetail makes one provider sentence safe to repeat.
func sanitizeProviderDetail(message string) string {
	if strings.TrimSpace(message) == "" {
		return ""
	}

	// Redaction runs before anything else, so a credential cannot survive by
	// being split across a line break or padded past the length cap.
	message = pcruntime.RedactText(message)
	message = urlPattern.ReplaceAllString(message, "<link>")
	message = highEntropyTokenPattern.ReplaceAllString(message, "<redacted>")

	// One line. Control characters, including the newlines a stack trace uses
	// to become a wall of text in a chat window, collapse to single spaces.
	message = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || unicode.IsControl(r) {
			return ' '
		}
		return r
	}, message)
	message = strings.Join(strings.Fields(message), " ")

	if message == "" {
		return ""
	}

	runes := []rune(message)
	if len(runes) > providerDetailMaxRunes {
		message = strings.TrimSpace(string(runes[:providerDetailMaxRunes])) + "…"
	}
	return message
}
