/**
 * What Telegram is actually doing, as opposed to what was configured.
 *
 * PC-DEF-027. This mirrors lib/src/core/telegram_runtime_state.dart exactly.
 * Two surfaces showing the same channel must not be able to disagree about
 * whether it is running, so the derivation is written once per language and
 * the contract tests on both sides assert the same table.
 *
 * Core publishes three separate booleans per channel and deliberately
 * publishes no reachability field, because nothing probes the network.
 */
export type TelegramRuntimeState =
  | "not-configured"
  | "configured-not-running"
  | "running"
  | "error"

export type RuntimeChannel = {
  name: string
  configured: boolean
  started: boolean
  running: boolean
}

/**
 * `null`/`undefined` channels mean the runtime has not reported yet -- the
 * gateway may be stopped or still starting. Not an error, and not running.
 */
export function resolveTelegramRuntimeState(
  channels: RuntimeChannel[] | null | undefined,
): TelegramRuntimeState {
  if (!channels) return "not-configured"

  const telegram = channels.find((c) => c.name?.toLowerCase() === "telegram")
  if (!telegram || !telegram.configured) return "not-configured"
  if (telegram.running) return "running"

  // Configured and not running splits on whether a start ever succeeded: a
  // channel that started and stopped has stopped, one that never started
  // failed to.
  return telegram.started ? "configured-not-running" : "error"
}

/**
 * Whether the product may show its "Connected" wording.
 *
 * It means exactly that the Telegram runtime channel is running. It does NOT
 * mean Telegram's servers were reached or the token independently verified:
 * Core performs no such probe, so the UI must not imply one.
 */
export function telegramMayReportConnected(
  state: TelegramRuntimeState,
): boolean {
  return state === "running"
}
