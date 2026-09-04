/// The languages the manual override menu offers.
///
/// Names are endonyms: a language is listed the way its own speakers write it,
/// so the entry stays recognisable no matter which locale the console is
/// currently rendering in. They are deliberately not translation keys — an
/// endonym is the same string in every locale.
export interface LanguageOption {
  /// The tag handed to i18next. It must be one of SUPPORTED_LANGUAGES.
  code: string
  endonym: string
}

/// Every locale the PocketClaw app itself can be set to. The host may reopen
/// the dashboard with any of these as ?lng=, so the manual menu has to be able
/// to reach all of them too.
export const APP_LANGUAGES: readonly LanguageOption[] = [
  { code: "ar", endonym: "العربية" },
  { code: "de", endonym: "Deutsch" },
  { code: "en", endonym: "English" },
  { code: "es", endonym: "Español" },
  { code: "fr", endonym: "Français" },
  { code: "hi", endonym: "हिन्दी" },
  { code: "id", endonym: "Bahasa Indonesia" },
  { code: "ja", endonym: "日本語" },
  { code: "ko", endonym: "한국어" },
  // The app sends plain "pt"; i18next resolves it onto the Brazilian bundle,
  // which is what the console actually ships.
  { code: "pt", endonym: "Português (Brasil)" },
  { code: "ru", endonym: "Русский" },
  { code: "zh", endonym: "简体中文" },
] as const

/// Locales the dashboard shipped before the app gained its own language
/// setting. The app cannot request them, but the bundles exist and the menu
/// has always offered them, so they stay reachable.
export const DASHBOARD_ONLY_LANGUAGES: readonly LanguageOption[] = [
  { code: "bn-IN", endonym: "বাংলা" },
  { code: "cs", endonym: "Čeština" },
] as const

/// What the menu renders, in a stable order.
export const LANGUAGE_OPTIONS: readonly LanguageOption[] = [
  ...APP_LANGUAGES,
  ...DASHBOARD_ONLY_LANGUAGES,
]

/// Maps whatever i18next currently reports onto the option the menu shows as
/// selected. The host may hand over a tag the menu does not list verbatim
/// ("pt-BR", "zh-CN"); those still belong to a listed language.
export function selectedLanguageCode(current: string | undefined | null) {
  if (!current) return "en"
  const lower = current.toLowerCase()
  const exact = LANGUAGE_OPTIONS.find((o) => o.code.toLowerCase() === lower)
  if (exact) return exact.code
  const base = lower.split("-")[0]
  const baseMatch = LANGUAGE_OPTIONS.find(
    (o) => o.code.toLowerCase().split("-")[0] === base,
  )
  return baseMatch ? baseMatch.code : "en"
}
