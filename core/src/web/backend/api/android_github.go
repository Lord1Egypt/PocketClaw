package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"

	"github.com/sipeed/picoclaw/pkg/pcruntime"
)

const androidGitHubValidatePath = "/api/pocketclaw/android/github/validate"
const androidGitHubStatusPath = "/api/pocketclaw/android/github/status"

// githubTokenValidation is how long one `gh api user` call may take. A device on
// a slow network should not leave the Settings screen waiting indefinitely, and
// a credential check that takes longer than this is not usable anyway.
const githubTokenValidation = 20 * time.Second

type androidGitHubValidateRequest struct {
	Token string `json:"token"`
}

type androidGitHubValidateResponse struct {
	Login string `json:"login"`
}

// GitHubTokenValidator checks one candidate credential and reports the account
// it belongs to.
//
// It is a field on the handler rather than a direct call so the endpoint can be
// tested without a gh binary on the host. The production implementation runs
// through the Managed Runtime, which is the only thing in PocketClaw allowed to
// execute a bundled tool.
type GitHubTokenValidator func(ctx context.Context, token string) (string, error)

func (h *Handler) validateGitHubToken(ctx context.Context, token string) (string, error) {
	if h.githubValidator != nil {
		return h.githubValidator(ctx, token)
	}
	return validateGitHubTokenThroughRuntime(ctx, token)
}

// validateGitHubTokenThroughRuntime asks GitHub who the candidate belongs to.
//
// The candidate reaches gh the same way a stored credential does — as GH_TOKEN
// in the constructed environment, never in the argument vector — and it is not
// written anywhere. Nothing about this call persists: gh is not asked to log in,
// so it writes no config of its own, and the token is gone when the process
// exits.
func validateGitHubTokenThroughRuntime(ctx context.Context, token string) (string, error) {
	manager, err := pcruntime.NewManager()
	if err != nil {
		return "", &GitHubFailure{
			Category: GitHubFailureUnavailable,
			Message:  "The PocketClaw runtime is not available.",
			Detail:   err.Error(),
		}
	}

	result, err := manager.Execute(ctx, pcruntime.ExecRequest{
		Tool: "gh",
		Args: []string{"api", "user"},
		EnvironmentAdditions: map[string]string{
			"GH_TOKEN": token,
			// gh's own summary is "error connecting to api.github.com", which
			// cannot distinguish a rejected credential from a network that never
			// carried the request. GH_DEBUG=1 makes it print the underlying error
			// as well. It prints no request headers -- that is GH_DEBUG=api -- so
			// nothing here can carry the credential, and it is scrubbed anyway.
			"GH_DEBUG": "1",
		},
		TimeoutMS: githubTokenValidation.Milliseconds(),
	})
	if err != nil {
		return "", &GitHubFailure{
			Category: GitHubFailureOther,
			Message:  "GitHub validation failed.",
			Detail:   scrubToken(err.Error(), token),
		}
	}
	if failure := classifyGitHubResult(result, token); failure != nil {
		return "", failure
	}

	var account struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &account); err != nil {
		return "", &GitHubFailure{
			Category: GitHubFailureOther,
			Message:  "GitHub's reply could not be read.",
			Detail:   "the response body was not the expected JSON",
		}
	}
	if strings.TrimSpace(account.Login) == "" {
		return "", &GitHubFailure{
			Category: GitHubFailureOther,
			Message:  "GitHub did not name an account for this token.",
		}
	}
	return account.Login, nil
}

// Categories a caller can act on. "Rejected" and "unreachable" call for
// completely different responses from the user, and reporting one as the other
// sends them to change a credential that was never checked.
const (
	GitHubFailureAuth         = "auth"
	GitHubFailureConnectivity = "connectivity"
	GitHubFailureTimeout      = "timeout"
	GitHubFailureUnavailable  = "unavailable"
	GitHubFailureOther        = "other"
)

// GitHubFailure is a validation failure the UI can classify, with a sanitized
// diagnostic for the logs. Detail never contains the credential.
type GitHubFailure struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

