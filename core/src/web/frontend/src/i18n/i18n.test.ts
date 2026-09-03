import { beforeEach, describe, expect, it } from "vitest"

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

/// The nine locales the dashboard localization milestone adds. Every finished
/// batch must render in all of them.
const BATCH_LOCALES = [
  "ar",
  "de",
  "es",
  "fr",
  "hi",
  "id",
  "ja",
  "ko",
  "ru",
] as const

// `pages` is translated in batches, so only the finished groups are required.
// Listing prefixes rather than the whole namespace keeps the enforcement
// honest: it can never imply that all 316 pages keys are done.

// The Skills page, the Agent page's shared load error, and the Logs page.
const PAGES_BATCH_1 = [
  "pages.agent.skills.",
  "pages.agent.load_error",
  "pages.logs.",
] as const

// The Tools page including its Web Search panel, the Configuration page's
// shared load error and section tabs, and the settings behind the Agent and
// Run Commands tabs: workspace, chatty mode, tool feedback, command
// execution, the pattern detector and the scheduled-command limits.
//
// `pages.config` is one flat group rather than a tree, so its members are
// listed key by key. A `pages.config.` prefix would silently claim batches 3
// and 4 as well.
const PAGES_BATCH_2 = [
  "pages.agent.tools.",
  "pages.config.load_error",
  "pages.config.sections.",
  "pages.config.workspace",
  "pages.config.workspace_hint",
  "pages.config.workspace_required",
  "pages.config.restrict_workspace",
  "pages.config.restrict_workspace_hint",
  "pages.config.split_on_marker",
  "pages.config.split_on_marker_hint",
  "pages.config.tool_feedback_enabled",
  "pages.config.tool_feedback_enabled_hint",
  "pages.config.tool_feedback_separate_messages",
  "pages.config.tool_feedback_separate_messages_hint",
  "pages.config.tool_feedback_max_args_length",
  "pages.config.tool_feedback_max_args_length_hint",
  "pages.config.exec_enabled",
  "pages.config.exec_enabled_hint",
  "pages.config.allow_remote",
  "pages.config.allow_remote_hint",
  "pages.config.enable_deny_patterns",
  "pages.config.enable_deny_patterns_hint",
  "pages.config.exec_timeout_seconds",
  "pages.config.exec_timeout_seconds_hint",
  "pages.config.custom_deny_patterns",
  "pages.config.custom_deny_patterns_hint",
  "pages.config.custom_allow_patterns",
  "pages.config.custom_allow_patterns_hint",
  "pages.config.custom_patterns_placeholder",
  "pages.config.pattern_detector_title",
  "pages.config.pattern_detector_hint",
  "pages.config.pattern_detector_input_placeholder",
  "pages.config.pattern_detector_test_button",
  "pages.config.pattern_detector_result_allowed",
  "pages.config.pattern_detector_result_blocked",
  "pages.config.pattern_detector_result_no_match",
  "pages.config.allow_shell_execution",
  "pages.config.allow_shell_execution_hint",
  "pages.config.cron_exec_timeout",
  "pages.config.cron_exec_timeout_hint",
] as const

