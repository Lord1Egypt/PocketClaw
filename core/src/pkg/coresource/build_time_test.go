package coresource

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// resolverPath locates core/resolve-build-time.sh relative to this package,
// rather than from a working directory that varies with how tests are invoked.
func resolverPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the resolver is a POSIX shell script")
	}
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "resolve-build-time.sh"))
	if err != nil {
		t.Fatalf("resolve script path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("core/resolve-build-time.sh is missing: %v", err)
	}
	return path
}

// runResolver executes the script with the given SOURCE_DATE_EPOCH ("" unsets
// it) and returns trimmed stdout plus any error.
func runResolver(t *testing.T, epoch string, dir string) (string, error) {
	t.Helper()
	cmd := exec.Command(resolverPath(t))
	if dir != "" {
		cmd.Dir = dir
	}
	env := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "SOURCE_DATE_EPOCH=") {
			env = append(env, kv)
		}
	}
	if epoch != "" {
		env = append(env, "SOURCE_DATE_EPOCH="+epoch)
	}
	cmd.Env = env
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

var buildTimeFormat = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{4}$`)

// The build stamped `date` at build time, so identical source produced
// different binaries purely because the clock had moved. Everything below is
// about that not being possible any more.
func TestSourceDateEpochIsHonouredExactly(t *testing.T) {
	got, err := runResolver(t, "1700000000", "")
	if err != nil {
		t.Fatalf("resolver failed: %v", err)
	}
	// Fixed to UTC so the output does not depend on the builder's timezone.
	const want = "2023-11-14T22:13:20+0000"
	if got != want {
		t.Errorf("resolved %q, want %q", got, want)
	}
	if !buildTimeFormat.MatchString(got) {
		t.Errorf("resolved %q does not match the historical BuildTime format", got)
	}
}

// The same input must give the same answer however much time passes between
// calls — which is the entire property the old `date` call lacked.
func TestResolutionDoesNotDependOnWallClock(t *testing.T) {
	first, err := runResolver(t, "1700000000", "")
	if err != nil {
		t.Fatalf("resolver failed: %v", err)
	}
	second, err := runResolver(t, "1700000000", "")
	if err != nil {
		t.Fatalf("resolver failed: %v", err)
	}
	if first != second {
		t.Fatalf("same epoch resolved differently: %q then %q", first, second)
	}

	// And with no epoch at all, the answer comes from the commit rather than
	// from now, so repeated calls still agree.
	firstGit, err := runResolver(t, "", "")
	if err != nil {
		t.Fatalf("resolver failed without SOURCE_DATE_EPOCH: %v", err)
	}
	secondGit, err := runResolver(t, "", "")
	if err != nil {
		t.Fatalf("resolver failed without SOURCE_DATE_EPOCH: %v", err)
	}
	if firstGit != secondGit {
		t.Fatalf("git-derived time changed between calls: %q then %q", firstGit, secondGit)
	}
	if !buildTimeFormat.MatchString(firstGit) {
		t.Errorf("git-derived %q does not match the historical BuildTime format", firstGit)
	}
}

// A malformed value must be an error, not something date(1) quietly
// reinterprets into a plausible-looking timestamp.
func TestMalformedSourceDateEpochFailsClearly(t *testing.T) {
	for _, bad := range []string{"notanumber", "-1", "17e8", "2023-11-14", " ", "0"} {
		t.Run(bad, func(t *testing.T) {
			out, err := runResolver(t, bad, "")
			if err == nil {
				t.Fatalf("SOURCE_DATE_EPOCH=%q was accepted and produced %q", bad, out)
			}
		})
	}
}

// Outside a git checkout and with no epoch, the build must stop rather than
// stamp the wall clock. A silent fallback would break reproducibility exactly
// where it is hardest to notice.
func TestNonGitWithoutEpochFailsRatherThanUsingNow(t *testing.T) {
	// A directory that is not inside any repository. git's discovery walks
	// upward, so this also sets the ceiling to keep it from finding ours.
	outside := t.TempDir()
	cmd := exec.Command(resolverPath(t))
	cmd.Dir = outside
	env := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "SOURCE_DATE_EPOCH=") &&
			!strings.HasPrefix(kv, "GIT_CEILING_DIRECTORIES=") {
			env = append(env, kv)
		}
	}
	// The script resolves the repository from its own location, so point that
	// resolution at a copy living outside any checkout.
	copied := filepath.Join(outside, "resolve-build-time.sh")
	source, err := os.ReadFile(resolverPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(copied, source, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command(copied)
	cmd.Dir = outside
	cmd.Env = append(env, "GIT_CEILING_DIRECTORIES="+filepath.Dir(outside))

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("a non-git build without SOURCE_DATE_EPOCH succeeded and produced: %s", out)
	}
	if !strings.Contains(string(out), "SOURCE_DATE_EPOCH") {
		t.Errorf("the failure does not tell the caller how to fix it:\n%s", out)
	}
}

// The Makefile must take the resolved value rather than deriving its own, and a
// command-line assignment — which is what the canonical build script uses —
// must win. An exported environment variable would not override the old `:=`
// form, which is the trap this shape exists to avoid.
func TestMakefileTakesTheResolvedBuildTime(t *testing.T) {
	makefile, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	body := string(makefile)

	if strings.Contains(body, "BUILD_TIME_RAW:=$(strip $(shell date") {
		t.Error("the Makefile derives BuildTime from the wall clock again")
	}
	if !strings.Contains(body, "BUILD_TIME_RESOLVER") {
		t.Error("the Makefile no longer routes BuildTime through the resolver")
	}
	// Recursive `=`, not `:=`: the subprocess must be deferred so a caller
	// passing BUILD_TIME on the command line never pays for it.
	if !strings.Contains(body, "BUILD_TIME_RAW=$(strip $(shell") {
		t.Error("BUILD_TIME_RAW must be lazily assigned so an override skips the resolver")
	}
}

// The canonical Android build resolves once and passes the value explicitly
// into both make invocations, so the two binaries cannot disagree.
func TestCanonicalBuildPassesBuildTimeExplicitly(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "..", "..", "build-android-arm64.sh"))
	if err != nil {
		t.Fatalf("read build script: %v", err)
	}
	body := string(script)

	if !strings.Contains(body, `BUILD_TIME="$("$REPO_ROOT/core/resolve-build-time.sh")"`) {
		t.Error("the build script does not resolve the build time from the one resolver")
	}
	if got := strings.Count(body, `BUILD_TIME="$BUILD_TIME"`); got != 2 {
		t.Errorf("BUILD_TIME is passed to %d make invocations, want 2 "+
			"(the gateway and the launcher must carry the same timestamp)", got)
	}
	if strings.Contains(body, "export BUILD_TIME") {
		t.Error("exporting BUILD_TIME does not override a Make assignment; pass it on the command line")
	}
}
