import { launcherFetch } from "@/api/http"

/**
 * The configured bot's own identity, from Core.
 *
 * PC-DEF-075. The Android host caches a username written only by the native
 * pairing launcher, so a bot paired any other way leaves that cache pointing at
 * a bot that may no longer exist. Core reads the identity from the committed
 * credential with getMe and canonicalises it, so this is the authority and a
 * cached username that disagrees with it is stale by definition.
 */
export interface TelegramIdentity {
  configured: boolean
  /** Canonical, never carrying a leading "@". Absent when it cannot be read. */
  username?: string
  /** The canonical https://t.me/<username> link, built by Core. */
  chat_url?: string
}

export async function fetchTelegramIdentity(): Promise<TelegramIdentity> {
  const response = await launcherFetch("/api/telegram/identity")
  if (!response.ok) return { configured: false }
  return (await response.json()) as TelegramIdentity
}