// The Configuration page's Agent tuning, session scope, runtime, launcher and
// security settings, plus the shared numeric-field validation messages. Listed
// key by key for the same reason batch 2 is: a `pages.config.` prefix would
// falsely claim batch 4's evolution, MCP and raw-JSON groups as finished.
const PAGES_BATCH_3 = [
  "pages.config.max_tokens",
  "pages.config.max_tokens_hint",
  "pages.config.context_window",
  "pages.config.context_window_hint",
  "pages.config.max_tool_iterations",
  "pages.config.max_tool_iterations_hint",
  "pages.config.summarize_threshold",
  "pages.config.summarize_threshold_hint",
  "pages.config.summarize_token_percent",
  "pages.config.summarize_token_percent_hint",
  "pages.config.turn_profile",
  "pages.config.turn_profile_hint",
  "pages.config.turn_profile_enabled",
  "pages.config.turn_profile_enabled_hint",
  "pages.config.turn_profile_mode_default",
  "pages.config.turn_profile_mode_off",
  "pages.config.turn_profile_mode_custom",
  "pages.config.turn_profile_history",
  "pages.config.turn_profile_history_hint",
  "pages.config.turn_profile_system_prompt",
  "pages.config.turn_profile_system_prompt_hint",
  "pages.config.turn_profile_skills",
  "pages.config.turn_profile_skills_hint",
  "pages.config.turn_profile_skills_allow_placeholder",
  "pages.config.turn_profile_tools",
  "pages.config.turn_profile_tools_hint",
  "pages.config.turn_profile_tools_allow_placeholder",
  "pages.config.session_scope",
  "pages.config.session_scope_hint",
  "pages.config.session_scope_required",
  "pages.config.session_scope_per_channel_peer",
  "pages.config.session_scope_per_channel_peer_desc",
  "pages.config.session_scope_per_channel",
  "pages.config.session_scope_per_channel_desc",
  "pages.config.session_scope_per_peer",
  "pages.config.session_scope_per_peer_desc",
  "pages.config.session_scope_global",
  "pages.config.session_scope_global_desc",
  "pages.config.heartbeat_enabled",
  "pages.config.heartbeat_enabled_hint",
  "pages.config.heartbeat_interval",
  "pages.config.heartbeat_interval_hint",
  "pages.config.devices_enabled",
  "pages.config.devices_enabled_hint",
  "pages.config.monitor_usb",
  "pages.config.monitor_usb_hint",
  "pages.config.autostart_label",
  "pages.config.autostart_hint",
  "pages.config.autostart_unsupported",
  "pages.config.autostart_load_error",
  "pages.config.server_port",
  "pages.config.server_port_hint",
  "pages.config.launcher_section_hint",
  "pages.config.gateway_restart_hint",
  "pages.config.dashboard_password",
  "pages.config.dashboard_password_hint",
  "pages.config.dashboard_password_placeholder",
  "pages.config.dashboard_password_confirm",
  "pages.config.dashboard_password_confirm_hint",
  "pages.config.dashboard_password_confirm_placeholder",
  "pages.config.dashboard_password_required",
  "pages.config.dashboard_password_mismatch",
  "pages.config.dashboard_password_min_length",
  "pages.config.lan_access",
  "pages.config.lan_access_hint",
  "pages.config.allowed_cidrs",
  "pages.config.allowed_cidrs_hint",
  "pages.config.allowed_cidrs_placeholder",
  "pages.config.allow_localhost_bypass",
  "pages.config.allow_localhost_bypass_hint",
  "pages.config.trusted_proxy_cidrs",
  "pages.config.trusted_proxy_cidrs_hint",
  "pages.config.trusted_proxy_cidrs_placeholder",
  "pages.config.validation_integer",
  "pages.config.validation_number",
  "pages.config.validation_min",
  "pages.config.validation_max",
] as const

