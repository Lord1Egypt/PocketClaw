package coresource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The canonical build stages two native binaries and this fingerprint speaks
// for both. Until N4K-A it spoke only for the first: web/ was outside the input
// set, so a dashboard change moved nothing and a stale libpocketclaw-web.so had
// no guard at all. These tests are the boundary, from both sides.

// TestAWebOnlyChangeMovesTheFingerprint is the N4J regression, stated directly.
//
// N4J edited core/src/web/backend/middleware/launcher_dashboard_auth.go — the
// dashboard session cookie, an auth change — and the fingerprint did not move.
// Nothing touched cmd/ or pkg/, so under the old scope nothing could have.
func TestAWebOnlyChangeMovesTheFingerprint(t *testing.T) {
	root := syntheticCore(t)
	before := fingerprintOf(t, root)

	rewrite(t, root, "web/backend/middleware/auth.go",
		"package middleware\n\nconst cookie = \"after\"\n")

	if fingerprintOf(t, root) == before {
		t.Error("a dashboard middleware change did not move the fingerprint; " +
			"a stale libpocketclaw-web.so would ship unnoticed")
	}

	// And nothing in the Core's own trees was touched, so this proves the web
	// tree carried the change rather than something leaking in from cmd/ or pkg/.
	for _, untouched := range []string{"cmd/picoclaw/main.go", "pkg/tools/python_tool.go"} {
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(untouched)))
		if err != nil {
			t.Fatalf("cannot read %s: %v", untouched, err)
		}
		if strings.Contains(string(body), "after") {
			t.Fatalf("%s was modified; the test must isolate the web tree", untouched)
		}
	}
}

// Every input that can reach either binary. Each of these alone must move the
// value, because each alone can change what ships.
func TestBothBinaryInputSetsMoveTheFingerprint(t *testing.T) {
	for _, change := range []struct {
		name     string
		binary   string
		relative string
		content  string
	}{
		{"Core entrypoint", "libpocketclaw.so", "cmd/picoclaw/main.go",
			"package main\n\nfunc main() { println(\"changed\") }\n"},
		{"shared package", "both", "pkg/tools/python_tool.go",
			"package tools\n\nconst marker = \"after\"\n"},
		{"dashboard entrypoint", "libpocketclaw-web.so", "web/backend/main.go",
			"package main\n\nfunc main() { println(\"changed\") }\n"},
		{"dashboard middleware", "libpocketclaw-web.so", "web/backend/middleware/auth.go",
			"package middleware\n\nconst cookie = \"after\"\n"},
		{"dashboard embedded icon", "libpocketclaw-web.so", "web/backend/icon.png",
			"different PNG bytes\n"},
		{"dashboard build recipe", "libpocketclaw-web.so", "web/Makefile",
			"build:\n\tgo build -tags stdjson ./backend/\n"},
		// The frontend is how the embedded bundle is produced, so its sources
		// are inputs to the dashboard binary even though the bundle itself is
		// not fingerprinted.
		{"frontend component", "libpocketclaw-web.so", "web/frontend/src/app.tsx",
			"export const App = () => 'changed'\n"},
		{"frontend entry document", "libpocketclaw-web.so", "web/frontend/index.html",
			"<!doctype html><title>changed</title>\n"},
		{"frontend bundler config", "libpocketclaw-web.so", "web/frontend/vite.config.ts",
			"export default { base: '/changed/' }\n"},
		{"frontend dependency pins", "libpocketclaw-web.so", "web/frontend/pnpm-lock.yaml",
			"lockfileVersion: 9.1\n"},
		{"frontend static asset", "libpocketclaw-web.so", "web/frontend/public/favicon.svg",
			"<svg viewBox=\"0 0 1 1\"/>\n"},
	} {
		t.Run(change.name, func(t *testing.T) {
			root := syntheticCore(t)
			before := fingerprintOf(t, root)
			rewrite(t, root, change.relative, change.content)
			if fingerprintOf(t, root) == before {
				t.Errorf("changing %s did not move the fingerprint; it can change %s",
					change.relative, change.binary)
			}
		})
	}
}

// The other direction. A fingerprint that moves for things that cannot reach
// either binary is a fingerprint nobody trusts, and two of these would be
// actively wrong rather than merely noisy.
func TestWebInputsThatCannotReachTheBinary(t *testing.T) {
	for _, change := range []struct {
		name     string
		relative string
		content  string
		why      string
	}{
		{"a Go test under web/", "web/backend/middleware/auth_test.go",
			"package middleware\n\n// rewritten test\n",
			"a test edit cannot change the shipped binary"},
		{"a Vitest file", "web/frontend/src/app.test.tsx",
			"it('is a rewritten test', () => {})\n",
			"vite build does not emit test files"},
		// These two are the ones that would make the value wrong. The bundle is
		// a build output of the tree already covered above, and node_modules is
		// installed rather than tracked; either would make the fingerprint
		// depend on whether the builder had run pnpm, so two clean checkouts
		// would disagree and the guard would fire at random.
		{"the generated bundle", "web/backend/dist/index.html",
			"<!doctype html><!-- rebuilt -->\n",
			"dist/ is a build output of web/frontend, which is covered instead"},
		{"an installed dependency", "web/frontend/node_modules/dep/index.js",
			"a reinstalled dependency\n",
			"node_modules is installed, not tracked"},
	} {
		t.Run(change.name, func(t *testing.T) {
			root := syntheticCore(t)
			before := fingerprintOf(t, root)
			rewrite(t, root, change.relative, change.content)
			if fingerprintOf(t, root) != before {
				t.Errorf("changing %s moved the fingerprint; %s", change.relative, change.why)
			}
		})
	}
}

