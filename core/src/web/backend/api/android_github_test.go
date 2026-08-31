package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// canaryToken is unique enough that finding it anywhere is proof, not
// coincidence. Every assertion below searches for this exact string.
const canaryToken = "ghp_canary_0f3a9c7e_must_never_be_written_down"

func newGitHubBridge(t *testing.T, validator GitHubTokenValidator) (*httptest.Server, string) {
	t.Helper()
	handler := NewHandler(t.TempDir() + "/config.json")
	handler.SetGitHubTokenValidator(validator)

	const bridgeToken = "bridge-token-for-tests"
	mux := http.NewServeMux()
	handler.RegisterAndroidBridgeRoutes(mux, bridgeToken)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, bridgeToken
}

func bridgeRequest(t *testing.T, server *httptest.Server, bridgeToken, method, path, body string) (int, string) {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatalf("cannot build request: %v", err)
	}
	if bridgeToken != "" {
		request.Header.Set("X-PocketClaw-Android-Bridge", bridgeToken)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("cannot read response: %v", err)
	}
	return response.StatusCode, string(payload)
}

func TestValidateReturnsOnlyTheAccountName(t *testing.T) {
	var seen string
	server, bridgeToken := newGitHubBridge(t, func(_ context.Context, token string) (string, error) {
		seen = token
		return "octocat", nil
	})

	status, body := bridgeRequest(t, server, bridgeToken, "POST",
		androidGitHubValidatePath, `{"token":"`+canaryToken+`"}`)

	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if seen != canaryToken {
		t.Errorf("the validator received %q, not the candidate", seen)
	}

	var decoded androidGitHubValidateResponse
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("reply is not JSON: %v", err)
	}
	if decoded.Login != "octocat" {
		t.Errorf("login = %q, want octocat", decoded.Login)
	}
	if strings.Contains(body, canaryToken) {
		t.Error("the reply echoed the credential back to the host")
	}
}

// gh's own output is not something PocketClaw controls, and a tool that quotes
// what it was given would otherwise put the credential into an error message
// that reaches the UI and the log.
func TestValidationFailureNeverQuotesTheCandidate(t *testing.T) {
	server, bridgeToken := newGitHubBridge(t, func(_ context.Context, token string) (string, error) {
		return "", fmt.Errorf("gh: HTTP 401 while sending token %s", token)
	})

	status, body := bridgeRequest(t, server, bridgeToken, "POST",
		androidGitHubValidatePath, `{"token":"`+canaryToken+`"}`)

	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", status, body)
	}
	if strings.Contains(body, canaryToken) {
		t.Errorf("the error message carried the credential: %s", body)
	}
	if !strings.Contains(body, "[redacted]") {
		t.Errorf("the credential was removed without saying so: %s", body)
	}
}

func TestValidateRejectsAnEmptyCandidate(t *testing.T) {
	server, bridgeToken := newGitHubBridge(t, func(context.Context, string) (string, error) {
		t.Error("an empty candidate reached the validator")
		return "", nil
	})

	status, _ := bridgeRequest(t, server, bridgeToken, "POST",
		androidGitHubValidatePath, `{"token":"   "}`)
	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", status)
	}
}

// The bridge is the Android host's private channel. Without the shared token it
// must not even admit the route exists.
func TestGitHubBridgeRequiresTheHostToken(t *testing.T) {
	server, _ := newGitHubBridge(t, func(context.Context, string) (string, error) {
		t.Error("an unauthenticated request reached the validator")
		return "", nil
	})

	for _, route := range []struct {
		method string
		path   string
		body   string
	}{
		{"POST", androidGitHubValidatePath, `{"token":"x"}`},
		{"GET", androidGitHubStatusPath, ""},
	} {
		status, _ := bridgeRequest(t, server, "", route.method, route.path, route.body)
		if status != http.StatusNotFound {
			t.Errorf("%s %s answered %d without the bridge token, want 404",
				route.method, route.path, status)
		}
	}
}

func TestStatusReportsTheAuthenticatedAccount(t *testing.T) {
	server, bridgeToken := newGitHubBridge(t, func(_ context.Context, token string) (string, error) {
		if token != "" {
			t.Errorf("the status check supplied a candidate %q; it must use the ambient credential", token)
		}
		return "octocat", nil
	})

	status, body := bridgeRequest(t, server, bridgeToken, "GET", androidGitHubStatusPath, "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if !strings.Contains(body, "octocat") {
		t.Errorf("body = %s, want the account name", body)
	}
}

// With no credential configured the answer has to be a clear failure. Falling
// back to an interactive gh login would persist credentials outside PocketClaw,
// which is the whole thing this milestone prevents.
func TestStatusFailsClearlyWhenNoCredentialIsConfigured(t *testing.T) {
	server, bridgeToken := newGitHubBridge(t, func(context.Context, string) (string, error) {
		return "", fmt.Errorf("GitHub authentication is not working: gh: not logged in")
	})

	status, body := bridgeRequest(t, server, bridgeToken, "GET", androidGitHubStatusPath, "")
	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", status)
	}
	if !strings.Contains(body, "not working") {
		t.Errorf("body = %s, want an actionable reason", body)
	}
}

func TestScrubTokenRemovesEveryOccurrence(t *testing.T) {
	text := "sent " + canaryToken + " and retried with " + canaryToken
	scrubbed := scrubToken(text, canaryToken)
	if strings.Contains(scrubbed, canaryToken) {
		t.Errorf("scrubbed text still contains the credential: %s", scrubbed)
	}
	if strings.Count(scrubbed, "[redacted]") != 2 {
		t.Errorf("not every occurrence was replaced: %s", scrubbed)
	}
	// An empty credential must not turn every string into redactions.
	if got := scrubToken("nothing to hide", ""); got != "nothing to hide" {
		t.Errorf("scrubToken with no credential changed the text: %q", got)
	}
}
