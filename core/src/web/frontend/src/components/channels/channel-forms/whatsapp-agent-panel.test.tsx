/**
 * Channels → WhatsApp Agent Channel (experimental), through the component the
 * physical APK renders.
 *
 * The cases that matter here are the security ones: the panel must never hold
 * the pairing payload as a string, it must refuse to pair without a sender to
 * allow, and it must be unmistakably marked experimental.
 */
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import en from "@/i18n/locales/en.json"

const getWhatsAppAgentStatus = vi.fn()
const forgetWhatsAppAgentSession = vi.fn()
const getChannelConfig = vi.fn()

vi.mock("@/api/channels", () => ({
  WHATSAPP_AGENT_QR_URL: "/api/channels/whatsapp-agent/qr.png",
  getWhatsAppAgentStatus: () => getWhatsAppAgentStatus(),
  forgetWhatsAppAgentSession: () => forgetWhatsAppAgentSession(),
  getChannelConfig: (name: string) => getChannelConfig(name),
}))

function translate(key: string, vars?: Record<string, unknown>): string {
  const value = key
    .split(".")
    .reduce<unknown>(
      (node, part) =>
        node && typeof node === "object"
          ? (node as Record<string, unknown>)[part]
          : undefined,
      en,
    )
  if (typeof value !== "string") return key
  if (!vars) return value
  return value.replace(/\{\{(\w+)\}\}/g, (_, name) => String(vars[name] ?? ""))
}

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: translate }),
}))

const { WhatsAppAgentPanel } = await import(
  "@/components/channels/channel-forms/whatsapp-agent-panel"
)

const SELF_NUMBER = "+201012345678"

function withSelfNumber(number: string | null) {
  getChannelConfig.mockResolvedValue({
    config: number === null ? {} : { self_number: number },
    configured_secrets: [],
    config_key: "whatsapp_self_chat",
  })
}

function status(overrides: Record<string, unknown> = {}) {
  return {
    available: true,
    enabled: false,
    state: "not_paired",
    has_qr: false,
    ...overrides,
  }
}

describe("WhatsAppAgentPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    withSelfNumber(SELF_NUMBER)
    getWhatsAppAgentStatus.mockResolvedValue(status())
  })

  it("marks itself experimental", async () => {
    render(<WhatsAppAgentPanel config={{}} onPersist={vi.fn()} />)

    expect(
      await screen.findByText(en.channels.whatsappAgent.experimentalTitle),
    ).toBeDefined()
  })

  it("offers no bridge URL, session store path, or use_native control", async () => {
    // These were the first questions the retired cards asked, and a phone user
    // can answer none of them.
    const { container } = render(
      <WhatsAppAgentPanel config={{}} onPersist={vi.fn()} />,
    )
    await screen.findByText(en.channels.whatsappAgent.experimentalTitle)

    expect(container.querySelectorAll("input")).toHaveLength(0)
    expect(container.textContent).not.toMatch(/bridge/i)
    expect(container.textContent).not.toMatch(/session.store/i)
    expect(container.textContent).not.toMatch(/use_native/i)
  })

  it("refuses to pair until a self number is configured", async () => {
    withSelfNumber(null)
    const onPersist = vi.fn()
    render(<WhatsAppAgentPanel config={{}} onPersist={onPersist} />)

    const pair = await screen.findByRole("button", {
      name: en.channels.whatsappAgent.pair,
    })
    await waitFor(() => expect(pair.hasAttribute("disabled")).toBe(true))
    expect(
      screen.getAllByText(en.channels.whatsappAgent.errorNoSelfNumber).length,
    ).toBeGreaterThan(0)
    expect(onPersist).not.toHaveBeenCalled()
  })

  it("seeds allow_from with only the user's own number", async () => {
    // This channel reaches an agent holding shell tools, so it must start
    // closed to everyone but the account owner.
    const onPersist = vi.fn().mockResolvedValue("applied")
    render(<WhatsAppAgentPanel config={{}} onPersist={onPersist} />)

    const pair = await screen.findByRole("button", {
      name: en.channels.whatsappAgent.pair,
    })
    await waitFor(() => expect(pair.hasAttribute("disabled")).toBe(false))
    await userEvent.click(pair)

    await waitFor(() => expect(onPersist).toHaveBeenCalledTimes(1))
    const [config, enabled] = onPersist.mock.calls[0]
    // The canonical "+" form is what Self-Chat stores and what the user reads
    // back in their config; Core normalizes it before matching.
    expect(config.allow_from).toEqual([SELF_NUMBER])
    expect(config.use_native).toBe(true)
    expect(enabled).toBe(true)
  })

  it("renders the pairing code as an image, never as text", async () => {
    getWhatsAppAgentStatus.mockResolvedValue(
      status({ state: "pairing", has_qr: true }),
    )
    const { container } = render(
      <WhatsAppAgentPanel config={{}} onPersist={vi.fn()} />,
    )

    const image = await screen.findByAltText(en.channels.whatsappAgent.qrAlt)
    expect(image.getAttribute("src")).toContain(
      "/api/channels/whatsapp-agent/qr.png",
    )
    // The payload is served as bytes by the backend and is never part of the
    // status response, so it cannot appear in the rendered markup.
    expect(container.textContent).not.toContain("2@")
  })

  it("reports each pairing state", async () => {
    for (const [state, key] of [
      ["not_paired", en.channels.whatsappAgent.stateNotPaired],
      ["pairing", en.channels.whatsappAgent.statePairing],
      ["connecting", en.channels.whatsappAgent.stateConnecting],
      ["connected", en.channels.whatsappAgent.stateConnected],
      ["disconnected", en.channels.whatsappAgent.stateDisconnected],
      ["logged_out", en.channels.whatsappAgent.stateLoggedOut],
    ] as const) {
      getWhatsAppAgentStatus.mockResolvedValue(status({ state }))
      const { unmount } = render(
        <WhatsAppAgentPanel config={{}} onPersist={vi.fn()} />,
      )
      const node = await screen.findByTestId("wa-agent-state")
      expect(node.textContent).toBe(key)
      unmount()
    }
  })

  it("disables the channel before erasing the session", async () => {
    // Erasing first would race the gateway, which still holds the database open
    // until it restarts.
    const order: string[] = []
    const onPersist = vi.fn().mockImplementation(async () => {
      order.push("persist")
      return "applied"
    })
    forgetWhatsAppAgentSession.mockImplementation(async () => {
      order.push("forget")
      return { status: "forgotten" }
    })
    getWhatsAppAgentStatus.mockResolvedValue(
      status({ state: "connected", enabled: true }),
    )
    render(<WhatsAppAgentPanel config={{}} onPersist={onPersist} />)

    const disconnect = await screen.findByRole("button", {
      name: en.channels.whatsappAgent.disconnect,
    })
    await waitFor(() => expect(disconnect.hasAttribute("disabled")).toBe(false))
    await userEvent.click(disconnect)

    await waitFor(() => expect(order).toEqual(["persist", "forget"]))
    expect(onPersist.mock.calls[0][1]).toBe(false)
  })

  it("says so when there is no Android host", async () => {
    getWhatsAppAgentStatus.mockResolvedValue(
      status({ available: false, state: "unavailable" }),
    )
    render(<WhatsAppAgentPanel config={{}} onPersist={vi.fn()} />)

    expect(
      await screen.findByText(en.channels.whatsappAgent.unavailable),
    ).toBeDefined()
  })
})
