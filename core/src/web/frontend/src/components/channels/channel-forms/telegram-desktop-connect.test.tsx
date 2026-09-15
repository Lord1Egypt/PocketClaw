import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { TelegramDesktopPairing } from "@/api/telegram-onboarding"

const createTelegramPairing = vi.fn()
const fetchTelegramPairingStatus = vi.fn()
const completeTelegramPairing = vi.fn()
const cancelTelegramPairing = vi.fn()
const fetchTelegramReadiness = vi.fn()
const toastSuccess = vi.fn()
const toastInfo = vi.fn()

vi.mock("@/api/telegram-onboarding", () => ({
  createTelegramPairing: (...a: unknown[]) => createTelegramPairing(...a),
  fetchTelegramPairingStatus: (...a: unknown[]) => fetchTelegramPairingStatus(...a),
  completeTelegramPairing: (...a: unknown[]) => completeTelegramPairing(...a),
  cancelTelegramPairing: (...a: unknown[]) => cancelTelegramPairing(...a),
}))

vi.mock("@/api/telegram-lifecycle", () => ({
  fetchTelegramReadiness: (...a: unknown[]) => fetchTelegramReadiness(...a),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock("sonner", () => ({
  toast: {
    success: (...a: unknown[]) => toastSuccess(...a),
    info: (...a: unknown[]) => toastInfo(...a),
    error: vi.fn(),
  },
}))

import { TelegramDesktopConnect } from "./telegram-desktop-connect"

/**
 * PC-DEF-060. Managed pairing driven from a browser with no Android host.
 *
 * Core does the pairing, so what matters here is that the UI never handles a credential,
 * never opens anything but a Telegram link, and reaches a finished state for each of the
 * outcomes the service can report.
 */
function pairing(overrides: Partial<TelegramDesktopPairing> = {}): TelegramDesktopPairing {
  return {
    pairing_id: "pair-1",
    suggested_username: "pocketclaw_abc_bot",
    suggested_name: "PocketClaw Agent",
    deep_link: "https://t.me/newbot/Mgr/pocketclaw_abc_bot",
    qr_payload: "https://t.me/newbot/Mgr/pocketclaw_abc_bot",
    expires_at: new Date(Date.now() + 600_000).toISOString(),
    poll_interval_seconds: 1,
    ...overrides,
  }
}

describe("TelegramDesktopConnect", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    createTelegramPairing.mockResolvedValue(pairing())
    fetchTelegramPairingStatus.mockResolvedValue({ state: "pending" })
    completeTelegramPairing.mockResolvedValue({ ok: true, bot_username: "pocketclaw_abc_bot" })
    cancelTelegramPairing.mockResolvedValue(undefined)
    fetchTelegramReadiness.mockResolvedValue({ state: "ready", ready: true })
  })

  it("offers Connect before anything has started", () => {
    render(<TelegramDesktopConnect onConnected={vi.fn()} />)

    expect(
      screen.getByText("channels.telegram.desktop.connect").closest("button"),
    ).toBeTruthy()
    expect(createTelegramPairing).not.toHaveBeenCalled()
  })

  it("starts a pairing and shows the suggested bot", async () => {
    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    await waitFor(() => expect(createTelegramPairing).toHaveBeenCalled())
    expect(await screen.findByText("@pocketclaw_abc_bot")).toBeTruthy()
  })

  // The only destination is Telegram. Core validates that before returning it, and this
  // asserts the UI does not send the user anywhere else.
  it("opens only the Telegram link Core returned", async () => {
    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    const open = await screen.findByText("channels.telegram.desktop.openTelegram")
    const anchor = open.closest("a") as HTMLAnchorElement
    expect(anchor.getAttribute("href")).toBe(
      "https://t.me/newbot/Mgr/pocketclaw_abc_bot",
    )
    // A new tab from a page that can be navigated back to needs both.
    expect(anchor.getAttribute("rel")).toContain("noopener")
  })

  // PC-DEF-061. Connected is announced from the gateway's own readiness, never
  // from the configuration having been applied.
  it("completes once the service reports ready, and reports it upward", async () => {
    const onConnected = vi.fn()
    fetchTelegramPairingStatus.mockResolvedValue({ state: "ready" })

    render(<TelegramDesktopConnect onConnected={onConnected} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    await waitFor(() => expect(completeTelegramPairing).toHaveBeenCalledWith("pair-1"))
    await waitFor(() => expect(onConnected).toHaveBeenCalled())
    expect(toastSuccess).toHaveBeenCalled()
  })

  // The service delivers the token exactly once, so a second poll must not try again.
  it("completes exactly once even if polling ticks again", async () => {
    fetchTelegramPairingStatus.mockResolvedValue({ state: "ready" })

    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    await waitFor(() => expect(completeTelegramPairing).toHaveBeenCalled())
    await new Promise((resolve) => setTimeout(resolve, 60))
    expect(completeTelegramPairing).toHaveBeenCalledTimes(1)
  })

  // Saved-but-not-live is PC-DEF-030's rule and is not a failure.
  it("reports a parked change as information, not an error", async () => {
    fetchTelegramPairingStatus.mockResolvedValue({ state: "ready" })
    completeTelegramPairing.mockResolvedValue({ ok: true, pending: true })
    // Parked means not receiving, so readiness never arrives here.
    fetchTelegramReadiness.mockResolvedValue({
      state: "gateway_stopped",
      ready: false,
    })

    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    await waitFor(() => expect(toastInfo).toHaveBeenCalled())
    expect(toastSuccess).not.toHaveBeenCalled()
  })

  // The defect this exists for: "Connected" used to be announced as soon as the
  // configuration was applied, which is before the channel is receiving.
  it("does not announce connected while Telegram is still starting", async () => {
    const onConnected = vi.fn()
    fetchTelegramPairingStatus.mockResolvedValue({ state: "ready" })
    fetchTelegramReadiness.mockResolvedValue({
      state: "channel_starting",
      ready: false,
    })

    render(<TelegramDesktopConnect onConnected={onConnected} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    await waitFor(() => expect(completeTelegramPairing).toHaveBeenCalled())
    // The stage is named rather than claimed as connected.
    expect(
      await screen.findByText("channels.telegram.readiness.channel_starting"),
    ).toBeTruthy()
    expect(toastSuccess).not.toHaveBeenCalled()
    expect(onConnected).not.toHaveBeenCalled()
  })

  // Each stage is shown, so the user is told why not to send a message yet.
  it("names the command-registration stage", async () => {
    fetchTelegramPairingStatus.mockResolvedValue({ state: "ready" })
    fetchTelegramReadiness.mockResolvedValue({
      state: "registering_commands",
      ready: false,
    })

    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    expect(
      await screen.findByText(
        "channels.telegram.readiness.registering_commands",
      ),
    ).toBeTruthy()
  })

  // Readiness arriving late must still announce, and only once.
  it("announces connected when readiness finally arrives", async () => {
    const onConnected = vi.fn()
    fetchTelegramPairingStatus.mockResolvedValue({ state: "ready" })
    fetchTelegramReadiness
      .mockResolvedValueOnce({ state: "gateway_starting", ready: false })
      .mockResolvedValue({ state: "ready", ready: true })

    render(<TelegramDesktopConnect onConnected={onConnected} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    // Long enough to cover a second readiness poll: the first answer is not
    // ready, so the announcement can only come from the one after it.
    await waitFor(() => expect(onConnected).toHaveBeenCalled(), {
      timeout: 5000,
    })
    expect(onConnected).toHaveBeenCalledTimes(1)
    expect(toastSuccess).toHaveBeenCalledTimes(1)
  })

  it("reports an expired pairing and offers a retry", async () => {
    fetchTelegramPairingStatus.mockResolvedValue({ state: "expired" })

    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    expect(
      await screen.findByText("channels.telegram.desktop.errorExpired"),
    ).toBeTruthy()
    expect(screen.getByText("channels.telegram.desktop.retry")).toBeTruthy()
    expect(completeTelegramPairing).not.toHaveBeenCalled()
  })

  // A newer service reporting a state this build has not heard of arrives as failed, and
  // must not look like progress.
  it("reports a failed pairing rather than waiting forever", async () => {
    fetchTelegramPairingStatus.mockResolvedValue({ state: "failed" })

    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    expect(
      await screen.findByText("channels.telegram.desktop.errorFailed"),
    ).toBeTruthy()
    expect(completeTelegramPairing).not.toHaveBeenCalled()
  })

  it("reports a rate-limited start distinctly", async () => {
    createTelegramPairing.mockRejectedValue(new Error("rate_limited"))

    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))

    expect(
      await screen.findByText("channels.telegram.desktop.errorRateLimited"),
    ).toBeTruthy()
  })

  it("cancels the pairing, which drops the token Core was holding", async () => {
    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))
    await screen.findByText("@pocketclaw_abc_bot")

    fireEvent.click(screen.getByText("common.cancel"))

    await waitFor(() => expect(cancelTelegramPairing).toHaveBeenCalledWith("pair-1"))
    // Back to the beginning, so the user can start again.
    expect(
      await screen.findByText("channels.telegram.desktop.connect"),
    ).toBeTruthy()
  })

  // An abandoned tab must not leave Core holding a poll token.
  it("cancels an unfinished pairing when unmounted", async () => {
    const view = render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))
    await screen.findByText("@pocketclaw_abc_bot")

    view.unmount()

    await waitFor(() => expect(cancelTelegramPairing).toHaveBeenCalledWith("pair-1"))
  })

  // A completed pairing is already spent; cancelling it would be a pointless call.
  it("does not cancel a completed pairing on unmount", async () => {
    fetchTelegramPairingStatus.mockResolvedValue({ state: "ready" })

    const view = render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))
    await waitFor(() => expect(completeTelegramPairing).toHaveBeenCalled())

    view.unmount()
    await new Promise((resolve) => setTimeout(resolve, 30))

    expect(cancelTelegramPairing).not.toHaveBeenCalled()
  })

  // Nothing rendered may carry a credential: the UI never receives one.
  it("renders no credential-shaped value", async () => {
    render(<TelegramDesktopConnect onConnected={vi.fn()} />)
    fireEvent.click(screen.getByText("channels.telegram.desktop.connect"))
    await screen.findByText("@pocketclaw_abc_bot")

    const text = document.body.textContent ?? ""
    expect(/\d{6,}:[A-Za-z0-9_-]{20,}/.test(text)).toBe(false)
    expect(text).not.toContain("poll_token")
  })
})
