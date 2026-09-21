/**
 * The one place a Telegram bot chat link is built.
 *
 * PC-DEF-075. "Open chat" interpolated a username straight into
 * `https://t.me/<value>` from a value the onboarding service supplied and the
 * Android host cached. Telegram's canonical bot link is `https://t.me/name`;
 * `https://t.me/@name` is a different, non-existent username and Telegram
 * answers "Username not found".
 *
 * Display formatting must never become URL identity. The card renders the
 * username as `@{name}`, and in an RTL locale the neutral `@` is reordered to
 * the visual right — so what is on screen and what belongs in the path are not
 * the same string, and only one of them is the destination.
 */

/**
 * Telegram's username shape, applied strictly.
 *
 * The job is not to be generous about what a username may look like; it is to
 * make sure a display string can never become a destination. This rejects a
 * second `@`, a path separator, a scheme, a dot, a space, and every
 * bidirectional control character.
 */
const TELEGRAM_USERNAME = /^[A-Za-z][A-Za-z0-9_]{3,30}[A-Za-z0-9]$/

/**
 * Trims, removes **exactly one** optional leading `@`, and validates.
 *
 * Returns null when the value is not a usable Telegram username, so a caller
 * cannot build a link from junk. Exactly one `@` is removed: stripping
 * repeatedly would quietly turn `@@name` — already evidence that something
 * upstream is wrong — into a working link, hiding the bug instead of refusing
 * it.
 */
export function canonicalTelegramUsername(raw: unknown): string | null {
  if (typeof raw !== "string") return null
  let value = raw.trim()
  if (value.startsWith("@")) value = value.slice(1)
  value = value.trim()
  return TELEGRAM_USERNAME.test(value) ? value : null
}

/**
 * The canonical chat link, or null when there is no usable username.
 *
 * Null means "offer no button", never "offer a broken one".
 */
export function telegramBotChatUrl(raw: unknown): string | null {
  const username = canonicalTelegramUsername(raw)
  return username === null ? null : `https://t.me/${username}`
}
