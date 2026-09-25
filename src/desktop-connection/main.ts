import { createApp } from "vue"
import { createI18n } from "vue-i18n"
import "@fontsource-variable/noto-sans/wght.css"
import "@fontsource-variable/noto-sans-jp/wght.css"
import { htmlLangFor, resolveInitialLocale } from "@/lib/locale-storage"
import { applyThemeToDocument, getStoredThemePreference } from "@/lib/theme-storage"
import { connectionMessages } from "./messages"
import "./style.css"
import ConnectionPage from "./ConnectionPage.vue"

// 与业务入口共用字体、语言和主题规则，首帧即应用本地偏好。
const locale = resolveInitialLocale()
document.documentElement.lang = htmlLangFor(locale)
applyThemeToDocument(getStoredThemePreference())
const i18n = createI18n({ legacy: false, locale, fallbackLocale: "zh-CN", messages: connectionMessages })
createApp(ConnectionPage).use(i18n).mount("#app")
