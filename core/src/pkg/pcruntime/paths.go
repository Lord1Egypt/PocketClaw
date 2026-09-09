package pcruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/canonicalenv"
)

// Environment keys the Android Service and the desktop host use to tell the
// runtime where its two storage areas are.
const (
	// EnvLibDir points at the directory the Android package manager unpacked
	// lib/<abi>/*.so into. It is read-only to the app, which is the entire
	// reason bundled executables live there.
	EnvLibDir = "POCKETCLAW_RUNTIME_LIB_DIR"
	// EnvRuntimeDir overrides where runtime metadata is kept.
	EnvRuntimeDir = "POCKETCLAW_RUNTIME_DIR"
	// EnvCoreBinary is the existing Core launch path. nativeLibraryDir is
	// derived from it when EnvLibDir is not set, because the Core binary is
	// already resolved from that directory.
	EnvCoreBinary = "PICOCLAW_BINARY"
	// EnvWorkspace is the user workspace. Runtime storage must never be inside
	// it; see Paths.
	EnvWorkspace = "PICOCLAW_HOME"
)

// systemBinDirs are the platform-owned executable directories, in probe order.
var systemBinDirs = []string{"/system/bin", "/system/xbin", "/vendor/bin"}

// Paths is the runtime's storage layout.
//
// The two areas are deliberately different kinds of place. LibDir holds
// executables and is read-only to the app. MetadataDir is writable and holds
// only inventory and probe records — never an executable, because on Android an
// executable there could not be run anyway.
type Paths struct {
	// LibDir is nativeLibraryDir. Empty when the host has no bundled payload
	// directory, which makes every bundled tool resolve unavailable.
	LibDir string
	// MetadataDir is app-private writable storage for runtime metadata.
	MetadataDir string
	// Workspace is the user workspace, kept only to prove MetadataDir is
	// outside it.
	Workspace string
}

// ResolvePaths derives the runtime storage layout from the process environment.
//
// It fails rather than guessing when metadata storage would land inside the user
// workspace: user Skills write freely there, and runtime inventory that a Skill
// can rewrite is worse than no inventory at all.
func ResolvePaths() (*Paths, error) {
	workspace := strings.TrimSpace(canonicalenv.Getenv(EnvWorkspace))

	libDir := strings.TrimSpace(os.Getenv(EnvLibDir))
	if libDir == "" {
		if coreBinary := strings.TrimSpace(canonicalenv.Getenv(EnvCoreBinary)); coreBinary != "" {
			libDir = filepath.Dir(coreBinary)
		}
	}

	metadataDir := strings.TrimSpace(os.Getenv(EnvRuntimeDir))
	if metadataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return nil, fmt.Errorf(
				"cannot locate runtime metadata storage: neither %s nor a home directory is set",
				EnvRuntimeDir,
			)
		}
		metadataDir = filepath.Join(home, "picoclaw", "runtime")
	}

	paths := &Paths{
		LibDir:      cleanOrEmpty(libDir),
		MetadataDir: filepath.Clean(metadataDir),
		Workspace:   cleanOrEmpty(workspace),
	}
	if err := paths.validate(); err != nil {
		return nil, err
	}
	return paths, nil
}

func (p *Paths) validate() error {
	if p.Workspace == "" {
		return nil
	}
	relative, err := filepath.Rel(p.Workspace, p.MetadataDir)
	if err != nil {
		return nil
	}
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "." {
		return fmt.Errorf(
			"runtime metadata storage %q is inside the user workspace %q; "+
				"managed runtime state must not be writable by user Skills",
			p.MetadataDir, p.Workspace,
		)
	}
	if relative == "." {
		return fmt.Errorf(
			"runtime metadata storage %q is the user workspace; "+
				"managed runtime state must not be writable by user Skills",
			p.MetadataDir,
		)
	}
	return nil
}

// EnsureMetadataDir creates the runtime metadata directory.
func (p *Paths) EnsureMetadataDir() error {
	if err := os.MkdirAll(p.MetadataDir, 0o700); err != nil {
		return fmt.Errorf("cannot create runtime metadata directory: %w", err)
	}
	return nil
}

// BundledPath is where a bundled tool's payload is expected on this host.
func (p *Paths) BundledPath(tool *Tool) string {
	if p.LibDir == "" || tool.LibraryName == "" {
		return ""
	}
	return filepath.Join(p.LibDir, tool.LibraryName)
}

func cleanOrEmpty(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
