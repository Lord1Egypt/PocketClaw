/**
 * What Telegram is, as opposed to what was configured — with configuration
 * truth and runtime truth kept apart.
 *
 * PC-DEF-027. This mirrors lib/src/core/telegram_runtime_state.dart exactly.
 * Two surfaces showing the same channel must not be able to disagree, so the
 * derivation is written once per language and the same contract table is
 * asserted on both sides.
 *
 * Runtime silence is not a verdict: a validly configured channel is still
 * configured while the Gateway is stopped, and "has not started" is not the
 * same fact as "failed to start".
 */
export type TelegramRuntimeState =
  /** No enabled, valid Telegram configuration exists. */
  | "not-configured"
  /**
   * Configured, but the runtime is not active or not observable: Gateway
   * stopped, still starting, not yet reporting, or the channel disabled. None
   * of those is a failure and none is Running.
   */
  | "configured-runtime-not-active"
  /**
   * Core reports the channel running. This is the product's "Connected", and
   * it means exactly that — not that Telegram was reached or the token
   * verified, because Core performs no such probe.
   */
  | "running"
  /**
   * Affirmative evidence that start or apply failed. Only ever produced from
   * an explicit failure signal; `started === false` does not imply it.
   */
  | "error"

export type RuntimeChannel = {
  name: string
  configured: boolean
  started: boolean
  running: boolean
}

export type TelegramStateInput = {
  /** From persisted configuration — the only thing that can answer this. */
  configuredAndValid: boolean
  /** Core's runtime report. `null`/`undefined` means not observed. */
  channels?: RuntimeChannel[] | null
  /** Affirmative sanitized failure evidence, never inferred. */
  startupError?: string | null
}

export function resolveTelegramRuntimeState({
  configuredAndValid,
  channels,
  startupError,
}: TelegramStateInput): TelegramRuntimeState {
  const telegram = channels?.find((c) => c.name?.toLowerCase() === "telegram")

  // Running is affirmative evidence and outranks everything: a channel that
  // reports running is running, whatever configuration was edited since.
  if (telegram?.running) return "running"

  if (startupError && startupError.trim() !== "") return "error"

  // Configuration decides configured-ness. Runtime silence never does.
  if (!configuredAndValid) return "not-configured"

  return "configured-runtime-not-active"
}

/** Whether the product may show its "Connected" wording. */
export function telegramMayReportConnected(
  state: TelegramRuntimeState,
): boolean {
  return state === "running"
}
