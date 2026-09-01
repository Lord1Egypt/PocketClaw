/**
 * The PocketClaw Android host, when the console is running inside the app.
 *
 * The console is also served to an ordinary browser, where no host exists.
 * Every caller must handle that: `getPocketClawHost()` returns null and the UI
 * falls back to the manual path rather than offering an action nothing can
 * perform.
 */
export interface PocketClawHost {
  platform: string
  /** Whether this build was compiled with an onboarding service endpoint. */
  onboardingConfigured: boolean
  /** The paired bot's public @username, if this device paired one. */
  telegramBotUsername: string | null
  /** Runs the native managed-bot pairing flow. */
  openTelegramOnboarding: () => void
  /** Opens a link outside the WebView. */
  openExternal: (url: string) => void
  /**
   * Opens WhatsApp on the given number with `message` prepared in the compose
   * box. Never sends: the Send button stays in WhatsApp, under the user.
   *
   * Optional because a host built before WhatsApp Self-Chat does not have it,
   * and the console must not offer a button that silently does nothing.
   */
  openWhatsAppSelfChat?: (selfNumber: string, message: string) => void
}

/** Fired by the host once `window.__pocketclawHost` is available. */
export const HOST_READY_EVENT = "pocketclaw:host-ready"

/** Fired by the host after the native flow changed the Telegram config. */
export const TELEGRAM_UPDATED_EVENT = "pocketclaw:telegram-updated"

declare global {
  interface Window {
    __pocketclawHost?: PocketClawHost
  }
}

export function getPocketClawHost(): PocketClawHost | null {
  if (typeof window === "undefined") return null
  const host = window.__pocketclawHost
  if (!host || typeof host.openTelegramOnboarding !== "function") return null
  return host
}

/**
 * Whether managed-bot onboarding can actually be offered right now.
 *
 * Both halves must hold: a host to run the flow, and an endpoint compiled into
 * that host. A build without the endpoint deliberately offers manual setup
 * instead of a button that cannot work.
 */
export function isTelegramOnboardingAvailable(
  host: PocketClawHost | null,
): boolean {
  return host !== null && host.onboardingConfigured === true
}

/**
 * Whether this host can open WhatsApp.
 *
 * Self-Chat is an Android Intent, so it exists only inside the app. In an
 * ordinary browser the console shows the number it has stored and hides Test,
 * rather than offering an action nothing can perform.
 */
export function isWhatsAppSelfChatAvailable(
  host: PocketClawHost | null,
): boolean {
  return host !== null && typeof host.openWhatsAppSelfChat === "function"
}
