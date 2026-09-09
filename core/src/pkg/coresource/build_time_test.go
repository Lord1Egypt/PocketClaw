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
		"android/app/src/main/jniLibs/arm64-v8a/libpocketclaw.so",
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
	commit(t, root, "android/app/src/main/jniLibs/arm64-v8a/libpocketclaw.so", "ELF\n", 1750000000)
	commit(t, root, "android/app/src/main/jniLibs/arm64-v8a/libpocketclaw-web.so", "ELF\n", 1750000001)
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

// A shallow clone is the same defect wearing a disguise.
//
// Git treats the graft boundary as a root commit, so every path looks like it
// was introduced by the tip and the path-scoped query returns the tip's
// timestamp — exactly the unscoped-HEAD behaviour the scoping removed. It is
// worse than the non-git case because it succeeds and produces a plausible
// wrong answer, so the resolver has to refuse it rather than date a build by
// whichever documentation or merge commit is checked out.
func TestShallowCloneFailsRatherThanDatingFromTheTip(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := gitInit(t)

	const (
		buildInputEpoch = 1700000000
		docsEpoch       = 1800000000
	)
	commit(t, root, "core/src/pkg/thing.go", "package thing\n", buildInputEpoch)
	commit(t, root, "docs/NOTES.md", "notes\n", docsEpoch)

	if got := resolveEpochIn(t, root); got != fmt.Sprint(buildInputEpoch) {
		t.Fatalf("full clone: expected the build-input epoch %d, got %s", buildInputEpoch, got)
	}

	shallow := filepath.Join(t.TempDir(), "shallow")
	clone := exec.Command("git", "clone", "-q", "--depth", "1", "file://"+root, shallow)
	if out, err := clone.CombinedOutput(); err != nil {
		t.Skipf("shallow clone unavailable in this environment: %v\n%s", err, out)
	}

	cmd := exec.Command(filepath.Join(shallow, "core/resolve-build-time.sh"), "--print-epoch")
	cmd.Dir = shallow
	env := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "SOURCE_DATE_EPOCH=") {
			env = append(env, kv)
		}
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("shallow clone resolved instead of failing, giving %q", strings.TrimSpace(string(out)))
	}
	if !strings.Contains(string(out), "shallow") {
		t.Errorf("failure does not name the cause:\n%s", out)
	}
	if strings.Contains(string(out), fmt.Sprint(docsEpoch)) {
		t.Errorf("resolver leaked the tip's epoch %d into a shallow clone:\n%s", docsEpoch, out)
	}

	// The documented escape hatch still works there, so a shallow CI checkout
	// is inconvenienced, not blocked.
	explicit := exec.Command(filepath.Join(shallow, "core/resolve-build-time.sh"), "--print-epoch")
	explicit.Dir = shallow
	explicit.Env = append(env, "SOURCE_DATE_EPOCH="+fmt.Sprint(buildInputEpoch))
	got, err := explicit.Output()
	if err != nil {
		t.Fatalf("explicit epoch failed in a shallow clone: %v", err)
	}
	if strings.TrimSpace(string(got)) != fmt.Sprint(buildInputEpoch) {
		t.Errorf("explicit epoch: got %s, want %d", strings.TrimSpace(string(got)), buildInputEpoch)
	}
}

