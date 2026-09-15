/**
 * Channels → Telegram, through the component the physical APK actually renders.
 *
 * Milestone D shipped a working managed-bot flow that no user could reach: it
 * was wired to the native settings list while this page — the one reached from
 * the Channels list in the embedded console — still opened straight onto Bot
 * Token. Widget tests of the onboarding screen all passed, because none of them
 * went through this route. These do.
 */
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import en from "@/i18n/locales/en.json"
import type { PocketClawHost } from "@/lib/pocketclaw-host"

const getChannelsCatalog = vi.fn()
const getChannelConfig = vi.fn()

vi.mock("@/api/channels", () => ({
  getChannelsCatalog: () => getChannelsCatalog(),
  getChannelConfig: (name: string) => getChannelConfig(name),
  patchAppConfig: vi.fn(),
}))

vi.mock("@/hooks/use-gateway", () => ({
  useGateway: () => ({ state: "running" }),
}))

vi.mock("@/store/gateway", () => ({ refreshGatewayState: vi.fn() }))

// PC-DEF-061. The connected card's wording comes from authoritative readiness
// now, so a page test has to state what the gateway reports rather than letting
// an unmocked probe decide it.
const fetchTelegramReadiness = vi.fn()
vi.mock("@/api/telegram-lifecycle", () => ({
  fetchTelegramReadiness: (...a: unknown[]) => fetchTelegramReadiness(...a),
  disconnectTelegram: vi.fn(),
}))

vi.mock("@/lib/restart-required", () => ({
  showSaveSuccessOrRestartToast: vi.fn(),
  // PC-DEF-030: saving a channel now applies it through this helper instead of
  // telling the user to restart. The stub runs the save so these tests keep
  // asserting what they always did -- what gets persisted.
  saveAndApplyGatewayConfig: vi.fn(
    async (_t: unknown, options: { save: () => Promise<unknown> }) =>
      options.save(),
  ),
}))

// The real header pulls in sidebar context this page does not otherwise need.
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

const { ChannelConfigPage } = await import(
  "@/components/channels/channel-config-page"
)

const TELEGRAM_CHANNEL = {
  name: "telegram",
  display_name: "Telegram",
  config_key: "telegram",
}

function installHost(overrides: Partial<PocketClawHost> = {}): PocketClawHost {
  const host: PocketClawHost = {
    platform: "android",
    onboardingConfigured: true,
    telegramBotUsername: null,
    openTelegramOnboarding: vi.fn(),
    openExternal: vi.fn(),
    ...overrides,
  }
  window.__pocketclawHost = host
  return host
}

function arrange({
  configuredSecrets = [] as string[],
  config = {} as Record<string, unknown>,
} = {}) {
  getChannelsCatalog.mockResolvedValue({ channels: [TELEGRAM_CHANNEL] })
  getChannelConfig.mockResolvedValue({
    config,
    configured_secrets: configuredSecrets,
    config_key: "telegram",
  })
}

async function renderTelegramPage() {
  render(<ChannelConfigPage channelName="telegram" />)
  await waitFor(() =>
    expect(screen.getByRole("heading", { name: "Telegram" })).toBeDefined(),
  )
}

