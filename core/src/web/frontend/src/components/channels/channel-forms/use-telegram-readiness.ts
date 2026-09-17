import { useCallback, useEffect, useState } from "react"

import {
  type TelegramReadiness,
  type TelegramReadinessState,
  fetchTelegramReadiness,
} from "@/api/telegram-lifecycle"

/**
 * Polls Telegram readiness while it matters, and stops when it does not.
 *
 * PC-DEF-061. Shared by the managed-connect flow and the connected card so both
 * read the same authoritative state — the gateway's own status snapshot —
 * rather than one of them inferring readiness from what a request just did.
 */

/** How often to ask. Frequent enough to feel live, not a busy loop. */
const POLL_INTERVAL_MS = 1500

/**
 * How long to keep waiting before calling it a failure.
 *
 * A bound, not a delay: nothing is claimed when it expires, and the user is
 * given something to do. Sized for the whole chain — gateway restart, channel
 * construction, the getMe round trip that measured four seconds on a phone, and
 * asynchronous command registration with its own retries.
 */
export const READINESS_TIMEOUT_MS = 90_000

export interface TelegramReadinessPoll {
  readiness: TelegramReadiness | null
  /** True once the bound expired without reaching ready. */
  timedOut: boolean
  /** Restarts the wait, for the "check again" action. */
  recheck: () => void
}

/**
 * @param active whether readiness is currently worth watching. Polling stops on
 * false, and on reaching `ready`, so a settled page makes no requests.
 */
export function useTelegramReadiness(active: boolean): TelegramReadinessPoll {
  const [readiness, setReadiness] = useState<TelegramReadiness | null>(null)
  const [timedOut, setTimedOut] = useState(false)
  const [attempt, setAttempt] = useState(0)

  const recheck = useCallback(() => {
    setTimedOut(false)
    setReadiness(null)
    setAttempt((n) => n + 1)
  }, [])

  useEffect(() => {
    if (!active) return
    let cancelled = false
    let timer: ReturnType<typeof setInterval> | null = null
    const startedAt = Date.now()

    const stop = () => {
      if (timer !== null) {
        clearInterval(timer)
        timer = null
      }
    }

    const tick = async () => {
      const next = await fetchTelegramReadiness()
      if (cancelled) return
      setReadiness(next)
      // Settled either way: a ready channel has nothing left to report, and an
      // expired bound has handed the decision to the user. Neither should keep
      // making requests.
      if (next.ready || next.state === "authentication_failed") {
        stop()
        return
      }
      if (Date.now() - startedAt >= READINESS_TIMEOUT_MS) {
        setTimedOut(true)
        stop()
      }
    }

    void tick()
    timer = setInterval(() => void tick(), POLL_INTERVAL_MS)
    return () => {
      cancelled = true
      stop()
    }
  }, [active, attempt])

  return { readiness, timedOut, recheck }
}

/**
 * The i18n key for a readiness state, so every surface words it identically.
 */
export function readinessLabelKey(state: TelegramReadinessState): string {
  return `channels.telegram.readiness.${state}`
}
