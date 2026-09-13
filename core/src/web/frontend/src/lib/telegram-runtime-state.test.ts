import { describe, expect, it } from "vitest"

import {
  resolveTelegramRuntimeState,
  telegramMayReportConnected,
  type RuntimeChannel,
  type TelegramRuntimeState,
} from "./telegram-runtime-state"

const telegram = (over: Partial<RuntimeChannel> = {}): RuntimeChannel => ({
  name: "telegram",
  configured: true,
  started: true,
  running: true,
  ...over,
})

describe("telegram runtime state", () => {
  it("reports Connected only for a running channel", () => {
    const state = resolveTelegramRuntimeState({
      configuredAndValid: true,
      channels: [telegram()],
    })
    expect(state).toBe("running")
    expect(telegramMayReportConnected(state)).toBe(true)
  })

  // The correction: runtime silence must never read as "not configured".
  it("keeps a configured channel configured while the Gateway is stopped", () => {
    const state = resolveTelegramRuntimeState({
      configuredAndValid: true,
      channels: null,
    })
    expect(state).toBe("configured-runtime-not-active")
    expect(state).not.toBe("not-configured")
    expect(state).not.toBe("error")
    expect(telegramMayReportConnected(state)).toBe(false)
  })

  // The second correction: not started is not failed.
  it("does not treat an unstarted channel as an error", () => {
    expect(
      resolveTelegramRuntimeState({
        configuredAndValid: true,
        channels: [telegram({ started: false, running: false })],
      }),
    ).toBe("configured-runtime-not-active")
  })

  it("produces an error only from affirmative failure evidence", () => {
    expect(
      resolveTelegramRuntimeState({
        configuredAndValid: true,
        startupError: "channel failed to start",
      }),
    ).toBe("error")
    expect(
      resolveTelegramRuntimeState({ configuredAndValid: true, startupError: "  " }),
    ).toBe("configured-runtime-not-active")
  })

  it("reports not configured only when configuration says so", () => {
    expect(
      resolveTelegramRuntimeState({ configuredAndValid: false, channels: null }),
    ).toBe("not-configured")
    expect(
      resolveTelegramRuntimeState({ configuredAndValid: false, channels: [] }),
    ).toBe("not-configured")
  })

  it("lets a live channel outrank stale configuration", () => {
    expect(
      resolveTelegramRuntimeState({
        configuredAndValid: false,
        channels: [telegram()],
      }),
    ).toBe("running")
  })

  it("does not mistake another channel for Telegram", () => {
    expect(
      resolveTelegramRuntimeState({
        configuredAndValid: false,
        channels: [telegram({ name: "pocketclaw" })],
      }),
    ).toBe("not-configured")
  })

  it("matches the channel name case-insensitively", () => {
    expect(
      resolveTelegramRuntimeState({
        configuredAndValid: true,
        channels: [telegram({ name: "Telegram" })],
      }),
    ).toBe("running")
  })

  // The contract table both languages must agree on.
  it("permits Connected for exactly one state", () => {
    const states: TelegramRuntimeState[] = [
      "not-configured",
      "configured-runtime-not-active",
      "running",
      "error",
    ]
    expect(states.filter(telegramMayReportConnected)).toEqual(["running"])
  })
})