// The toolchain had its own opinion about when this was built.
//
// Go stamps build.vcs.revision, build.vcs.time and build.vcs.modified into a
// binary automatically, reading the enclosing repository's HEAD. It ignores the
// resolver entirely, so identical build inputs produced different bytes after
// any unrelated commit — including the staging commit itself, which meant a
// staged Core could never be reproduced from the commit that contained it.
//
// Nothing else in the tree notices if the flag is dropped: the binary still
// builds, still runs, still carries the right fingerprint and the right
// BuildTime. Only its bytes stop being reproducible, silently. Hence a test on
// the recipe and a gate check on the artifact.
func TestCanonicalBuildDisablesToolchainVCSStamping(t *testing.T) {
	for _, makefile := range []string{
		filepath.Join("..", "..", "Makefile"),
		filepath.Join("..", "..", "web", "Makefile"),
	} {
		body, err := os.ReadFile(makefile)
		if err != nil {
			t.Fatalf("read %s: %v", makefile, err)
		}
		text := string(body)

		if !strings.Contains(text, "REPRODUCIBLE_BUILD_FLAGS=-trimpath -buildvcs=false") {
			t.Errorf("%s does not define the reproducible build flags", makefile)
		}
		// Every android/arm64 recipe must go through that variable. A literal
		// -trimpath on one of them is the exact regression this catches: it
		// looks deliberate and drops the VCS stamping fix.
		for _, line := range strings.Split(text, "\n") {
			if !strings.Contains(line, "GOOS=android") || !strings.Contains(line, "GOARCH=arm64") {
				continue
			}
			if strings.Contains(line, "-trimpath") {
				t.Errorf("%s: android/arm64 recipe uses -trimpath directly instead of "+
					"$(REPRODUCIBLE_BUILD_FLAGS), which silently drops -buildvcs=false:\n  %s",
					makefile, strings.TrimSpace(line))
			}
			if !strings.Contains(line, "REPRODUCIBLE_BUILD_FLAGS") {
				t.Errorf("%s: android/arm64 recipe does not use $(REPRODUCIBLE_BUILD_FLAGS):\n  %s",
					makefile, strings.TrimSpace(line))
			}
		}
	}
}

// What counts as a build input, stated one rule at a time.
//
// The exclusion of Core *_test.go arrived after the two guards disagreed in
// practice: a four-line edit to a test file left the source fingerprint
// identical and staged freshness green, while core.staged_build_time went red
// against binaries that provably could not differ. The fingerprint had always
// excluded _test.go; the timestamp had not. These pin both halves of the fixed
// rule, and in particular that the exclusion did not quietly widen.
func TestBuildTimeInputRules(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	const (
		baseEpoch  = 1700000000
		laterEpoch = 1800000000
	)

	cases := []struct {
		name    string
		relPath string
		content string
		moves   bool
		because string
	}{
		{
			name:    "a Core test file does not move it",
			relPath: "core/src/pkg/thing/thing_test.go",
			content: "package thing\n\nfunc TestX(*testing.T) {}\n",
			moves:   false,
			because: "a test edit cannot change the shipped binary",
		},
		{
			name:    "a Core test file in another package does not move it either",
			relPath: "core/src/web/backend/api/gateway_test.go",
			content: "package api\n",
			moves:   false,
			because: "the rule is about *_test.go under core/src, not one directory",
		},
		{
			name:    "production Core source moves it",
			relPath: "core/src/pkg/thing/thing.go",
			content: "package thing\n\nvar X = 1\n",
			moves:   true,
			because: "it is compiled into the binary",
		},
		{
			name:    "a file merely named like a test does not get excluded",
			relPath: "core/src/pkg/thing/testdata_loader.go",
			content: "package thing\n",
			moves:   true,
			because: "only the _test.go suffix is excluded, not anything test-ish",
		},
		{
			name:    "dashboard production source moves it",
			relPath: "core/src/web/backend/middleware/auth.go",
			content: "package middleware\n\nconst cookie = \"pocketclaw_launcher_auth\"\n",
			moves:   true,
			because: "it is compiled into libpocketclaw-web.so, which ships",
		},
		{
			name:    "a frontend source file moves it",
			relPath: "core/src/web/frontend/src/app.tsx",
			content: "export const App = () => null\n",
			moves:   true,
			because: "it is compiled into the bundle libpocketclaw-web.so embeds",
		},
		{
			name:    "a frontend test file does not move it",
			relPath: "core/src/web/frontend/src/app.test.tsx",
			content: "it('is a test', () => {})\n",
			moves:   false,
			because: "vite build does not emit test files, so it cannot reach the binary",
		},
		{
			name:    "the build script moves it",
			relPath: "core/build-android-arm64.sh",
			content: "#!/usr/bin/env bash\necho rebuilt\n",
			moves:   true,
			because: "changing how the build runs changes the build",
		},
		{
			name:    "the resolver itself moves it",
			relPath: "core/resolve-build-time.sh",
			content: "",
			moves:   true,
			because: "the resolver must be self-provenancing; excluding it would let " +
				"a change to the dating rule go unrecorded in what it dates",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := gitInit(t)
			commit(t, root, "core/src/pkg/base/base.go", "package base\n", baseEpoch)
			atBase := resolveEpochIn(t, root)
			if atBase != fmt.Sprint(baseEpoch) {
				t.Fatalf("baseline epoch = %s, want %d", atBase, baseEpoch)
			}

			content := tc.content
			if tc.relPath == "core/resolve-build-time.sh" {
				// Keep it a working resolver; change only a comment.
				source, err := os.ReadFile(resolverPath(t))
				if err != nil {
					t.Fatal(err)
				}
				content = string(source) + "\n# touched by a test\n"
			}
			commit(t, root, tc.relPath, content, laterEpoch)

			got := resolveEpochIn(t, root)
			want := fmt.Sprint(baseEpoch)
			if tc.moves {
				want = fmt.Sprint(laterEpoch)
			}
			if got != want {
				verb := "should not have moved"
				if tc.moves {
					verb = "should have moved"
				}
				t.Errorf("committing %s: epoch = %s, want %s — it %s, because %s",
					tc.relPath, got, want, verb, tc.because)
			}
		})
	}
}