func (f *GitHubFailure) Error() string { return f.Message }

// HTTPStatus separates "your credential is wrong" from "PocketClaw could not
// ask", so the host is not left inferring it from prose.
func (f *GitHubFailure) HTTPStatus() int {
	switch f.Category {
	case GitHubFailureAuth:
		return http.StatusBadRequest
	case GitHubFailureConnectivity, GitHubFailureTimeout, GitHubFailureUnavailable:
		return http.StatusBadGateway
	default:
		return http.StatusBadRequest
	}
}

// classifyGitHubResult turns one gh run into a category, or nil if it succeeded.
//
// The evidence is gh's stderr, which with GH_DEBUG=1 carries the underlying
// transport error as well as gh's own summary. Everything returned from here is
// scrubbed of the candidate first: gh does not echo it, but a tool's output is
// not something this code controls.
func classifyGitHubResult(result *pcruntime.ExecResult, token string) *GitHubFailure {
	if result.Status == pcruntime.StatusUnavailable {
		return &GitHubFailure{
			Category: GitHubFailureUnavailable,
			Message:  "The bundled gh is not available on this device.",
			Detail:   scrubToken(result.Diagnostics, token),
		}
	}
	if result.TimedOut {
		return &GitHubFailure{
			Category: GitHubFailureTimeout,
			Message:  "GitHub connection timed out.",
			Detail:   fmt.Sprintf("gh was terminated after %dms", result.DurationMS),
			ExitCode: result.ExitCode,
		}
	}
	if result.ExitCode == 0 {
		return nil
	}

	detail := scrubToken(strings.TrimSpace(result.Stderr), token)
	lowered := strings.ToLower(detail)
	failure := &GitHubFailure{Detail: detail, ExitCode: result.ExitCode}

	switch {
	case containsAny(lowered, "http 401", "bad credentials", "requires authentication"):
		failure.Category = GitHubFailureAuth
		failure.Message = "GitHub rejected this token."
	case containsAny(lowered, "http 403", "forbidden", "insufficient scope", "missing the required scope"):
		failure.Category = GitHubFailureAuth
		failure.Message = "GitHub rejected this token: it lacks the required access."
	case containsAny(lowered, "no such host", "server misbehaving", "lookup "):
		failure.Category = GitHubFailureConnectivity
		failure.Message = "Could not connect to GitHub: its address could not be resolved."
	case containsAny(lowered, "x509", "certificate", "tls handshake"):
		failure.Category = GitHubFailureConnectivity
		failure.Message = "Could not connect to GitHub: the secure connection failed."
	case containsAny(lowered, "i/o timeout", "context deadline exceeded", "timeout awaiting"):
		failure.Category = GitHubFailureTimeout
		failure.Message = "GitHub connection timed out."
	case containsAny(lowered, "connection refused", "network is unreachable", "no route to host",
		"error connecting to", "dial tcp", "connection reset"):
		failure.Category = GitHubFailureConnectivity
		failure.Message = "Could not connect to GitHub."
	case result.ExitCode == 4:
		// gh's own "not logged in" exit. It means gh never made a request.
		failure.Category = GitHubFailureAuth
		failure.Message = "GitHub authentication is not configured."
	default:
		failure.Category = GitHubFailureOther
		failure.Message = "GitHub validation failed."
	}
	return failure
}

func containsAny(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

// scrubToken removes a credential from text that is about to be shown or
// logged. A tool's own error message is untrusted for this purpose: it may quote
// what it was given.
func scrubToken(text, token string) string {
	if strings.TrimSpace(token) == "" {
		return text
	}
	return strings.ReplaceAll(text, token, "[redacted]")
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		return strings.TrimSpace(text[:index])
	}
	if text == "" {
		return "no reason given"
	}
	return text
}

