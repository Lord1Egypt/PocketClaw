package pcruntime

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ManifestVersion is the runtime contract version this build understands.
// Bump it when the shape of a catalog entry changes, never for catalog edits.
const ManifestVersion = 1

// DeliveryType is how a tool's executable reaches the device. Both members are
// read-only to the app; see the package comment for why there is no third.
type DeliveryType string

const (
	// DeliverySystem is an executable the platform ships in /system/bin.
	// Its integrity is the platform's, so it carries no PocketClaw hash.
	DeliverySystem DeliveryType = "system"
	// DeliveryBundled is an executable shipped inside the APK as lib*.so and
	// unpacked by the package manager into nativeLibraryDir at install time.
	DeliveryBundled DeliveryType = "bundled"
)

// SecurityClass records what integrity guarantee a tool actually has. It exists
// so diagnostics never overstate the guarantee for platform-owned binaries.
type SecurityClass string

const (
	// SecurityClassSystem: integrity comes from the OS image. PocketClaw does
	// not pin a hash and must not claim to have verified one.
	SecurityClassSystem SecurityClass = "system"
	// SecurityClassBundledVerified: PocketClaw built the payload and pinned its
	// SHA-256 at build time; the resolver checks the on-device file against it.
	SecurityClassBundledVerified SecurityClass = "bundled_verified"
)

// TimeoutProfile names a bounded execution budget. Every tool declares one, so
// no catalog entry can be added without an answer to "how long may this run".
type TimeoutProfile string

const (
	TimeoutQuick    TimeoutProfile = "quick"
	TimeoutStandard TimeoutProfile = "standard"
	TimeoutExtended TimeoutProfile = "extended"
	// TimeoutTransfer is for tools whose work is bounded by a network peer
	// rather than by local computation: cloning a large repository or uploading
	// a release asset is not a utility command and must not share their budget.
	TimeoutTransfer TimeoutProfile = "transfer"
)

var timeoutProfileDurations = map[TimeoutProfile]time.Duration{
	TimeoutQuick:    15 * time.Second,
	TimeoutStandard: 2 * time.Minute,
	TimeoutExtended: 10 * time.Minute,
	TimeoutTransfer: 30 * time.Minute,
}

// Duration returns the budget for a profile, and whether the profile is known.
func (p TimeoutProfile) Duration() (time.Duration, bool) {
	d, ok := timeoutProfileDurations[p]
	return d, ok
}

// EnvironmentProfile names the per-tool environment the runtime prepares before
// execution. Tools that need nothing beyond the base allowlist leave it empty.
type EnvironmentProfile string

const (
	EnvironmentProfileNone EnvironmentProfile = ""
	EnvironmentProfileGit  EnvironmentProfile = "git"
	EnvironmentProfileGH   EnvironmentProfile = "gh"
	// EnvironmentProfilePython points the interpreter at the standard library
	// appended to its own payload. It injects no credential of any kind.
	EnvironmentProfilePython EnvironmentProfile = "python"
)

var knownEnvironmentProfiles = map[EnvironmentProfile]struct{}{
	EnvironmentProfileNone:   {},
	EnvironmentProfileGit:    {},
	EnvironmentProfileGH:     {},
	EnvironmentProfilePython: {},
}

// Helper is an auxiliary executable a tool invokes by a logical name that the
// APK cannot use as a filename.
//
// Android's package manager only unpacks lib/<abi>/*.so, so git's transport
// helper cannot ship as "git-remote-https". It ships under a lib*.so name and
// the runtime presents it under its logical name at execution time. Several
// logical names may share one payload: git-remote-http and git-remote-https are
// the same binary upstream.
type Helper struct {
	LogicalName string `json:"logical_name"`
	LibraryName string `json:"library_name"`
	SHA256      string `json:"sha256"`
}

