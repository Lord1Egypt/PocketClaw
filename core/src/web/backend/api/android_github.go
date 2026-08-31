package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
		return "", fmt.Errorf("the managed runtime is unavailable: %w", err)
	}

	result, err := manager.Execute(ctx, pcruntime.ExecRequest{
		Tool:                 "gh",
		Args:                 []string{"api", "user"},
		EnvironmentAdditions: map[string]string{"GH_TOKEN": token},
		TimeoutMS:            githubTokenValidation.Milliseconds(),
	})
	if err != nil {
		return "", err
	}
	if result.Status == pcruntime.StatusUnavailable {
		return "", fmt.Errorf("gh is not available on this device: %s", result.Diagnostics)
	}
	if result.ExitCode != 0 {
		// gh does not echo GH_TOKEN, but its output is not something this code
		// controls, so it is scrubbed before it can reach a response or a log.
		return "", fmt.Errorf("GitHub rejected this token: %s",
			firstLine(scrubToken(result.Stderr, token)))
	}

	var account struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &account); err != nil {
		return "", fmt.Errorf("GitHub's reply could not be read")
	}
	if strings.TrimSpace(account.Login) == "" {
		return "", fmt.Errorf("GitHub did not name an account for this token")
	}
	return account.Login, nil
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
		// The candidate never appears in the reply, whatever gh said about it.
		http.Error(w, scrubToken(err.Error(), token), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		return "", fmt.Errorf("the managed runtime is unavailable: %w", err)
	}
	result, err := manager.Execute(ctx, pcruntime.ExecRequest{
		Tool:      "gh",
		Args:      []string{"api", "user"},
		TimeoutMS: githubTokenValidation.Milliseconds(),
	})
	if err != nil {
		return "", err
	}
	if result.Status == pcruntime.StatusUnavailable {
		return "", fmt.Errorf("gh is not available on this device")
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("GitHub authentication is not working: %s", firstLine(result.Stderr))
	}

	var account struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &account); err != nil {
		return "", fmt.Errorf("GitHub's reply could not be read")
	}
	if strings.TrimSpace(account.Login) == "" {
		return "", fmt.Errorf("GitHub did not name an authenticated account")
	}
	return account.Login, nil
}
