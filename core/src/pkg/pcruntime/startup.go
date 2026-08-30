package pcruntime

import (
	"context"
	"sync"
)

// startupDiagnosticsOnce keeps the startup record to one per process. The
// runtime tool is constructed once per agent instance, and several agents may
// exist, but the platform probe and the tool inventory describe the device — not
// the agent — so repeating them would only pad the Debug Logs.
var startupDiagnosticsOnce sync.Once

// LogStartupDiagnostics records what the runtime found on this device: the
// execution probe's verdict and the full tool inventory.
//
// It runs without waiting for a tool to be invoked, so a device's Debug Logs
// show whether the runtime initialised and what it resolved even when the agent
// never reaches for a tool. That is also what makes the probe's answer visible
// on real hardware rather than assumed.
func (m *Manager) LogStartupDiagnostics(ctx context.Context) {
	startupDiagnosticsOnce.Do(func() {
		probe := ProbeExecution(ctx, m.registry.paths)
		m.Inventory(probe)
	})
}
