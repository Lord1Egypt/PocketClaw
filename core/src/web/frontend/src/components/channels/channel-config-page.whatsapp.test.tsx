/**
 * Channels → WhatsApp, through the component the physical APK renders.
 *
 * The console used to offer two WhatsApp cards — "WhatsApp" and "WhatsApp
 * Native" — whose first questions were a bridge URL and a session store path.
 * A phone user has neither. These assert the replacement: one card, one number,
 * and no auto-send.
 */
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import en from "@/i18n/locales/en.json"
import type { PocketClawHost } from "@/lib/pocketclaw-host"

const getChannelsCatalog = vi.fn()
const getChannelConfig = vi.fn()
const patchAppConfig = vi.fn()

vi.mock("@/api/channels", () => ({
  getChannelsCatalog: () => getChannelsCatalog(),
  getChannelConfig: (name: string) => getChannelConfig(name),
  patchAppConfig: (patch: unknown) => patchAppConfig(patch),
}))

vi.mock("@/hooks/use-gateway", () => ({
  useGateway: () => ({ state: "running" }),
}))

// restart-required is deliberately NOT mocked. The point of these cases is the
// real decision it makes, so only the two things it talks to are stubbed.
const refreshGatewayState = vi.fn()
const applyGatewayConfig = vi.fn()

vi.mock("@/store/gateway", () => ({
  refreshGatewayState: (...args: unknown[]) => refreshGatewayState(...args),
}))
vi.mock("@/api/gateway", () => ({
  applyGatewayConfig: (...args: unknown[]) => applyGatewayConfig(...args),
}))
vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    loading: vi.fn(() => "toast-id"),
  },
}))

vi.mock("@/components/page-header", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}))

function translate(key: string): string {
  const value = key
    .split(".")
    .reduce<unknown>(
      (node, part) =>
        node && typeof node === "object"
          ? (node as Record<string, unknown>)[part]
          : undefined,
      en,
    )
  return typeof value === "string" ? value : key
}

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: translate }),
}))

const { ChannelConfigPage } =
  await import("@/components/channels/channel-config-page")

const SELF_CHAT_CHANNEL = {
  name: "whatsapp_self_chat",
  display_name: "WhatsApp Self-Chat",
  config_key: "whatsapp_self_chat",
}

function installHost(withWhatsApp = true): PocketClawHost {
  const host: PocketClawHost = {
    platform: "android",
    onboardingConfigured: true,
    telegramBotUsername: null,
    openTelegramOnboarding: vi.fn(),
    openExternal: vi.fn(),
    ...(withWhatsApp ? { openWhatsAppSelfChat: vi.fn() } : {}),
  }
  window.__pocketclawHost = host
  return host
}

function arrange(config: Record<string, unknown> = {}) {
  getChannelsCatalog.mockResolvedValue({ channels: [SELF_CHAT_CHANNEL] })
  getChannelConfig.mockResolvedValue({
    config,
    configured_secrets: [],
    config_key: "whatsapp_self_chat",
  })
  patchAppConfig.mockResolvedValue({ status: "ok" })
}

const STOPPED = { status: "stopped", canStart: true, restartRequired: false }
const STARTING = { status: "starting", canStart: false, restartRequired: false }
const RUNNING = { status: "running", canStart: true, restartRequired: false }
const NEEDS_RESTART = {
  status: "running",
  canStart: true,
  restartRequired: true,
}
const ERRORED = { status: "error", canStart: true, restartRequired: true }

/** The gateway needs a restart, and comes back healthy once it happens. */
function gatewayRestartsCleanly() {
  refreshGatewayState.mockResolvedValueOnce(NEEDS_RESTART)
  refreshGatewayState.mockResolvedValue(RUNNING)
  applyGatewayConfig.mockResolvedValue({ status: "ok" })
}

async function connectWith(number: string) {
  await userEvent.type(
    screen.getByLabelText(translate("channels.whatsappSelfChat.numberLabel")),
    number,
  )
  await userEvent.click(
    screen.getByRole("button", {
      name: translate("channels.whatsappSelfChat.connect"),
    }),
  )
}

