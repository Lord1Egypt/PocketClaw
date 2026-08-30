package pcruntime

import (
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Availability is whether a catalog tool can actually be run on this device.
// It is always measured, never read from the catalog: the catalog says what
// PocketClaw supports, the device says what is present.
type Availability string

const (
	AvailabilityAvailable   Availability = "available"
	AvailabilityUnavailable Availability = "unavailable"
)

// UnavailableReason explains an unavailable tool in a form the agent can act on.
type UnavailableReason string

const (
	ReasonNotInCatalog       UnavailableReason = "not_in_catalog"
	ReasonNotInstalled       UnavailableReason = "not_installed"
	ReasonNotExecutable      UnavailableReason = "not_executable"
	ReasonABIMismatch        UnavailableReason = "abi_mismatch"
	ReasonChecksumMismatch   UnavailableReason = "checksum_mismatch"
	ReasonNoBundleDirectory  UnavailableReason = "no_bundle_directory"
	ReasonUnsupportedRuntime UnavailableReason = "unsupported_runtime"
)

// VerificationResult records what integrity check actually ran, so a log reader
// can tell a hash-verified payload from one that was merely found.
type VerificationResult string

const (
	VerificationSHA256Match    VerificationResult = "sha256_match"
	VerificationSHA256Mismatch VerificationResult = "sha256_mismatch"
	VerificationABIRejected    VerificationResult = "abi_rejected"
	// VerificationPlatformOwned is the honest answer for a system binary:
	// PocketClaw did not verify it and must not imply that it did.
	VerificationPlatformOwned VerificationResult = "platform_owned"
	VerificationNotPerformed  VerificationResult = "not_performed"
)

// ErrToolUnavailable is returned when a resolvable tool is not usable here.
var ErrToolUnavailable = errors.New("runtime tool unavailable")

// ErrNotInCatalog is returned for a name the runtime does not support at all.
// It is deliberately distinct from ErrToolUnavailable: "PocketClaw does not
// support this" and "this device does not have it" call for different answers.
var ErrNotInCatalog = errors.New("tool is not in the PocketClaw runtime catalog")

// ResolvedTool is a catalog entry plus everything measured about it on device.
type ResolvedTool struct {
	Tool              Tool
	Availability      Availability
	UnavailableReason UnavailableReason
	// ExecutablePath is set only when Availability is available.
	ExecutablePath  string
	ResolvedVersion string
	Verification    VerificationResult
	ObservedSHA256  string
	Diagnostics     string
	ResolvedAt      time.Time

	// HelperPaths maps each verified helper's logical name to its packaged
	// payload. Set only when Availability is available.
	HelperPaths map[string]string
}

// Available reports whether the tool can be executed.
func (r *ResolvedTool) Available() bool { return r.Availability == AvailabilityAvailable }

// Registry is the single place tool names become executables. Nothing else in
// PocketClaw may construct a managed tool path.
type Registry struct {
	manifest *Manifest
	paths    *Paths

	byToolID  map[string]*Tool
	byCommand map[string]*Tool

	// resolveMu serialises resolution per tool so concurrent callers share one
	// verification pass instead of each hashing the same payload.
	resolveMu sync.Map // tool_id -> *sync.Mutex

	cacheMu sync.RWMutex
	cache   map[string]*ResolvedTool
}

// NewRegistry builds the registry from the manifest compiled into this build.
func NewRegistry() (*Registry, error) {
	manifest, err := LoadEmbeddedManifest()
	if err != nil {
		return nil, err
	}
	paths, err := ResolvePaths()
	if err != nil {
		return nil, err
	}
	return NewRegistryWith(manifest, paths)
}

// NewRegistryWith builds a registry from an explicit manifest and layout.
func NewRegistryWith(manifest *Manifest, paths *Paths) (*Registry, error) {
	if manifest == nil || paths == nil {
		return nil, fmt.Errorf("runtime registry needs both a manifest and a storage layout")
	}
	r := &Registry{
		manifest:  manifest,
		paths:     paths,
		byToolID:  make(map[string]*Tool, len(manifest.Tools)),
		byCommand: make(map[string]*Tool, len(manifest.Tools)),
		cache:     make(map[string]*ResolvedTool, len(manifest.Tools)),
	}
	for i := range manifest.Tools {
		tool := &manifest.Tools[i]
		r.byToolID[tool.ToolID] = tool
		for _, command := range tool.commandNames() {
			r.byCommand[command] = tool
		}
	}
	return r, nil
}

// RuntimeVersion is the contract version this registry implements.
func (r *Registry) RuntimeVersion() int { return r.manifest.RuntimeVersion }

// CatalogVersion is the version of the tool catalog itself.
func (r *Registry) CatalogVersion() string { return r.manifest.CatalogVersion }

// Paths exposes the storage layout for diagnostics.
func (r *Registry) Paths() *Paths { return r.paths }

// Lookup finds a catalog entry by tool id or by any command name it answers to.
func (r *Registry) Lookup(name string) (*Tool, bool) {
	key := strings.TrimSpace(name)
	if tool, ok := r.byToolID[key]; ok {
		return tool, true
	}
	tool, ok := r.byCommand[key]
	return tool, ok
}

// Resolve turns a tool name into a verified executable, measuring the device
// rather than trusting the catalog. Results are cached per process; the set of
// installed system binaries and the read-only bundled payload cannot change
// while the app is running.
func (r *Registry) Resolve(name string) (*ResolvedTool, error) {
	tool, ok := r.Lookup(name)
	if !ok {
		emitWarn(EventResolveFailed, map[string]any{
			"tool":   name,
			"status": string(AvailabilityUnavailable),
			"reason": string(ReasonNotInCatalog),
		})
		return nil, fmt.Errorf("%w: %q", ErrNotInCatalog, name)
	}

	if cached := r.cached(tool.ToolID); cached != nil {
		return cached, nil
	}

	lock := r.lockFor(tool.ToolID)
	lock.Lock()
	defer lock.Unlock()

	// Another caller may have resolved this tool while we waited for the lock.
	if cached := r.cached(tool.ToolID); cached != nil {
		return cached, nil
	}

	emitDebug(EventResolveStarted, map[string]any{
		"tool":   tool.ToolID,
		"source": string(tool.Delivery),
		"abi":    tool.ABI,
		"stage":  "resolve",
	})

	started := time.Now()
	resolved := r.resolveUncached(tool)
	resolved.ResolvedAt = time.Now()

	fields := map[string]any{
		"tool":                tool.ToolID,
		"source":              string(tool.Delivery),
		"status":              string(resolved.Availability),
		"verification_result": string(resolved.Verification),
		"duration_ms":         time.Since(started).Milliseconds(),
	}
	if resolved.Available() {
		fields["tool_version"] = resolved.ResolvedVersion
		fields["path"] = resolved.ExecutablePath
		emitInfo(EventResolveCompleted, fields)
	} else {
		fields["reason"] = string(resolved.UnavailableReason)
		if resolved.Diagnostics != "" {
			fields["diagnostics"] = resolved.Diagnostics
		}
		emitInfo(EventResolveFailed, fields)
	}

	r.store(tool.ToolID, resolved)
	return resolved, nil
}

func (r *Registry) resolveUncached(tool *Tool) *ResolvedTool {
	switch tool.Delivery {
	case DeliverySystem:
		return r.resolveSystem(tool)
	case DeliveryBundled:
		return r.resolveBundled(tool)
	default:
		return &ResolvedTool{
			Tool:              *tool,
			Availability:      AvailabilityUnavailable,
			UnavailableReason: ReasonUnsupportedRuntime,
			Verification:      VerificationNotPerformed,
			Diagnostics: fmt.Sprintf(
				"delivery type %q has no resolver in this build", tool.Delivery,
			),
		}
	}
}

// resolveSystem probes the platform's own executable directories. There is no
// hash to check: the OS owns these files and replaces them on system updates,
// so the honest verification result is platform_owned.
func (r *Registry) resolveSystem(tool *Tool) *ResolvedTool {
	resolved := &ResolvedTool{Tool: *tool, Verification: VerificationPlatformOwned}

	var checked []string
	for _, dir := range systemBinDirs {
		candidate := filepath.Join(dir, tool.CommandName)
		checked = append(checked, candidate)

		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		if info.Mode().Perm()&0o111 == 0 {
			resolved.Availability = AvailabilityUnavailable
			resolved.UnavailableReason = ReasonNotExecutable
			resolved.Diagnostics = fmt.Sprintf("%s exists but is not executable", candidate)
			return resolved
		}
		resolved.Availability = AvailabilityAvailable
		resolved.ExecutablePath = candidate
		resolved.ResolvedVersion = tool.Version
		return resolved
	}

	resolved.Availability = AvailabilityUnavailable
	resolved.UnavailableReason = ReasonNotInstalled
	resolved.Verification = VerificationNotPerformed
	resolved.Diagnostics = fmt.Sprintf(
		"%s is not provided by this platform image; checked %s",
		tool.CommandName, strings.Join(checked, ", "),
	)
	return resolved
}

// resolveBundled verifies a payload the APK shipped. The file lives in
// nativeLibraryDir, which is read-only to the app, so a mismatch means a broken
// or tampered install rather than something the runtime should repair.
func (r *Registry) resolveBundled(tool *Tool) *ResolvedTool {
	resolved := &ResolvedTool{Tool: *tool, Verification: VerificationNotPerformed}

	path := r.paths.BundledPath(tool)
	if path == "" {
		resolved.Availability = AvailabilityUnavailable
		resolved.UnavailableReason = ReasonNoBundleDirectory
		resolved.Diagnostics = fmt.Sprintf(
			"no bundled payload directory is known on this host; set %s", EnvLibDir,
		)
		return resolved
	}

	info, err := os.Stat(path)
	if err != nil {
		resolved.Availability = AvailabilityUnavailable
		resolved.UnavailableReason = ReasonNotInstalled
		if errors.Is(err, fs.ErrNotExist) {
			resolved.Diagnostics = fmt.Sprintf(
				"%s was not unpacked into %s; the APK must ship it as lib/%s/%s "+
					"with android:extractNativeLibs=\"true\"",
				tool.LibraryName, r.paths.LibDir, tool.ABI, tool.LibraryName,
			)
		} else {
			resolved.Diagnostics = fmt.Sprintf("cannot stat %s: %v", path, err)
		}
		return resolved
	}
	if info.Mode().Perm()&0o111 == 0 {
		resolved.Availability = AvailabilityUnavailable
		resolved.UnavailableReason = ReasonNotExecutable
		resolved.Diagnostics = fmt.Sprintf("%s is not executable", path)
		return resolved
	}

	emitDebug(EventVerifyStarted, map[string]any{
		"tool":            tool.ToolID,
		"stage":           "verify",
		"expected_sha256": tool.SHA256,
	})

	if err := verifyELFTarget(path, tool.ABI); err != nil {
		resolved.Availability = AvailabilityUnavailable
		resolved.UnavailableReason = ReasonABIMismatch
		resolved.Verification = VerificationABIRejected
		resolved.Diagnostics = err.Error()
		emitError(EventVerifyFailed, map[string]any{
			"tool":                tool.ToolID,
			"verification_result": string(VerificationABIRejected),
			"diagnostics":         err.Error(),
		})
		return resolved
	}

	sum, err := fileSHA256(path)
	if err != nil {
		resolved.Availability = AvailabilityUnavailable
		resolved.UnavailableReason = ReasonNotInstalled
		resolved.Diagnostics = fmt.Sprintf("cannot read %s for verification: %v", path, err)
		emitError(EventVerifyFailed, map[string]any{
			"tool":        tool.ToolID,
			"diagnostics": resolved.Diagnostics,
		})
		return resolved
	}
	resolved.ObservedSHA256 = sum

	if sum != tool.SHA256 {
		resolved.Availability = AvailabilityUnavailable
		resolved.UnavailableReason = ReasonChecksumMismatch
		resolved.Verification = VerificationSHA256Mismatch
		resolved.Diagnostics = fmt.Sprintf(
			"%s does not match the pinned checksum for %s %s; the install is corrupt",
			tool.LibraryName, tool.ToolID, tool.Version,
		)
		emitError(EventVerifyFailed, map[string]any{
			"tool":                tool.ToolID,
			"verification_result": string(VerificationSHA256Mismatch),
			"expected_sha256":     tool.SHA256,
			"observed_sha256":     sum,
		})
		return resolved
	}

	// A tool whose helper is missing or altered is not usable: git without a
	// verified transport helper would fail at the first https:// URL, and
	// reporting it available would move that failure somewhere less legible.
	helperPaths, helperErr := r.verifyHelpers(tool)
	if helperErr != nil {
		resolved.Availability = AvailabilityUnavailable
		resolved.UnavailableReason = helperErr.reason
		resolved.Verification = helperErr.verification
		resolved.Diagnostics = helperErr.Error()
		emitError(EventVerifyFailed, map[string]any{
			"tool":                tool.ToolID,
			"verification_result": string(helperErr.verification),
			"diagnostics":         helperErr.Error(),
		})
		return resolved
	}

	resolved.Availability = AvailabilityAvailable
	resolved.Verification = VerificationSHA256Match
	resolved.ExecutablePath = path
	resolved.ResolvedVersion = tool.Version
	resolved.HelperPaths = helperPaths
	emitInfo(EventVerifyCompleted, map[string]any{
		"tool":                tool.ToolID,
		"tool_version":        tool.Version,
		"verification_result": string(VerificationSHA256Match),
		"expected_sha256":     tool.SHA256,
	})
	return resolved
}

// verifyELFTarget rejects a payload built for the wrong machine. Running an
// x86-64 binary on an ARM64 device fails with a bare ENOEXEC; this turns that
// into a diagnosis.
func verifyELFTarget(path, declaredABI string) error {
	file, err := elf.Open(path)
	if err != nil {
		return fmt.Errorf("%s is not a readable ELF executable: %w", filepath.Base(path), err)
	}
	defer file.Close()

	wantMachine, wantClass, ok := abiELFTarget(declaredABI)
	if !ok {
		return fmt.Errorf("runtime does not know how to check ABI %q", declaredABI)
	}
	if file.Class != wantClass || file.Machine != wantMachine {
		return fmt.Errorf(
			"%s is %s/%s but the catalog declares %s",
			filepath.Base(path), file.Class, file.Machine, declaredABI,
		)
	}
	return nil
}

func abiELFTarget(abi string) (elf.Machine, elf.Class, bool) {
	switch abi {
	case "arm64-v8a":
		return elf.EM_AARCH64, elf.ELFCLASS64, true
	case "armeabi-v7a":
		return elf.EM_ARM, elf.ELFCLASS32, true
	case "x86_64":
		return elf.EM_X86_64, elf.ELFCLASS64, true
	default:
		return 0, 0, false
	}
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func (r *Registry) lockFor(toolID string) *sync.Mutex {
	lock, _ := r.resolveMu.LoadOrStore(toolID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func (r *Registry) cached(toolID string) *ResolvedTool {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	return r.cache[toolID]
}

func (r *Registry) store(toolID string, resolved *ResolvedTool) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.cache[toolID] = resolved
}

// hostSupportsManagedExecution reports whether managed execution is meaningful
// on this GOOS. The runtime targets Android and the desktop hosts PocketClaw
// already builds for; it makes no claim about others.
func hostSupportsManagedExecution() bool {
	switch runtime.GOOS {
	case "android", "linux", "darwin":
		return true
	default:
		return false
	}
}

// runtimeGOARCH is a seam for tests that need the host architecture name.
func runtimeGOARCH() string { return runtime.GOARCH }

// helperError carries the reason and verification outcome for a helper failure
// so the caller can report it the same way as a main-payload failure.
type helperError struct {
	message      string
	reason       UnavailableReason
	verification VerificationResult
}

func (e *helperError) Error() string { return e.message }

// verifyHelpers checks every distinct helper payload and returns the logical
// name to path mapping the execution environment needs.
func (r *Registry) verifyHelpers(tool *Tool) (map[string]string, *helperError) {
	if len(tool.Helpers) == 0 {
		return nil, nil
	}

	// Each payload is hashed once even when several logical names share it.
	verified := make(map[string]string, len(tool.Helpers))
	for libraryName, expected := range tool.helperPayloads() {
		path := filepath.Join(r.paths.LibDir, libraryName)

		info, err := os.Stat(path)
		if err != nil {
			return nil, &helperError{
				message: fmt.Sprintf(
					"%s needs helper payload %s, which was not unpacked into %s; "+
						"the APK must ship it as lib/%s/%s",
					tool.ToolID, libraryName, r.paths.LibDir, tool.ABI, libraryName,
				),
				reason:       ReasonNotInstalled,
				verification: VerificationNotPerformed,
			}
		}
		if info.Mode().Perm()&0o111 == 0 {
			return nil, &helperError{
				message:      fmt.Sprintf("%s is not executable", path),
				reason:       ReasonNotExecutable,
				verification: VerificationNotPerformed,
			}
		}
		if err := verifyELFTarget(path, tool.ABI); err != nil {
			return nil, &helperError{
				message:      err.Error(),
				reason:       ReasonABIMismatch,
				verification: VerificationABIRejected,
			}
		}

		sum, err := fileSHA256(path)
		if err != nil {
			return nil, &helperError{
				message:      fmt.Sprintf("cannot read %s for verification: %v", path, err),
				reason:       ReasonNotInstalled,
				verification: VerificationNotPerformed,
			}
		}
		if sum != expected {
			return nil, &helperError{
				message: fmt.Sprintf(
					"helper payload %s does not match the pinned checksum for %s %s; "+
						"the install is corrupt",
					libraryName, tool.ToolID, tool.Version,
				),
				reason:       ReasonChecksumMismatch,
				verification: VerificationSHA256Mismatch,
			}
		}
		verified[libraryName] = path
	}

	paths := make(map[string]string, len(tool.Helpers))
	for _, helper := range tool.Helpers {
		paths[helper.LogicalName] = verified[helper.LibraryName]
	}
	return paths, nil
}
