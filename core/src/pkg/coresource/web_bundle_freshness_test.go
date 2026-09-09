package coresource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The dashboard binary embeds web/backend/dist, and the fingerprint covers the
// frontend *sources* that produce it rather than the bundle itself — the bundle
// is a build output, untracked, and hashing it would fold an output into the
// fingerprint of its own inputs.
//
// That trade is only sound if the canonical build always regenerates the bundle
// from those sources. If it could reuse whatever happens to be sitting in dist/,
// the fingerprint would claim a frontend the binary does not contain: a source
// edit would move the value, the build would succeed, and the embedded UI would
// still be the previous one. These pin the chain that makes it sound.

func webMakefile(t *testing.T) string {
	t.Helper()
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}
	body, err := os.ReadFile(filepath.Join(root, "core", "src", "web", "Makefile"))
	if err != nil {
		t.Fatalf("cannot read core/src/web/Makefile: %v", err)
	}
	return string(body)
}

// The Android launcher target must depend on the frontend build, so the bundle
// is rebuilt before web/backend is compiled rather than after, or not at all.
func TestTheLauncherBuildDependsOnTheFrontendBuild(t *testing.T) {
	makefile := webMakefile(t)

	const target = "build-android-arm64:"
	idx := strings.Index(makefile, "\n"+target)
	if idx < 0 {
		t.Fatal("web/Makefile has no build-android-arm64 target")
	}
	rule := makefile[idx+1 : idx+1+len(target)+40]
	if !strings.Contains(rule, "build-frontend") {
		t.Errorf("build-android-arm64 does not depend on build-frontend: %q\n"+
			"Without it the launcher would embed whatever bundle was left in "+
			"web/backend/dist, which the fingerprint does not describe.", rule)
	}
}

// build-frontend must actually run the bundler every time. The dependency
// install above it is allowed to be conditional — reinstalling unchanged
// node_modules proves nothing — but the build must not be, or a stale bundle
// survives whenever the sources changed and the lockfile did not.
func TestTheFrontendBundleIsAlwaysRebuilt(t *testing.T) {
	makefile := webMakefile(t)

	idx := strings.Index(makefile, "\nbuild-frontend:")
	if idx < 0 {
		t.Fatal("web/Makefile has no build-frontend target")
	}
	// The recipe runs to the next target definition: a line starting at column
	// zero that is neither blank nor one of Make's conditional keywords.
	body := makefile[idx+1:]
	lines := strings.Split(body, "\n")
	recipe := []string{lines[0]}
	for _, line := range lines[1:] {
		if line == "" || strings.HasPrefix(line, "\t") ||
			strings.HasPrefix(line, "ifeq") || strings.HasPrefix(line, "ifneq") ||
			line == "else" || line == "endif" {
			recipe = append(recipe, line)
			continue
		}
		break
	}
	body = strings.Join(recipe, "\n")
	if !strings.Contains(body, "pnpm build:backend") {
		t.Error("build-frontend does not run `pnpm build:backend`")
	}
	// The guarded block is the install; the build line must sit outside any
	// conditional, which in this recipe means after the endif.
	after := body
	if e := strings.LastIndex(body, "endif"); e >= 0 {
		after = body[e:]
	}
	if !strings.Contains(after, "pnpm build:backend") {
		t.Error("`pnpm build:backend` is inside a conditional; a stale bundle " +
			"would survive a source-only change")
	}
}

// vite must write the bundle with --emptyOutDir. Without it a file removed from
// the frontend keeps shipping: vite overwrites what it emits and leaves
// everything else, so a deleted route or asset stays in the binary forever.
func TestTheBundleIsWrittenCleanIntoTheEmbeddedDirectory(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}
	raw, err := os.ReadFile(filepath.Join(root, "core", "src", "web", "frontend", "package.json"))
	if err != nil {
		t.Fatalf("cannot read the frontend package.json: %v", err)
	}
	var pkg struct {
		Name    string            `json:"name"`
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatalf("cannot parse the frontend package.json: %v", err)
	}

	build := pkg.Scripts["build:backend"]
	if build == "" {
		t.Fatal("the frontend has no build:backend script")
	}
	for _, required := range []string{"vite build", "--outDir ../backend/dist", "--emptyOutDir"} {
		if !strings.Contains(build, required) {
			t.Errorf("build:backend is missing %q: %s", required, build)
		}
	}
	// tsc -b before vite: a type error must stop the build rather than produce
	// a bundle the fingerprint then vouches for.
	if !strings.Contains(build, "tsc -b") {
		t.Errorf("build:backend does not typecheck before bundling: %s", build)
	}
}

// The bundle directory itself must stay out of the fingerprint, and its source
// tree must stay in. This is the pairing the whole arrangement rests on, so it
// is asserted here as well as in the provenance tests: if someone ever adds
// dist/ to the input set, these two must not both keep passing.
func TestTheBundleIsExcludedButItsSourceIsNot(t *testing.T) {
	if covers("web/backend/dist/index.html") {
		t.Error("the generated bundle is a fingerprint input; it must be covered " +
			"through web/frontend instead")
	}
	if !covers("web/frontend/src/main.tsx") {
		t.Error("the frontend source is not a fingerprint input, so nothing " +
			"covers the embedded bundle at all")
	}
}
