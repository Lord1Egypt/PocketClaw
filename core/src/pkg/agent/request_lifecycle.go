package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// traceRequestLifecycle records only process-local correlation metadata. Never
// add message content, platform IDs, tokens, session keys, or tool arguments.
func traceRequestLifecycle(event string, inbound *bus.InboundContext, fields map[string]any) {
	lifecycleID := bus.InboundLifecycleID(inbound)
	if lifecycleID == "" {
		return
	}
	safe := map[string]any{
		"event":        strings.TrimSpace(event),
		"lifecycle_id": lifecycleID,
	}
	if inbound != nil && strings.TrimSpace(inbound.Channel) != "" {
		safe["channel"] = strings.TrimSpace(inbound.Channel)
	}
	for key, value := range fields {
		safe[key] = value
	}
	logger.InfoCF("request_lifecycle", "Request lifecycle", safe)
}

func traceTurnLifecycle(event string, ts *turnState, fields map[string]any) {
	if ts == nil {
		return
	}
	traceRequestLifecycle(event, ts.opts.Dispatch.InboundContext, fields)
}
