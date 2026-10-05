import { createApp } from "vue"
import { createI18n } from "vue-i18n"
import "@fontsource-variable/noto-sans/wght.css"
import "@fontsource-variable/noto-sans-jp/wght.css"
import "@fontsource-variable/outfit/wght.css"
import "../native-player-prototype/style.css"
import DesktopPlayer from "./DesktopPlayer.vue"
import { nativeMessages } from "../native-player-prototype/messages"
const messages = Object.fromEntries(Object.entries(nativeMessages).map(([locale, text]) => [locale, { player: {
  playbackSettings: () => text.settings, playbackSettingsAria: () => text.settingsAria,
  playbackSpeed: () => text.speed, selectPart: () => text.file,
  partLabel: (context: { named(key: string): unknown }) => text.partLabel.replace("{number}", String(context.named("number"))),
} }]))
// 独立视频窗口始终使用深色 HUD，包含 portal 到 body 的分部菜单。
document.documentElement.classList.add("dark")
createApp(DesktopPlayer).use(createI18n({ legacy: false, locale: "zh-CN", messages })).mount("#app")
