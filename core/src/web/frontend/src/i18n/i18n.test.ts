import { describe, expect, it, beforeEach } from "vitest"

import i18n, { SUPPORTED_LANGUAGES, applyDocumentDirection } from "./index"

/// The locales the PocketClaw app can be set to. The dashboard must resolve
/// every one of them rather than silently falling back to English.
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

describe("dashboard i18n", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en")
  })

  it("declares every app locale as supported", () => {
    for (const locale of APP_LOCALES) {
      expect(SUPPORTED_LANGUAGES).toContain(locale)
    }
  })

  it("keeps the locales the dashboard already shipped", () => {
    for (const locale of ["bn-IN", "cs", "pt-BR", "zh", "en"]) {
      expect(SUPPORTED_LANGUAGES).toContain(locale)
    }
  })

  it("resolves every app locale to a real resource", async () => {
    for (const locale of APP_LOCALES) {
      await i18n.changeLanguage(locale)
      expect(
        i18n.resolvedLanguage,
        `${locale} did not resolve to a bundled resource`,
      ).toBeTruthy()
      // navigation.models exists in every bundle; a missing namespace would
      // return the key itself.
      expect(i18n.t("navigation.models")).not.toBe("navigation.models")
    }
  })

  // "pt" is what the app sends. It must reach the existing Brazilian
  // Portuguese translations rather than English.
  it("maps pt onto the existing pt-BR translations", async () => {
    await i18n.changeLanguage("pt-BR")
    const brazilian = i18n.t("navigation.models")

    await i18n.changeLanguage("pt")
    expect(i18n.t("navigation.models")).toBe(brazilian)
  })

  it("reuses the existing zh resource", async () => {
    await i18n.changeLanguage("zh")
    expect(i18n.t("navigation.models")).toBe("模型")
  })

  it("renders translated navigation in every app locale", async () => {
    const expected: Record<string, string> = {
      ar: "الموديلات",
      de: "Modelle",
      en: "Models",
      es: "Modelos",
      fr: "Modèles",
      hi: "मॉडल",
      id: "Model",
      ja: "モデル",
      ko: "모델",
      ru: "Модели",
      zh: "模型",
    }
    for (const [locale, text] of Object.entries(expected)) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("navigation.models"), `${locale}`).toBe(text)
    }
  })

  it("translates shared buttons rather than leaving them English", async () => {
    const save: Record<string, string> = {
      ar: "حفظ",
      de: "Speichern",
      es: "Guardar",
      fr: "Enregistrer",
      hi: "सहेजें",
      id: "Simpan",
      ja: "保存",
      ko: "저장",
      ru: "Сохранить",
    }
    for (const [locale, text] of Object.entries(save)) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("common.save"), `${locale}`).toBe(text)
    }
  })

  it("reports Arabic as right-to-left and the others as left-to-right", () => {
    expect(i18n.dir("ar")).toBe("rtl")
    for (const locale of APP_LOCALES.filter((l) => l !== "ar")) {
      expect(i18n.dir(locale), `${locale}`).toBe("ltr")
    }
  })

  it("sets the document language and direction, and restores it", async () => {
    await i18n.changeLanguage("ar")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expect(document.documentElement.getAttribute("lang")).toBe("ar")

    await i18n.changeLanguage("de")
    expect(document.documentElement.getAttribute("dir")).toBe("ltr")
    expect(document.documentElement.getAttribute("lang")).toBe("de")
  })

  it("applies direction directly for a locale it is handed", () => {
    applyDocumentDirection("ar")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    applyDocumentDirection("ja")
    expect(document.documentElement.getAttribute("dir")).toBe("ltr")
  })

  // The locale must survive route changes and reloads. i18next persists the
  // detected language, so a later navigation reads it back instead of
  // re-detecting from scratch and landing on English.
  it("persists the chosen language for later navigations", async () => {
    await i18n.changeLanguage("ja")
    expect(localStorage.getItem("i18nextLng")).toBe("ja")
    expect(i18n.t("navigation.models")).toBe("モデル")
  })
})

describe("translation coverage", () => {
  // What this milestone actually delivers: the shared chrome every route shows.
  // The page bodies are not translated yet and fall back to English per key,
  // which is why this asserts the chrome rather than the whole bundle.
  const CHROME = ["common", "navigation", "header", "footer", "labels"] as const

  const bundles: Record<string, unknown> = {
    ar: enBundleFor("ar"),
    de: enBundleFor("de"),
    es: enBundleFor("es"),
    fr: enBundleFor("fr"),
    hi: enBundleFor("hi"),
    id: enBundleFor("id"),
    ja: enBundleFor("ja"),
    ko: enBundleFor("ko"),
    ru: enBundleFor("ru"),
  }

  function enBundleFor(lng: string) {
    return i18n.getResourceBundle(lng, "translation") as Record<string, unknown>
  }

  function leafCount(value: unknown): number {
    if (value && typeof value === "object") {
      return Object.values(value as Record<string, unknown>).reduce<number>(
        (total, child) => total + leafCount(child),
        0,
      )
    }
    return 1
  }

  it("translates the whole shared chrome in every added locale", () => {
    const english = enBundleFor("en")
    for (const [locale, bundle] of Object.entries(bundles)) {
      for (const namespace of CHROME) {
        const theirs = (bundle as Record<string, unknown>)[namespace]
        expect(theirs, `${locale} is missing ${namespace}`).toBeTruthy()
        expect(
          leafCount(theirs),
          `${locale}.${namespace} has fewer strings than English`,
        ).toBe(leafCount((english as Record<string, unknown>)[namespace]))
      }
    }
  })

  it("does not ship English copies as if they were translations", () => {
    const english = enBundleFor("en") as Record<string, Record<string, string>>
    for (const [locale, bundle] of Object.entries(bundles)) {
      const common = (bundle as Record<string, Record<string, string>>).common
      expect(common.save, `${locale} copied the English "Save"`).not.toBe(
        english.common.save,
      )
    }
  })
})
