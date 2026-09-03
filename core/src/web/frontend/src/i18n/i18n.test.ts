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
  const CHROME = [
    "common",
    "navigation",
    "header",
    "footer",
    "labels",
    // Manage Models and Manage Telegram are first-class entry points from
    // native Settings, so these namespaces are required in full rather than
    // falling back to English.
    "models",
    "channels",
  ] as const

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

  it("translates the whole shared chrome and Models in every added locale", () => {
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

  // Proper nouns that are identical in every language. Kept deliberately tiny:
  // a broad whitelist would hide real untranslated work.
  const PROPER_NOUNS = new Set([
    "channels.name.telegram",
    "channels.name.discord",
    "channels.name.slack",
    "channels.name.feishu",
    "channels.name.dingtalk",
    "channels.name.line",
    "channels.name.qq",
    "channels.name.onebot",
    "channels.name.wecom",
    "channels.name.maixcam",
    "channels.name.matrix",
    "channels.name.irc",
    "channels.name.weixin",
    "channels.name.mqtt",
    // "text" is the wire field name in the MQTT payload, not prose.
    "channels.mqtt.fieldText",
  ])

  // Latin-script languages legitimately share loanwords with English — German
  // "Chat", "Agent" and "Version", French "Services" and "Documentation",
  // Spanish "Hub" are correct translations that happen to be identical. An
  // equality check cannot tell those from untranslated strings, so the
  // English-copy rule is applied where it is unambiguous: in these scripts any
  // identical Latin string is genuinely untranslated.
  const NON_LATIN = ["ar", "hi", "ja", "ko", "ru"] as const

  function flatten(
    value: unknown,
    prefix: string,
    out: Map<string, string>,
  ): Map<string, string> {
    if (value && typeof value === "object") {
      for (const [key, child] of Object.entries(
        value as Record<string, unknown>,
      )) {
        flatten(child, prefix ? `${prefix}.${key}` : key, out)
      }
    } else if (typeof value === "string") {
      out.set(prefix, value)
    }
    return out
  }

  function keysOfNamespace(english: Map<string, string>, namespace: string) {
    return [...english.keys()].filter((key) => key.startsWith(`${namespace}.`))
  }

  it("has every required key in every locale", () => {
    const english = flatten(enBundleFor("en"), "", new Map())
    const failures: string[] = []

    for (const [locale, bundle] of Object.entries(bundles)) {
      const theirs = flatten(bundle, "", new Map())
      for (const namespace of CHROME) {
        for (const key of keysOfNamespace(english, namespace)) {
          if (theirs.get(key) === undefined) {
            failures.push(`${locale} | ${namespace} | ${key} | MISSING`)
          }
        }
      }
    }

    expect(failures.join("\n"), failures.join("\n")).toBe("")
  })

  it("leaves nothing in English in the non-Latin locales", () => {
    const english = flatten(enBundleFor("en"), "", new Map())
    const failures: string[] = []

    for (const locale of NON_LATIN) {
      const theirs = flatten(enBundleFor(locale), "", new Map())
      for (const namespace of CHROME) {
        for (const key of keysOfNamespace(english, namespace)) {
          if (PROPER_NOUNS.has(key)) continue
          const value = theirs.get(key)
          if (value !== undefined && value === english.get(key)) {
            failures.push(
              `${locale} | ${namespace} | ${key} | ENGLISH COPY: "${value}"`,
            )
          }
        }
      }
    }

    expect(failures.join("\n"), failures.join("\n")).toBe("")
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

describe("Models route body", () => {
  it("renders translated Models page content in every app locale", async () => {
    const expected: Record<string, string> = {
      ar: "إضافة موديل",
      de: "Modell hinzufügen",
      es: "Añadir modelo",
      fr: "Ajouter un modèle",
      hi: "मॉडल जोड़ें",
      id: "Tambah Model",
      ja: "モデルを追加",
      ko: "모델 추가",
      ru: "Добавить модель",
    }
    for (const [locale, text] of Object.entries(expected)) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("models.add.button"), locale).toBe(text)
      // Body prose, not just a button.
      expect(i18n.t("models.description"), locale).not.toBe(
        "Configure API keys for AI providers. Only configured models are available for chat.",
      )
      expect(i18n.t("models.field.apiKey"), locale).not.toBe("API Key")
    }
  })
})

describe("Channels / Telegram route body", () => {
  // Manage Telegram opens /channels/telegram, so its body must be translated,
  // not merely reachable.
  it("renders translated Telegram page content in every app locale", async () => {
    const connectTitle: Record<string, string> = {
      ar: "اربط PocketClaw بتيليجرام",
      de: "PocketClaw mit Telegram verbinden",
      es: "Conecta PocketClaw a Telegram",
      fr: "Connecter PocketClaw à Telegram",
      hi: "PocketClaw को Telegram से जोड़ें",
      id: "Hubungkan PocketClaw ke Telegram",
      ja: "PocketClaw を Telegram に接続",
      ko: "PocketClaw를 Telegram에 연결",
      ru: "Подключить PocketClaw к Telegram",
    }
    for (const [locale, text] of Object.entries(connectTitle)) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("channels.telegram.connectTitle"), locale).toBe(text)

      // Body prose and form text, not just the headline.
      expect(i18n.t("channels.telegram.connectBody"), locale).not.toBe(
        "Create your personal PocketClaw bot in a few seconds. No BotFather token copy/paste required.",
      )
      expect(i18n.t("channels.page.enableLabel"), locale).not.toBe(
        "Enable channel",
      )
      expect(i18n.t("channels.form.desc.allowFrom"), locale).not.toBe(
        "Allowed user or group IDs. Add items one by one, or paste multiple values at once.",
      )
      expect(i18n.t("channels.validation.requiredField"), locale).not.toBe(
        "This field is required.",
      )
    }
  })

  it("keeps English and the pre-existing locales intact", async () => {
    await i18n.changeLanguage("en")
    expect(i18n.t("channels.telegram.connected")).toBe("Connected")

    for (const locale of ["pt", "zh"]) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("channels.telegram.connectTitle"), locale).toBeTruthy()
      expect(i18n.t("channels.telegram.connectTitle"), locale).not.toBe(
        "channels.telegram.connectTitle",
      )
    }
  })

  it("keeps Arabic right-to-left on the Telegram route", async () => {
    await i18n.changeLanguage("ar")
    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expect(i18n.t("channels.telegram.ownerLabel")).toBe("المالك")
  })

  it("preserves interpolation placeholders", async () => {
    for (const locale of ["ar", "de", "es", "fr", "hi", "id", "ja", "ko", "ru"]) {
      await i18n.changeLanguage(locale)
      expect(
        i18n.t("channels.page.notFound", { name: "telegram" }),
        locale,
      ).toContain("telegram")
      expect(
        i18n.t("channels.field.removeListItem", { value: "123456" }),
        locale,
      ).toContain("123456")
    }
  })

  it("does not translate platform names", async () => {
    for (const locale of ["ar", "hi", "ja", "ko", "ru"]) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("channels.name.telegram"), locale).toBe("Telegram")
      expect(i18n.t("channels.name.matrix"), locale).toBe("Matrix")
    }
  })
})
