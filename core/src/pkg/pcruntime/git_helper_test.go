package pcruntime

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// gitCatalogEntry returns the shipped git entry.
func gitCatalogEntry(t *testing.T) *Tool {
	t.Helper()
	manifest, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatalf("cannot load catalog: %v", err)
	}
	for i := range manifest.Tools {
		if manifest.Tools[i].ToolID == "git" {
			return &manifest.Tools[i]
		}
	}
	t.Fatal("the catalog declares no git tool")
	return nil
}

// git does not exec its transport helper directly. It spawns "git remote-https"
// and resolves the literal name "git" through PATH, which setup_path() builds by
// prepending GIT_EXEC_PATH. If the helper directory holds only the two remote
// helpers, that lookup fails with ENOENT and git reports the thoroughly
// misleading "unable to find remote helper for 'https'" — which is exactly the
// bug this test exists to prevent recurring.
func TestGitDeclaresItselfAsAHelperSoItsOwnLookupSucceeds(t *testing.T) {
	git := gitCatalogEntry(t)

	var found bool
	for _, helper := range git.Helpers {
		if helper.LogicalName == "git" {
			found = true
			if helper.LibraryName != git.LibraryName {
				t.Fatalf("the git helper must point at git's own payload %q, got %q",
					git.LibraryName, helper.LibraryName)
			}
			if helper.SHA256 != git.SHA256 {
				t.Fatalf("the git helper pins %s but git's payload pins %s",
					helper.SHA256, git.SHA256)
			}
		}
	}
	if !found {
		t.Fatal(
			"git does not declare itself as a helper. It spawns remote helpers as " +
				"`git remote-<protocol>` and resolves the literal name `git` through " +
				"PATH, so without this entry every https:// clone fails with " +
				"\"unable to find remote helper for 'https'\".",
		)
	}
}

func TestGitDeclaresBothRemoteHelpers(t *testing.T) {
	git := gitCatalogEntry(t)

	names := make([]string, 0, len(git.Helpers))
	for _, helper := range git.Helpers {
		names = append(names, helper.LogicalName)
	}
	sort.Strings(names)

	for _, required := range []string{"git", "git-remote-http", "git-remote-https"} {
		var present bool
		for _, name := range names {
			if name == required {
				present = true
			}
		}
		if !present {
			t.Fatalf("git must declare helper %q; declares %v", required, names)
		}
	}
}

// http and https are one upstream binary under two names. The catalog must say
// so, because two payloads would double the shipped size for nothing and two
// different checksums for one file would make verification order-dependent.
func TestGitRemoteHelpersShareOnePayload(t *testing.T) {
	git := gitCatalogEntry(t)

	var http, https string
	for _, helper := range git.Helpers {
		switch helper.LogicalName {
		case "git-remote-http":
			http = helper.LibraryName
		case "git-remote-https":
			https = helper.LibraryName
		}
	}
	if http == "" || https == "" {
		t.Fatal("both remote helpers must be declared")
	}
	if http != https {
		t.Fatalf("git-remote-http and git-remote-https must share one payload, got %q and %q",
			http, https)
	}
}

func TestGitUsesTheGitEnvironmentProfileAndTransferTimeout(t *testing.T) {
	git := gitCatalogEntry(t)

	if git.EnvironmentProfile != EnvironmentProfileGit {
		t.Fatalf("git must use the git environment profile, got %q", git.EnvironmentProfile)
	}
	// A clone is bounded by a network peer, not by local computation.
	if git.TimeoutProfile != TimeoutTransfer {
		t.Fatalf("git must use the transfer timeout profile, got %q", git.TimeoutProfile)
	}
}

// gh shells out to git, so it needs the same three entries on its PATH.
func TestGHDeclaresGitAndItsRemoteHelpers(t *testing.T) {
	manifest, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatalf("cannot load catalog: %v", err)
	}
	var gh *Tool
	for i := range manifest.Tools {
		if manifest.Tools[i].ToolID == "gh" {
			gh = &manifest.Tools[i]
		}
	}
	if gh == nil {
		t.Fatal("the catalog declares no gh tool")
	}
	for _, required := range []string{"git", "git-remote-https"} {
		var present bool
		for _, helper := range gh.Helpers {
			if helper.LogicalName == required {
				present = true
			}
		}
		if !present {
			t.Fatalf("gh must declare helper %q so `gh repo clone` can reach git", required)
		}
	}
}

