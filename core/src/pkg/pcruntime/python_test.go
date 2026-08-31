package pcruntime

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pythonCatalogEntry returns the shipped python tool, or fails.
func pythonCatalogEntry(t *testing.T) Tool {
	t.Helper()
	manifest, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatalf("cannot load catalog: %v", err)
	}
	for _, tool := range manifest.Tools {
		if tool.ToolID == "python" {
			return tool
		}
	}
	t.Fatal("the catalog has no python entry")
	return Tool{}
}

func TestPythonCatalogEntryIsBundledAndPinned(t *testing.T) {
	tool := pythonCatalogEntry(t)

	if tool.Delivery != DeliveryBundled {
		t.Errorf("python delivery = %q, want bundled", tool.Delivery)
	}
	if tool.LibraryName != "libpocketclaw-python.so" {
		t.Errorf("python library_name = %q", tool.LibraryName)
	}
	if tool.Version != "3.14.7" {
		t.Errorf("python version = %q, want the pinned 3.14.7", tool.Version)
	}
	if tool.ABI != "arm64-v8a" {
		t.Errorf("python abi = %q", tool.ABI)
	}
	if len(tool.SHA256) != 64 {
		t.Errorf("python sha256 = %q, want a 64-character payload checksum", tool.SHA256)
	}
	if !strings.HasPrefix(tool.TrustedSource, "https://www.python.org/") {
		t.Errorf("python trusted_source = %q, want an upstream python.org URL", tool.TrustedSource)
	}
	if tool.EnvironmentProfile != EnvironmentProfilePython {
		t.Errorf("python environment_profile = %q", tool.EnvironmentProfile)
	}
	if tool.SecurityClass != SecurityClassBundledVerified {
		t.Errorf("python security_class = %q", tool.SecurityClass)
	}
}

// -S has no environment-variable equivalent, and without it site.py imports a
// workspace sitecustomize.py automatically. That would execute workspace code
// on every single run, before the caller's own program.
func TestPythonDefaultArgsFixTheExecutionMode(t *testing.T) {
	tool := pythonCatalogEntry(t)
	for _, flag := range []string{"-P", "-s", "-S", "-B", "-u"} {
		found := false
		for _, arg := range tool.DefaultArgs {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("python default_args is missing %s (have %v)", flag, tool.DefaultArgs)
		}
	}
}

func TestDefaultArgsPrecedeCallerArguments(t *testing.T) {
	resolved := &ResolvedTool{Tool: Tool{
		CommandName: "python",
		DefaultArgs: []string{"-P", "-S"},
	}}
	argv := buildArgv(resolved, "python", []string{"-c", "print(1)"})

	want := []string{"python", "-P", "-S", "-c", "print(1)"}
	if len(argv) != len(want) {
		t.Fatalf("argv = %v, want %v", argv, want)
	}
	for i := range want {
		if argv[i] != want[i] {
			t.Fatalf("argv = %v, want %v", argv, want)
		}
	}
}

func TestPythonProfileOrdersStdlibBeforeWorkspace(t *testing.T) {
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("true", TimeoutQuick, 4096)))
	resolved := &ResolvedTool{
		Tool:           Tool{ToolID: "python", EnvironmentProfile: EnvironmentProfilePython},
		ExecutablePath: "/nativeLibraryDir/libpocketclaw-python.so",
	}

	prepared := newPreparedEnvironment()
	manager.applyPythonProfile(prepared, resolved)

	path := prepared.variables["PYTHONPATH"]
	entries := strings.Split(path, string(os.PathListSeparator))
	if len(entries) != 2 {
		t.Fatalf("PYTHONPATH = %q, want exactly the payload and the workspace", path)
	}
	if entries[0] != resolved.ExecutablePath {
		t.Errorf("PYTHONPATH[0] = %q, want the payload so the stdlib cannot be shadowed", entries[0])
	}
	if entries[1] != manager.workspace {
		t.Errorf("PYTHONPATH[1] = %q, want the workspace so user modules stay importable", entries[1])
	}
	if got := prepared.variables["PYTHONHOME"]; got != pythonHome {
		t.Errorf("PYTHONHOME = %q, want %q", got, pythonHome)
	}
	for key, want := range map[string]string{
		"PYTHONDONTWRITEBYTECODE": "1",
		"PYTHONUTF8":              "1",
		"PYTHONNOUSERSITE":        "1",
	} {
		if got := prepared.variables[key]; got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

// Python runs model-authored code. A credential in its environment would be
// readable by that code, so the profile must inject none.
func TestPythonProfileInjectsNoCredential(t *testing.T) {
	t.Setenv(EnvGitHubToken, "ghp_not-a-real-token-0123456789")
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("true", TimeoutQuick, 4096)))
	resolved := &ResolvedTool{
		Tool:           Tool{ToolID: "python", EnvironmentProfile: EnvironmentProfilePython},
		ExecutablePath: "/nativeLibraryDir/libpocketclaw-python.so",
	}

	prepared := newPreparedEnvironment()
	manager.applyPythonProfile(prepared, resolved)

	if len(prepared.secretKeys) != 0 {
		t.Errorf("python profile declared secrets %v; it must inject none", prepared.secretKeys)
	}
	for _, forbidden := range []string{
		"GH_TOKEN", "GITHUB_TOKEN", "GIT_CONFIG_VALUE_0", "GIT_CONFIG_KEY_0",
		"GIT_CONFIG_COUNT", EnvGitHubToken,
	} {
		if value, present := prepared.variables[forbidden]; present {
			t.Errorf("python profile set %s=%q; it must inject no credential", forbidden, value)
		}
	}
	for key, value := range prepared.variables {
		if strings.Contains(value, "ghp_") {
			t.Errorf("python profile leaked a token through %s", key)
		}
	}
}

