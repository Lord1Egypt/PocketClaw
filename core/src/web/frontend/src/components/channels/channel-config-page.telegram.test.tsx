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

// PC-DEF-075. Core's getMe identity is the authority for the Open chat
// destination; the host-injected username is only a cache.
const fetchTelegramIdentity = vi.fn()
vi.mock("@/api/telegram-identity", () => ({
  fetchTelegramIdentity: () => fetchTelegramIdentity(),
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
    fetchTelegramIdentity.mockResolvedValue({ configured: false })
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

  // A valid token with no owner is an explicit incomplete setup. It must never
  // be presented as Connected, and the owner field is opened for the user.
  it("case 3c: a valid token with no owner shows setup incomplete, not connected", async () => {
    installHost({
      onboardingConfigured: true,
      telegramBotUsername: "pocketclaw_ab12cd34_bot",
    })
    fetchTelegramReadiness.mockResolvedValue({
      state: "setup_required",
      ready: false,
      detail: "owner_missing",
    })
    arrange({ configuredSecrets: ["token"], config: { enabled: true } })

    await renderTelegramPage()

    expect(
      await screen.findByText(
        translate("channels.telegram.setupIncompleteTitle"),
      ),
    ).toBeDefined()
    expect(
      screen.queryByText(translate("channels.telegram.connected")),
    ).toBeNull()
    // The direct route to the owner field is revealed, not hidden behind a tap.
    expect(screen.getByText(translate("channels.field.allowFrom"))).toBeDefined()
  })

  // PC-DEF-071. The heading was an else-chain ending in "Starting Telegram…", so
  // every readiness state without a branch of its own was dressed as a stage of
  // starting -- spinner included -- directly above a body sentence that said the
  // opposite. None of the three below is a stage of anything.

  it.each([
    ["authentication_failed", undefined],
    ["not_configured", undefined],
    ["gateway_stopped", undefined],
  ] as const)(
    "case 3f: %s is not presented as Telegram starting",
    async (state, detail) => {
      installHost({
        onboardingConfigured: true,
        telegramBotUsername: "pocketclaw_ab12cd34_bot",
      })
      fetchTelegramReadiness.mockResolvedValue({ state, ready: false, detail })
      arrange({
        configuredSecrets: ["token"],
        config: { enabled: true, allow_from: ["123456789"] },
      })

      await renderTelegramPage()

      const heading = await waitFor(() => {
        const node = document.querySelector("[data-telegram-heading]")
        expect(node?.getAttribute("data-telegram-heading")).toBe("stalled")
        return node as HTMLElement
      })
      // The readiness sentence is the heading, stated exactly once.
      expect(heading.textContent).toBe(
        translate(`channels.telegram.readiness.${state}`),
      )
      expect(
        screen.queryByText(translate("channels.telegram.startingTitle")),
      ).toBeNull()
      expect(
        screen.queryByText(translate("channels.telegram.connected")),
      ).toBeNull()
      expect(
        screen.queryAllByText(
          translate(`channels.telegram.readiness.${state}`),
        ),
      ).toHaveLength(1)
    },
  )

  // The starting stages keep the stage heading they always had: this fix names
  // them positively rather than widening the terminal branch.
  it.each(["gateway_starting", "channel_starting", "registering_commands"] as const)(
    "case 3g: %s is still presented as Telegram starting",
    async (state) => {
      installHost({
        onboardingConfigured: true,
        telegramBotUsername: "pocketclaw_ab12cd34_bot",
      })
      fetchTelegramReadiness.mockResolvedValue({ state, ready: false })
      arrange({
        configuredSecrets: ["token"],
        config: { enabled: true, allow_from: ["123456789"] },
      })

      await renderTelegramPage()

      expect(
        await screen.findByText(translate("channels.telegram.startingTitle")),
      ).toBeDefined()
      expect(
        screen.getByText(translate(`channels.telegram.readiness.${state}`)),
      ).toBeDefined()
    },
  )

  // PC-DEF-075. "Open chat" sent the owner to https://t.me/@name and Telegram
  // answered "Username not found" while the bot itself worked. Two causes, one
  // destination: an un-stripped "@", and a host-cached username that only the
  // native pairing launcher ever writes, so a bot paired any other way leaves it
  // pointing at a bot that may no longer exist.

  it("case 3h: Open chat uses the canonical link, with no @ in the path", async () => {
    const host = installHost({
      onboardingConfigured: true,
      // The cache carries the "@" shape the service can return.
      telegramBotUsername: "@pocketclaw_ab12cd34_bot",
    })
    fetchTelegramIdentity.mockResolvedValue({ configured: true })
    arrange({
      configuredSecrets: ["token"],
      config: { enabled: true, allow_from: ["123456789"] },
    })

    await renderTelegramPage()
    const openChat = await screen.findByRole("button", {
      name: translate("channels.telegram.openChat"),
    })
    await userEvent.click(openChat)

    expect(host.openExternal).toHaveBeenCalledWith(
      "https://t.me/pocketclaw_ab12cd34_bot",
    )
    const url = (host.openExternal as ReturnType<typeof vi.fn>).mock
      .calls[0][0] as string
    expect(url).not.toContain("@")
  })

  it("case 3i: Core's getMe identity replaces a stale cached username", async () => {
    const host = installHost({
      onboardingConfigured: true,
      // A bot from an earlier pairing that no longer exists.
      telegramBotUsername: "pocketclaw_stale0000_bot",
    })
    fetchTelegramIdentity.mockResolvedValue({
      configured: true,
      username: "pocketclaw_current1_bot",
      chat_url: "https://t.me/pocketclaw_current1_bot",
    })
    arrange({
      configuredSecrets: ["token"],
      config: { enabled: true, allow_from: ["123456789"] },
    })

    await renderTelegramPage()
    await waitFor(() =>
      expect(screen.getByText("@pocketclaw_current1_bot")).toBeDefined(),
    )
    await userEvent.click(
      screen.getByRole("button", {
        name: translate("channels.telegram.openChat"),
      }),
    )

    expect(host.openExternal).toHaveBeenCalledWith(
      "https://t.me/pocketclaw_current1_bot",
    )
    expect(host.openExternal).not.toHaveBeenCalledWith(
      expect.stringContaining("stale"),
    )
  })

  it("case 3j: an unusable username offers no Open chat button at all", async () => {
    installHost({
      onboardingConfigured: true,
      telegramBotUsername: "@@pocketclaw_ab12cd34_bot",
    })
    fetchTelegramIdentity.mockResolvedValue({ configured: true })
    arrange({
      configuredSecrets: ["token"],
      config: { enabled: true, allow_from: ["123456789"] },
    })

    await renderTelegramPage()
    await waitFor(() =>
      expect(screen.getByTestId("telegram-surface-connected")).toBeDefined(),
    )
    // No button beats a button that leads to "Username not found".
    expect(
      screen.queryByRole("button", {
        name: translate("channels.telegram.openChat"),
      }),
    ).toBeNull()
  })

  // A bot owned by another service is never Connected. The two ownerships need
  // different instructions, so the body differs by reason.
  it("case 3d: a webhook conflict shows an actionable conflict, not connected", async () => {
    installHost({
      onboardingConfigured: true,
      telegramBotUsername: "pocketclaw_ab12cd34_bot",
    })
    fetchTelegramReadiness.mockResolvedValue({
      state: "telegram_conflict",
      ready: false,
      detail: "webhook_active",
    })
    arrange({ configuredSecrets: ["token"], config: { enabled: true, allow_from: ["1"] } })

    await renderTelegramPage()

    expect(
      await screen.findByText(translate("channels.telegram.conflictTitle")),
    ).toBeDefined()
    expect(
      screen.getByText(translate("channels.telegram.conflictWebhook")),
    ).toBeDefined()
    expect(
      screen.queryByText(translate("channels.telegram.connected")),
    ).toBeNull()
  })

  it("case 3e: another poller shows the bot-in-use guidance", async () => {
    installHost({
      onboardingConfigured: true,
      telegramBotUsername: "pocketclaw_ab12cd34_bot",
    })
    fetchTelegramReadiness.mockResolvedValue({
      state: "telegram_conflict",
      ready: false,
      detail: "bot_in_use",
    })
    arrange({ configuredSecrets: ["token"], config: { enabled: true, allow_from: ["1"] } })

    await renderTelegramPage()

    expect(
      await screen.findByText(translate("channels.telegram.conflictTitle")),
    ).toBeDefined()
    expect(
      screen.getByText(translate("channels.telegram.conflictInUse")),
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
