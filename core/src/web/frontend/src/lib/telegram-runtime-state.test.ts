import { describe, expect, it } from "vitest"

import {
  resolveTelegramRuntimeState,
  telegramMayReportConnected,
  type RuntimeChannel,
} from "./telegram-runtime-state"

const telegram = (over: Partial<RuntimeChannel> = {}): RuntimeChannel => ({
  name: "telegram",
  configured: true,
  started: true,
  running: true,
  ...over,
})

describe("telegram runtime state", () => {
  it("never reports Connected for a channel Core refused", () => {
    const state = resolveTelegramRuntimeState([
      telegram({ started: false, running: false }),
    ])
    expect(state).toBe("error")
    expect(telegramMayReportConnected(state)).toBe(false)
  })

  it("reports Connected only for a running channel", () => {
    expect(resolveTelegramRuntimeState([telegram()])).toBe("running")
    expect(telegramMayReportConnected("running")).toBe(true)
  })

  it("treats a stopped channel as configured but not running", () => {
    const state = resolveTelegramRuntimeState([telegram({ running: false })])
    expect(state).toBe("configured-not-running")
    expect(telegramMayReportConnected(state)).toBe(false)
  })

  it("treats an absent or unconfigured channel as not configured", () => {
    expect(resolveTelegramRuntimeState([])).toBe("not-configured")
    expect(
      resolveTelegramRuntimeState([
        telegram({ configured: false, started: false, running: false }),
      ]),
    ).toBe("not-configured")
  })

  it("never guesses when the runtime has not reported", () => {
    expect(resolveTelegramRuntimeState(null)).toBe("not-configured")
    expect(resolveTelegramRuntimeState(undefined)).toBe("not-configured")
  })

  it("does not mistake another running channel for Telegram", () => {
    expect(
      resolveTelegramRuntimeState([telegram({ name: "pocketclaw" })]),
    ).toBe("not-configured")
  })

  it("matches the channel name case-insensitively", () => {
    expect(resolveTelegramRuntimeState([telegram({ name: "Telegram" })])).toBe(
      "running",
    )
  })

  // The contract table both languages must agree on.
  it("permits Connected for exactly one state", () => {
    const states: TelegramRuntimeState[] = [
      "not-configured",
      "configured-not-running",
      "running",
      "error",
    ]
    expect(states.filter(telegramMayReportConnected)).toEqual(["running"])
  })
})

type TelegramRuntimeState = ReturnType<typeof resolveTelegramRuntimeState>
