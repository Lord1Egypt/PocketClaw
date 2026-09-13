package openai_compat

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Structured DEBUG for one upstream request.
//
// The line a developer needs to reconstruct an inference failure: which provider
// and model, over which protocol, to which endpoint, with which headers present,
// how big the request was, and what came back. None of it is the request body,
// the user's message, or a credential.
//
// Presence booleans rather than values, deliberately: `authorization_present` and
// `session_header_present` answer the question a 401 or an OpenCode 400 actually
// raises, and neither can carry a secret. The logger's central redaction would
// catch a credential placed here by mistake; not placing one here is the first
// line.

// safeEndpoint reduces a URL to scheme, host and path.
//
// The query string and fragment are dropped rather than inspected: providers put
// keys in `?key=`, signed URLs carry their signature in the query, and a
// diagnostic does not need either. The path is kept because it is what
// distinguishes /chat/completions from /messages.
func safeEndpoint(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		// Not parseable, so not repeatable either. Report the shape, not the
		// string, which may be a misconfigured value containing anything.
		return "<unparseable>"
	}
	if parsed.Host == "" {
		return "<relative>"
	}
	return parsed.Scheme + "://" + parsed.Host + parsed.Path
}

// logProviderRequest records the request about to be sent.
func logProviderRequest(
	protocol string,
	req *http.Request,
	model string,
	messages, tools int,
	stream bool,
	requestBytes int,
) {
	if req == nil {
		return
	}
	logger.DebugCF("provider", "provider.request", map[string]any{
		"operation":              "chat",
		"protocol":               protocol,
		"model":                  model,
		"method":                 req.Method,
		"endpoint":               safeEndpoint(req.URL.String()),
		"stream":                 stream,
		"messages":               messages,
		"tools":                  tools,
		"request_bytes":          requestBytes,
		"authorization_present":  req.Header.Get("Authorization") != "",
		"session_header_present": hasSessionHeader(req),
		"custom_header_count":    len(req.Header),
	})
}

// logProviderResponse records what came back, and how long it took.
func logProviderResponse(
	protocol string,
	resp *http.Response,
	model string,
	started time.Time,
) {
	if resp == nil {
		return
	}
	logger.DebugCF("provider", "provider.response", map[string]any{
		"operation":      "chat",
		"protocol":       protocol,
		"model":          model,
		"status":         resp.StatusCode,
		"content_type":   resp.Header.Get("Content-Type"),
		"duration_ms":    time.Since(started).Milliseconds(),
		"content_length": resp.ContentLength,
	})
}

// logProviderTransportFailure records a request that never got a status.
//
// The error text goes through the logger's redaction like any other, and a
// transport error can quote the URL it failed to reach -- so the endpoint is
// reported separately, already reduced.
func logProviderTransportFailure(
	protocol string,
	endpoint string,
	model string,
	started time.Time,
	err error,
) {
	logger.DebugCF("provider", "provider.transport_failed", map[string]any{
		"operation":   "chat",
		"protocol":    protocol,
		"model":       model,
		"endpoint":    safeEndpoint(endpoint),
		"duration_ms": time.Since(started).Milliseconds(),
		"error":       err,
	})
}

// hasSessionHeader reports whether any session-style header was attached,
// without naming its value. OpenCode Go rejects a request that lacks one, so
// whether it was present is the first thing to check.
func hasSessionHeader(req *http.Request) bool {
	for name := range req.Header {
		if strings.Contains(strings.ToLower(name), "session") {
			return true
		}
	}
	return false
}
