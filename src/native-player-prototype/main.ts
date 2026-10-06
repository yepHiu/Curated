import { createApp } from "vue"
import { createI18n } from "vue-i18n"
import "@fontsource-variable/noto-sans/wght.css"
import "@fontsource-variable/noto-sans-jp/wght.css"
import "@fontsource-variable/outfit/wght.css"
import "./style.css"
import { applyThemeToDocument, getStoredThemePreference } from "@/lib/theme-storage"
import { htmlLangFor, resolveInitialLocale } from "@/lib/locale-storage"
import NativePlayerPrototype from "./NativePlayerPrototype.vue"
import { nativeMessages } from "./messages"

document.documentElement.lang = htmlLangFor(resolveInitialLocale())
applyThemeToDocument(getStoredThemePreference())
// 共享控件只加载所需的翻译，不初始化正式路由或媒体库服务。
const messages = Object.fromEntries(Object.entries(nativeMessages).map(([locale, text]) => {
  /* 复用分段选择与播放设置组件已有的翻译键。 */
  // CSP 禁止运行时 Function 编译，直接提供有限的消息函数。
  return [locale, { player: {
    playbackSettings: () => { return text.settings }, playbackSettingsAria: () => { return text.settingsAria },
    playbackSpeed: () => { return text.speed }, selectPart: () => { return text.file },
    partLabel: (context: { named(key: string): unknown }) => { return text.partLabel.replace("{number}", String(context.named("number"))) },
  } }]
}))
createApp(NativePlayerPrototype).use(createI18n({ legacy: false, locale: resolveInitialLocale(), messages })).mount("#app")
