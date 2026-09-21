import { launcherFetch } from "@/api/http"

/**
 * Telegram readiness and removal, both same-origin and both answered by Core.
 *
 * PC-DEF-061 and PC-DEF-062. Two facts the Dashboard could not previously get:
 * whether Telegram is actually receiving, and how to undo a pairing without the
 * Android host.
 */

/**
 * How far Telegram has got, in lifecycle order.
 *
 * `ready` is the only state that means the bot will receive the owner's next
 * message. `unknown` is reported when the gateway would not say, and is never
 * treated as ready.
 */
export type TelegramReadinessState =
  | "not_configured"
  | "gateway_stopped"
  | "gateway_starting"
  | "channel_starting"
  | "registering_commands"
  | "authentication_failed"
  | "setup_required"
  | "telegram_conflict"
  | "ready"
  | "unknown"

export interface TelegramReadiness {
  state: TelegramReadinessState
  ready: boolean
  detail?: string
}

/**
 * Reads readiness once. The caller polls; this never waits.
 *
 * A failed request is reported as `unknown` rather than thrown: readiness is
 * polled, and a single missed poll during a gateway restart is expected rather
 * than an error to show the user.
 */
export async function fetchTelegramReadiness(): Promise<TelegramReadiness> {
  try {
    const response = await launcherFetch("/api/telegram/readiness")
    if (!response.ok) return { state: "unknown", ready: false }
    const body = (await response.json()) as Partial<TelegramReadiness>
    if (typeof body.state !== "string")
      return { state: "unknown", ready: false }
    return {
      state: body.state as TelegramReadinessState,
      ready: body.ready === true,
      detail: typeof body.detail === "string" ? body.detail : undefined,
    }
  } catch {
    return { state: "unknown", ready: false }
  }
}

export interface TelegramDisconnectResult {
  ok: boolean
  applied?: boolean
  pending?: boolean
}

/**
 * Removes the configured bot through Core.
 *
 * Deliberately a single call to the one authoritative endpoint: the token, the
 * owner allowlist and the enabled flag are cleared together there, and the
 * runtime apply that stops the old bot happens there too. Clearing fields from
 * here instead would leave the previous bot polling.
 */
export async function disconnectTelegram(): Promise<TelegramDisconnectResult> {
  const response = await launcherFetch("/api/telegram/configuration", {
    method: "DELETE",
  })
  if (!response.ok) {
    let kind = "configuration_failed"
    try {
      const body = (await response.json()) as { error?: string }
      if (typeof body.error === "string" && body.error !== "") kind = body.error
    } catch {
      // A non-JSON failure keeps the generic kind, which is honest.
    }
    throw new Error(kind)
  }
  return (await response.json()) as TelegramDisconnectResult
}
