package logger

import "regexp"

// Normalizing Android's app-private path prefixes out of log text.
//
// The physical DEBUG logs carried absolute paths like
// `/data/user/0/<package>/cache/tmp/...` and
// `/data/app/~~<hash>/<package>-<hash>/lib/arm64/...`. The useful half of such
// a path is the part *inside* the app -- which subtree, which file -- and that
// is what a reader needs to diagnose anything. The prefix is install-specific
// noise: the package name, the Android user id, and the installer's random
// per-install directory hashes.
//
// So this normalizes rather than redacts: the prefix becomes a stable marker
// and the remainder survives intact. `<app-private>/cache/tmp/pocketclaw_media/x.png`
// says everything the absolute path said that anyone could act on.
//
// Deliberately not touched: `/storage/emulated/0/...`. That is shared external
// storage, a well-known public location, and the workspace path under it is
// something the user chose and needs to recognise.

var (
	// /data/user/<id>/<package>, /data/user_de/<id>/<package> and the older
	// /data/data/<package>. All three are the same app-private root.
	appPrivatePathPattern = regexp.MustCompile(
		`/data/(?:user|user_de)/\d+/[A-Za-z0-9_][A-Za-z0-9_.]*|/data/data/[A-Za-z0-9_][A-Za-z0-9_.]*`)

	// /data/app/~~<hash>==/<package>-<hash>== is where the APK and its native
	// libraries are unpacked. Both hashes change on every install.
	appInstallPathPattern = regexp.MustCompile(
		`/data/app/(?:~~[A-Za-z0-9_=+-]+/)?[A-Za-z0-9_][A-Za-z0-9_.]*-[A-Za-z0-9_=+-]+`)
)

const (
	appPrivatePathMarker = "<app-private>"
	appInstallPathMarker = "<app-install>"
)

// normalizeAndroidPaths replaces app-private and install prefixes with stable
// markers, leaving the path inside them readable.
func normalizeAndroidPaths(s string) string {
	// Install paths first: /data/app never overlaps the private roots, but
	// ordering it first keeps the two rules independent of each other.
	s = appInstallPathPattern.ReplaceAllString(s, appInstallPathMarker)
	return appPrivatePathPattern.ReplaceAllString(s, appPrivatePathMarker)
}
