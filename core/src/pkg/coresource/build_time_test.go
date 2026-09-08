package coresource

import (
	"fmt"
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

// gitInit creates a throwaway repository containing a copy of the resolver at
// the same relative location it occupies in the real tree, so the script's own
// repo-root discovery works unchanged.
func gitInit(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.invalid"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "core/src/pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(resolverPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core/resolve-build-time.sh"), source, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// commit writes a file and commits it at a fixed author/committer date, so the
// test controls the timestamps it is asserting about.
func commit(t *testing.T, root, relPath, content string, epoch int64) {
	t.Helper()
	full := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	add := exec.Command("git", "add", "-A")
	add.Dir = root
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	stamp := fmt.Sprintf("%d +0000", epoch)
	c := exec.Command("git", "commit", "-q", "-m", "change "+relPath)
	c.Dir = root
	c.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+stamp, "GIT_COMMITTER_DATE="+stamp)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func resolveEpochIn(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.Command(filepath.Join(root, "core/resolve-build-time.sh"), "--print-epoch")
	cmd.Dir = root
	env := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "SOURCE_DATE_EPOCH=") {
			env = append(env, kv)
		}
	}
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolver failed in %s: %v", root, err)
	}
	return strings.TrimSpace(string(out))
}

// The defect the path scoping exists to fix.
//
// An unscoped `git log -1` dated the build from HEAD, so committing
// documentation — or the staged binaries themselves — gave identical Core source
// a different timestamp and therefore different bytes. The guarantee would have
// held only until the next unrelated commit, which is worse than not claiming
// it.
func TestDefaultEpochIgnoresCommitsThatAreNotBuildInputs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := gitInit(t)

	const (
		buildInputEpoch = 1700000000
		unrelatedEpoch  = 1800000000
		laterInputEpoch = 1900000000
	)

	// A — a real canonical Core build input.
	commit(t, root, "core/src/pkg/thing.go", "package thing\n", buildInputEpoch)
	atA := resolveEpochIn(t, root)
	if atA != fmt.Sprint(buildInputEpoch) {
		t.Fatalf("after the build-input commit the epoch is %s, want %d", atA, buildInputEpoch)
	}

	// B — documentation, an acceptance baseline, a staged binary and an
	// application file. None of these changes what the canonical build
	// compiles, so none may move the timestamp.
	for _, path := range []string{
		"docs/notes.md",
		"android/release-baseline.properties",
		"android/app/src/main/jniLibs/arm64-v8a/libpicoclaw.so",
		"lib/main.dart",
		"PROJECT_STATE.md",
	} {
		commit(t, root, path, "unrelated change to "+path+"\n", unrelatedEpoch)
	}
	atB := resolveEpochIn(t, root)
	if atB != atA {
		t.Fatalf("an unrelated commit moved the build timestamp: %s then %s\n"+
			"identical Core source would now produce different bytes", atA, atB)
	}

	// C — a canonical build input again. Now it must advance.
	commit(t, root, "core/src/Makefile", "# changed\n", laterInputEpoch)
	atC := resolveEpochIn(t, root)
	if atC != fmt.Sprint(laterInputEpoch) {
		t.Fatalf("a build-input commit did not advance the epoch: got %s, want %d",
			atC, laterInputEpoch)
	}
}

// The staging commit is the specific case that motivated this: the accepted
// binaries land in a commit of their own, which must not redate the build that
// produced them.
func TestStagingCommitDoesNotChangeTheBuildTimestamp(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := gitInit(t)

	commit(t, root, "core/src/pkg/core.go", "package core\n", 1700000000)
	before := resolveEpochIn(t, root)

	// Exactly the shape of a real staging commit: the built binaries, then the
	// documentation that records their acceptance.
	commit(t, root, "android/app/src/main/jniLibs/arm64-v8a/libpicoclaw.so", "ELF\n", 1750000000)
	commit(t, root, "android/app/src/main/jniLibs/arm64-v8a/libpicoclaw-web.so", "ELF\n", 1750000001)
	commit(t, root, "PROJECT_STATE.md", "accepted\n", 1750000002)

	after := resolveEpochIn(t, root)
	if after != before {
		t.Fatalf("staging the build output redated the build: %s then %s", before, after)
	}
}

// The build recipe itself is a build input: changing how the binary is produced
// must move the timestamp even though the Go source did not change.
func TestBuildRecipeChangesAdvanceTheEpoch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := gitInit(t)

	commit(t, root, "core/src/pkg/core.go", "package core\n", 1700000000)
	before := resolveEpochIn(t, root)

	commit(t, root, "core/build-android-arm64.sh", "#!/bin/sh\n# changed recipe\n", 1800000000)
	after := resolveEpochIn(t, root)

	if after == before {
		t.Fatal("a change to the canonical build recipe did not move the build timestamp")
	}
	if after != "1800000000" {
		t.Fatalf("epoch = %s, want the recipe commit's timestamp", after)
	}
}
