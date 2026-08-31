package tools

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These tests run the bootstrap that is actually packaged — read out of the
// staged payload, not out of runtime/ — against a real interpreter. It is the
// closest this host can get to the device: the same module, the same flags, the
// same program-on-stdin, and a CPython that has to produce the streams, the
// argv, the traceback and the exit status the contract promises.
//
// What it cannot reproduce is Android's TextLogStream, which is the bug the
// module exists to work around. On this host sys.stdout already points at
// descriptor 1, so a passing run proves the bootstrap is correct, not that it is
// necessary. Necessity was established on the device.

func shippedBootstrap(t *testing.T) string {
	t.Helper()

	root := toolsRepoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout; there is no payload to read")
	}
	payload := filepath.Join(root, "android", "app", "src", "main", "jniLibs",
		"arm64-v8a", "libpocketclaw-python.so")
	data, err := os.ReadFile(payload)
	if err != nil {
		t.Skipf("no staged python payload: %v", err)
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("the payload has no readable appended stdlib: %v", err)
	}
	for _, file := range archive.File {
		if file.Name != "pocketclaw_bootstrap.py" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("cannot read the packaged bootstrap: %v", err)
		}
		defer reader.Close()
		source, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("cannot read the packaged bootstrap: %v", err)
		}

		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "pocketclaw_bootstrap.py"), source, 0o644); err != nil {
			t.Fatalf("cannot stage the bootstrap: %v", err)
		}
		return dir
	}
	t.Fatal("the staged payload does not contain pocketclaw_bootstrap.py")
	return ""
}

// runShippedBootstrap executes one program through the packaged entry point,
// with the catalog's own interpreter flags.
func runShippedBootstrap(t *testing.T, program string, args ...string) (string, string, int) {
	t.Helper()

	interpreter, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("no python3 on this host to run the bootstrap through")
	}
	dir := shippedBootstrap(t)

	argv := append([]string{"-P", "-s", "-S", "-B", "-u", "-m", "pocketclaw_bootstrap"}, args...)
	command := exec.Command(interpreter, argv...)
	command.Stdin = strings.NewReader(program)
	command.Env = []string{"PATH=/usr/bin:/bin", "PYTHONPATH=" + dir}

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	_ = command.Run()

	return stdout.String(), stderr.String(), command.ProcessState.ExitCode()
}

func TestShippedBootstrapPrintsToStdout(t *testing.T) {
	stdout, stderr, code := runShippedBootstrap(t, `print("PYTHON-FINAL-PASS")`)

	if !strings.Contains(stdout, "PYTHON-FINAL-PASS") {
		t.Errorf("stdout = %q, want the printed line", stdout)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
}

func TestShippedBootstrapReportsTheRealTraceback(t *testing.T) {
	stdout, stderr, code := runShippedBootstrap(t, `raise ValueError("TEST-ERROR")`)

	for _, want := range []string{
		"Traceback (most recent call last):",
		`File "<stdin>", line 1, in <module>`,
		"ValueError: TEST-ERROR",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not contain %q:\n%s", want, stderr)
		}
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
	// The bootstrap's own frames must not appear: a line number in a traceback
	// has to refer to the code the caller sent.
	if strings.Contains(stderr, "pocketclaw_bootstrap") {
		t.Errorf("the traceback exposes the entry point's frames:\n%s", stderr)
	}
}

func TestShippedBootstrapPassesUnicodeThrough(t *testing.T) {
	const text = "مرحبا 🐍"
	stdout, _, code := runShippedBootstrap(t, `print("`+text+`")`)

	if strings.TrimRight(stdout, "\n") != text {
		t.Errorf("stdout = %q, want exactly %q", stdout, text)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

// The Phase C contract is the one `python -` gives: argv[0] is "-", and the
// caller's values follow. The entry point's own module path must not leak in.
func TestShippedBootstrapKeepsTheArgvContract(t *testing.T) {
	stdout, stderr, code := runShippedBootstrap(t,
		"import sys\nprint(sys.argv)", "alpha", "beta gamma")

	if !strings.Contains(stdout, `['-', 'alpha', 'beta gamma']`) {
		t.Errorf("sys.argv = %q, want ['-', 'alpha', 'beta gamma'] (stderr: %s)", stdout, stderr)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestShippedBootstrapKeepsExitStatusAndMainName(t *testing.T) {
	stdout, _, code := runShippedBootstrap(t, "print(__name__)")
	if strings.TrimSpace(stdout) != "__main__" {
		t.Errorf("__name__ = %q, want __main__", stdout)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	_, _, requested := runShippedBootstrap(t, "import sys\nsys.exit(7)")
	if requested != 7 {
		t.Errorf("sys.exit(7) produced exit code %d", requested)
	}

	_, syntaxErr, syntaxCode := runShippedBootstrap(t, "def (\n")
	if !strings.Contains(syntaxErr, "SyntaxError") ||
		!strings.Contains(syntaxErr, `File "<stdin>", line 1`) {
		t.Errorf("a syntax error was not reported against <stdin>:\n%s", syntaxErr)
	}
	if syntaxCode != 1 {
		t.Errorf("a syntax error produced exit code %d, want 1", syntaxCode)
	}
}

// An empty program is a valid program: `python -` with nothing on stdin exits 0
// having done nothing, and the entry point must not turn that into an error.
func TestShippedBootstrapAcceptsAnEmptyProgram(t *testing.T) {
	stdout, stderr, code := runShippedBootstrap(t, "")
	if code != 0 || stdout != "" || stderr != "" {
		t.Errorf("empty program gave exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func toolsRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "pubspec.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
