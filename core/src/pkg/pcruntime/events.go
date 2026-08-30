package pcruntime

import (
	"github.com/sipeed/picoclaw/pkg/logger"
)

// logComponent is the component tag every runtime log line carries, so the
// whole lifecycle can be filtered out of the Debug Logs screen as one family.
const logComponent = "runtime"

// Runtime lifecycle events. Every managed operation emits a started event and
// exactly one terminal event, so an operation can never be left with no record
// of how it ended.
const (
	EventResolveStarted   = "runtime.resolve.started"
	EventResolveCompleted = "runtime.resolve.completed"
	EventResolveFailed    = "runtime.resolve.failed"

	EventVerifyStarted   = "runtime.verify.started"
	EventVerifyCompleted = "runtime.verify.completed"
	EventVerifyFailed    = "runtime.verify.failed"

	EventProvisionStarted     = "runtime.provision.started"
	EventProvisionUnsupported = "runtime.provision.unsupported"

	EventExecQueued    = "runtime.exec.queued"
	EventExecStarted   = "runtime.exec.started"
	EventExecStdout    = "runtime.exec.stdout"
	EventExecStderr    = "runtime.exec.stderr"
	EventExecCompleted = "runtime.exec.completed"
	EventExecFailed    = "runtime.exec.failed"
	EventExecTimeout   = "runtime.exec.timeout"
	EventExecCancelled = "runtime.exec.cancelled"

	EventCleanupStarted   = "runtime.cleanup.started"
	EventCleanupCompleted = "runtime.cleanup.completed"
	EventCleanupFailed    = "runtime.cleanup.failed"

	EventProbeStarted   = "runtime.probe.started"
	EventProbeCompleted = "runtime.probe.completed"

	EventInventoryCompleted = "runtime.inventory.completed"
)

// emit writes one lifecycle event. Fields are redacted here rather than at the
// call sites, so a new call site cannot forget to redact. logger applies its own
// secret patterns afterwards; this is the runtime's own first pass.
func emit(level logger.LogLevel, event string, fields map[string]any) {
	safe := make(map[string]any, len(fields)+2)
	safe["event"] = event
	safe["runtime_version"] = ManifestVersion
	for key, value := range fields {
		safe[key] = redactFieldValue(key, value)
	}

	switch level {
	case logger.ERROR:
		logger.ErrorCF(logComponent, event, safe)
	case logger.WARN:
		logger.WarnCF(logComponent, event, safe)
	case logger.DEBUG:
		logger.DebugCF(logComponent, event, safe)
	default:
		logger.InfoCF(logComponent, event, safe)
	}
}

func emitInfo(event string, fields map[string]any)  { emit(logger.INFO, event, fields) }
func emitDebug(event string, fields map[string]any) { emit(logger.DEBUG, event, fields) }
func emitWarn(event string, fields map[string]any)  { emit(logger.WARN, event, fields) }
func emitError(event string, fields map[string]any) { emit(logger.ERROR, event, fields) }
