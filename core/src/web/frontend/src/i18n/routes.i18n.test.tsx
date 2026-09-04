/**
 * The routes a phone actually reaches, rendered through the real i18next.
 *
 * The bundle-level gates in `i18n.test.ts` prove every string is translated.
 * They cannot prove a component reads the key it should, so a page can pass
 * them and still show English on the device. These mount the components the
 * embedded console renders — Models, Channels → Telegram, Chat and the
 * Configuration console — in five representative locales and read the rendered
 * text back out.
 *
 * Nothing here mocks react-i18next: that is the whole point.
 */
import { render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { EMPTY_FORM } from "@/components/config/form-model"
import i18n from "@/i18n"

/// Arabic proves right-to-left; German, Japanese and Chinese cover the Latin,
/// CJK and existing-bundle cases; "pt" proves the app's tag reaches pt-BR.
const LOCALES = ["ar", "de", "ja", "pt", "zh"] as const

const getModels = vi.fn()
const getChannelsCatalog = vi.fn()
const getChannelConfig = vi.fn()

vi.mock("@/api/models", () => ({
  getModels: () => getModels(),
  setDefaultModel: vi.fn(),
  getCatalogs: vi.fn(),
  deleteCatalog: vi.fn(),
  addModel: vi.fn(),
}))

vi.mock("@/api/channels", () => ({
  getChannelsCatalog: () => getChannelsCatalog(),
  getChannelConfig: (name: string) => getChannelConfig(name),
  patchAppConfig: vi.fn(),
}))

vi.mock("@/hooks/use-gateway", () => ({
  useGateway: () => ({ state: "running" }),
}))

vi.mock("@/store/gateway", () => ({ refreshGatewayState: vi.fn() }))

vi.mock("@/lib/restart-required", () => ({
  showSaveSuccessOrRestartToast: vi.fn(),
  saveAndApplyGatewayConfig: vi.fn(),
}))

// The real header needs sidebar context these pages do not otherwise use.
vi.mock("@/components/page-header", () => ({
  PageHeader: ({
    title,
    children,
  }: {
    title: string
    children?: React.ReactNode
  }) => (
    <div>
      <h1>{title}</h1>
      {children}
    </div>
  ),
}))

// Chat's empty state links to /models; the router is not what is under test.
vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: React.ReactNode }) => <a>{children}</a>,
}))

const { ModelsPage } = await import("@/components/models/models-page")
const { ChannelConfigPage } =
  await import("@/components/channels/channel-config-page")
const { ChatEmptyState } = await import("@/components/chat/chat-empty-state")
const { EvolutionSection } = await import("@/components/config/config-sections")

/// The literal English the bundle ships for a key. Seeing it on screen in
/// another locale is the failure this file exists to catch.
function english(key: string): string {
  const value = i18n.getFixedT("en")(key)
  return typeof value === "string" ? value : key
}

function translated(key: string): string {
  const value = i18n.t(key)
  return typeof value === "string" ? value : key
}

/// Asserts the rendered page shows the locale's own text for `key`, and that
/// the English wording is nowhere on the page.
function expectTranslated(key: string, locale: string) {
  const localized = translated(key)
  expect(localized, `${locale} | ${key} is untranslated`).not.toBe(english(key))
  expect(
    screen.getAllByText(localized).length,
    `${locale} | ${key} is not rendered`,
  ).toBeGreaterThan(0)
  expect(
    screen.queryByText(english(key)),
    `${locale} | ${key} rendered in English`,
  ).toBeNull()
}

beforeEach(async () => {
  vi.clearAllMocks()
  delete window.__pocketclawHost
  await i18n.changeLanguage("en")
})

