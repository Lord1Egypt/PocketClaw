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
 * Whether the legacy token/base-URL/proxy form is visible without a further
 * click. It is never removed — only demoted behind Advanced when there is a
 * friendlier primary action to show first.
 */
export function isAdvancedFormAlwaysVisible(surface: TelegramSurface): boolean {
  return surface === "manual-only"
}
