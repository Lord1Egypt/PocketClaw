package pcruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// profileFixture stages a system tool carrying an environment profile so the
// prepared environment can be observed through a real execution.
func profileFixture(t *testing.T, profile EnvironmentProfile) (*Manager, string) {
	t.Helper()
	tool := systemTool("dumpenv", TimeoutQuick, 65536)
	tool.EnvironmentProfile = profile
	manager, binDir := newTestManager(t, newTestManifest(t, tool))
	writeScript(t, binDir, "dumpenv", "env\n")
	return manager, binDir
}

func TestGitProfileDisablesInteractivePrompts(t *testing.T) {
	manager, _ := profileFixture(t, EnvironmentProfileGit)

	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	// Without these git can block on a terminal that does not exist, turning a
	// missing credential into an opaque timeout.
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("git profile did not set %s: %q", want, result.Stdout)
		}
	}
}

// git and gh must not read or write the user's workspace as their HOME, where a
// Skill could edit their config.
func TestGitProfileUsesAPrivateHomeOutsideTheWorkspace(t *testing.T) {
	manager, _ := profileFixture(t, EnvironmentProfileGit)

	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	var home string
	for _, line := range strings.Split(result.Stdout, "\n") {
		if strings.HasPrefix(line, "HOME=") {
			home = strings.TrimPrefix(line, "HOME=")
		}
	}
	if home == "" {
		t.Fatalf("git profile set no HOME: %q", result.Stdout)
	}
	if strings.HasPrefix(home, manager.workspace) {
		t.Fatalf("git HOME %q is inside the user workspace", home)
	}
	if !strings.HasPrefix(home, manager.registry.paths.MetadataDir) {
		t.Fatalf("git HOME %q is not in runtime metadata storage", home)
	}
}

// The credential is injected through git's environment-based config, never
// through argv, so it cannot be read from a process listing.
func TestGitCredentialIsInjectedThroughConfigEnvironmentNotArgv(t *testing.T) {
	const token = "ghp_1234567890abcdefghijklmnopqrstuvwx"
	t.Setenv(EnvGitHubToken, token)

	manager, _ := profileFixture(t, EnvironmentProfileGit)
	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}

	if !strings.Contains(result.Stdout, "GIT_CONFIG_KEY_0=http.https://github.com/.extraheader") {
		t.Fatalf("git credential config was not injected: %q", result.Stdout)
	}
	// The token must not appear verbatim: it is carried as Basic credentials.
	if strings.Contains(result.Stdout, token) {
		t.Fatal("the raw token reached the child environment in plaintext form")
	}
	if !strings.Contains(result.Stdout, "GIT_CONFIG_VALUE_0=Authorization: Basic ") {
		t.Fatalf("expected Basic credentials in git config: %q", result.Stdout)
	}
}

func TestNoCredentialIsInjectedWhenNoneIsConfigured(t *testing.T) {
	t.Setenv(EnvGitHubToken, "")

	manager, _ := profileFixture(t, EnvironmentProfileGit)
	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if strings.Contains(result.Stdout, "GIT_CONFIG_COUNT") {
		t.Fatalf("credential config was injected with no credential configured: %q", result.Stdout)
	}
}

// The Android Service can hold the token in memory, but a file fallback lets it
// live app-private, outside the workspace, without a UI existing yet.
func TestGitHubTokenIsReadFromAppPrivateStorage(t *testing.T) {
	t.Setenv(EnvGitHubToken, "")

	manager, _ := profileFixture(t, EnvironmentProfileGit)
	credentials := filepath.Join(manager.registry.paths.MetadataDir, "credentials")
	if err := os.MkdirAll(credentials, 0o700); err != nil {
		t.Fatalf("cannot create credential dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(credentials, "github_token"),
		[]byte("ghp_abcdefghijklmnopqrstuvwxyz012345\n"), 0o600); err != nil {
		t.Fatalf("cannot write token: %v", err)
	}

	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !strings.Contains(result.Stdout, "GIT_CONFIG_VALUE_0=Authorization: Basic ") {
		t.Fatalf("token file was not used: %q", result.Stdout)
	}
}

func TestGHProfileSuppressesInteractionAndUpdateChecks(t *testing.T) {
	manager, _ := profileFixture(t, EnvironmentProfileGH)

	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	for _, want := range []string{"GH_NO_UPDATE_NOTIFIER=1", "GH_PROMPT_DISABLED=1"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("gh profile did not set %s: %q", want, result.Stdout)
		}
	}
}

func TestGHProfileInjectsTheTokenAsGHToken(t *testing.T) {
	const token = "ghp_1234567890abcdefghijklmnopqrstuvwx"
	t.Setenv(EnvGitHubToken, token)

	manager, _ := profileFixture(t, EnvironmentProfileGH)
	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !strings.Contains(result.Stdout, "GH_TOKEN="+token) {
		t.Fatalf("gh did not receive its token: %q", result.Stdout)
	}
}

// A caller must not be able to undo the loader protections by overwriting a
// value the profile set.
func TestCallerAdditionsCannotOverrideDeniedVariables(t *testing.T) {
	manager, _ := profileFixture(t, EnvironmentProfileGH)

	_, err := manager.Execute(context.Background(), ExecRequest{
		Tool:                 "dumpenv",
		EnvironmentAdditions: map[string]string{"PATH": "/tmp/evil"},
	})
	if err == nil || !strings.Contains(err.Error(), "may not be set") {
		t.Fatalf("PATH must stay refused even when a profile sets it, got %v", err)
	}
}

// Credentials must never reach a log, whichever variable carries them.
func TestInjectedCredentialsAreNotPersistedToLogs(t *testing.T) {
	const token = "ghp_1234567890abcdefghijklmnopqrstuvwx"
	t.Setenv(EnvGitHubToken, token)

	manager, _ := profileFixture(t, EnvironmentProfileGH)
	events := captureRuntimeLog(t, func() {
		if _, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"}); err != nil {
			t.Fatalf("execute was rejected: %v", err)
		}
	})

	for _, event := range events {
		for key, value := range event {
			text, ok := value.(string)
			if ok && strings.Contains(text, token) {
				t.Fatalf("the GitHub token reached the log in field %q", key)
			}
		}
	}
}

func TestTransferTimeoutProfileIsLongerThanExtended(t *testing.T) {
	transfer, ok := TimeoutTransfer.Duration()
	if !ok {
		t.Fatal("the transfer profile must be known")
	}
	extended, _ := TimeoutExtended.Duration()
	if transfer <= extended {
		t.Fatalf("a clone or a release upload needs longer than %s, got %s", extended, transfer)
	}
}