function savedSelfNumber(): string | undefined {
  const call = patchAppConfig.mock.calls.at(-1)?.[0] as
    | {
        channel_list?: Record<string, { settings?: { self_number?: string } }>
      }
    | undefined
  return call?.channel_list?.whatsapp_self_chat?.settings?.self_number
}

async function renderSelfChatPage() {
  render(<ChannelConfigPage channelName="whatsapp_self_chat" />)
  await waitFor(() =>
    expect(
      screen.getByRole("heading", { name: "WhatsApp Self-Chat" }),
    ).toBeDefined(),
  )
}

describe("Channels → WhatsApp Self-Chat", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    delete window.__pocketclawHost
    refreshGatewayState.mockResolvedValue(RUNNING)
    applyGatewayConfig.mockResolvedValue({ status: "ok" })
  })

  it("the retired WhatsApp entries are no longer reachable", async () => {
    getChannelsCatalog.mockResolvedValue({ channels: [SELF_CHAT_CHANNEL] })
    getChannelConfig.mockResolvedValue({
      config: {},
      configured_secrets: [],
      config_key: "whatsapp",
    })

    for (const retired of ["whatsapp", "whatsapp_native"]) {
      const { unmount } = render(<ChannelConfigPage channelName={retired} />)
      await waitFor(() =>
        expect(
          screen.getByText(translate("channels.page.notFound")),
        ).toBeDefined(),
      )
      unmount()
    }
  })

  it("unconfigured shows one number field and Connect, with no legacy fields", async () => {
    installHost()
    arrange()

    await renderSelfChatPage()

    expect(screen.getByTestId("whatsapp-self-chat-setup")).toBeDefined()
    expect(
      screen.getByLabelText(translate("channels.whatsappSelfChat.numberLabel")),
    ).toBeDefined()
    expect(
      screen.getByRole("button", {
        name: translate("channels.whatsappSelfChat.connect"),
      }),
    ).toBeDefined()

    // The whole point of the milestone: none of the old configuration survives.
    expect(screen.queryByText(/bridge url/i)).toBeNull()
    expect(screen.queryByText(/session store/i)).toBeNull()
    expect(screen.queryByText(/websocket url/i)).toBeNull()
    expect(
      screen.queryByText(translate("channels.page.enableLabel")),
    ).toBeNull()
    expect(
      screen.queryByRole("button", { name: translate("common.save") }),
    ).toBeNull()
  })

  it("Connect stores the number in canonical international form", async () => {
    installHost()
    arrange()

    await renderSelfChatPage()
    await userEvent.type(
      screen.getByLabelText(translate("channels.whatsappSelfChat.numberLabel")),
      "0020 101 234 5678",
    )
    await userEvent.click(
      screen.getByRole("button", {
        name: translate("channels.whatsappSelfChat.connect"),
      }),
    )

    await waitFor(() => expect(patchAppConfig).toHaveBeenCalledTimes(1))
    expect(patchAppConfig).toHaveBeenCalledWith({
      channel_list: {
        whatsapp_self_chat: {
          enabled: false,
          type: "whatsapp_self_chat",
          settings: { self_number: "+201012345678" },
        },
      },
    })
  })

  it("Connect refuses an ambiguous national number without saving", async () => {
    installHost()
    arrange()

    await renderSelfChatPage()
    await userEvent.type(
      screen.getByLabelText(translate("channels.whatsappSelfChat.numberLabel")),
      "0101 234 5678",
    )
    await userEvent.click(
      screen.getByRole("button", {
        name: translate("channels.whatsappSelfChat.connect"),
      }),
    )

    expect(
      screen.getByText(
        translate("channels.whatsappSelfChat.errorNotInternational"),
      ),
    ).toBeDefined()
    expect(patchAppConfig).not.toHaveBeenCalled()
  })

  it("a stored number shows Test, Change and Disconnect", async () => {
    installHost()
    arrange({ self_number: "+201012345678" })

    await renderSelfChatPage()

    expect(screen.getByTestId("whatsapp-self-chat-connected")).toBeDefined()
    expect(screen.getByText("+201012345678")).toBeDefined()
    for (const key of ["test", "change", "disconnect"]) {
      expect(
        screen.getByRole("button", {
          name: translate(`channels.whatsappSelfChat.${key}`),
        }),
      ).toBeDefined()
    }
  })

  it("Test asks the host to prepare the marker message and never sends", async () => {
    const host = installHost()
    arrange({ self_number: "+201012345678" })

    await renderSelfChatPage()
    await userEvent.click(
      screen.getByRole("button", {
        name: translate("channels.whatsappSelfChat.test"),
      }),
    )

    expect(host.openWhatsAppSelfChat).toHaveBeenCalledWith(
      "+201012345678",
      "PocketClaw WhatsApp test",
    )
  })

  it("Test is hidden in a plain browser, where no host can open WhatsApp", async () => {
    arrange({ self_number: "+201012345678" })

    await renderSelfChatPage()

    expect(
      screen.queryByRole("button", {
        name: translate("channels.whatsappSelfChat.test"),
      }),
    ).toBeNull()
    expect(
      screen.getByText(translate("channels.whatsappSelfChat.testNeedsApp")),
    ).toBeDefined()
  })

  it("Change reopens the number field prefilled", async () => {
    installHost()
    arrange({ self_number: "+201012345678" })

    await renderSelfChatPage()
    await userEvent.click(
      screen.getByRole("button", {
        name: translate("channels.whatsappSelfChat.change"),
      }),
    )

    const field = screen.getByLabelText(
      translate("channels.whatsappSelfChat.numberLabel"),
    ) as HTMLInputElement
    expect(field.value).toBe("+201012345678")
  })

  it("Disconnect clears the stored number", async () => {
    installHost()
    arrange({ self_number: "+201012345678" })

    await renderSelfChatPage()
    await userEvent.click(
      screen.getByRole("button", {
        name: translate("channels.whatsappSelfChat.disconnect"),
      }),
    )

    await waitFor(() => expect(patchAppConfig).toHaveBeenCalledTimes(1))
    expect(patchAppConfig).toHaveBeenCalledWith({
      channel_list: {
        whatsapp_self_chat: {
          enabled: false,
          type: "whatsapp_self_chat",
          settings: { self_number: "" },
        },
      },
    })
  })
})

