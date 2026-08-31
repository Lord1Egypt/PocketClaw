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

	"github.com/sipeed/picoclaw/pkg/pcruntime"
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

// The message the user sees has to match what actually happened. "GitHub
// rejected this token" for a request that never left the device is not a
// wording problem: it sends the user to regenerate a credential that was never
// checked.
func TestFailuresAreClassifiedByWhatActuallyHappened(t *testing.T) {
	cases := []struct {
		name     string
		result   *pcruntime.ExecResult
		category string
		message  string
	}{
		{
			name: "android has no resolv.conf so Go falls back to localhost",
			result: &pcruntime.ExecResult{
				ExitCode: 1,
				Stderr: "* Request to https://api.github.com/user\n" +
					"* dial tcp: lookup api.github.com on [::1]:53: read udp [::1]:36500->[::1]:53: read: connection refused\n" +
					"error connecting to api.github.com\n",
			},
			category: GitHubFailureConnectivity,
			message:  "Could not connect to GitHub",
		},
		{
			name: "name resolution failed outright",
			result: &pcruntime.ExecResult{
				ExitCode: 1,
				Stderr:   "dial tcp: lookup api.github.com: no such host\n",
			},
			category: GitHubFailureConnectivity,
			message:  "address could not be resolved",
		},
		{
			name: "the certificate chain did not verify",
			result: &pcruntime.ExecResult{
				ExitCode: 1,
				Stderr:   "x509: certificate signed by unknown authority\n",
			},
			category: GitHubFailureConnectivity,
			message:  "secure connection failed",
		},
		{
			name: "the credential was actually rejected",
			result: &pcruntime.ExecResult{
				ExitCode: 1,
				Stderr:   "gh: Bad credentials (HTTP 401)\n",
			},
			category: GitHubFailureAuth,
			message:  "GitHub rejected this token",
		},
		{
			name: "the credential lacks a scope",
			result: &pcruntime.ExecResult{
				ExitCode: 1,
				Stderr:   "gh: Resource not accessible by personal access token (HTTP 403)\n",
			},
			category: GitHubFailureAuth,
			message:  "lacks the required access",
		},
		{
			name:     "the run exceeded its budget",
			result:   &pcruntime.ExecResult{ExitCode: -1, TimedOut: true, DurationMS: 20000},
			category: GitHubFailureTimeout,
			message:  "timed out",
		},
		{
			name: "the transport gave up waiting",
			result: &pcruntime.ExecResult{
				ExitCode: 1,
				Stderr:   "dial tcp 140.82.121.6:443: i/o timeout\n",
			},
			category: GitHubFailureTimeout,
			message:  "timed out",
		},
		{
			name: "gh has no credential at all",
			result: &pcruntime.ExecResult{
				ExitCode: 4,
				Stderr:   "To get started with GitHub CLI, please run: gh auth login\n",
			},
			category: GitHubFailureAuth,
			message:  "not configured",
		},
		{
			name: "something this code has not seen",
			result: &pcruntime.ExecResult{
				ExitCode: 1,
				Stderr:   "gh: an entirely novel problem\n",
			},
			category: GitHubFailureOther,
			message:  "GitHub validation failed",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			failure := classifyGitHubResult(testCase.result, "")
			if failure == nil {
				t.Fatal("a failing run was classified as a success")
			}
			if failure.Category != testCase.category {
				t.Errorf("category = %q, want %q (detail: %s)",
					failure.Category, testCase.category, failure.Detail)
			}
			if !strings.Contains(failure.Message, testCase.message) {
				t.Errorf("message = %q, want it to contain %q", failure.Message, testCase.message)
			}
			if failure.Category != GitHubFailureAuth &&
				strings.Contains(strings.ToLower(failure.Message), "rejected this token") {
				t.Errorf("a %s failure claimed the token was rejected: %q",
					failure.Category, failure.Message)
			}
		})
	}
}

func TestASuccessfulRunIsNotClassifiedAsAFailure(t *testing.T) {
	if failure := classifyGitHubResult(&pcruntime.ExecResult{
		ExitCode: 0, Stdout: `{"login":"octocat"}`,
	}, ""); failure != nil {
		t.Errorf("a successful run was classified as %+v", failure)
	}
}

// The sanitized diagnostic is what reaches the Debug Logs and the reply, so it
// is the one place the credential could still escape.
func TestTheDiagnosticNeverCarriesTheCandidate(t *testing.T) {
	failure := classifyGitHubResult(&pcruntime.ExecResult{
		ExitCode: 1,
		Stderr:   "gh: request with token " + canaryToken + " failed (HTTP 401)\n",
	}, canaryToken)

	if failure == nil {
		t.Fatal("a failing run was classified as a success")
	}
	if strings.Contains(failure.Detail, canaryToken) {
		t.Errorf("the diagnostic carried the credential: %s", failure.Detail)
	}
	if !strings.Contains(failure.Detail, "[redacted]") {
		t.Errorf("the credential was removed without saying so: %s", failure.Detail)
	}
	if failure.Category != GitHubFailureAuth {
		t.Errorf("category = %q, want auth", failure.Category)
	}
}

// A connectivity failure must not answer with the status a rejected credential
// would get, or the host cannot tell them apart without reading prose.
func TestConnectivityFailuresAnswerWithABadGateway(t *testing.T) {
	server, bridgeToken := newGitHubBridge(t, func(context.Context, string) (string, error) {
		return "", &GitHubFailure{
			Category: GitHubFailureConnectivity,
			Message:  "Could not connect to GitHub.",
			Detail:   "dial tcp: lookup api.github.com on [::1]:53: connection refused",
		}
	})

	status, body := bridgeRequest(t, server, bridgeToken, "POST",
		androidGitHubValidatePath, `{"token":"`+canaryToken+`"}`)

	if status != http.StatusBadGateway {
		t.Errorf("status = %d, want 502 for a connectivity failure", status)
	}
	var decoded GitHubFailure
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("the reply is not a structured failure: %v (%s)", err, body)
	}
	if decoded.Category != GitHubFailureConnectivity {
		t.Errorf("category = %q, want connectivity", decoded.Category)
	}
	if strings.Contains(body, canaryToken) {
		t.Error("the reply carried the credential")
	}
	if strings.Contains(strings.ToLower(body), "rejected this token") {
		t.Error("a connectivity failure was reported as a rejected credential")
	}
}
