package pcruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/androiddns"
)

// Environment keys for the credentials the git and gh profiles inject. Both are
// read at execution time and never written to a log, an event, or an argument
// vector.
const (
	// EnvGitHubToken is the GitHub credential the git and gh profiles use, and
	// the only place the runtime reads it from.
	//
	// The Android host holds the credential encrypted under a key that lives in
	// the Android Keystore and never leaves it, decrypts it when it starts Core,
	// and passes it here. There is deliberately no file fallback: a credential
	// this process could read off disk is one that survives on disk, which is
	// what the encrypted store exists to prevent.
	EnvGitHubToken = "POCKETCLAW_GITHUB_TOKEN"
	// pythonHome is the prefix compiled into the interpreter by
	// runtime/build-python-android-arm64.sh. It intentionally does not exist on
	// the device: setting it stops getpath searching the filesystem around the
	// executable, and the standard library is supplied through PYTHONPATH
	// instead. Change it in both places or not at all.
	pythonHome = "/pocketclaw/python"
)

// systemCACandidates are the platform certificate stores, newest layout first.
// Android 14 moved the store into the Conscrypt APEX, so a single hard-coded
// path would silently break TLS on either older or newer devices.
var systemCACandidates = []string{
	"/apex/com.android.conscrypt/cacerts",
	"/system/etc/security/cacerts",
}

// preparedEnvironment is the result of applying a tool's environment profile.
type preparedEnvironment struct {
	// variables are added to the constructed base environment.
	variables map[string]string
	// secretKeys names variables whose values must never be logged, even when
	// their name would not look secret on its own.
	secretKeys map[string]struct{}
}

func newPreparedEnvironment() *preparedEnvironment {
	return &preparedEnvironment{
		variables:  make(map[string]string, 8),
		secretKeys: make(map[string]struct{}, 2),
	}
}

func (p *preparedEnvironment) set(key, value string) {
	p.variables[key] = value
}

func (p *preparedEnvironment) setSecret(key, value string) {
	p.variables[key] = value
	p.secretKeys[key] = struct{}{}
}

// prepareEnvironment applies a tool's environment profile, materialising the
// helper directory when the tool has helpers.
func (m *Manager) prepareEnvironment(resolved *ResolvedTool) (*preparedEnvironment, error) {
	prepared := newPreparedEnvironment()

	var helperDir string
	if len(resolved.HelperPaths) > 0 {
		dir, err := m.materialiseHelpers(resolved)
		if err != nil {
			return nil, err
		}
		helperDir = dir
	}

	switch resolved.Tool.EnvironmentProfile {
	case EnvironmentProfileNone:
	case EnvironmentProfileGit:
		m.applyGitProfile(prepared, helperDir)
	case EnvironmentProfileGH:
		m.applyGHProfile(prepared, helperDir)
	case EnvironmentProfilePython:
		m.applyPythonProfile(prepared, resolved)
	default:
		return nil, fmt.Errorf(
			"tool %q declares environment profile %q, which this build cannot prepare",
			resolved.Tool.ToolID, resolved.Tool.EnvironmentProfile,
		)
	}
	return prepared, nil
}

// applyPythonProfile prepares the interpreter's environment.
//
// It injects no credential. Python runs model-authored code, so a token in its
// environment would be readable by that code; the git and gh profiles hand out
// credentials because those tools use them for a fixed purpose, and Python has
// no such purpose.
func (m *Manager) applyPythonProfile(prepared *preparedEnvironment, resolved *ResolvedTool) {
	prepared.set("PYTHONHOME", pythonHome)
	// The payload is both the interpreter and its standard library: the stdlib
	// zip is appended to the ELF and zipimport reads it out of the same file.
	// The stdlib must come first so a workspace file called json.py or re.py
	// cannot shadow it, while the workspace still follows so a user's own
	// modules remain importable.
	prepared.set("PYTHONPATH", strings.Join(
		[]string{resolved.ExecutablePath, m.workspace}, string(os.PathListSeparator),
	))
	prepared.set("PYTHONDONTWRITEBYTECODE", "1")
	prepared.set("PYTHONUTF8", "1")
	prepared.set("PYTHONNOUSERSITE", "1")
}

