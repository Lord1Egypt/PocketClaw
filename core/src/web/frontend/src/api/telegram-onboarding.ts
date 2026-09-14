import { launcherFetch } from "@/api/http"

/**
 * Managed Telegram onboarding driven through PocketClaw's own backend.
 *
 * PC-DEF-060. The native flow launches Telegram and writes the token itself, which a
 * desktop browser cannot do. Core runs the pairing instead, so every call here is
 * same-origin: the browser never reaches the hosted onboarding service, never learns its
 * URL, never holds the poll token that authorises token collection, and never sees the
 * bot token at all.
 */

/** Where a pairing has got to, as Core reports it. */
export type TelegramPairingState =
  | "pending"
  | "created"
  | "ready"
  | "expired"
  | "failed"

export interface TelegramOnboardingAvailability {
  available: boolean
}

export interface TelegramDesktopPairing {
  pairing_id: string
  suggested_username: string
  suggested_name: string
  /** A Telegram link. Core does not return anything else. */
  deep_link: string
  qr_payload: string
  expires_at: string
  poll_interval_seconds: number
}

export interface TelegramDesktopStatus {
  state: TelegramPairingState
  bot_username?: string
  reason?: string
}

export interface TelegramDesktopCompletion {
  ok: boolean
  bot_username?: string
  /** Whether the change reached the running gateway, or is parked for it. */
  applied?: boolean
  pending?: boolean
}

/**
 * A classified failure, matching the `error` kinds Core returns.
 *
 * The service's own detail stays in Core's log: it can quote a request.
 */
export class TelegramOnboardingRequestError extends Error {
  // Declared as fields rather than constructor parameter properties: the project
  // builds with erasableSyntaxOnly, which forbids the shorthand.
  readonly kind: string
  readonly status: number

  constructor(kind: string, status: number) {
    super(kind)
    this.name = "TelegramOnboardingRequestError"
    this.kind = kind
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await launcherFetch(path, init)
  if (!response.ok) {
    let kind = "service_error"
    try {
      const body = (await response.json()) as { error?: string }
      if (typeof body.error === "string" && body.error !== "") kind = body.error
    } catch {
      // A non-JSON failure leaves the generic kind, which is honest.
    }
    throw new TelegramOnboardingRequestError(kind, response.status)
  }
  return (await response.json()) as T
}

/** Whether this deployment can run managed onboarding at all. */
export async function getTelegramOnboardingAvailability(): Promise<boolean> {
  try {
    const body = await request<TelegramOnboardingAvailability>(
      "/api/telegram/onboarding",
    )
    return body.available === true
  } catch {
    // An older Core has no such route. Treat that as unavailable rather than
    // offering a button nothing can serve.
    return false
  }
}

export async function createTelegramPairing(): Promise<TelegramDesktopPairing> {
  return request<TelegramDesktopPairing>("/api/telegram/onboarding/pairings", {
    method: "POST",
  })
}

export async function fetchTelegramPairingStatus(
  pairingId: string,
): Promise<TelegramDesktopStatus> {
  return request<TelegramDesktopStatus>(
    `/api/telegram/onboarding/pairings/${encodeURIComponent(pairingId)}`,
  )
}

/**
 * Collects the token and configures Telegram, entirely inside Core.
 *
 * Nothing about the credential passes through the browser: the response says which bot
 * was paired and whether the change reached the running gateway.
 */
export async function completeTelegramPairing(
  pairingId: string,
): Promise<TelegramDesktopCompletion> {
  return request<TelegramDesktopCompletion>(
    `/api/telegram/onboarding/pairings/${encodeURIComponent(pairingId)}/complete`,
    { method: "POST" },
  )
}

/** Forgets a pairing, which drops the poll token Core was holding. */
export async function cancelTelegramPairing(pairingId: string): Promise<void> {
  try {
    await request<{ ok: boolean }>(
      `/api/telegram/onboarding/pairings/${encodeURIComponent(pairingId)}`,
      { method: "DELETE" },
    )
  } catch {
    // Cancelling is best-effort: the pairing expires on its own, and failing to
    // forget it must not block the user from starting another.
  }
}
