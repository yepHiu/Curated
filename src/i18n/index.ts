import { createI18n } from "vue-i18n"
import zhCN from "@/locales/zh-CN.json"
import enMessagesUrl from '@/locales/en.json?url'
import jaMessagesUrl from '@/locales/ja.json?url'
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

  // Locale dictionaries are data assets: fetch/parse them on demand instead
  // of shipping another executable JavaScript chunk for each language.
  const response = await fetch(locale === 'en' ? enMessagesUrl : jaMessagesUrl)
  if (!response.ok) throw new Error(`Failed to load locale ${locale}`)
  const messages = await response.json()
  i18n.global.setLocaleMessage(locale, messages)
  loadedLocales.add(locale)
}

export function syncHtmlLang(locale: SupportedLocale) {
  if (typeof document === "undefined") return
  document.documentElement.lang = htmlLangFor(locale)
}

syncHtmlLang(initialLocale)