func (m *Manager) applyGitProfile(prepared *preparedEnvironment, helperDir string) {
	// git finds git-remote-https here. Without it, every https:// URL fails.
	if helperDir != "" {
		prepared.set("GIT_EXEC_PATH", helperDir)
	}
	// git must never block waiting for a terminal that does not exist. Without
	// these it can hang until the runtime's timeout kills it, turning a missing
	// credential into an opaque timeout.
	prepared.set("GIT_TERMINAL_PROMPT", "0")
	prepared.set("GIT_ASKPASS", "")
	prepared.set("GIT_CONFIG_NOSYSTEM", "1")
	// A private HOME keeps git's own config and credential state inside runtime
	// storage rather than in the user workspace, where a Skill could edit it.
	prepared.set("HOME", m.gitHome())
	if caPath := systemCAPath(); caPath != "" {
		prepared.set("GIT_SSL_CAPATH", caPath)
	}

	m.applyGitCredentials(prepared)
}

// applyGitCredentials injects a GitHub token through git's environment-based
// config, never through argv.
//
// git reads config from GIT_CONFIG_KEY_n/GIT_CONFIG_VALUE_n pairs, so the
// credential never appears on a command line where /proc/<pid>/cmdline, a
// process listing, or an argv diagnostic could expose it. This is deliberately
// not the `https://TOKEN@github.com/...` form, which leaks the token into the
// URL, into git's remote config on disk, and into any error message quoting it.
func (m *Manager) applyGitCredentials(prepared *preparedEnvironment) {
	token := m.githubToken()
	if token == "" {
		return
	}
	prepared.set("GIT_CONFIG_COUNT", "1")
	prepared.set("GIT_CONFIG_KEY_0", "http.https://github.com/.extraheader")
	prepared.setSecret("GIT_CONFIG_VALUE_0", "Authorization: Basic "+basicAuthValue(token))
}

func (m *Manager) applyGHProfile(prepared *preparedEnvironment, helperDir string) {
	// gh shells out to git for clone and for repository context. The helper
	// directory carries a git entry so gh finds the runtime's verified git
	// rather than depending on whatever the platform happens to expose.
	if helperDir != "" {
		prepared.set("PATH", helperDir+":"+strings.Join(systemBinDirs, ":"))
	}
	// gh must not try to be interactive or to reach out for update checks.
	prepared.set("GH_NO_UPDATE_NOTIFIER", "1")
	prepared.set("GH_PROMPT_DISABLED", "1")
	prepared.set("GH_PAGER", "cat")
	prepared.set("NO_COLOR", "1")
	prepared.set("HOME", m.gitHome())

	// gh is a statically linked Go binary, and Android provides no
	// /etc/resolv.conf for Go's resolver to read. Without this it falls back to
	// [::1]:53, where nothing listens, and every request fails as "error
	// connecting to api.github.com" without leaving the device. The payload
	// carries the same resolver shim Core and the launcher use; this is what
	// gives it the servers to use.
	//
	// It is set on the gh profile rather than inherited by every managed tool:
	// gh is the only bundled tool that resolves names in Go. curl, git and its
	// transport helper go through bionic and Android's own resolver, so widening
	// this would add reach without adding capability.
	if servers := strings.TrimSpace(os.Getenv(androiddns.EnvServer)); servers != "" {
		prepared.set(androiddns.EnvServer, servers)
	}

	if token := m.githubToken(); token != "" {
		prepared.setSecret("GH_TOKEN", token)
	}
}

// githubToken reads the GitHub credential from PocketClaw-controlled storage.
//
// The environment is checked first so the Android Service can pass a credential
// it holds without it ever touching disk. The file fallback lives under runtime
// metadata storage, which is app-private and outside the user workspace.
func (m *Manager) githubToken() string {
	return strings.TrimSpace(os.Getenv(EnvGitHubToken))
}

// gitHome is the private HOME given to git and gh.
func (m *Manager) gitHome() string {
	home := filepath.Join(m.registry.paths.MetadataDir, "home")
	// A missing HOME makes git fall back to reading /etc/passwd and can make gh
	// fail to write its config; creating it is cheaper than diagnosing that.
	_ = os.MkdirAll(home, 0o700)
	return home
}

// systemCAPath returns the platform certificate store, or "" if neither
// known layout exists.
func systemCAPath() string {
	for _, candidate := range systemCACandidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}