// Tool is one catalog entry: everything the runtime needs to resolve, verify,
// and bound a single managed executable.
type Tool struct {
	ToolID         string         `json:"tool_id"`
	DisplayName    string         `json:"display_name"`
	CommandName    string         `json:"command_name"`
	Version        string         `json:"version"`
	ABI            string         `json:"abi"`
	Delivery       DeliveryType   `json:"delivery_type"`
	TrustedSource  string         `json:"trusted_source"`
	SHA256         string         `json:"sha256,omitempty"`
	Capabilities   []string       `json:"capabilities"`
	Dependencies   []string       `json:"dependencies,omitempty"`
	TimeoutProfile TimeoutProfile `json:"timeout_profile"`
	MaxOutputBytes int64          `json:"max_output_bytes"`
	SecurityClass  SecurityClass  `json:"security_class"`

	// LibraryName is the lib*.so entry the package manager unpacks into
	// nativeLibraryDir. Bundled tools only; the name must match that pattern or
	// the installer will not extract it and the tool can never resolve.
	LibraryName string `json:"library_name,omitempty"`

	// Applets are the command names a multicall binary answers to. Empty for a
	// single-purpose tool.
	Applets []string `json:"applets,omitempty"`

	// Helpers are auxiliary executables this tool invokes by logical name.
	Helpers []Helper `json:"helpers,omitempty"`

	// EnvironmentProfile selects the per-tool environment preparation.
	EnvironmentProfile EnvironmentProfile `json:"environment_profile,omitempty"`

	// DefaultArgs are prepended to every invocation, ahead of the caller's own
	// arguments. They exist for interpreters whose execution mode has to be
	// fixed by the runtime rather than left to the caller: Python's -S, for
	// one, is what stops a workspace sitecustomize.py from being imported and
	// executed automatically. There is no env-var equivalent for it.
	DefaultArgs []string `json:"default_args,omitempty"`
}

// Manifest is the versioned runtime catalog.
type Manifest struct {
	RuntimeVersion int    `json:"runtime_version"`
	CatalogVersion string `json:"catalog_version"`
	Tools          []Tool `json:"tools"`
}

//go:embed manifest.json
var embeddedManifestJSON []byte

// LoadEmbeddedManifest parses and validates the catalog compiled into this
// build. A malformed catalog is a build defect, so callers get the error rather
// than a runtime with a silently empty tool list.
func LoadEmbeddedManifest() (*Manifest, error) {
	return ParseManifest(embeddedManifestJSON)
}

// ParseManifest decodes and validates a runtime catalog.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		return nil, fmt.Errorf("runtime manifest is not valid JSON: %w", err)
	}
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Manifest) validate() error {
	if m.RuntimeVersion != ManifestVersion {
		return fmt.Errorf(
			"runtime manifest version %d is not supported by this build (expected %d)",
			m.RuntimeVersion, ManifestVersion,
		)
	}
	if strings.TrimSpace(m.CatalogVersion) == "" {
		return fmt.Errorf("runtime manifest is missing catalog_version")
	}
	if len(m.Tools) == 0 {
		return fmt.Errorf("runtime manifest declares no tools")
	}

	seenID := make(map[string]struct{}, len(m.Tools))
	seenCommand := make(map[string]string, len(m.Tools))
	for i := range m.Tools {
		tool := &m.Tools[i]
		if err := tool.validate(); err != nil {
			return err
		}
		if _, dup := seenID[tool.ToolID]; dup {
			return fmt.Errorf("runtime manifest declares tool_id %q twice", tool.ToolID)
		}
		seenID[tool.ToolID] = struct{}{}

		// One command name must resolve to exactly one tool, or the resolver
		// would silently pick an arbitrary provider.
		for _, command := range tool.commandNames() {
			if owner, dup := seenCommand[command]; dup {
				return fmt.Errorf(
					"runtime manifest maps command %q to both %q and %q",
					command, owner, tool.ToolID,
				)
			}
			seenCommand[command] = tool.ToolID
		}
	}
	return nil
}

func (t *Tool) validate() error {
	if strings.TrimSpace(t.ToolID) == "" {
		return fmt.Errorf("runtime manifest has a tool with an empty tool_id")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"display_name", t.DisplayName},
		{"command_name", t.CommandName},
		{"version", t.Version},
		{"abi", t.ABI},
		{"trusted_source", t.TrustedSource},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("runtime tool %q is missing %s", t.ToolID, field.name)
		}
	}
	if _, ok := t.TimeoutProfile.Duration(); !ok {
		return fmt.Errorf(
			"runtime tool %q declares unknown timeout_profile %q",
			t.ToolID, t.TimeoutProfile,
		)
	}
	if t.MaxOutputBytes <= 0 {
		return fmt.Errorf("runtime tool %q must declare a positive max_output_bytes", t.ToolID)
	}
	if len(t.Capabilities) == 0 {
		return fmt.Errorf("runtime tool %q declares no capabilities", t.ToolID)
	}

	switch t.Delivery {
	case DeliverySystem:
		if t.SecurityClass != SecurityClassSystem {
			return fmt.Errorf(
				"runtime tool %q is system-delivered and must use security_class %q",
				t.ToolID, SecurityClassSystem,
			)
		}
		// A hash on a platform binary would be a guarantee PocketClaw cannot
		// keep: the OS owns the file and replaces it on every system update.
		if t.SHA256 != "" {
			return fmt.Errorf("runtime tool %q is system-delivered and must not pin a sha256", t.ToolID)
		}
		if t.LibraryName != "" {
			return fmt.Errorf("runtime tool %q is system-delivered and must not declare library_name", t.ToolID)
		}
	case DeliveryBundled:
		if t.SecurityClass != SecurityClassBundledVerified {
			return fmt.Errorf(
				"runtime tool %q is bundled and must use security_class %q",
				t.ToolID, SecurityClassBundledVerified,
			)
		}
		if !isSHA256Hex(t.SHA256) {
			return fmt.Errorf("runtime tool %q is bundled and must pin a 64-character sha256", t.ToolID)
		}
		// The package manager only unpacks lib/<abi>/*.so. A name outside that
		// pattern never reaches nativeLibraryDir, so the tool could never run.
		if !strings.HasPrefix(t.LibraryName, "lib") || !strings.HasSuffix(t.LibraryName, ".so") {
			return fmt.Errorf(
				"runtime tool %q declares library_name %q; bundled payloads must be named lib*.so "+
					"or the Android package manager will not extract them",
				t.ToolID, t.LibraryName,
			)
		}
	default:
		return fmt.Errorf("runtime tool %q declares unknown delivery_type %q", t.ToolID, t.Delivery)
	}

	for i, arg := range t.DefaultArgs {
		if strings.TrimSpace(arg) == "" {
			return fmt.Errorf(
				"runtime tool %q declares an empty default_args entry at index %d",
				t.ToolID, i,
			)
		}
	}
	if _, known := knownEnvironmentProfiles[t.EnvironmentProfile]; !known {
		return fmt.Errorf(
			"runtime tool %q declares unknown environment_profile %q",
			t.ToolID, t.EnvironmentProfile,
		)
	}
	return t.validateHelpers()
}