// The timestamp and the source fingerprint must describe the same production
// universe.
//
// They did not before N4K-A: this resolver already covered all of core/src
// while the fingerprint named only cmd/, pkg/, workspace/ and three root files.
// A dashboard change moved the timestamp and not the fingerprint, so
// core.staged_build_time went red while core.staged_freshness stayed green —
// the two guards disagreeing about the same binaries, which is how the N4J gap
// stayed invisible. This holds them together in the direction that matters:
// anything the fingerprint calls a production input must also date the build.
func TestBuildTimeCoversEverythingTheFingerprintDoes(t *testing.T) {
	source, err := os.ReadFile(resolverPath(t))
	if err != nil {
		t.Fatal(err)
	}
	resolver := string(source)

	// BUILD_INPUTS covers core/src wholesale, so every fingerprint root is in
	// by construction. State it, so narrowing BUILD_INPUTS later fails here.
	if !strings.Contains(resolver, `"core/src"`) {
		t.Fatal("BUILD_INPUTS no longer covers core/src wholesale; every fingerprint " +
			"root must still be a build-time input")
	}

	// And the exclusions must not remove anything the fingerprint keeps. The
	// fingerprint excludes exactly test files; so must this.
	for _, excluded := range []string{
		"core/src/**/*_test.go",
		"core/src/web/frontend/**/*.test.ts",
		"core/src/web/frontend/**/*.test.tsx",
	} {
		if !strings.Contains(resolver, excluded) {
			t.Errorf("the resolver does not exclude %s, but the fingerprint does; "+
				"a test edit would move the timestamp against binaries that cannot differ",
				excluded)
		}
	}

	// The pairing, checked against covers() rather than restated by hand.
	for _, relative := range []string{
		"web/backend/middleware/launcher_dashboard_auth.go",
		"web/frontend/src/app.tsx",
		"cmd/picoclaw/main.go",
	} {
		if !covers(relative) {
			t.Errorf("%s is a build-time input but not a fingerprint input", relative)
		}
	}
}

// The exclusion must not reach past Core's own tests.
func TestTestExclusionDoesNotLeakOutsideCoreSrc(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := gitInit(t)
	commit(t, root, "core/src/pkg/base/base.go", "package base\n", 1700000000)

	// A test file outside core/src is not a build input at all, so it cannot
	// move the epoch — for a different reason than the exclusion, and the
	// result must be the same.
	commit(t, root, "test/unit/something_test.dart", "void main() {}\n", 1800000000)
	if got := resolveEpochIn(t, root); got != "1700000000" {
		t.Errorf("a non-Core test moved the epoch: %s", got)
	}
}

// An explicit epoch still wins over everything above.
func TestExplicitEpochStillOverridesTheInputRules(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := gitInit(t)
	commit(t, root, "core/src/pkg/base/base.go", "package base\n", 1700000000)

	cmd := exec.Command(filepath.Join(root, "core/resolve-build-time.sh"), "--print-epoch")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "SOURCE_DATE_EPOCH=1234567890")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolver failed: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "1234567890" {
		t.Errorf("explicit epoch = %s, want 1234567890", got)
	}
}
