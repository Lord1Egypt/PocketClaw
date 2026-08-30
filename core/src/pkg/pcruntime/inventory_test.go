package pcruntime

import (
	"context"
	"testing"
)

func TestInventoryReportsEveryCatalogToolWithMeasuredAvailability(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t,
		systemTool("present", TimeoutQuick, 4096),
		systemTool("absent", TimeoutQuick, 4096),
	))
	writeScript(t, binDir, "present", "exit 0\n")

	inventory := manager.Inventory(nil)
	if inventory.TotalCount != 2 {
		t.Fatalf("expected 2 catalog tools, got %d", inventory.TotalCount)
	}
	if inventory.AvailableCount != 1 {
		t.Fatalf("expected exactly one available tool, got %d", inventory.AvailableCount)
	}

	byID := make(map[string]InventoryTool, len(inventory.Tools))
	for _, entry := range inventory.Tools {
		byID[entry.ToolID] = entry
	}
	if byID["present"].Availability != AvailabilityAvailable {
		t.Fatal("an installed tool must be reported available")
	}
	if byID["present"].SizeBytes <= 0 {
		t.Fatal("an available tool must report its size for the runtime inventory")
	}
	if byID["absent"].Availability != AvailabilityUnavailable {
		t.Fatal("a missing tool must be reported unavailable")
	}
	if byID["absent"].ExecutablePath != "" {
		t.Fatal("an unavailable tool must not expose an executable path")
	}
}

// The probe measures the platform rather than asserting it, so the assumption
// the delivery model rests on is visible in the logs of every device.
func TestExecutionProbeReportsAConclusionOrSaysItCouldNot(t *testing.T) {
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("present", TimeoutQuick, 4096)))

	probe := ProbeExecution(context.Background(), manager.Registry().Paths())
	switch probe.WritableExec {
	case WritableExecSupported, WritableExecBlocked, WritableExecInconclusive:
	default:
		t.Fatalf("probe returned an unrecognised verdict %q", probe.WritableExec)
	}
	if probe.WritableExecDetail == "" {
		t.Fatal("the probe must always explain its verdict")
	}
	if probe.GOOS == "" {
		t.Fatal("the probe must record which platform it measured")
	}
}

// The probe copies a binary into writable storage; leaving it behind would put
// an executable in the one place the runtime promises never to keep one.
func TestExecutionProbeCleansUpAfterItself(t *testing.T) {
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("present", TimeoutQuick, 4096)))
	paths := manager.Registry().Paths()

	ProbeExecution(context.Background(), paths)

	if entries, err := readDirNames(paths.MetadataDir); err == nil {
		for _, name := range entries {
			if name == ".exec-probe" {
				t.Fatal("the probe left its staged binary in writable runtime storage")
			}
		}
	}
}

// The probe and the inventory exist to appear in a device's Debug Logs. If
// nothing invoked them, a physical test could not tell an initialised runtime
// from a broken one, so the startup record must actually be emitted.
func TestStartupDiagnosticsEmitTheProbeAndTheInventory(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("present", TimeoutQuick, 4096)))
	writeScript(t, binDir, "present", "exit 0\n")

	events := captureRuntimeLog(t, func() {
		manager.LogStartupDiagnostics(context.Background())
	})
	requireEvents(t, events, EventProbeStarted, EventProbeCompleted, EventInventoryCompleted)
}