const PAGES_DONE_PREFIXES = [
  ...PAGES_BATCH_1,
  ...PAGES_BATCH_2,
  ...PAGES_BATCH_3,
] as readonly string[]

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
    "chat",
    "credentials",
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
    // OAuth is a protocol name. Arabic, Hindi, Japanese, Korean and Russian
    // technical interfaces all write it "OAuth"; transliterating it would be
    // less recognisable, not more localized.
    "chat.modelGroup.oauth",
    // "URL" is the standard term in Hindi, Japanese, Korean and Russian
    // technical interfaces; it is a field label beside Name and Description,
    // not prose. Arabic uses "الرابط" and is unaffected.
    "pages.agent.skills.metadata.url",
    // MCP is the protocol's name and is written "MCP" in every one of these
    // languages, the same way the channel platform names above are.
    "pages.config.sections.mcp",
    // Not prose at all: the two sample regular expressions shown greyed out in
    // the custom-pattern textareas. Translating a regex would make it wrong.
    "pages.config.custom_patterns_placeholder",
    // Also literals rather than copy: web_search and web_fetch are the tools'
    // actual names, and the CIDR samples are addresses. All three are shown as
    // the format to type, so translating them would make them wrong.
    "pages.config.turn_profile_tools_allow_placeholder",
    "pages.config.allowed_cidrs_placeholder",
    "pages.config.trusted_proxy_cidrs_placeholder",
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

  function batchedPagesKeys(english: Map<string, string>) {
    return [...english.keys()].filter((key) =>
      PAGES_DONE_PREFIXES.some(
        (prefix) => key === prefix || key.startsWith(prefix),
      ),
    )
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
      for (const key of batchedPagesKeys(english)) {
        if (theirs.get(key) === undefined) {
          failures.push(`${locale} | pages | ${key} | MISSING`)
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
      const checked: Array<[string, string[]]> = [
        ...CHROME.map(
          (namespace) =>
            [namespace, keysOfNamespace(english, namespace)] as [
              string,
              string[],
            ],
        ),
        ["pages", batchedPagesKeys(english)],
      ]
      for (const [namespace, keys] of checked) {
        for (const key of keys) {
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
    for (const locale of [
      "ar",
      "de",
      "es",
      "fr",
      "hi",
      "id",
      "ja",
      "ko",
      "ru",
    ]) {
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

describe("Chat route body", () => {
  const CHAT_LOCALES = [
    "ar",
    "de",
    "es",
    "fr",
    "hi",
    "id",
    "ja",
    "ko",
    "ru",
  ] as const

  // Body prose, an action control and a status/empty state — not nav.chat.
  it("renders translated Chat body, actions and states in every locale", async () => {
    const welcome: Record<string, string> = {
      ar: "كيف يمكنني مساعدتك اليوم؟",
      de: "Wie kann ich Ihnen heute helfen?",
      es: "¿En qué puedo ayudarte hoy?",
      fr: "Comment puis-je vous aider aujourd'hui ?",
      hi: "आज मैं आपकी क्या मदद कर सकता हूँ?",
      id: "Ada yang bisa saya bantu hari ini?",
      ja: "今日はどのようなご用件でしょうか？",
      ko: "오늘 무엇을 도와드릴까요?",
      ru: "Чем я могу помочь сегодня?",
    }

    for (const locale of CHAT_LOCALES) {
      await i18n.changeLanguage(locale)

      // Page/body prose.
      expect(i18n.t("chat.welcome"), locale).toBe(welcome[locale])
      expect(i18n.t("chat.welcomeDesc"), locale).not.toBe(
        "Ask me about weather, settings, or any other tasks. I'm here to assist you.",
      )

      // Input and action controls.
      expect(i18n.t("chat.placeholder"), locale).not.toBe(
        "Start a new message...",
      )
      expect(i18n.t("chat.sendMessage"), locale).not.toBe("Send message")
      expect(i18n.t("chat.newChat"), locale).not.toBe("New Chat")

      // Status, error and empty states.
      expect(i18n.t("chat.thinking.step1"), locale).not.toBe("Thinking...")
      expect(i18n.t("chat.historyLoadFailed"), locale).not.toBe(
        "Failed to load chat history",
      )
      expect(i18n.t("chat.noHistory"), locale).not.toBe("No chat history yet")
      expect(i18n.t("chat.empty.notRunning"), locale).not.toBe(
        "Gateway Not Running",
      )
      expect(
        i18n.t("chat.disabledPlaceholder.gatewayStopped"),
        locale,
      ).not.toBe(
        "Unable to chat: Gateway is not started. Click Start Gateway in the top bar, then retry.",
      )
    }
  })

  it("keeps Arabic Chat right-to-left with a translated body", async () => {
    await i18n.changeLanguage("ar")
    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expect(i18n.t("chat.newChat")).toBe("دردشة جديدة")
    expect(i18n.t("chat.empty.noSelectedModel")).toBe("لم يُحدَّد موديل")
  })

  it("preserves Chat interpolation placeholders", async () => {
    for (const locale of CHAT_LOCALES) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("chat.messagesCount", { count: 12 }), locale).toContain(
        "12",
      )
      expect(
        i18n.t("chat.invalidImage", { name: "photo.heic" }),
        locale,
      ).toContain("photo.heic")
      const tooLarge = i18n.t("chat.imageTooLarge", {
        name: "photo.png",
        size: "5 MB",
      })
      expect(tooLarge, locale).toContain("photo.png")
      expect(tooLarge, locale).toContain("5 MB")
    }
  })

  it("keeps English and the pre-existing locales intact for Chat", async () => {
    await i18n.changeLanguage("en")
    expect(i18n.t("chat.newChat")).toBe("New Chat")

    for (const locale of ["pt", "zh"]) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("chat.newChat"), locale).not.toBe("chat.newChat")
    }
  })
})

describe("Credentials route body", () => {
  const CRED_LOCALES = [
    "ar",
    "de",
    "es",
    "fr",
    "hi",
    "id",
    "ja",
    "ko",
    "ru",
  ] as const

  it("renders translated Credentials body, fields, actions and states", async () => {
    for (const locale of CRED_LOCALES) {
      await i18n.changeLanguage(locale)

      // Title / description prose.
      expect(i18n.t("credentials.description"), locale).not.toBe(
        "Manage OAuth and token-based credentials for supported providers.",
      )
      expect(
        i18n.t("credentials.providers.anthropic.description"),
        locale,
      ).not.toBe("Uses token login for Claude access.")

      // Field and helper strings.
      expect(i18n.t("credentials.labels.account"), locale).not.toBe("Account")
      expect(i18n.t("credentials.device.description"), locale).not.toBe(
        "Open the verification page and enter the code below. This page will refresh automatically.",
      )

      // Action buttons.
      expect(i18n.t("credentials.actions.saveToken"), locale).not.toBe("Save")
      expect(i18n.t("credentials.actions.logout"), locale).not.toBe("Logout")

      // Validation / error / status.
      expect(i18n.t("credentials.errors.loginFailed"), locale).not.toBe(
        "Login failed",
      )
      expect(i18n.t("credentials.errors.popupBlocked"), locale).not.toBe(
        "Unable to open a new tab. Please allow popups and try again.",
      )
      expect(i18n.t("credentials.status.notLoggedIn"), locale).not.toBe(
        "Not logged in",
      )
      expect(i18n.t("credentials.flow.pending"), locale).not.toBe(
        "Waiting for authorization...",
      )
    }
  })

  it("keeps Arabic Credentials right-to-left with a translated body", async () => {
    await i18n.changeLanguage("ar")
    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expect(i18n.t("credentials.status.connected")).toBe("متصل")
    expect(i18n.t("credentials.labels.email")).toBe("البريد الإلكتروني")
  })

  it("preserves the Credentials interpolation placeholder", async () => {
    for (const locale of CRED_LOCALES) {
      await i18n.changeLanguage(locale)
      expect(
        i18n.t("credentials.logoutDialog.description", { provider: "OpenAI" }),
        locale,
      ).toContain("OpenAI")
    }
  })

  it("keeps provider and protocol names untranslated", async () => {
    for (const locale of ["ar", "hi", "ja", "ko", "ru"]) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("credentials.fields.openaiToken"), locale).toContain(
        "OpenAI",
      )
      expect(i18n.t("credentials.fields.anthropicToken"), locale).toContain(
        "Anthropic",
      )
      expect(
        i18n.t("credentials.providers.anthropic.description"),
        locale,
      ).toContain("Claude")
    }
  })

  it("keeps English and the pre-existing locales intact for Credentials", async () => {
    await i18n.changeLanguage("en")
    expect(i18n.t("credentials.status.connected")).toBe("Connected")

    for (const locale of ["pt", "zh"]) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("credentials.status.connected"), locale).not.toBe(
        "credentials.status.connected",
      )
    }
  })
})

