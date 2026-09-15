import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const disconnectTelegram = vi.fn()
const toastSuccess = vi.fn()
const toastInfo = vi.fn()
const toastError = vi.fn()

vi.mock("@/api/telegram-lifecycle", () => ({
  disconnectTelegram: (...a: unknown[]) => disconnectTelegram(...a),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock("sonner", () => ({
  toast: {
    success: (...a: unknown[]) => toastSuccess(...a),
    info: (...a: unknown[]) => toastInfo(...a),
    error: (...a: unknown[]) => toastError(...a),
  },
}))

import { TelegramDisconnectDialog } from "./telegram-disconnect-dialog"

/**
 * PC-DEF-062. Removing a bot from a client that is not the Android host.
 *
 * What matters here is that removal is destructive and confirmed, that it goes
 * through the single authoritative call rather than editing fields, and that a
 * parked removal is not reported as a completed one.
 */
describe("TelegramDisconnectDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    disconnectTelegram.mockResolvedValue({ ok: true })
  })

  it("does not remove anything until it is confirmed", () => {
    render(<TelegramDisconnectDialog onDisconnected={vi.fn()} />)

    fireEvent.click(screen.getByText("channels.telegram.disconnect.action"))

    expect(screen.getByTestId("telegram-disconnect-dialog")).toBeTruthy()
    expect(disconnectTelegram).not.toHaveBeenCalled()
  })

  it("removes through the one authoritative call and reports it upward", async () => {
    const onDisconnected = vi.fn()
    render(<TelegramDisconnectDialog onDisconnected={onDisconnected} />)

    fireEvent.click(screen.getByText("channels.telegram.disconnect.action"))
    fireEvent.click(screen.getByText("channels.telegram.disconnect.confirm"))

    await waitFor(() => expect(disconnectTelegram).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(onDisconnected).toHaveBeenCalled())
    expect(toastSuccess).toHaveBeenCalled()
  })

  // Removed from configuration but still running until the gateway is idle.
  // PC-DEF-030's rule, and saying "removed" flatly would be the dishonest half.
  it("reports a parked removal as information, not completion", async () => {
    disconnectTelegram.mockResolvedValue({ ok: true, pending: true })
    render(<TelegramDisconnectDialog onDisconnected={vi.fn()} />)

    fireEvent.click(screen.getByText("channels.telegram.disconnect.action"))
    fireEvent.click(screen.getByText("channels.telegram.disconnect.confirm"))

    await waitFor(() => expect(toastInfo).toHaveBeenCalled())
    expect(toastSuccess).not.toHaveBeenCalled()
  })

  it("keeps the dialog open and says so when removal fails", async () => {
    const onDisconnected = vi.fn()
    disconnectTelegram.mockRejectedValue(new Error("configuration_failed"))
    render(<TelegramDisconnectDialog onDisconnected={onDisconnected} />)

    fireEvent.click(screen.getByText("channels.telegram.disconnect.action"))
    fireEvent.click(screen.getByText("channels.telegram.disconnect.confirm"))

    await waitFor(() => expect(toastError).toHaveBeenCalled())
    // Nothing was removed, so the page must not be told it was.
    expect(onDisconnected).not.toHaveBeenCalled()
    expect(screen.getByTestId("telegram-disconnect-dialog")).toBeTruthy()
  })
})
