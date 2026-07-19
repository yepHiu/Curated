import { createI18n } from "vue-i18n"
import zhCN from "@/locales/zh-CN.json"
import { htmlLangFor, resolveInitialLocale, type SupportedLocale } from "@/lib/locale-storage"

export type { SupportedLocale }

export const initialLocale = resolveInitialLocale()

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: "zh-CN",
  messages: {
    en: {},
    ja: {},
    "zh-CN": zhCN,
    /** 与浏览器 `navigator.language` 的 `zh` 对齐，避免回退到 en 时出现缺键警告 */
    zh: zhCN,
  },
})

const loadedLocales = new Set<SupportedLocale>(["zh-CN"])

export async function ensureLocaleMessages(locale: SupportedLocale): Promise<void> {
  if (loadedLocales.has(locale)) return

  const module = locale === "en"
    ? await import("@/locales/en.json")
    : await import("@/locales/ja.json")
  i18n.global.setLocaleMessage(locale, module.default)
  loadedLocales.add(locale)
}

export function syncHtmlLang(locale: SupportedLocale) {
  if (typeof document === "undefined") return
  document.documentElement.lang = htmlLangFor(locale)
}

syncHtmlLang(initialLocale)