describe("Pages batch 1 — Skills and Logs", () => {
  // Interpolation parity: every {{var}} English uses must survive translation,
  // and no translation may invent one English does not have.
  it("preserves placeholder parity with English for every batch 1 key", async () => {
    const placeholders = (value: string) =>
      [...value.matchAll(/\{\{\s*([A-Za-z0-9_]+)\s*\}\}/g)]
        .map((match) => match[1])
        .sort()

    const english = i18n.getResourceBundle("en", "translation") as Record<
      string,
      unknown
    >
    const failures: string[] = []

    const walk = (
      value: unknown,
      path: string,
      visit: (key: string, text: string) => void,
    ) => {
      if (value && typeof value === "object") {
        for (const [key, child] of Object.entries(
          value as Record<string, unknown>,
        )) {
          walk(child, path ? `${path}.${key}` : key, visit)
        }
      } else if (typeof value === "string") {
        visit(path, value)
      }
    }

    for (const locale of BATCH_LOCALES) {
      const bundle = i18n.getResourceBundle(locale, "translation") as Record<
        string,
        unknown
      >
      for (const group of [
        ["pages.agent.skills", (english.pages as never)["agent"]["skills"]],
        ["pages.logs", (english.pages as never)["logs"]],
      ] as Array<[string, unknown]>) {
        walk(group[1], "", (key, englishText) => {
          const path = `${group[0]}.${key}`
          const translated = i18n.getResource(locale, "translation", path) as
            | string
            | undefined
          if (typeof translated !== "string") return
          const want = placeholders(englishText).join(",")
          const got = placeholders(translated).join(",")
          if (want !== got) {
            failures.push(
              `${locale} | pages | ${path} | placeholders "${want}" vs "${got}"`,
            )
          }
        })
      }
      void bundle
    }

    expect(failures.join("\n"), failures.join("\n")).toBe("")
  })

  it("renders translated Skills page body, actions and states", async () => {
    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)

      // Title / description prose.
      expect(i18n.t("pages.agent.skills.marketplace_title"), locale).not.toBe(
        "Discover Skills",
      )
      expect(
        i18n.t("pages.agent.skills.marketplace_description"),
        locale,
      ).not.toBe(
        "Search the skill registries and install useful skills into this workspace",
      )

      // Form / helper content.
      expect(i18n.t("pages.agent.skills.search_placeholder"), locale).not.toBe(
        "Search by name, description, or registry",
      )
      expect(i18n.t("pages.agent.skills.import_constraints"), locale).not.toBe(
        "Import a Markdown or ZIP skill file up to 1 MB",
      )

      // Actions.
      expect(i18n.t("pages.agent.skills.import"), locale).not.toBe(
        "Import Skill",
      )
      expect(
        i18n.t("pages.agent.skills.marketplace_install_action"),
        locale,
      ).not.toBe("Install")

      // Status / error / empty states.
      expect(i18n.t("pages.agent.skills.empty"), locale).not.toBe(
        "No skills are currently available.",
      )
      expect(i18n.t("pages.agent.skills.no_results"), locale).not.toBe(
        "No skills matched the current filters.",
      )
      expect(i18n.t("pages.agent.skills.install_error"), locale).not.toBe(
        "Failed to install skill.",
      )
      expect(i18n.t("pages.agent.load_error"), locale).not.toBe(
        "Failed to load agent support information.",
      )
    }
  })

  it("renders the translated Logs page", async () => {
    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("pages.logs.clear"), locale).not.toBe("Clear logs")
      expect(i18n.t("pages.logs.empty"), locale).not.toBe("Waiting for logs...")
      expect(i18n.t("pages.logs.log_level_error"), locale).not.toBe(
        "Failed to update log level.",
      )
    }
  })

  it("keeps Arabic batch 1 pages right-to-left and translated", async () => {
    await i18n.changeLanguage("ar")
    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expect(i18n.t("pages.agent.skills.import")).toBe("استيراد مهارة")
    expect(i18n.t("pages.logs.clear")).toBe("مسح السجلات")
  })

  it("keeps English and the pre-existing locales intact for batch 1", async () => {
    await i18n.changeLanguage("en")
    expect(i18n.t("pages.logs.clear")).toBe("Clear logs")

    for (const locale of ["pt", "zh"]) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("pages.logs.clear"), locale).not.toBe("pages.logs.clear")
    }
  })
})

