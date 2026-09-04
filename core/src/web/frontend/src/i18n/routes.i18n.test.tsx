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

/**
 * pt-BR and zh shipped values byte-identical to English long after the other
 * locales were finished — the whole Fallback Models block, the gateway-restart
 * lifecycle, the provider picker, several channel field labels. The bundle gate
 * in i18n.test.ts now forbids that. These prove the translations actually reach
 * the screen on the surfaces those strings belong to.
 */
describe("pt-BR and zh rendered quality", () => {
  const CLEANED = ["pt", "zh"] as const

  it.each(CLEANED)(
    "renders translated Fallback Models on /models in %s",
    async (locale) => {
      await i18n.changeLanguage(locale)
      const { FallbackModelsSection } =
        await import("@/components/models/fallback-models-section")

      render(
        <FallbackModelsSection
          models={[]}
          fallbacks={[]}
          defaultModelName=""
          onSaved={vi.fn()}
        />,
      )

      expectTranslated("models.fallbacks.title", locale)
      expectTranslated("models.fallbacks.description", locale)
      expectTranslated("models.fallbacks.empty", locale)
    },
  )

  it("renders the Brazilian Portuguese wording, not European or English", async () => {
    await i18n.changeLanguage("pt")
    expect(i18n.t("models.fallbacks.title")).toBe("Modelos de Fallback")
    expect(i18n.t("models.picker.addProvider")).toBe("Adicionar Provedor")
    expect(i18n.t("models.field.provider")).toBe("Provedor")
    expect(i18n.t("credentials.labels.email")).toBe("E-mail")
    expect(i18n.t("pages.config.evolution_mode_draft")).toBe("Rascunho")
  })

  it("renders Simplified Chinese consistent with the existing bundle", async () => {
    await i18n.changeLanguage("zh")
    expect(i18n.t("models.fallbacks.title")).toBe("备用模型")
    expect(i18n.t("models.field.provider")).toBe("服务商")
    expect(i18n.t("channels.field.token")).toBe("机器人 Token")
    // The gateway is 服务 throughout this bundle, never 网关服务器.
    expect(i18n.t("common.restartFailedTitle")).toBe("服务重启失败")
  })

  // Telegram's manual surface is where the channel field labels render, and zh
  // had left Bot Token and API Base URL in English.
  it.each(CLEANED)(
    "renders translated Telegram field labels in %s",
    async (locale) => {
      await i18n.changeLanguage(locale)
      getChannelsCatalog.mockResolvedValue({
        channels: [
          {
            name: "telegram",
            display_name: "Telegram",
            config_key: "telegram",
          },
        ],
      })
      getChannelConfig.mockResolvedValue({
        config: {},
        configured_secrets: [],
        config_key: "telegram",
      })
      window.__pocketclawHost = {
        platform: "android",
        onboardingConfigured: false,
        telegramBotUsername: null,
        openTelegramOnboarding: vi.fn(),
        openExternal: vi.fn(),
      }

      render(<ChannelConfigPage channelName="telegram" />)
      await waitFor(() =>
        expect(screen.getByRole("heading", { name: "Telegram" })).toBeDefined(),
      )

      expectTranslated("channels.field.token", locale)
      expectTranslated("channels.field.baseUrl", locale)
    },
  )

  it.each(CLEANED)("renders a translated Chat body in %s", async (locale) => {
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
  })

  // pt-BR had left the Evolution mode options in English. Only the selected one
  // is mounted — the rest live inside a closed Select — so the other two are
  // asserted on the bundle in the per-locale wording tests above.
  it.each(CLEANED)(
    "renders a translated Configuration section in %s",
    async (locale) => {
      await i18n.changeLanguage(locale)
      render(<EvolutionSection form={EMPTY_FORM} onFieldChange={vi.fn()} />)

      expectTranslated("pages.config.evolution_mode", locale)
      expectTranslated("pages.config.evolution_mode_observe", locale)
      expectTranslated("pages.config.evolution_section_hint", locale)
    },
  )

  // The shared chrome: the restart notice every settings page can raise.
  it.each(CLEANED)(
    "renders the shared restart notice in %s",
    async (locale) => {
      await i18n.changeLanguage(locale)
      const { ConfigChangeNotice } =
        await import("@/components/config-change-notice")

      render(
        <ConfigChangeNotice
          kind="restart"
          title={i18n.t("common.restartFailedTitle")}
          description={i18n.t("common.restartFailedDesc", { name: "Telegram" })}
        />,
      )

      expectTranslated("common.restartFailedTitle", locale)
      // The interpolated name must survive translation.
      expect(
        screen.getByText(new RegExp("Telegram")),
        `${locale} dropped the interpolated name`,
      ).toBeTruthy()
    },
  )
})