describe("Channels → Telegram", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    delete window.__pocketclawHost
    // A receiving channel unless a test says otherwise: these cases are about
    // which surface is shown, not about the readiness lifecycle.
    fetchTelegramReadiness.mockResolvedValue({ state: "ready", ready: true })
  })

  it("case 1: unconfigured with an onboarding URL puts managed onboarding first", async () => {
    installHost({ onboardingConfigured: true })
    arrange()

    await renderTelegramPage()

    expect(screen.getByTestId("telegram-surface-managed-onboarding")).toBeDefined()
    expect(screen.getByText(translate("channels.telegram.connectTitle"))).toBeDefined()
    expect(
      screen.getByRole("button", { name: translate("channels.telegram.openTelegram") }),
    ).toBeDefined()
    expect(screen.getByText(translate("channels.telegram.statusReady"))).toBeDefined()

    // The regression itself: the raw token form must not be the landing page.
    expect(screen.queryByText(translate("channels.field.token"))).toBeNull()
    expect(screen.queryByText(translate("channels.field.baseUrl"))).toBeNull()
  })

  it("case 1: the connect button hands off to the native pairing flow", async () => {
    const host = installHost({ onboardingConfigured: true })
    arrange()

    await renderTelegramPage()
    await userEvent.click(
      screen.getByRole("button", { name: translate("channels.telegram.openTelegram") }),
    )

    expect(host.openTelegramOnboarding).toHaveBeenCalledTimes(1)
  })

  it("case 2: unconfigured with no onboarding URL falls back to a clear manual setup", async () => {
    installHost({ onboardingConfigured: false })
    arrange()

    await renderTelegramPage()

    expect(screen.getByTestId("telegram-surface-manual-only")).toBeDefined()
    expect(screen.getByText(translate("channels.telegram.manualOnlyTitle"))).toBeDefined()
    // The legacy form is present immediately, with nothing to expand.
    expect(screen.getByText(translate("channels.field.token"))).toBeDefined()
    expect(screen.getByText(translate("channels.field.baseUrl"))).toBeDefined()
    expect(
      screen.queryByRole("button", { name: translate("channels.telegram.openTelegram") }),
    ).toBeNull()
  })

  it("case 2: a plain browser with no host at all also gets manual setup", async () => {
    arrange()

    await renderTelegramPage()

    expect(screen.getByTestId("telegram-surface-manual-only")).toBeDefined()
    expect(screen.getByText(translate("channels.field.token"))).toBeDefined()
  })

  // PC-DEF-061. Configured is not the same as receiving, and the card may not
  // say Connected until the gateway says the channel is running.
  it("case 3b: a configured Telegram that is still starting does not say connected", async () => {
    installHost({
      onboardingConfigured: true,
      telegramBotUsername: "pocketclaw_ab12cd34_bot",
    })
    fetchTelegramReadiness.mockResolvedValue({
      state: "channel_starting",
      ready: false,
    })
    arrange({
      configuredSecrets: ["token"],
      config: { enabled: true, allow_from: ["123456789"] },
    })

    await renderTelegramPage()

    expect(
      await screen.findByText(translate("channels.telegram.startingTitle")),
    ).toBeDefined()
    expect(
      screen.queryByText(translate("channels.telegram.connected")),
    ).toBeNull()
  })

  it("case 3: an already-configured Telegram shows the connected summary first", async () => {
    const host = installHost({
      onboardingConfigured: true,
      telegramBotUsername: "pocketclaw_ab12cd34_bot",
    })
    arrange({
      configuredSecrets: ["token"],
      config: { enabled: true, allow_from: ["123456789"] },
    })

    await renderTelegramPage()

    expect(screen.getByTestId("telegram-surface-connected")).toBeDefined()
    expect(screen.getByText(translate("channels.telegram.connected"))).toBeDefined()
    expect(screen.getByText("@pocketclaw_ab12cd34_bot")).toBeDefined()
    expect(screen.getByText(translate("channels.telegram.ownerConfigured"))).toBeDefined()
    expect(
      screen.getByRole("button", { name: translate("channels.telegram.advancedSettings") }),
    ).toBeDefined()

    // Not dumped straight into raw token fields.
    expect(screen.queryByText(translate("channels.field.token"))).toBeNull()

    await userEvent.click(
      screen.getByRole("button", { name: translate("channels.telegram.openChat") }),
    )
    expect(host.openExternal).toHaveBeenCalledWith(
      "https://t.me/pocketclaw_ab12cd34_bot",
    )

    await userEvent.click(
      screen.getByRole("button", { name: translate("channels.telegram.reconnect") }),
    )
    expect(host.openTelegramOnboarding).toHaveBeenCalledTimes(1)
    expect(screen.getByTestId("telegram-surface-connected")).toBeDefined()
  })

  it("case 4: Advanced / Manual setup reveals the legacy Bot Token form", async () => {
    installHost({ onboardingConfigured: true })
    arrange()

    await renderTelegramPage()
    expect(screen.queryByText(translate("channels.field.token"))).toBeNull()

    await userEvent.click(
      screen.getByRole("button", { name: translate("channels.telegram.advancedManual") }),
    )

    // Every legacy field the manual path depends on is still here.
    expect(screen.getByText(translate("channels.field.token"))).toBeDefined()
    expect(screen.getByText(translate("channels.field.baseUrl"))).toBeDefined()
    expect(screen.getByText(translate("channels.field.proxy"))).toBeDefined()
    expect(screen.getByText(translate("channels.field.allowFrom"))).toBeDefined()
    expect(screen.getByText(translate("channels.field.typingEnabled"))).toBeDefined()
    expect(screen.getByText(translate("channels.field.placeholderEnabled"))).toBeDefined()
  })

  it("case 4: Advanced Settings reveals the same form from the connected state", async () => {
    installHost({
      onboardingConfigured: true,
      telegramBotUsername: "pocketclaw_ab12cd34_bot",
    })
    arrange({ configuredSecrets: ["token"], config: { enabled: true } })

    await renderTelegramPage()
    await userEvent.click(
      screen.getByRole("button", { name: translate("channels.telegram.advancedSettings") }),
    )

    expect(screen.getByText(translate("channels.field.token"))).toBeDefined()
    expect(screen.getByText(translate("channels.field.baseUrl"))).toBeDefined()
  })

  it("picks up a host that injects itself after the page has rendered", async () => {
    arrange()
    await renderTelegramPage()
    expect(screen.getByTestId("telegram-surface-manual-only")).toBeDefined()

    installHost({ onboardingConfigured: true })
    window.dispatchEvent(new CustomEvent("pocketclaw:host-ready"))

    await waitFor(() =>
      expect(screen.getByTestId("telegram-surface-managed-onboarding")).toBeDefined(),
    )
  })
})
