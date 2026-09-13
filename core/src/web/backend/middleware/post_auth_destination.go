package middleware

import (
	"net/url"
	"strings"
)

// Where the launcher login page may be told to return to.
//
// PC-DEF-059, second attempt. The first fix was entirely in the frontend: the
// router guard appended `?next=` and the login page honoured it. It did nothing on
// the device, because the guard never ran.
//
// The real path, traced: PocketClaw's native Settings opens the WebView at
// `http://127.0.0.1:18800/models?lng=en` (canonicalModelsConsolePath), and for an
// unauthenticated request rejectLauncherDashboardAuth answers **302 to
// /launcher-login** — server-side, before a single line of JavaScript loads. So the
// destination was already gone by the time the SPA existed to remember it, and the
// login page read an empty `next` and fell back to `/`.
//
// The destination therefore has to survive the server redirect, which is what this
// file is for.
//
// The value is put into a URL that a browser follows, so it is matched against the
// route set rather than sanitised. The path arrives from `r.URL.Path` — attacker
// influenced, since anyone may request any path — so the same allowlist discipline
// applies here as in the frontend: an allowlist cannot be talked into an external
// host, and does not depend on completing a denylist.

// PostAuthDestinationParam is the query parameter the login page reads.
// Spelled identically in web/frontend/src/lib/post-auth-destination.ts.
const PostAuthDestinationParam = "next"

// launcherLoginPath is where an unauthenticated page request is sent.
const launcherLoginPath = "/launcher-login"

// postAuthDestinations are the Dashboard routes worth returning to.
//
// Deliberately duplicated from STATIC_DESTINATIONS in
// web/frontend/src/lib/post-auth-destination.ts rather than shared: the backend
// cannot import TypeScript, and a redirect decision must not depend on a file the
// client owns. TestPostAuthDestinationsMatchTheFrontendAllowlist reads that file
// and fails if the two drift, so the duplication cannot rot silently.
//
// "/" is absent on purpose: it is where login already goes, so remembering it adds
// a parameter that changes nothing. The auth pages are absent because returning to
// a login page after logging in is a loop.
var postAuthDestinations = map[string]struct{}{
	"/models":       {},
	"/credentials":  {},
	"/logs":         {},
	"/config":       {},
	"/config/raw":   {},
	"/channels":     {},
	"/agent":        {},
	"/agent/hub":    {},
	"/agent/skills": {},
	"/agent/tools":  {},
}

// channelDestinationPrefix covers the one parameterised route, which is what the
// native Telegram card opens (`/channels/telegram`).
const channelDestinationPrefix = "/channels/"

// isSafePostAuthDestination reports whether canonicalPath names a Dashboard route
// the login page may be sent back to.
//
// canonicalPath is expected to have been through canonicalAuthPath, which applies
// path.Clean — so traversal is already resolved rather than merely rejected. The
// checks here do not rely on that: a value that still contains "..", a scheme, a
// protocol-relative prefix, a backslash or a control character is refused outright.
func isSafePostAuthDestination(canonicalPath string) bool {
	if canonicalPath == "" || canonicalPath[0] != '/' {
		return false
	}
	// Protocol-relative. A browser reads "//host/x" as an absolute URL.
	if strings.HasPrefix(canonicalPath, "//") {
		return false
	}
	if strings.Contains(canonicalPath, "..") {
		return false
	}
	// A backslash can be read as a separator, and a control character can split a
	// header. Neither belongs in a path this hands to a Location.
	for _, r := range canonicalPath {
		if r == '\\' || r < 0x20 || r == 0x7f {
			return false
		}
	}
	// A scheme cannot appear in a rooted path, but a colon in the first segment is
	// enough for some parsers to try, so it is refused.
	if first := strings.SplitN(strings.TrimPrefix(canonicalPath, "/"), "/", 2)[0]; strings.Contains(first, ":") {
		return false
	}

	if _, ok := postAuthDestinations[canonicalPath]; ok {
		return true
	}
	return isChannelDestination(canonicalPath)
}

// isChannelDestination matches /channels/<name> with one plain segment, so the
// name cannot carry a second path segment, a host or traversal.
func isChannelDestination(canonicalPath string) bool {
	if !strings.HasPrefix(canonicalPath, channelDestinationPrefix) {
		return false
	}
	name := strings.TrimPrefix(canonicalPath, channelDestinationPrefix)
	if name == "" || len(name) > 64 || strings.Contains(name, "/") {
		return false
	}
	for i, r := range name {
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if i == 0 && !(isLower || isDigit) {
			return false
		}
		if !(isLower || isDigit || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// launcherLoginRedirectTarget is the Location for an unauthenticated page request.
//
// It remembers canonicalPath when that is a Dashboard route, and is otherwise the
// bare login path — so an ordinary unauthenticated visit behaves exactly as before.
func launcherLoginRedirectTarget(canonicalPath string) string {
	if !isSafePostAuthDestination(canonicalPath) {
		return launcherLoginPath
	}
	return launcherLoginPath + "?" + PostAuthDestinationParam + "=" +
		url.QueryEscape(canonicalPath)
}
