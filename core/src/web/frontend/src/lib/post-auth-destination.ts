/**
 * Where the Dashboard may send a user after authentication.
 *
 * PC-DEF-059. Opening Manage Models or Manage Telegram from PocketClaw's native
 * Settings navigates the console to `/models` or `/channels/telegram`. If the
 * session had expired, the auth guard redirected to `/launcher-login` and the
 * login page then sent the user to `/` on success — so the destination they asked
 * for was lost and they had to navigate again, or go back to native Settings and
 * tap the same thing a second time.
 *
 * The destination now travels through the auth flow as `?next=`. That parameter is
 * **untrusted input**: it reaches this module from a URL anybody can craft, and it
 * ends up in a navigation. So it is matched against the route set rather than
 * sanitised, and anything unrecognised falls back to home. An allowlist cannot be
 * talked into `https://evil.example`, `//evil.example`, `javascript:` or a path
 * traversal, and does not depend on getting a denylist complete.
 */

/** The parameter name, spelled once. */
export const POST_AUTH_DESTINATION_PARAM = "next"

/** Where the Dashboard goes when there is nothing valid to return to. */
export const DEFAULT_POST_AUTH_DESTINATION = "/"

/**
 * Every route the Dashboard will navigate to after login.
 *
 * Taken from the generated route tree. The auth pages are deliberately absent:
 * returning to a login page after logging in is a loop.
 */
const STATIC_DESTINATIONS: ReadonlySet<string> = new Set([
  "/",
  "/models",
  "/credentials",
  "/logs",
  "/config",
  "/config/raw",
  "/channels",
  "/agent",
  "/agent/hub",
  "/agent/skills",
  "/agent/tools",
])

/**
 * `/channels/<name>` is the one parameterised destination, and the native
 * Telegram card uses it. The segment is constrained to the shape a channel name
 * actually has, so it cannot carry a second path segment, a scheme, a host, or
 * traversal.
 */
const CHANNEL_DESTINATION = /^\/channels\/[a-z0-9][a-z0-9_-]{0,63}$/

/**
 * Whether a candidate contains a character that must never appear in a returned
 * path: any control character, DEL, or a backslash.
 *
 * Checked by code point rather than a regular expression, because a control-class
 * regex is exactly what the lint rule forbids and the intent reads more clearly
 * this way.
 */
function hasForbiddenCharacter(value: string): boolean {
  for (const character of value) {
    const code = character.codePointAt(0) ?? 0
    if (code <= 0x1f || code === 0x7f || character === "\\") return true
  }
  return false
}

/** Any scheme, with or without slashes: https:, javascript:, data:, mailto:. */
const HAS_SCHEME = /^[a-zA-Z][a-zA-Z0-9+.-]*:/

/**
 * Normalizes a candidate to a comparable internal path, or null if it is not one.
 *
 * Rejected before any matching: anything with a scheme, anything
 * protocol-relative, anything that is not rooted at a single `/`, and anything
 * containing a traversal segment, a backslash, or a control character. A browser
 * can read `\` as `/` and `//host` as an absolute URL, which is how an allowlist
 * gets bypassed when it only inspects the start of the string.
 */
function normalizeCandidate(raw: string): string | null {
  const value = raw.trim()
  if (value === "") return null

  if (hasForbiddenCharacter(value)) return null
  if (HAS_SCHEME.test(value)) return null
  // Protocol-relative.
  if (value.startsWith("//")) return null
  if (!value.startsWith("/")) return null
  if (value.includes("..")) return null

  // Compare the path only. A query or hash on a returned destination is not
  // something the native entry points produce, and carrying one would widen what
  // this has to reason about for no benefit.
  const path = value.split(/[?#]/)[0] ?? ""
  if (path === "") return null

  // Trailing slashes are cosmetic; "/models/" is "/models".
  const trimmed = path.replace(/\/+$/, "")
  return trimmed === "" ? "/" : trimmed
}

/** Whether a candidate names a Dashboard route this may navigate to. */
export function isSafePostAuthDestination(
  raw: string | null | undefined,
): boolean {
  if (raw == null) return false
  const path = normalizeCandidate(raw)
  if (path === null) return false
  return STATIC_DESTINATIONS.has(path) || CHANNEL_DESTINATION.test(path)
}

/**
 * The path to navigate to after a successful login.
 *
 * Always returns something safe, so a caller cannot forget the fallback.
 */
export function resolvePostAuthDestination(
  raw: string | null | undefined,
): string {
  if (!isSafePostAuthDestination(raw)) return DEFAULT_POST_AUTH_DESTINATION
  return normalizeCandidate(raw as string) as string
}

/** Reads the requested destination out of a search string. */
export function readPostAuthDestination(search: string): string {
  try {
    const params = new URLSearchParams(search)
    return resolvePostAuthDestination(params.get(POST_AUTH_DESTINATION_PARAM))
  } catch {
    return DEFAULT_POST_AUTH_DESTINATION
  }
}

/**
 * Builds the login URL that remembers where the user was going.
 *
 * A destination that is not worth returning to — home, or an auth page — produces
 * a bare login URL, so the common case stays unchanged.
 */
export function launcherLoginUrlFor(pathname: string, search = ""): string {
  const candidate = `${pathname}${search}`
  if (!isSafePostAuthDestination(candidate)) return "/launcher-login"
  const path = normalizeCandidate(candidate) as string
  if (path === DEFAULT_POST_AUTH_DESTINATION) return "/launcher-login"
  return `/launcher-login?${POST_AUTH_DESTINATION_PARAM}=${encodeURIComponent(path)}`
}