describe("Pages batch 2 — Tools and Configuration", () => {
  // Interpolation parity, applied to the batch 2 keys the same way batch 1
  // applies it to its own: no translation may drop a {{var}} English uses or
  // invent one it does not.
  it("preserves placeholder parity with English for every batch 2 key", () => {
    const placeholders = (value: string) =>
      [...value.matchAll(/\{\{\s*([A-Za-z0-9_]+)\s*\}\}/g)]
        .map((match) => match[1])
        .sort()
        .join(",")

    const walk = (
      value: unknown,
      path: string,
      visit: (key: string, text: string) => void,
    ) => {
      if (value && typeof value === "object") {
        for (const [key, child] of Object.entries(
          value as Record<string, unknown>,
        )) {
          walk(child, path ? `${path}.${key}` : key, visit)
        }
      } else if (typeof value === "string") {
        visit(path, value)
      }
    }

    const english = i18n.getResourceBundle("en", "translation") as Record<
      string,
      unknown
    >
    const inBatch2 = (key: string) =>
      PAGES_BATCH_2.some(
        (prefix) => key === prefix || key.startsWith(prefix as string),
      )

    const failures: string[] = []
    walk(english.pages, "pages", (key, englishText) => {
      if (!inBatch2(key)) return
      for (const locale of BATCH_LOCALES) {
        const translated = i18n.getResource(locale, "translation", key) as
          | string
          | undefined
        if (typeof translated !== "string") continue
        const want = placeholders(englishText)
        const got = placeholders(translated)
        if (want !== got) {
          failures.push(
            `${locale} | pages | ${key} | placeholders "${want}" vs "${got}"`,
          )
        }
      }
    })

    expect(failures.join("\n"), failures.join("\n")).toBe("")
  })

  it("renders translated Tools page body, actions and states", async () => {
    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)

      // Title / description prose.
      expect(i18n.t("pages.agent.tools.library_title"), locale).not.toBe(
        "Tool Library",
      )
      expect(i18n.t("pages.agent.tools.library_description"), locale).not.toBe(
        "Browse and manage the toolset available to your AI agents.",
      )

      // Form / helper content.
      expect(i18n.t("pages.agent.tools.search_placeholder"), locale).not.toBe(
        "Search tools...",
      )
      expect(i18n.t("pages.agent.tools.filter.all"), locale).not.toBe(
        "All Status",
      )

      // Status.
      expect(i18n.t("pages.agent.tools.enable_success"), locale).not.toBe(
        "Tool enabled.",
      )
      expect(i18n.t("pages.agent.tools.status.enabled"), locale).not.toBe(
        "Enabled",
      )

      // Empty / no-result states.
      expect(i18n.t("pages.agent.tools.empty"), locale).not.toBe(
        "No tools are available.",
      )
      expect(i18n.t("pages.agent.tools.no_results"), locale).not.toBe(
        "No tools match your criteria.",
      )
      expect(i18n.t("pages.agent.tools.no_results_hint"), locale).not.toBe(
        "Try adjusting your search criteria or status filters.",
      )

      // Errors and the reasons a tool is blocked.
      expect(i18n.t("pages.agent.tools.toggle_error"), locale).not.toBe(
        "Failed to update tool state.",
      )
      expect(
        i18n.t("pages.agent.tools.reasons.requires_web_search_provider"),
        locale,
      ).not.toBe("Configure at least one ready external web-search provider.")
    }
  })

  it("renders the translated Web Search panel", async () => {
    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)

      expect(i18n.t("pages.agent.tools.web_search.title"), locale).not.toBe(
        "Web Search",
      )
      expect(
        i18n.t("pages.agent.tools.web_search.provider_description"),
        locale,
      ).not.toBe(
        "Select the default provider to use when the web search tool handles a request.",
      )

      // Form helper text.
      expect(
        i18n.t("pages.agent.tools.web_search.api_key_placeholder"),
        locale,
      ).not.toBe("Enter API key, leave it blank to keep the original key")

      // Actions.
      expect(i18n.t("pages.agent.tools.web_search.save"), locale).not.toBe(
        "Save Changes",
      )
      expect(
        i18n.t("pages.agent.tools.web_search.open_settings"),
        locale,
      ).not.toBe("Open Settings")

      // Status and error states.
      expect(
        i18n.t("pages.agent.tools.web_search.save_success"),
        locale,
      ).not.toBe("Settings saved successfully.")
      expect(
        i18n.t("pages.agent.tools.web_search.load_error"),
        locale,
      ).not.toBe("Failed to load web search configuration.")
      expect(i18n.t("pages.agent.tools.web_search.none"), locale).not.toBe(
        "Unavailable",
      )
      // "Model" itself is a Latin cognate that several of these languages
      // legitimately spell the same way, so the field asserted here is its
      // helper text rather than the one-word label.
      expect(
        i18n.t("pages.agent.tools.web_search.model_placeholder"),
        locale,
      ).not.toBe("Optional model override")
    }
  })

  it("renders the translated Configuration page", async () => {
    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)

      // Section tabs. Agent, Runtime, Evolution, MCP and Launcher are the same
      // word in several of these languages, so the tabs asserted here are the
      // ones that genuinely differ.
      expect(i18n.t("pages.config.sections.exec"), locale).not.toBe(
        "Run Commands",
      )
      expect(i18n.t("pages.config.sections.cron"), locale).not.toBe(
        "Cron Tasks",
      )
      expect(i18n.t("pages.config.sections.devices"), locale).not.toBe(
        "Devices",
      )

      // Field labels and their helper text.
      expect(i18n.t("pages.config.workspace"), locale).not.toBe(
        "Workspace Directory",
      )
      expect(i18n.t("pages.config.workspace_hint"), locale).not.toBe(
        "Base directory for agent file operations.",
      )
      expect(i18n.t("pages.config.tool_feedback_enabled"), locale).not.toBe(
        "Tool Feedback",
      )
      expect(i18n.t("pages.config.exec_timeout_seconds_hint"), locale).not.toBe(
        "Maximum runtime for command requests. Set to 0 to use the default timeout.",
      )
      expect(i18n.t("pages.config.cron_exec_timeout"), locale).not.toBe(
        "Scheduled Command Timeout (minutes)",
      )

      // The pattern detector: prompt, input placeholder, action and each of
      // its three verdicts.
      expect(i18n.t("pages.config.pattern_detector_title"), locale).not.toBe(
        "Pattern Detection Tool",
      )
      expect(
        i18n.t("pages.config.pattern_detector_input_placeholder"),
        locale,
      ).not.toBe("Enter a command to test, e.g., rm -rf /tmp")
      expect(
        i18n.t("pages.config.pattern_detector_test_button"),
        locale,
      ).not.toBe("Test")
      expect(
        i18n.t("pages.config.pattern_detector_result_allowed"),
        locale,
      ).not.toBe("Allowed (matches whitelist)")
      expect(
        i18n.t("pages.config.pattern_detector_result_blocked"),
        locale,
      ).not.toBe("Blocked (matches blacklist)")
      expect(
        i18n.t("pages.config.pattern_detector_result_no_match"),
        locale,
      ).not.toBe("No match (will use default rules)")

      // Validation and error states.
      expect(i18n.t("pages.config.workspace_required"), locale).not.toBe(
        "Workspace path is required.",
      )
      expect(i18n.t("pages.config.load_error"), locale).not.toBe(
        "Failed to load configuration. Please refresh and try again.",
      )
    }
  })

  // The sample regular expressions are code, not copy: they must be byte
  // identical everywhere or the hint they give would be wrong.
  it("keeps the pattern samples identical in every locale", async () => {
    await i18n.changeLanguage("en")
    const english = i18n.t("pages.config.custom_patterns_placeholder")
    expect(english).toContain("^rm")

    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)
      expect(i18n.t("pages.config.custom_patterns_placeholder"), locale).toBe(
        english,
      )
    }
  })

  it("keeps Arabic batch 2 pages right-to-left and translated", async () => {
    await i18n.changeLanguage("ar")
    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expect(i18n.t("pages.agent.tools.library_title")).toBe("مكتبة الأدوات")
    expect(i18n.t("pages.agent.tools.web_search.title")).toBe("البحث على الويب")
    expect(i18n.t("pages.config.sections.exec")).toBe("تشغيل الأوامر")
    expect(i18n.t("pages.config.pattern_detector_test_button")).toBe("اختبار")
  })

  it("keeps English and the pre-existing locales intact for batch 2", async () => {
    await i18n.changeLanguage("en")
    expect(i18n.t("pages.agent.tools.library_title")).toBe("Tool Library")
    expect(i18n.t("pages.config.sections.exec")).toBe("Run Commands")

    for (const locale of ["pt", "zh"]) {
      await i18n.changeLanguage(locale)
      for (const key of [
        "pages.agent.tools.library_title",
        "pages.config.sections.exec",
      ]) {
        expect(i18n.t(key), `${locale} ${key}`).not.toBe(key)
      }
    }
  })
})

