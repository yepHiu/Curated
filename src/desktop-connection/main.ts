import { createApp } from "vue"
import { createI18n } from "vue-i18n"
import "@fontsource-variable/noto-sans/wght.css"
import "@fontsource-variable/noto-sans-jp/wght.css"
import "@fontsource-variable/outfit/wght.css"
import outfitLicenseURL from "@fontsource-variable/outfit/LICENSE?url"
import { htmlLangFor, resolveInitialLocale } from "@/lib/locale-storage"
import { applyThemeToDocument, getStoredThemePreference } from "@/lib/theme-storage"
import { connectionMessages } from "./messages"
import "./style.css"
import ConnectionPage from "./ConnectionPage.vue"

// 与业务入口共用字体、语言和主题规则，首帧即应用本地偏好。
const locale = resolveInitialLocale()
document.documentElement.lang = htmlLangFor(locale)
applyThemeToDocument(getStoredThemePreference())
// 字体许可随连接页分发，通过文档 license 链接保留访问入口。
const outfitLicense = document.createElement("link")
outfitLicense.rel = "license"
outfitLicense.title = "Outfit — SIL Open Font License"
outfitLicense.href = outfitLicenseURL
document.head.append(outfitLicense)
const i18n = createI18n({ legacy: false, locale, fallbackLocale: "zh-CN", messages: connectionMessages })
createApp(ConnectionPage).use(i18n).mount("#app")