func TestPythonExecutionControlVariablesCannotBeOverridden(t *testing.T) {
	for _, key := range []string{
		"PYTHONPATH", "PYTHONHOME", "PYTHONSTARTUP", "PYTHONINSPECT",
		"PYTHONUSERBASE", "PYTHONDONTWRITEBYTECODE", "PYTHONEXECUTABLE",
	} {
		if _, err := buildEnvironment(nil, map[string]string{key: "/attacker"}); err == nil {
			t.Errorf("%s was accepted as a caller-supplied variable; it must be refused", key)
		}
	}
	// The denial is a namespace, not a fixed list, but it must stay a namespace
	// and not swallow unrelated variables.
	if _, err := buildEnvironment(nil, map[string]string{"PYTEST_ADDOPTS": "-x"}); err != nil {
		t.Errorf("PYTEST_ADDOPTS was refused, but it does not control the interpreter: %v", err)
	}
}

// Python source arrives on stdin and must never reach a log, an event or an
// argument vector. Only its size may be recorded.
func TestStdinContentIsNeverRecordedInEvents(t *testing.T) {
	manifest := newTestManifest(t, systemTool("cat-tool", TimeoutQuick, 1<<20))
	manager, binDir := newTestManager(t, manifest)
	writeScript(t, binDir, "cat-tool", "cat\n")

	const secretSource = "PRIVATE_KEY = 'super-secret-value-in-source'"
	var result *ExecResult
	events := captureRuntimeLog(t, func() {
		var err error
		result, err = manager.Execute(context.Background(), ExecRequest{
			Tool:  "cat-tool",
			Stdin: secretSource,
		})
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(result.Stdout, "super-secret") {
		t.Fatal("the tool did not receive stdin; the test would prove nothing")
	}

	sawBytesIn := false
	for _, event := range events {
		for key, value := range event {
			if text, ok := value.(string); ok && strings.Contains(text, "super-secret") {
				t.Errorf("event field %q carried stdin content", key)
			}
			if key == "bytes_in" {
				sawBytesIn = true
				if got, ok := value.(float64); ok && int(got) != len(secretSource) {
					t.Errorf("bytes_in = %v, want %d", got, len(secretSource))
				}
			}
		}
	}
	if !sawBytesIn {
		t.Error("no event recorded bytes_in; the size of stdin should stay observable")
	}
}

func TestPythonPayloadIsPackagedForTheInstaller(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}
	tool := pythonCatalogEntry(t)
	payload := filepath.Join(root, "android", "app", "src", "main", "jniLibs",
		"arm64-v8a", tool.LibraryName)
	data, err := os.ReadFile(payload)
	if err != nil {
		t.Fatalf("python payload is not staged for packaging: %v", err)
	}
	if len(data) < 1<<20 {
		t.Fatalf("python payload is %d bytes, far too small to be an interpreter", len(data))
	}
	if !bytes.HasPrefix(data, []byte("\x7fELF")) {
		t.Error("python payload does not start with an ELF header")
	}

	// The standard library is appended to the ELF as a zip. If the build ever
	// strips the payload after appending instead of before, the interpreter
	// still looks fine here and cannot import anything on the device, so look
	// for the archive rather than trusting the file size.
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("python payload has no readable appended standard library: %v", err)
	}
	reader, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	found := false
	for _, file := range reader.File {
		if file.Name == "json/__init__.pyc" {
			found = true
			break
		}
	}
	if !found {
		t.Error("the appended standard library does not contain json/__init__.pyc")
	}
}

// TestPythonShapedRequestDeliversSourceOnStdin proves end to end what the
// agent-facing Python tool relies on: catalog default_args lead, the caller's
// "-" marker follows, script arguments land after it, and the program itself
// arrives on standard input rather than anywhere a process listing could show.
func TestPythonShapedRequestDeliversSourceOnStdin(t *testing.T) {
	manifest := newTestManifest(t, Tool{
		ToolID: "pyfake", DisplayName: "pyfake", CommandName: "pyfake",
		Version: "3.14.7", ABI: "arm64-v8a",
		Delivery: DeliverySystem, TrustedSource: "test",
		Capabilities:   []string{"script.execute"},
		DefaultArgs:    []string{"-P", "-s", "-S", "-B", "-u"},
		TimeoutProfile: TimeoutQuick, MaxOutputBytes: 1 << 16,
		SecurityClass: SecurityClassSystem,
	})
	manager, binDir := newTestManager(t, manifest)
	writeScript(t, binDir, "pyfake", "echo \"argv:$*\"\necho \"stdin:$(cat)\"\n")

	const source = "print('PYTHON-AGENT-PASS')"
	result, err := manager.Execute(context.Background(), ExecRequest{
		Tool:  "pyfake",
		Args:  []string{"-", "alpha", "beta"},
		Stdin: source,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !strings.Contains(result.Stdout, "stdin:"+source) {
		t.Errorf("the program did not arrive on stdin:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "argv:-P -s -S -B -u - alpha beta") {
		t.Errorf("argv is not default_args, then the stdin marker, then script args:\n%s",
			result.Stdout)
	}
	for _, line := range strings.Split(result.Stdout, "\n") {
		if strings.HasPrefix(line, "argv:") && strings.Contains(line, "print(") {
			t.Errorf("source reached the argument vector: %q", line)
		}
	}
}
