package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/pcruntime"
)

// ghCanary is unique enough that finding it anywhere is proof rather than
// coincidence.
const ghCanary = "ghp_canary_5b21d4af_never_in_argv_or_logs"

// newCredentialedRuntime stages the test binary as a gh-profiled bundled tool,
// so the runtime injects the GitHub credential exactly as it does on a device.
func newCredentialedRuntime(t *testing.T) *RuntimeTool {
	t.Helper()

	self, err := os.Executable()
	if err != nil {
		t.Fatalf("cannot find the test binary: %v", err)
	}
	root := t.TempDir()
	libDir := filepath.Join(root, "lib")
	workspace := filepath.Join(root, "workspace")
	metadata := filepath.Join(root, "runtime")
	for _, dir := range []string{libDir, workspace, metadata} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", dir, err)
		}
	}
	payload := filepath.Join(libDir, "libpocketclaw-ghfake.so")
	if err := os.Symlink(self, payload); err != nil {
		t.Fatalf("cannot stage the payload: %v", err)
	}

	manifest := &pcruntime.Manifest{
		RuntimeVersion: pcruntime.ManifestVersion,
		CatalogVersion: "test",
		Tools: []pcruntime.Tool{{
			ToolID: "ghfake", DisplayName: "ghfake", CommandName: interpreterArgv0,
			Version: "stand-in", ABI: hostABI(t),
			Delivery: pcruntime.DeliveryBundled, TrustedSource: "test",
			SHA256:             digestOf(t, payload),
			Capabilities:       []string{"vcs.github"},
			TimeoutProfile:     pcruntime.TimeoutQuick,
			MaxOutputBytes:     1 << 20,
			SecurityClass:      pcruntime.SecurityClassBundledVerified,
			LibraryName:        "libpocketclaw-ghfake.so",
			EnvironmentProfile: pcruntime.EnvironmentProfileGH,
		}},
	}
	registry, err := pcruntime.NewRegistryWith(manifest, &pcruntime.Paths{
		LibDir: libDir, MetadataDir: metadata, Workspace: workspace,
	})
	if err != nil {
		t.Fatalf("cannot build the registry: %v", err)
	}
	manager, err := pcruntime.NewManagerWith(registry, workspace)
	if err != nil {
		t.Fatalf("cannot build the manager: %v", err)
	}
	return &RuntimeTool{manager: manager}
}

