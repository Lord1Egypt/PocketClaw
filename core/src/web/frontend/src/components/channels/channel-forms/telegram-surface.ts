/**
 * Which Telegram surface the Channels → Telegram page shows.
 *
 * Kept as a pure function so the decision that regressed on a physical device
 * — managed onboarding existing but never being reached — is asserted directly
 * rather than inferred from a rendered tree.
 */
export type TelegramSurface =
  /** Telegram already has a bot token: connected summary first. */
  | "connected"
  /** No token, and a host that can pair one: managed onboarding first. */
  | "managed-onboarding"
  /** No token and no way to pair: the manual form is the whole page. */
  | "manual-only"

export interface TelegramSurfaceInput {
  configured: boolean
  onboardingAvailable: boolean
}

export function resolveTelegramSurface({
  configured,
  onboardingAvailable,
}: TelegramSurfaceInput): TelegramSurface {
  if (configured) return "connected"
  if (onboardingAvailable) return "managed-onboarding"
  return "manual-only"
}

/**
 * Why managed onboarding is not on offer.
 *
 * PC-DEF-060. Both reasons produced the same sentence — "One-tap bot creation is
 * not available in this build" — and on a desktop browser that is simply untrue:
 * the build has the feature, this client cannot run it. Managed pairing needs the
 * native flow to launch Telegram and write the token, which only the Android host
 * provides. Telling a desktop user their build lacks the feature sends them looking
 * for a different build.
 */
export type TelegramManualReason =
  /** No PocketClaw host: an ordinary browser. Managed setup lives in the app. */
  | "no-host"
  /** A host that was compiled without an onboarding endpoint. */
  | "host-without-endpoint"

export interface TelegramManualReasonInput {
  hostPresent: boolean
  onboardingConfigured: boolean
}

export function resolveTelegramManualReason({
  hostPresent,
  onboardingConfigured,
}: TelegramManualReasonInput): TelegramManualReason {
  if (!hostPresent) return "no-host"
  // A host is present and still cannot pair, so the endpoint is what is missing.
  return onboardingConfigured ? "no-host" : "host-without-endpoint"
}

/**
 * Whether an already-connected Telegram channel stays manageable here.
 *
 * Always true, and asserted as its own rule: a connected configuration is Core's,
 * not the Android host's, so viewing and editing it must never depend on which
 * client opened the console. Only *pairing a new bot* needs the host.
 */
export function isTelegramManageableWithoutHost(configured: boolean): boolean {
  return configured
}

/**
 * Whether the legacy token/base-URL/proxy form is visible without a further
 * click. It is never removed — only demoted behind Advanced when there is a
 * friendlier primary action to show first.
 */
export function isAdvancedFormAlwaysVisible(surface: TelegramSurface): boolean {
  return surface === "manual-only"
}

/**
 * Whether a readiness state is a stage of Telegram starting up.
 *
 * PC-DEF-071. The connected card built its heading as an else-chain whose last
 * branch was "Starting Telegram…" with a spinner, so every state that had no
 * branch of its own was presented as a stage of starting. Three of them are not:
 * `not_configured` and `gateway_stopped` describe something that is not
 * happening at all, and `authentication_failed` is terminal — its poll stops, so
 * the spinner never stopped spinning and the heading sat directly above a body
 * sentence saying the opposite ("No bot is configured.", "Telegram rejected the
 * bot credentials."). Naming the starting stages positively is what stops a new
 * state inheriting that claim by default.
 */
export function isTelegramStartingState(state: string): boolean {
  return (
    state === "gateway_starting" ||
    state === "channel_starting" ||
    state === "registering_commands"
  )
}