// The prepared directory must contain every declared logical name, as a symlink
// to the packaged payload, and never a copy.
func TestPreparedGitExecPathContainsEveryDeclaredHelper(t *testing.T) {
	root := t.TempDir()
	libDir := filepath.Join(root, "lib")
	workspace := filepath.Join(root, "workspace")
	for _, dir := range []string{libDir, workspace} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", dir, err)
		}
	}

	self, err := os.Executable()
	if err != nil {
		t.Skipf("cannot locate an ELF fixture: %v", err)
	}
	mainPayload := filepath.Join(libDir, "libpocketclaw-git.so")
	helperPayload := filepath.Join(libDir, "libpocketclaw-git-remote-http.so")
	for _, target := range []string{mainPayload, helperPayload} {
		if err := copyFileMode(self, target, 0o755); err != nil {
			t.Fatalf("cannot stage payload: %v", err)
		}
	}
	mainSum, err := fileSHA256(mainPayload)
	if err != nil {
		t.Fatalf("cannot hash payload: %v", err)
	}

	tool := Tool{
		ToolID: "git", DisplayName: "git", CommandName: "git", Version: "2.51.0",
		ABI: hostABI(t), Delivery: DeliveryBundled, TrustedSource: "test",
		SHA256: mainSum, Capabilities: []string{"vcs.repository"},
		TimeoutProfile: TimeoutTransfer, MaxOutputBytes: 4096,
		SecurityClass: SecurityClassBundledVerified,
		LibraryName:   "libpocketclaw-git.so",
		Helpers: []Helper{
			{LogicalName: "git", LibraryName: "libpocketclaw-git.so", SHA256: mainSum},
			{LogicalName: "git-remote-http", LibraryName: "libpocketclaw-git-remote-http.so", SHA256: mainSum},
			{LogicalName: "git-remote-https", LibraryName: "libpocketclaw-git-remote-http.so", SHA256: mainSum},
		},
		EnvironmentProfile: EnvironmentProfileGit,
	}
	registry, err := NewRegistryWith(newTestManifest(t, tool), &Paths{
		LibDir: libDir, MetadataDir: filepath.Join(root, "runtime"), Workspace: workspace,
	})
	if err != nil {
		t.Fatalf("cannot build registry: %v", err)
	}
	manager, err := NewManagerWith(registry, workspace)
	if err != nil {
		t.Fatalf("cannot build manager: %v", err)
	}

	resolved, err := registry.Resolve("git")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	prepared, err := manager.prepareEnvironment(resolved)
	if err != nil {
		t.Fatalf("cannot prepare environment: %v", err)
	}

	execPath := prepared.variables["GIT_EXEC_PATH"]
	if execPath == "" {
		t.Fatal("the git profile set no GIT_EXEC_PATH")
	}
	for _, name := range []string{"git", "git-remote-http", "git-remote-https"} {
		link := filepath.Join(execPath, name)
		info, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("%s is missing from GIT_EXEC_PATH: %v", name, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s is a real file; it must be a symlink to the packaged payload", name)
		}
		target, err := os.Readlink(link)
		if err != nil {
			t.Fatalf("cannot read link %s: %v", name, err)
		}
		if !strings.HasPrefix(target, libDir) {
			t.Fatalf("%s points at %q, outside the packaged payload directory", name, target)
		}
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("%s points at a payload that does not exist: %v", name, err)
		}
	}
}

// Nothing may ever fall back to copying an executable into writable storage.
func TestHelperPreparationNeverCopiesAnExecutable(t *testing.T) {
	manager, helperPath := helperFixture(t, false)

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	dir, err := manager.materialiseHelpers(resolved)
	if err != nil {
		t.Fatalf("cannot materialise helpers: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read helper dir: %v", err)
	}
	for _, entry := range entries {
		info, err := os.Lstat(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("cannot stat %s: %v", entry.Name(), err)
		}
		if info.Mode().IsRegular() {
			t.Fatalf("%s is a regular file in writable storage; Android will not execute it",
				entry.Name())
		}
		if info.Size() > 512 {
			t.Fatalf("%s looks like a copied payload, not a link", entry.Name())
		}
	}
	if _, err := os.Stat(helperPath); err != nil {
		t.Fatalf("the packaged payload should be untouched: %v", err)
	}
}

func TestHelperPreparationIsLogged(t *testing.T) {
	manager, _ := helperFixture(t, false)
	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	events := captureRuntimeLog(t, func() {
		if _, err := manager.materialiseHelpers(resolved); err != nil {
			t.Fatalf("cannot materialise helpers: %v", err)
		}
	})
	requireEvents(t, events, EventHelpersPrepared)

	for _, event := range events {
		if event["event"] != EventHelpersPrepared {
			continue
		}
		if path, _ := event["exec_path"].(string); path == "" {
			t.Fatal("helper preparation must record the exec path it built")
		}
	}
}

// A working directory is what makes `git status` meaningful after `git init`.
func TestWorkingDirectoryAppliesToTheChildProcess(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("here", TimeoutQuick, 4096)))
	writeScript(t, binDir, "here", "pwd\n")

	nested := filepath.Join(manager.workspace, "test_repo")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("cannot create nested dir: %v", err)
	}

	result, err := manager.Execute(context.Background(), ExecRequest{
		Tool: "here", WorkingDirectory: "test_repo",
	})
	if err != nil {
		t.Fatalf("execute was rejected: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d", result.ExitCode)
	}
	if !strings.HasSuffix(strings.TrimSpace(result.Stdout), "test_repo") {
		t.Fatalf("the tool did not run in test_repo: %q", result.Stdout)
	}
}