describe("Pages batch 3 — Agent tuning, runtime and security", () => {
  const inBatch3 = (key: string) =>
    PAGES_BATCH_3.some(
      (prefix) => key === prefix || key.startsWith(`${prefix}.`),
    )

  it("preserves placeholder parity with English for every batch 3 key", () => {
    const placeholders = (value: string) =>
      [...value.matchAll(/\{\{\s*([A-Za-z0-9_]+)\s*\}\}/g)]
        .map((match) => match[1])
        .sort()
        .join(",")

    const english = i18n.getResourceBundle("en", "translation") as Record<
      string,
      unknown
    >
    const config = (english.pages as Record<string, unknown>).config as Record<
      string,
      string
    >

    const failures: string[] = []
    for (const [name, englishText] of Object.entries(config)) {
      const key = `pages.config.${name}`
      if (!inBatch3(key) || typeof englishText !== "string") continue
      for (const locale of BATCH_LOCALES) {
        const translated = i18n.getResource(locale, "translation", key) as
          | string
          | undefined
        if (typeof translated !== "string") continue
        const want = placeholders(englishText)
        const got = placeholders(translated)
        if (want !== got) {
          failures.push(
            `${locale} | pages | ${key} | placeholders "${want}" vs "${got}"`,
          )
        }
      }
    }

    expect(failures.join("\n"), failures.join("\n")).toBe("")
  })

  it("renders the translated Agent tuning settings", async () => {
    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)

      // Field labels and helper text.
      expect(i18n.t("pages.config.max_tokens"), locale).not.toBe("Max Tokens")
      expect(i18n.t("pages.config.max_tokens_hint"), locale).not.toBe(
        "Upper token limit per model response.",
      )
      expect(i18n.t("pages.config.context_window_hint"), locale).not.toBe(
        "Model input context capacity in tokens. Leave empty to use the default (4x max tokens).",
      )
      expect(i18n.t("pages.config.summarize_threshold"), locale).not.toBe(
        "Summarize Message Threshold",
      )

      // The request-context policy, its title, its explanation and its modes.
      expect(i18n.t("pages.config.turn_profile"), locale).not.toBe(
        "Request Context Policy",
      )
      expect(i18n.t("pages.config.turn_profile_hint"), locale).not.toBe(
        "Controls what context each request carries. Leave disabled to keep the normal chat behavior.",
      )
      expect(i18n.t("pages.config.turn_profile_mode_custom"), locale).not.toBe(
        "Allow List",
      )
      expect(i18n.t("pages.config.turn_profile_history_hint"), locale).not.toBe(
        "Default includes earlier messages from this session. Off makes the turn behave like a fresh chat and skips saving its result back to history.",
      )

      // Session scope: the selector, its options and their descriptions.
      expect(i18n.t("pages.config.session_scope"), locale).not.toBe(
        "Session Scope",
      )
      expect(
        i18n.t("pages.config.session_scope_per_channel_peer"),
        locale,
      ).not.toBe("Per Channel + Peer")
      expect(
        i18n.t("pages.config.session_scope_per_channel_peer_desc"),
        locale,
      ).not.toBe("Separate context for each user in each channel.")
    }
  })

  it("renders the translated runtime, launcher and security settings", async () => {
    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)

      // Runtime toggles.
      expect(i18n.t("pages.config.heartbeat_enabled_hint"), locale).not.toBe(
        "Send periodic heartbeat messages.",
      )
      expect(i18n.t("pages.config.monitor_usb_hint"), locale).not.toBe(
        "Watch USB plug/unplug events when devices are enabled.",
      )

      // Launcher.
      expect(i18n.t("pages.config.autostart_label"), locale).not.toBe(
        "Launch at Login",
      )
      expect(i18n.t("pages.config.server_port"), locale).not.toBe(
        "Service Port",
      )
      expect(i18n.t("pages.config.launcher_section_hint"), locale).not.toBe(
        "Changes in this section take effect after the launcher restarts.",
      )

      // Password fields, including their placeholders.
      expect(i18n.t("pages.config.dashboard_password"), locale).not.toBe(
        "Login Password",
      )
      expect(
        i18n.t("pages.config.dashboard_password_placeholder"),
        locale,
      ).not.toBe("At least 8 characters")

      // Network security.
      expect(i18n.t("pages.config.lan_access_hint"), locale).not.toBe(
        "Allow access from other devices on your local network.",
      )
      expect(i18n.t("pages.config.allowed_cidrs"), locale).not.toBe(
        "Allowed Network CIDRs",
      )
      expect(
        i18n.t("pages.config.allow_localhost_bypass_hint"),
        locale,
      ).not.toBe(
        "When enabled, localhost requests are allowed even when they do not match the allowed CIDRs. Disable this when the launcher is behind a same-host proxy.",
      )

      // Status / error states.
      expect(i18n.t("pages.config.autostart_unsupported"), locale).not.toBe(
        "Launch at login is not supported on this platform.",
      )
      expect(i18n.t("pages.config.autostart_load_error"), locale).not.toBe(
        "Failed to load launch-at-login status.",
      )
    }
  })

  // Everything here reaches the user through toast.error(err.message) when a
  // save is rejected, so each one has to be translated prose, not source text.
  it("renders translated validation messages with their values interpolated", async () => {
    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)

      expect(i18n.t("pages.config.session_scope_required"), locale).not.toBe(
        "Session scope is required.",
      )
      expect(
        i18n.t("pages.config.dashboard_password_mismatch"),
        locale,
      ).not.toBe("The login passwords do not match.")
      expect(
        i18n.t("pages.config.dashboard_password_min_length"),
        locale,
      ).not.toBe("Login password must be at least 8 characters.")

      // The numeric validators substitute the field's own label and bound.
      const label = i18n.t("pages.config.max_tokens")
      const integer = i18n.t("pages.config.validation_integer", { label })
      expect(integer, locale).toContain(label)
      expect(integer, locale).not.toContain("{{")
      expect(integer, locale).not.toBe(`${label} must be an integer.`)

      const min = i18n.t("pages.config.validation_min", { label, min: 1 })
      expect(min, locale).toContain("1")
      expect(min, locale).not.toContain("{{")

      const max = i18n.t("pages.config.validation_max", { label, max: 100 })
      expect(max, locale).toContain("100")
      expect(max, locale).not.toContain("{{")
    }
  })

  // The samples are the format to type, so they must survive translation byte
  // for byte the way the pattern samples do.
  it("keeps the tool and CIDR samples identical in every locale", async () => {
    const literalKeys = [
      "pages.config.turn_profile_tools_allow_placeholder",
      "pages.config.allowed_cidrs_placeholder",
      "pages.config.trusted_proxy_cidrs_placeholder",
    ]
    await i18n.changeLanguage("en")
    const english = Object.fromEntries(
      literalKeys.map((key) => [key, i18n.t(key)]),
    )
    expect(english["pages.config.allowed_cidrs_placeholder"]).toContain(
      "192.168.1.0/24",
    )

    for (const locale of BATCH_LOCALES) {
      await i18n.changeLanguage(locale)
      for (const key of literalKeys) {
        expect(i18n.t(key), `${locale} ${key}`).toBe(english[key])
      }
    }
  })

  it("keeps Arabic batch 3 pages right-to-left and translated", async () => {
    await i18n.changeLanguage("ar")
    expect(i18n.dir()).toBe("rtl")
    expect(document.documentElement.getAttribute("dir")).toBe("rtl")
    expect(i18n.t("pages.config.session_scope")).toBe("نطاق الجلسة")
    expect(i18n.t("pages.config.dashboard_password")).toBe("كلمة مرور الدخول")
    expect(i18n.t("pages.config.autostart_label")).toBe(
      "التشغيل عند تسجيل الدخول",
    )
  })

  it("keeps English and the pre-existing locales intact for batch 3", async () => {
    await i18n.changeLanguage("en")
    expect(i18n.t("pages.config.session_scope")).toBe("Session Scope")
    expect(
      i18n.t("pages.config.validation_min", { label: "Max Tokens", min: 1 }),
    ).toBe("Max Tokens must be 1 or more.")

    for (const locale of ["pt", "zh"]) {
      await i18n.changeLanguage(locale)
      for (const key of [
        "pages.config.session_scope",
        "pages.config.dashboard_password",
      ]) {
        expect(i18n.t(key), `${locale} ${key}`).not.toBe(key)
      }
    }
  })
})
