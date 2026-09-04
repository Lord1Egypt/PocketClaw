import dayjs from "dayjs"
import "dayjs/locale/bn"
import "dayjs/locale/cs"
import "dayjs/locale/en"
import "dayjs/locale/pt-br"
import "dayjs/locale/zh-cn"
import localizedFormat from "dayjs/plugin/localizedFormat"
import relativeTime from "dayjs/plugin/relativeTime"
import i18n from "i18next"
import LanguageDetector from "i18next-browser-languagedetector"
import { initReactI18next } from "react-i18next"

import ar from "./locales/ar.json"
import bnIn from "./locales/bn-in.json"
import cs from "./locales/cs.json"
import de from "./locales/de.json"
import en from "./locales/en.json"
import es from "./locales/es.json"
import fr from "./locales/fr.json"
import hi from "./locales/hi.json"
import id from "./locales/id.json"
import ja from "./locales/ja.json"
import ko from "./locales/ko.json"
import ptBr from "./locales/pt-br.json"
import ru from "./locales/ru.json"
import zh from "./locales/zh.json"

/// Every locale the PocketClaw app can be set to, plus the ones the dashboard
/// already shipped. The host passes its selected locale as ?lng=, which
/// i18next's own query-string detector reads.
export const SUPPORTED_LANGUAGES = [
  "en",
  "ar",
  "de",
  "es",
  "fr",
  "hi",
  "id",
  "ja",
  "ko",
  "pt",
  "pt-BR",
  "ru",
  "zh",
  "bn-IN",
  "cs",
] as const

dayjs.extend(relativeTime)
dayjs.extend(localizedFormat)

/// The single init contract, exported so a test can stand up an isolated
/// instance and assert what the host actually depends on — notably that
/// LanguageDetector still reads ?lng= ahead of the cached localStorage value.
/// for all options read: https://www.i18next.com/overview/configuration-options
export const I18N_OPTIONS = {
  resources: {
    en: { translation: en },
    ar: { translation: ar },
    de: { translation: de },
    es: { translation: es },
    fr: { translation: fr },
    hi: { translation: hi },
    id: { translation: id },
    ja: { translation: ja },
    ko: { translation: ko },
    ru: { translation: ru },
    // The host app sends plain "pt" and "zh"; these carry the existing
    // Portuguese and Chinese translations rather than duplicating them.
    "pt-BR": { translation: ptBr },
    "bn-IN": { translation: bnIn },
    zh: { translation: zh },
    cs: { translation: cs },
  },
  // "pt" resolves to the existing pt-BR resource instead of falling all the
  // way back to English, and regional tags land on their base language.
  fallbackLng: {
    pt: ["pt-BR", "en"],
    "pt-PT": ["pt-BR", "en"],
    bn: ["bn-IN", "en"],
    default: ["en"],
  },
  supportedLngs: SUPPORTED_LANGUAGES,
  nonExplicitSupportedLngs: true,
  debug: false,

  interpolation: {
    escapeValue: false, // not needed for react as it escapes by default
  },
}

i18n
  // detect user language
  // learn more: https://github.com/i18next/i18next-browser-languageDetector
  .use(LanguageDetector)
  // pass the i18n instance to react-i18next.
  .use(initReactI18next)
  .init(I18N_OPTIONS)

/// Applies the document language and writing direction.
///
/// i18next knows the direction of every language it ships, so this asks it
/// rather than testing for Arabic in each component. The attributes live on the
/// root element, which is what CSS logical properties and the layout read.
export function applyDocumentDirection(lng: string) {
  if (typeof document === "undefined") return
  const root = document.documentElement
  root.setAttribute("lang", lng)
  root.setAttribute("dir", i18n.dir(lng))
}

i18n.on("languageChanged", (lng) => {
  applyDocumentDirection(lng)

  if (lng.startsWith("zh")) {
    dayjs.locale("zh-cn")
  } else if (lng.startsWith("pt")) {
    dayjs.locale("pt-br")
  } else if (lng.startsWith("bn")) {
    dayjs.locale("bn")
  } else if (lng.startsWith("cs")) {
    dayjs.locale("cs")
  } else {
    dayjs.locale("en")
  }
})

applyDocumentDirection(i18n.resolvedLanguage ?? i18n.language ?? "en")

export default i18n