func (t *Tool) validateHelpers() error {
	if len(t.Helpers) == 0 {
		return nil
	}
	if t.Delivery != DeliveryBundled {
		return fmt.Errorf(
			"runtime tool %q declares helpers but is %s-delivered; only bundled payloads have helpers",
			t.ToolID, t.Delivery,
		)
	}

	// One payload may back several logical names, but the catalog must agree
	// with itself about that payload's checksum, or verification would depend
	// on which entry happened to be read first.
	checksums := make(map[string]string, len(t.Helpers))
	logical := make(map[string]struct{}, len(t.Helpers))
	for _, helper := range t.Helpers {
		if strings.TrimSpace(helper.LogicalName) == "" {
			return fmt.Errorf("runtime tool %q declares a helper with no logical_name", t.ToolID)
		}
		// The logical name becomes a filename in the runtime's helper directory.
		// Anything path-like there would let a catalog entry place a symlink
		// outside it.
		if strings.ContainsAny(helper.LogicalName, `/\`) || helper.LogicalName == "." || helper.LogicalName == ".." {
			return fmt.Errorf(
				"runtime tool %q declares helper logical_name %q; it is used as a filename and must not be a path",
				t.ToolID, helper.LogicalName,
			)
		}
		if _, dup := logical[helper.LogicalName]; dup {
			return fmt.Errorf(
				"runtime tool %q declares helper %q twice", t.ToolID, helper.LogicalName,
			)
		}
		logical[helper.LogicalName] = struct{}{}

		if !strings.HasPrefix(helper.LibraryName, "lib") || !strings.HasSuffix(helper.LibraryName, ".so") {
			return fmt.Errorf(
				"runtime tool %q helper %q declares library_name %q; bundled payloads must be named lib*.so "+
					"or the Android package manager will not extract them",
				t.ToolID, helper.LogicalName, helper.LibraryName,
			)
		}
		if !isSHA256Hex(helper.SHA256) {
			return fmt.Errorf(
				"runtime tool %q helper %q must pin a 64-character sha256",
				t.ToolID, helper.LogicalName,
			)
		}
		if existing, seen := checksums[helper.LibraryName]; seen && existing != helper.SHA256 {
			return fmt.Errorf(
				"runtime tool %q pins two different checksums for payload %q",
				t.ToolID, helper.LibraryName,
			)
		}
		checksums[helper.LibraryName] = helper.SHA256
	}
	return nil
}

// helperPayloads returns each distinct packaged payload the tool's helpers use.
func (t *Tool) helperPayloads() map[string]string {
	payloads := make(map[string]string, len(t.Helpers))
	for _, helper := range t.Helpers {
		payloads[helper.LibraryName] = helper.SHA256
	}
	return payloads
}

// commandNames is every command this tool answers to.
func (t *Tool) commandNames() []string {
	if len(t.Applets) == 0 {
		return []string{t.CommandName}
	}
	names := make([]string, 0, len(t.Applets)+1)
	names = append(names, t.CommandName)
	for _, applet := range t.Applets {
		if applet != t.CommandName {
			names = append(names, applet)
		}
	}
	return names
}

func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
		if !isHex {
			return false
		}
	}
	return true
}