/**
 * Applying a Self-Chat change.
 *
 * The physical gap this closes: Connect, Change and Disconnect all saved, then
 * asked the user to restart Core by hand. The apply now goes through the same
 * machinery every other configuration change uses — which is also what keeps
 * it from ever interrupting an answer in progress.
 */
describe("Channels → WhatsApp Self-Chat → applying the change", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    delete window.__pocketclawHost
    refreshGatewayState.mockResolvedValue(RUNNING)
    applyGatewayConfig.mockResolvedValue({ status: "ok" })
  })

  it("case 1: Connect while Core is running applies it automatically", async () => {
    installHost()
    arrange()
    gatewayRestartsCleanly()

    await renderSelfChatPage()
    await connectWith("+201012345678")

    await waitFor(() => expect(applyGatewayConfig).toHaveBeenCalledTimes(1))
    expect(savedSelfNumber()).toBe("+201012345678")

    // The whole point: nothing anywhere tells the user to restart by hand.
    await waitFor(() =>
      expect(
        screen.queryByTestId("whatsapp-self-chat-apply-pending"),
      ).toBeNull(),
    )
    expect(screen.queryByTestId("whatsapp-self-chat-apply-warning")).toBeNull()
  })

  it("case 2: Change while Core is running applies the new number", async () => {
    installHost()
    arrange({ self_number: "+201012345678" })
    gatewayRestartsCleanly()

    await renderSelfChatPage()
    await userEvent.click(
      screen.getByRole("button", {
        name: translate("channels.whatsappSelfChat.change"),
      }),
    )
    const field = screen.getByLabelText(
      translate("channels.whatsappSelfChat.numberLabel"),
    )
    await userEvent.clear(field)
    await userEvent.type(field, "+90 532 123 4567")
    await userEvent.click(
      screen.getByRole("button", {
        name: translate("channels.whatsappSelfChat.connect"),
      }),
    )

    await waitFor(() => expect(applyGatewayConfig).toHaveBeenCalledTimes(1))
    expect(savedSelfNumber()).toBe("+905321234567")
  })

  it("case 3: Disconnect removes the number and applies it", async () => {
    installHost()
    arrange({ self_number: "+201012345678" })
    gatewayRestartsCleanly()

    await renderSelfChatPage()
    await userEvent.click(
      screen.getByRole("button", {
        name: translate("channels.whatsappSelfChat.disconnect"),
      }),
    )

    await waitFor(() => expect(applyGatewayConfig).toHaveBeenCalledTimes(1))
    // Empty, not absent: the stored number has to be cleared, which is what
    // makes the agent tool report Self-Chat as not configured.
    expect(savedSelfNumber()).toBe("")
  })

  it("case 4: a stopped Core is saved to and left alone", async () => {
    installHost()
    arrange()
    refreshGatewayState.mockResolvedValue(STOPPED)

    await renderSelfChatPage()
    await connectWith("+201012345678")

    await waitFor(() => expect(patchAppConfig).toHaveBeenCalledTimes(1))
    // Nothing is running, so there is nothing to apply and nothing to start:
    // the next start reads the saved configuration.
    expect(applyGatewayConfig).not.toHaveBeenCalled()
    expect(screen.queryByTestId("whatsapp-self-chat-apply-warning")).toBeNull()
  })

  it("case 5: a Core that is still starting is never interrupted", async () => {
    installHost()
    arrange()
    refreshGatewayState.mockResolvedValue(STARTING)

    await renderSelfChatPage()
    await connectWith("+201012345678")

    await waitFor(() => expect(patchAppConfig).toHaveBeenCalledTimes(1))
    expect(applyGatewayConfig).not.toHaveBeenCalled()
  })

  it("case 6: a busy Core is not restarted, and the card says so", async () => {
    installHost()
    arrange()
    refreshGatewayState.mockResolvedValue(NEEDS_RESTART)
    // The backend held the restart back because a turn was in flight. Saved,
    // deliberately not applied — not a failure.
    applyGatewayConfig.mockResolvedValue({
      status: "saved_not_applied",
      outcome: "busy_timeout",
    })

    await renderSelfChatPage()
    await connectWith("+201012345678")

    await waitFor(() =>
      expect(
        screen.getByTestId("whatsapp-self-chat-apply-pending"),
      ).toBeDefined(),
    )
    expect(
      screen.getByText(translate("channels.whatsappSelfChat.applyDeferred")),
    ).toBeDefined()
    // Never an instruction to restart something by hand.
    expect(screen.queryByText(/Restart Gateway/i)).toBeNull()
  })

  it("case 7: a failed apply never claims the change went live", async () => {
    installHost()
    arrange()
    refreshGatewayState.mockResolvedValueOnce(NEEDS_RESTART)
    refreshGatewayState.mockResolvedValue(ERRORED)
    applyGatewayConfig.mockResolvedValue({ status: "ok" })

    await renderSelfChatPage()
    await connectWith("+201012345678")

    await waitFor(() =>
      expect(
        screen.getByTestId("whatsapp-self-chat-apply-warning"),
      ).toBeDefined(),
    )
    expect(
      screen.getByText(translate("channels.whatsappSelfChat.applyFailed")),
    ).toBeDefined()
    expect(screen.queryByText(/Restart Gateway/i)).toBeNull()
  })

  it("a save that the gateway refuses outright is reported, not swallowed", async () => {
    installHost()
    arrange()
    patchAppConfig.mockRejectedValue(new Error("config rejected"))

    await renderSelfChatPage()
    await connectWith("+201012345678")

    await waitFor(() =>
      expect(screen.getByText("config rejected")).toBeDefined(),
    )
    expect(applyGatewayConfig).not.toHaveBeenCalled()
  })
})
