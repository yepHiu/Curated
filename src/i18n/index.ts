import { createI18n } from "vue-i18n"
import en from "@/locales/en.json"
import ja from "@/locales/ja.json"
import zhCN from "@/locales/zh-CN.json"
import { htmlLangFor, resolveInitialLocale, type SupportedLocale } from "@/lib/locale-storage"

export type { SupportedLocale }

const initial = resolveInitialLocale()

export const i18n = createI18n({
  legacy: false,
  locale: initial,
  fallbackLocale: "zh-CN",
  messages: {
    en,
    ja,
    "zh-CN": zhCN,
    /** 与浏览器 `navigator.language` 的 `zh` 对齐，避免回退到 en 时出现缺键警告 */
    zh: zhCN,
  },
})

export function syncHtmlLang(locale: SupportedLocale) {
  if (typeof document === "undefined") return
  document.documentElement.lang = htmlLangFor(locale)
}

syncHtmlLang(initial)