func digestOfFile(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		t.Fatalf("cannot hash %s: %v", path, err)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

// The credential must reach the tool that needs it and nothing else. This runs
// both halves of that in one pass: the child reports the length it received,
// and the whole runtime log is searched for the value.
func TestGitHubCredentialIsInjectedButNeverLogged(t *testing.T) {
	t.Setenv(pcruntime.EnvGitHubToken, ghCanary)
	runtime := newCredentialedRuntime(t)

	sink := filepath.Join(t.TempDir(), "runtime.log")
	previous := logger.GetLevel()
	logger.SetLevel(logger.DEBUG)
	logger.DisableConsole()
	if err := logger.EnableFileLogging(sink); err != nil {
		t.Fatalf("cannot capture logs: %v", err)
	}
	t.Cleanup(func() {
		logger.DisableFileLogging()
		logger.EnableConsole()
		logger.SetLevel(previous)
	})

	result := runtime.Execute(context.Background(), map[string]any{
		"action": "run",
		"tool":   "ghfake",
		"stdin":  "ENVLEN:GH_TOKEN\nARGV\nEXIT:0",
	})
	logger.DisableFileLogging()

	report := result.ContentForLLM()
	if !strings.Contains(report, fmt.Sprintf("GH_TOKEN_len=%d", len(ghCanary))) {
		t.Errorf("the credential did not reach the tool:\n%s", report)
	}
	if strings.Contains(report, ghCanary) {
		t.Error("the credential appeared in the agent-visible result")
	}
	for _, line := range strings.Split(report, "\n") {
		if strings.HasPrefix(line, "argv=") && strings.Contains(line, ghCanary) {
			t.Errorf("the credential reached the argument vector: %s", line)
		}
	}

	captured, err := os.ReadFile(sink)
	if err != nil {
		t.Fatalf("cannot read the captured log: %v", err)
	}
	if strings.Contains(string(captured), ghCanary) {
		t.Error("the credential was written to the runtime log")
	}
	if !strings.Contains(string(captured), "runtime.exec.completed") {
		t.Fatal("no runtime events were captured; the test would prove nothing")
	}
}

// With nothing configured, no credential is invented and no interactive login
// is attempted: gh is told not to prompt, so the agent gets a plain failure it
// can report rather than a hung process or a login that would persist a
// credential outside PocketClaw.
func TestNoCredentialIsInjectedWhenGitHubIsNotConnected(t *testing.T) {
	t.Setenv(pcruntime.EnvGitHubToken, "")
	runtime := newCredentialedRuntime(t)

	result := runtime.Execute(context.Background(), map[string]any{
		"action": "run",
		"tool":   "ghfake",
		"stdin":  "ENVLEN:GH_TOKEN\nENVLEN:GH_PROMPT_DISABLED\nEXIT:0",
	})

	report := result.ContentForLLM()
	if !strings.Contains(report, "GH_TOKEN_len=0") {
		t.Errorf("a credential was injected with none connected:\n%s", report)
	}
	if strings.Contains(report, "GH_PROMPT_DISABLED_len=0") {
		t.Errorf("gh was left able to prompt for a login:\n%s", report)
	}
}

// The agent has no way to ask for the credential. This is a contract about the
// tool surface, not about any single call: a tool that returned the token would
// make every other guarantee here decorative.
func TestNoAgentToolCanReturnTheGitHubCredential(t *testing.T) {
	runtime := newCredentialedRuntime(t)

	registry := NewToolRegistry()
	registry.Register(runtime)
	registry.Register(NewPythonTool(runtime))

	for _, definition := range registry.GetDefinitions() {
		name, _ := definition["name"].(string)
		lowered := strings.ToLower(name)
		for _, forbidden := range []string{"token", "credential", "secret", "auth"} {
			if strings.Contains(lowered, forbidden) {
				t.Errorf("tool %q looks like a credential accessor", name)
			}
		}
	}

	// The runtime tool is the only way to reach a managed tool, and it takes no
	// environment: a caller cannot ask for GH_TOKEN to be echoed back by
	// setting it, nor read it out of a parameter this schema does not have.
	params := runtime.Parameters()
	properties, _ := params["properties"].(map[string]any)
	for _, forbidden := range []string{"env", "environment", "token", "credentials"} {
		if _, present := properties[forbidden]; present {
			t.Errorf("the runtime tool exposes %q to the model", forbidden)
		}
	}
}

// git authenticates through a process-scoped config value, so nothing is
// written into a repository's .git/config and no credential-bearing URL is ever
// constructed. That is what keeps a cloned repo from carrying the token.
func TestGitCredentialTravelsInTheEnvironmentNotTheRepository(t *testing.T) {
	t.Setenv(pcruntime.EnvGitHubToken, ghCanary)

	self, err := os.Executable()
	if err != nil {
		t.Fatalf("cannot find the test binary: %v", err)
	}
	root := t.TempDir()
	libDir := filepath.Join(root, "lib")
	workspace := filepath.Join(root, "workspace")
	metadata := filepath.Join(root, "runtime")
	for _, dir := range []string{libDir, workspace, metadata} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", dir, err)
		}
	}
	payload := filepath.Join(libDir, "libpocketclaw-gitfake.so")
	if err := os.Symlink(self, payload); err != nil {
		t.Fatalf("cannot stage the payload: %v", err)
	}

	manifest := &pcruntime.Manifest{
		RuntimeVersion: pcruntime.ManifestVersion,
		CatalogVersion: "test",
		Tools: []pcruntime.Tool{{
			ToolID: "gitfake", DisplayName: "gitfake", CommandName: interpreterArgv0,
			Version: "stand-in", ABI: hostABI(t),
			Delivery: pcruntime.DeliveryBundled, TrustedSource: "test",
			SHA256:             digestOfFile(t, payload),
			Capabilities:       []string{"vcs.git"},
			TimeoutProfile:     pcruntime.TimeoutQuick,
			MaxOutputBytes:     1 << 20,
			SecurityClass:      pcruntime.SecurityClassBundledVerified,
			LibraryName:        "libpocketclaw-gitfake.so",
			EnvironmentProfile: pcruntime.EnvironmentProfileGit,
		}},
	}
	registry, err := pcruntime.NewRegistryWith(manifest, &pcruntime.Paths{
		LibDir: libDir, MetadataDir: metadata, Workspace: workspace,
	})
	if err != nil {
		t.Fatalf("cannot build the registry: %v", err)
	}
	manager, err := pcruntime.NewManagerWith(registry, workspace)
	if err != nil {
		t.Fatalf("cannot build the manager: %v", err)
	}

	result := (&RuntimeTool{manager: manager}).Execute(context.Background(), map[string]any{
		"action": "run",
		"tool":   "gitfake",
		"stdin":  "ENVLEN:GIT_CONFIG_VALUE_0\nENVLEN:GIT_CONFIG_KEY_0\nARGV\nEXIT:0",
	})

	report := result.ContentForLLM()
	if strings.Contains(report, "GIT_CONFIG_VALUE_0_len=0") {
		t.Errorf("git received no credential:\n%s", report)
	}
	if strings.Contains(report, ghCanary) {
		t.Error("the credential appeared in the agent-visible result")
	}
	// The credential is base64 inside an Authorization header, never a bare
	// token and never part of a URL.
	if strings.Contains(report, "https://"+ghCanary) || strings.Contains(report, ghCanary+"@") {
		t.Error("a credential-bearing URL was constructed")
	}
}
