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

vi.mock("@/store/gateway", () => ({ refreshGatewayState: vi.fn() }))

vi.mock("@/lib/restart-required", () => ({
  showSaveSuccessOrRestartToast: vi.fn(),
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
