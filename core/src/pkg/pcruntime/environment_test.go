package pcruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/androiddns"
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
// The runtime reads the GitHub credential from its environment and from
// nowhere else.
//
// A file the runtime would read is a credential that lives on disk in a form
// this process can use, which is the thing the encrypted store exists to
// prevent: the Android host holds it under a Keystore key and passes the
// plaintext in at launch. An earlier build did read such a file, so this test
// exists to keep that path from coming back by habit.
func TestGitHubTokenIsNeverReadFromDisk(t *testing.T) {
	t.Setenv(EnvGitHubToken, "")

	manager, _ := profileFixture(t, EnvironmentProfileGit)
	credentials := filepath.Join(manager.registry.paths.MetadataDir, "credentials")
	if err := os.MkdirAll(credentials, 0o700); err != nil {
		t.Fatalf("cannot create credential dir: %v", err)
	}
	const planted = "ghp_planted_credential_that_must_not_be_read_0001"
	for _, name := range []string{"github_token", "github.bin", "github_token.txt"} {
		if err := os.WriteFile(filepath.Join(credentials, name),
			[]byte(planted+"\n"), 0o600); err != nil {
			t.Fatalf("cannot write %s: %v", name, err)
		}
	}

	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if strings.Contains(result.Stdout, planted) {
		t.Error("a credential file on disk was read and injected")
	}
	if strings.Contains(result.Stdout, "GIT_CONFIG_VALUE_0=") {
		t.Errorf("a credential was injected with none configured:\n%s", result.Stdout)
	}
}

// The host supplies the credential at launch, and that is the whole source.
func TestGitHubTokenComesFromTheProcessEnvironment(t *testing.T) {
	const token = "ghp_environment_supplied_credential_0002"
	t.Setenv(EnvGitHubToken, token)

	manager, _ := profileFixture(t, EnvironmentProfileGH)
	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !strings.Contains(result.Stdout, "GH_TOKEN="+token) {
		t.Errorf("the credential from the environment was not injected:\n%s", result.Stdout)
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

// gh is the only bundled tool that resolves hostnames in Go, and Android gives
// Go's resolver no /etc/resolv.conf to read. Without the servers the host read
// from ConnectivityManager it falls back to [::1]:53, where nothing listens, and
// every request fails as "error connecting to api.github.com" without leaving
// the device. That was physically reproduced before this was added.
func TestGHProfileCarriesTheAndroidDNSServers(t *testing.T) {
	t.Setenv(androiddns.EnvServer, "8.8.8.8:53;1.1.1.1:53")
	t.Setenv(EnvGitHubToken, "")

	manager, _ := profileFixture(t, EnvironmentProfileGH)
	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if !strings.Contains(result.Stdout, androiddns.EnvServer+"=8.8.8.8:53;1.1.1.1:53") {
		t.Errorf("gh did not receive the host's DNS servers:\n%s", result.Stdout)
	}
}

// Nothing is invented when the host supplied nothing: off Android there is a
// working resolver already, and setting an empty value would be a way to break
// one that worked.
func TestGHProfileAddsNoDNSWhenTheHostSuppliedNone(t *testing.T) {
	t.Setenv(androiddns.EnvServer, "")
	t.Setenv(EnvGitHubToken, "")

	manager, _ := profileFixture(t, EnvironmentProfileGH)
	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if strings.Contains(result.Stdout, androiddns.EnvServer+"=") {
		t.Errorf("an empty DNS override was injected:\n%s", result.Stdout)
	}
}

// The DNS servers go to gh alone. git and its transport helper resolve through
// bionic, so widening this would add reach without adding capability.
func TestGitProfileDoesNotCarryTheDNSOverride(t *testing.T) {
	t.Setenv(androiddns.EnvServer, "8.8.8.8:53")
	t.Setenv(EnvGitHubToken, "")

	manager, _ := profileFixture(t, EnvironmentProfileGit)
	result, err := manager.Execute(context.Background(), ExecRequest{Tool: "dumpenv"})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if strings.Contains(result.Stdout, androiddns.EnvServer+"=") {
		t.Errorf("the DNS override reached a tool that does not need it:\n%s", result.Stdout)
	}
}
