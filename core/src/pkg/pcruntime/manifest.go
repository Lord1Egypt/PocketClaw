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
)

var timeoutProfileDurations = map[TimeoutProfile]time.Duration{
	TimeoutQuick:    15 * time.Second,
	TimeoutStandard: 2 * time.Minute,
	TimeoutExtended: 10 * time.Minute,
}

// Duration returns the budget for a profile, and whether the profile is known.
func (p TimeoutProfile) Duration() (time.Duration, bool) {
	d, ok := timeoutProfileDurations[p]
	return d, ok
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
	return nil
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