describe("/models route body", () => {
  it.each(LOCALES)("renders the Models page in %s", async (locale) => {
    await i18n.changeLanguage(locale)
    getModels.mockResolvedValue({ models: [], fallbacks: [], providers: [] })

    render(<ModelsPage />)

    await waitFor(() =>
      expect(screen.getByText(translated("navigation.models"))).toBeDefined(),
    )
    expectTranslated("navigation.models", locale)
    expectTranslated("models.description", locale)
    expectTranslated("models.catalog.button", locale)
  })

  it("keeps Arabic Models right-to-left", async () => {
    await i18n.changeLanguage("ar")
    getModels.mockResolvedValue({ models: [], fallbacks: [], providers: [] })

    render(<ModelsPage />)
    await waitFor(() =>
      expect(screen.getByText(translated("navigation.models"))).toBeDefined(),
    )

    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
  })
})

describe("/channels/telegram route body", () => {
  function arrangeTelegram() {
    getChannelsCatalog.mockResolvedValue({
      channels: [
        { name: "telegram", display_name: "Telegram", config_key: "telegram" },
      ],
    })
    getChannelConfig.mockResolvedValue({
      config: {},
      configured_secrets: [],
      config_key: "telegram",
    })
    window.__pocketclawHost = {
      platform: "android",
      onboardingConfigured: true,
      telegramBotUsername: null,
      openTelegramOnboarding: vi.fn(),
      openExternal: vi.fn(),
    }
  }

  it.each(LOCALES)("renders the Telegram page in %s", async (locale) => {
    await i18n.changeLanguage(locale)
    arrangeTelegram()

    render(<ChannelConfigPage channelName="telegram" />)
    await waitFor(() =>
      expect(screen.getByRole("heading", { name: "Telegram" })).toBeDefined(),
    )

    expectTranslated("channels.telegram.connectTitle", locale)
    expectTranslated("channels.telegram.openTelegram", locale)
    expectTranslated("channels.telegram.statusReady", locale)
  })

  it("keeps Arabic Telegram right-to-left with a translated body", async () => {
    await i18n.changeLanguage("ar")
    arrangeTelegram()

    render(<ChannelConfigPage channelName="telegram" />)
    await waitFor(() =>
      expect(screen.getByRole("heading", { name: "Telegram" })).toBeDefined(),
    )

    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expectTranslated("channels.telegram.connectTitle", "ar")
  })

  // Telegram is the platform's name, not copy to be localized.
  it("leaves the platform name alone in every locale", async () => {
    for (const locale of LOCALES) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("channels.name.telegram"), locale).toBe("Telegram")
    }
  })
})

describe("Chat route body", () => {
  it.each(LOCALES)("renders the Chat body in %s", async (locale) => {
    await i18n.changeLanguage(locale)

    render(
      <ChatEmptyState
        hasAvailableModels={false}
        defaultModelName=""
        isConnected
      />,
    )

    expectTranslated("chat.empty.noConfiguredModel", locale)
    expectTranslated("chat.empty.noConfiguredModelDescription", locale)
    expectTranslated("chat.empty.goToModels", locale)
  })

  it("keeps Arabic Chat right-to-left", async () => {
    await i18n.changeLanguage("ar")
    render(
      <ChatEmptyState
        hasAvailableModels={false}
        defaultModelName=""
        isConnected
      />,
    )
    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expectTranslated("chat.empty.noConfiguredModel", "ar")
  })
})

describe("Config console body", () => {
  it.each(LOCALES)("renders a Configuration section in %s", async (locale) => {
    await i18n.changeLanguage(locale)

    render(<EvolutionSection form={EMPTY_FORM} onFieldChange={vi.fn()} />)

    expectTranslated("pages.config.evolution_section_hint", locale)
    expectTranslated("pages.config.evolution_enabled", locale)
    expectTranslated("pages.config.evolution_enabled_hint", locale)
  })

  it("keeps the Arabic Configuration console right-to-left", async () => {
    await i18n.changeLanguage("ar")
    render(<EvolutionSection form={EMPTY_FORM} onFieldChange={vi.fn()} />)
    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expectTranslated("pages.config.evolution_enabled", "ar")
  })
})
