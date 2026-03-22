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
  },
})

export function syncHtmlLang(locale: SupportedLocale) {
  if (typeof document === "undefined") return
  document.documentElement.lang = htmlLangFor(locale)
}

syncHtmlLang(initial)