// Adding a file to the generated bundle must be as inert as changing one.
// Walking dist/ at all is the mistake, not just hashing what is in it.
func TestGrowingTheGeneratedBundleIsInert(t *testing.T) {
	root := syntheticCore(t)
	before := fingerprintOf(t, root)

	full := filepath.Join(root, filepath.FromSlash("web/backend/dist/assets/index-def456.js"))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("cannot create the bundle directory: %v", err)
	}
	if err := os.WriteFile(full, []byte("a newly emitted chunk\n"), 0o644); err != nil {
		t.Fatalf("cannot write the bundle chunk: %v", err)
	}

	if fingerprintOf(t, root) != before {
		t.Error("a newly emitted bundle chunk moved the fingerprint; " +
			"rebuilding the frontend would make the tree look like new source")
	}
}

// The staged pair is one provenance unit. A freshness check that reads only the
// Core passes while the dashboard on the device is from other source, which is
// exactly the shape N4J left behind.
func TestStagedFreshnessCoversBothShippingBinaries(t *testing.T) {
	if len(StagedCoreBinaries) < 2 {
		t.Fatalf("StagedCoreBinaries = %v, want both shipping binaries", StagedCoreBinaries)
	}
	for _, want := range []string{"libpocketclaw.so", "libpocketclaw-web.so"} {
		found := false
		for _, lib := range StagedCoreBinaries {
			if lib == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s is not checked for staleness; it ships", want)
		}
	}

	fingerprint := fingerprintOf(t, syntheticCore(t))
	fresh := []byte("ELF...\x00" + fingerprint + "\x00")
	stale := []byte("ELF...\x00" + strings.Repeat("0", 64) + "\x00")

	// The asymmetric case the pair check exists for: Core current, dashboard not.
	if !StagedCoreMatches(fresh, fingerprint) {
		t.Fatal("the fresh fixture does not match; the case below would prove nothing")
	}
	if StagedCoreMatches(stale, fingerprint) {
		t.Error("a dashboard built from other source was reported as current")
	}
}

// The real repository, not a fixture. A synthetic tree proves the rule; this
// proves the rule is pointed at the files that actually ship.
//
// It asserts against collect() rather than covers() alone, because those can
// disagree: covers() is only a predicate, and a file it accepts is still
// invisible if the walk never reaches its directory. That is precisely the
// pre-N4K-A shape — covers() said yes to web/backend/*.go all along, while
// includedRoots meant nothing ever asked.
func TestRealShippingSourceIsActuallyHashed(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout; there is no Core source to read")
	}
	hashed, err := collect(filepath.Join(root, "core", "src"))
	if err != nil {
		t.Fatalf("cannot collect the Core source: %v", err)
	}

	for _, relative := range []string{
		// The file N4J changed, and the reason this phase exists.
		"web/backend/middleware/launcher_dashboard_auth.go",
		"web/backend/main.go",
		"web/frontend/src/main.tsx",
		"web/frontend/package.json",
		"cmd/picoclaw/main.go",
	} {
		if _, ok := hashed[relative]; !ok {
			t.Errorf("%s ships but is not hashed into the fingerprint", relative)
		}
	}
	for _, relative := range []string{
		"web/backend/middleware/launcher_dashboard_auth_test.go",
		"web/backend/dist/index.html",
	} {
		if _, ok := hashed[relative]; ok {
			t.Errorf("%s is not a build input but is hashed", relative)
		}
	}
}

func TestRealShippingSourceIsInsideTheContract(t *testing.T) {
	for _, relative := range []string{
		// The file N4J changed.
		"web/backend/middleware/launcher_dashboard_auth.go",
		"web/backend/main.go",
		"web/backend/api/auth.go",
		"web/Makefile",
		"web/frontend/package.json",
		"web/frontend/vite.config.ts",
		"cmd/picoclaw/main.go",
	} {
		if !covers(relative) {
			t.Errorf("%s ships but is not a fingerprint input", relative)
		}
	}
	for _, relative := range []string{
		"web/backend/middleware/launcher_dashboard_auth_test.go",
		"web/backend/dist/index.html",
		"web/frontend/src/aperture-tokens.test.ts",
	} {
		if covers(relative) {
			t.Errorf("%s is not a build input but is fingerprinted", relative)
		}
	}
}

// Both build roots the canonical script compiles must be represented, or the
// contract has a side it does not describe.
func TestBothBuildRootsAreRepresented(t *testing.T) {
	roots := make(map[string]bool, len(includedRoots))
	for _, r := range includedRoots {
		roots[r] = true
	}
	// ./cmd/picoclaw -> libpocketclaw.so, ./web/backend -> libpocketclaw-web.so.
	for root, binary := range map[string]string{
		"cmd": "libpocketclaw.so",
		"web": "libpocketclaw-web.so",
	} {
		if !roots[root] {
			t.Errorf("%s/ is not a fingerprint root, so %s has no provenance", root, binary)
		}
	}
}

// Determinism, restated for the widened set: the web tree brought in a
// directory that is pruned during the walk, and a prune that depended on walk
// order would show up here.
func TestTheWidenedFingerprintIsStableAcrossInvocations(t *testing.T) {
	root := syntheticCore(t)
	first := fingerprintOf(t, root)
	for i := 0; i < 4; i++ {
		if again := fingerprintOf(t, root); again != first {
			t.Fatalf("invocation %d produced %s, want %s", i+2, again, first)
		}
	}

	other := syntheticCore(t)
	if fingerprintOf(t, other) != first {
		t.Error("the same source in two directories produced different fingerprints")
	}
}
