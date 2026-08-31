package pcruntime

import (
	"os"
	"sort"

	"github.com/sipeed/picoclaw/pkg/coresource"
)

// InventoryTool is one catalog entry as it stands on this device.
type InventoryTool struct {
	ToolID            string             `json:"tool_id"`
	DisplayName       string             `json:"display_name"`
	CommandName       string             `json:"command_name"`
	Version           string             `json:"version"`
	ResolvedVersion   string             `json:"resolved_version,omitempty"`
	ABI               string             `json:"abi"`
	Delivery          DeliveryType       `json:"delivery_type"`
	Availability      Availability       `json:"availability"`
	UnavailableReason UnavailableReason  `json:"unavailable_reason,omitempty"`
	SecurityClass     SecurityClass      `json:"security_class"`
	Verification      VerificationResult `json:"verification_result"`
	Capabilities      []string           `json:"capabilities"`
	Dependencies      []string           `json:"dependencies,omitempty"`
	TimeoutProfile    TimeoutProfile     `json:"timeout_profile"`
	MaxOutputBytes    int64              `json:"max_output_bytes"`
	// ExecutablePath and SizeBytes are set for available tools only.
	ExecutablePath string `json:"executable_path,omitempty"`
	SizeBytes      int64  `json:"size_bytes,omitempty"`
	Diagnostics    string `json:"diagnostics,omitempty"`
}

// Inventory is the read-only view of the managed runtime. It is what a future
// Runtime/Statistics page would render; this milestone only exposes the data.
type Inventory struct {
	RuntimeVersion int             `json:"runtime_version"`
	CatalogVersion string          `json:"catalog_version"`
	LibDir         string          `json:"lib_dir"`
	MetadataDir    string          `json:"metadata_dir"`
	Workspace      string          `json:"workspace"`
	Probe          *ExecutionProbe `json:"probe,omitempty"`
	Tools          []InventoryTool `json:"tools"`
	AvailableCount int             `json:"available_count"`
	TotalCount     int             `json:"total_count"`
	BundledBytes   int64           `json:"bundled_bytes"`
}

// Inventory resolves every catalog tool and reports the result. It performs no
// mutation, which is what makes it safe to call from a status endpoint.
func (m *Manager) Inventory(probe *ExecutionProbe) *Inventory {
	registry := m.registry
	inventory := &Inventory{
		RuntimeVersion: registry.RuntimeVersion(),
		CatalogVersion: registry.CatalogVersion(),
		LibDir:         registry.paths.LibDir,
		MetadataDir:    registry.paths.MetadataDir,
		Workspace:      registry.paths.Workspace,
		Probe:          probe,
		Tools:          make([]InventoryTool, 0, len(registry.manifest.Tools)),
	}

	for i := range registry.manifest.Tools {
		tool := &registry.manifest.Tools[i]
		resolved, err := registry.Resolve(tool.ToolID)
		if err != nil {
			continue
		}

		entry := InventoryTool{
			ToolID:            tool.ToolID,
			DisplayName:       tool.DisplayName,
			CommandName:       tool.CommandName,
			Version:           tool.Version,
			ResolvedVersion:   resolved.ResolvedVersion,
			ABI:               tool.ABI,
			Delivery:          tool.Delivery,
			Availability:      resolved.Availability,
			UnavailableReason: resolved.UnavailableReason,
			SecurityClass:     tool.SecurityClass,
			Verification:      resolved.Verification,
			Capabilities:      tool.Capabilities,
			Dependencies:      tool.Dependencies,
			TimeoutProfile:    tool.TimeoutProfile,
			MaxOutputBytes:    tool.MaxOutputBytes,
			Diagnostics:       resolved.Diagnostics,
		}
		if resolved.Available() {
			entry.ExecutablePath = resolved.ExecutablePath
			inventory.AvailableCount++
			if info, statErr := os.Stat(resolved.ExecutablePath); statErr == nil {
				entry.SizeBytes = info.Size()
				if tool.Delivery == DeliveryBundled {
					inventory.BundledBytes += info.Size()
				}
			}
		}
		inventory.Tools = append(inventory.Tools, entry)
	}

	sort.Slice(inventory.Tools, func(a, b int) bool {
		return inventory.Tools[a].ToolID < inventory.Tools[b].ToolID
	})
	inventory.TotalCount = len(inventory.Tools)

	emitInfo(EventInventoryCompleted, map[string]any{
		"runtime_version": inventory.RuntimeVersion,
		"catalog_version": inventory.CatalogVersion,
		"available_count": inventory.AvailableCount,
		"total_count":     inventory.TotalCount,
		"bundled_bytes":   inventory.BundledBytes,
		// Which Core source this binary was built from. Paired with
		// catalog_version it separates the two ways a device can be behind:
		// a Core that predates the catalog, and a Core that predates the code.
		"core_source": coresource.Describe(),
	})
	return inventory
}
