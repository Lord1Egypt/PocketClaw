package agent

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

func httpError(status int, body string) error {
	return &common.HTTPError{StatusCode: status, BodyPreview: body, ContentType: "application/json"}
}

// The physical defect: OpenCode Zen rejects a model that only OpenCode Go
// serves, and the user was shown "request rejected (400)" with nothing to act
// on. The provider's own sentence names the model and the problem.
func TestProviderErrorDetailSurfacesOpenCodeModelRejection(t *testing.T) {
	err := httpError(400,
		`{"type":"error","error":{"type":"ModelError","message":"Model deepseek-v4.1-flash is not supported"}}`)

	got := providerErrorDetail(err)
	want := "Model deepseek-v4.1-flash is not supported"
	if got != want {
		t.Fatalf("providerErrorDetail = %q, want %q", got, want)
	}
}

// The whole reason this reaches the user at all is that it reaches them through
// the sentence they actually read.
func TestFormatProcessingErrorCarriesTheProviderSentence(t *testing.T) {
	err := httpError(400,
		`{"error":{"message":"Model deepseek-v4.1-flash is not supported"}}`)

	got := formatProcessingError(err)
	if !strings.Contains(got, "request rejected (400)") {
		t.Fatalf("lost the classification: %q", got)
	}
	if !strings.Contains(got, "Model deepseek-v4.1-flash is not supported") {
		t.Fatalf("lost the provider detail: %q", got)
	}
}

func TestProviderErrorDetailReadsCommonBodyShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"openai nested", `{"error":{"message":"model not found","type":"invalid_request_error"}}`, "model not found"},
		{"flat error string", `{"error":"model not found"}`, "model not found"},
		{"top level message", `{"message":"model not found"}`, "model not found"},
		{"detail field", `{"detail":"model not found"}`, "model not found"},
		{"not json", `Bad Request`, ""},
		{"empty message", `{"error":{"message":"   "}}`, ""},
		{"no known field", `{"code":400,"request_id":"abc"}`, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := providerErrorDetail(httpError(400, tc.body)); got != tc.want {
				t.Fatalf("providerErrorDetail = %q, want %q", got, tc.want)
			}
		})
	}
}

// A provider that echoes the key back is the case this whole file has to be
// safe against, because several of them do.
func TestProviderErrorDetailRedactsEchoedCredentials(t *testing.T) {
	secrets := []string{
		"sk-proj-AbCdEf0123456789AbCdEf0123456789",
		"AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7",
		"ghp_AbCdEf0123456789AbCdEf0123456789AbCd",
		"123456789:AAHfSomeTelegramBotTokenValue0123456789",
		"Bearer AbCdEf0123456789AbCdEf0123456789",
	}

	for _, secret := range secrets {
		body := fmt.Sprintf(`{"error":{"message":"Incorrect API key provided: %s"}}`, secret)
		got := providerErrorDetail(httpError(401, body))
		if got == "" {
			t.Fatalf("detail was dropped entirely for %q; the explanation is still wanted", secret[:8])
		}
		if strings.Contains(got, secret) {
			t.Fatalf("credential survived redaction: %q", got)
		}
		// The distinctive part of any of these must be gone, not just the prefix.
		if strings.Contains(got, "AbCdEf0123456789") || strings.Contains(got, "SomeTelegramBotToken") {
			t.Fatalf("credential body survived redaction: %q", got)
		}
	}
}

// Model ids are the thing the message exists to name, so redaction must not eat
// them.
func TestProviderErrorDetailKeepsModelIdentifiers(t *testing.T) {
	for _, model := range []string{
		"deepseek-v4.1-flash",
		"claude-sonnet-4-5",
		"gpt-5.1-codex-max",
		"gemini-3.5-flash-lite",
		"muse-spark-1.3-contributor",
	} {
		body := fmt.Sprintf(`{"error":{"message":"Model %s is not supported"}}`, model)
		got := providerErrorDetail(httpError(400, body))
		if !strings.Contains(got, model) {
			t.Fatalf("model id %q did not survive sanitisation: %q", model, got)
		}
	}
}

func TestProviderErrorDetailDropsLinksAndCollapsesToOneLine(t *testing.T) {
	body := `{"error":{"message":"Quota exhausted.\nSee https://example.test/billing?token=abc for details."}}`
	got := providerErrorDetail(httpError(429, body))

	if strings.ContainsAny(got, "\n\r") {
		t.Fatalf("detail is not one line: %q", got)
	}
	if strings.Contains(got, "https://") || strings.Contains(got, "token=abc") {
		t.Fatalf("link survived: %q", got)
	}
	if !strings.Contains(got, "Quota exhausted.") {
		t.Fatalf("lost the explanation: %q", got)
	}
}

func TestProviderErrorDetailCapsLength(t *testing.T) {
	long := strings.Repeat("word ", 200)
	body := fmt.Sprintf(`{"error":{"message":%q}}`, long)

	got := providerErrorDetail(httpError(400, body))
	if len([]rune(got)) > providerDetailMaxRunes+1 {
		t.Fatalf("detail was not capped: %d runes", len([]rune(got)))
	}
}

// An HTML body means the request never reached the provider's API. Quoting a
// captive portal or a proxy error page as "the provider said" would be wrong.
func TestProviderErrorDetailIgnoresHTMLBodies(t *testing.T) {
	err := &common.HTTPError{
		StatusCode:  400,
		BodyPreview: "<html><body>Bad Request</body></html>",
		ContentType: "text/html",
		IsHTML:      true,
	}
	if got := providerErrorDetail(err); got != "" {
		t.Fatalf("quoted an HTML error page: %q", got)
	}
}

func TestProviderErrorDetailIgnoresNonHTTPErrors(t *testing.T) {
	if got := providerErrorDetail(errors.New("dial tcp: connection refused")); got != "" {
		t.Fatalf("providerErrorDetail = %q, want empty", got)
	}
}