// handleAndroidGitHubValidate checks a candidate credential on the Android
// host's behalf and returns only the account name.
//
// The Android host owns the credential: it encrypts it under a key held in the
// Android Keystore and hands it back to Core as an environment variable when it
// starts the process. Core never writes it down, which is why validation lives
// here rather than in the host — running a bundled tool is the Managed
// Runtime's job, and duplicating that in Kotlin would be a second execution path
// with none of the resolution, verification or bounding this one has.
func (h *Handler) handleAndroidGitHubValidate(w http.ResponseWriter, r *http.Request) {
	var request androidGitHubValidateRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(request.Token)
	if token == "" {
		http.Error(w, "A GitHub token is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), githubTokenValidation+5*time.Second)
	defer cancel()

	login, err := h.validateGitHubToken(ctx, token)
	if err != nil {
		writeGitHubFailure(w, asGitHubFailure(err, token))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(androidGitHubValidateResponse{Login: login})
}

// handleAndroidGitHubStatus reports whether the credential Core was started
// with actually authenticates.
//
// It runs gh with no environment override, so it exercises the same path every
// agent gh call takes rather than re-checking what is in storage. "Is the token
// in the store still valid" and "is this running Core authenticated" are
// different questions, and only the second one tells the user whether their
// next gh command will work.
func (h *Handler) handleAndroidGitHubStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), githubTokenValidation+5*time.Second)
	defer cancel()

	login, err := h.checkGitHubAuthentication(ctx)
	if err != nil {
		writeGitHubFailure(w, asGitHubFailure(err, ""))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(androidGitHubValidateResponse{Login: login})
}

func (h *Handler) checkGitHubAuthentication(ctx context.Context) (string, error) {
	if h.githubValidator != nil {
		// Tests supply the same seam; an empty candidate means "whatever the
		// process is already configured with".
		return h.githubValidator(ctx, "")
	}
	return authenticatedGitHubLogin(ctx)
}

func authenticatedGitHubLogin(ctx context.Context) (string, error) {
	manager, err := pcruntime.NewManager()
	if err != nil {
		return "", &GitHubFailure{
			Category: GitHubFailureUnavailable,
			Message:  "The PocketClaw runtime is not available.",
			Detail:   err.Error(),
		}
	}
	result, err := manager.Execute(ctx, pcruntime.ExecRequest{
		Tool:                 "gh",
		Args:                 []string{"api", "user"},
		EnvironmentAdditions: map[string]string{"GH_DEBUG": "1"},
		TimeoutMS:            githubTokenValidation.Milliseconds(),
	})
	if err != nil {
		return "", &GitHubFailure{
			Category: GitHubFailureOther,
			Message:  "GitHub validation failed.",
			Detail:   err.Error(),
		}
	}
	if failure := classifyGitHubResult(result, ""); failure != nil {
		if failure.Category == GitHubFailureAuth && result.ExitCode == 4 {
			failure.Message = "GitHub authentication is not configured."
		}
		return "", failure
	}

	var account struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &account); err != nil {
		return "", &GitHubFailure{
			Category: GitHubFailureOther,
			Message:  "GitHub's reply could not be read.",
		}
	}
	if strings.TrimSpace(account.Login) == "" {
		return "", &GitHubFailure{
			Category: GitHubFailureOther,
			Message:  "GitHub did not name an authenticated account.",
		}
	}
	return account.Login, nil
}

// asGitHubFailure normalises anything the validator returned into a category.
func asGitHubFailure(err error, token string) *GitHubFailure {
	var failure *GitHubFailure
	if errors.As(err, &failure) {
		return failure
	}
	return &GitHubFailure{
		Category: GitHubFailureOther,
		Message:  "GitHub validation failed.",
		Detail:   scrubToken(err.Error(), token),
	}
}

// writeGitHubFailure answers the host and records the diagnostic.
//
// The detail goes to the log so it reaches the Debug Logs screen, where a
// connectivity fault can actually be diagnosed. It has already been scrubbed of
// the candidate, and no environment is dumped with it.
func writeGitHubFailure(w http.ResponseWriter, failure *GitHubFailure) {
	logger.WarnCF("github", "GitHub credential check failed", map[string]any{
		"category":  failure.Category,
		"exit_code": failure.ExitCode,
		"detail":    failure.Detail,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(failure.HTTPStatus())
	_ = json.NewEncoder(w).Encode(failure)
}
