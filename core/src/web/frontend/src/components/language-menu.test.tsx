/**
 * The manual language override, through the component the console renders.
 *
 * The menu used to list two languages while the app itself could be set to
 * twelve, so a user whose phone was in Japanese had no way to put the embedded
 * console back into Japanese after touching the override. These tests hold the
 * menu to the app's own locale set, and hold the selection to i18next — there
 * is deliberately no second language state to go out of sync with it.
 */
import { fireEvent, render, screen } from "@testing-library/react"
import { useTranslation } from "react-i18next"
import { beforeEach, describe, expect, it } from "vitest"

import i18n from "@/i18n"
import {
  APP_LANGUAGES,
  DASHBOARD_ONLY_LANGUAGES,
  LANGUAGE_OPTIONS,
  selectedLanguageCode,
} from "@/i18n/languages"

import { LanguageMenu } from "./language-menu"

/// The locales the PocketClaw app can be set to. All twelve are mandatory.
const APP_LOCALES = [
  "ar",
  "de",
  "en",
  "es",
  "fr",
  "hi",
  "id",
  "ja",
  "ko",
  "pt",
  "ru",
  "zh",
] as const

function openMenu() {
  return render(<LanguageMenu defaultOpen />)
}

function item(name: string) {
  return screen.getByRole("menuitemradio", { name })
}

/// Picking an entry closes the menu, so each selection opens its own.
function selectLanguage(endonym: string) {
  const view = render(<LanguageMenu defaultOpen />)
  fireEvent.click(item(endonym))
  view.unmount()
}

describe("language menu contents", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en")
  })

  it("offers every locale the app can be set to", () => {
    for (const locale of APP_LOCALES) {
      expect(
        APP_LANGUAGES.map((option) => option.code),
        `${locale} is not offered by the manual override`,
      ).toContain(locale)
    }
  })

  it("keeps the dashboard-only locales the console already shipped", () => {
    const codes = LANGUAGE_OPTIONS.map((option) => option.code)
    expect(codes).toContain("bn-IN")
    expect(codes).toContain("cs")
    expect(DASHBOARD_ONLY_LANGUAGES).toHaveLength(2)
  })

  it("names each language in its own script", () => {
    openMenu()
    const endonyms: Record<string, string> = {
      ar: "العربية",
      de: "Deutsch",
      en: "English",
      es: "Español",
      fr: "Français",
      hi: "हिन्दी",
      id: "Bahasa Indonesia",
      ja: "日本語",
      ko: "한국어",
      pt: "Português (Brasil)",
      ru: "Русский",
      zh: "简体中文",
      "bn-IN": "বাংলা",
      cs: "Čeština",
    }
    for (const [code, endonym] of Object.entries(endonyms)) {
      const option = LANGUAGE_OPTIONS.find((o) => o.code === code)
      expect(option?.endonym, code).toBe(endonym)
      expect(item(endonym), code).toBeTruthy()
    }
  })

  // An endonym is the same string in every language, so it must not be routed
  // through the resource bundles — a "translated" endonym would be wrong.
  it("does not translate the entries with the console's language", async () => {
    openMenu()
    expect(item("日本語")).toBeTruthy()
    await i18n.changeLanguage("ar")
    expect(item("日本語")).toBeTruthy()
    expect(item("English")).toBeTruthy()
  })
})

describe("language menu selection", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en")
  })

  it("shows the current language as the checked entry", async () => {
    await i18n.changeLanguage("de")
    openMenu()
    expect(item("Deutsch").getAttribute("aria-checked")).toBe("true")
    expect(item("English").getAttribute("aria-checked")).toBe("false")
  })

  // The host hands over regional tags the menu does not list verbatim. They
  // still belong to a listed language and must not read as "English".
  it("maps regional tags onto the language they belong to", () => {
    expect(selectedLanguageCode("pt-BR")).toBe("pt")
    expect(selectedLanguageCode("zh-CN")).toBe("zh")
    expect(selectedLanguageCode("bn-IN")).toBe("bn-IN")
    expect(selectedLanguageCode(undefined)).toBe("en")
  })

  it.each([
    ["ar", "العربية", "الموديلات", "rtl"],
    ["de", "Deutsch", "Modelle", "ltr"],
    ["ja", "日本語", "モデル", "ltr"],
    ["pt", "Português (Brasil)", "Modelos", "ltr"],
    ["zh", "简体中文", "模型", "ltr"],
  ])(
    "switches the console to %s and applies its direction",
    async (code, endonym, translatedNav, dir) => {
      selectLanguage(endonym)

      expect(i18n.language).toBe(code)
      expect(i18n.t("navigation.models")).toBe(translatedNav)
      expect(document.documentElement.getAttribute("lang")).toBe(code)
      expect(document.documentElement.getAttribute("dir")).toBe(dir)
      expect(i18n.dir()).toBe(dir)
    },
  )

  it("restores left-to-right after leaving Arabic", () => {
    selectLanguage("العربية")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")

    selectLanguage("Deutsch")
    expect(document.documentElement.getAttribute("dir")).toBe("ltr")
    expect(i18n.t("navigation.models")).toBe("Modelle")
  })

  // Selecting through the menu is what persists the override; nothing else in
  // the console writes a language anywhere.
  it("persists the manual choice through i18next's own storage", () => {
    selectLanguage("한국어")
    expect(localStorage.getItem("i18nextLng")).toBe("ko")
  })

  it("re-renders the surrounding UI immediately", () => {
    function Probe() {
      const { t } = useTranslation()
      return <p>{t("common.language")}</p>
    }

    render(
      <>
        <LanguageMenu defaultOpen />
        <Probe />
      </>,
    )
    expect(screen.getByText("Language")).toBeTruthy()

    fireEvent.click(item("Français"))
    expect(screen.getByText("Langue")).toBeTruthy()
  })
})
