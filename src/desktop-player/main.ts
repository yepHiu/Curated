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
  clipRecording: () => text.clipRecording, clipProcessing: () => text.clipProcessing, clipSavedToLibrary: () => text.clipSaved,
  clipExportFailed: () => text.clipFailed, clipExportTimedOut: () => text.clipTimedOut, clipStatusUnavailable: () => text.clipStatusUnavailable,
}, curated: {
  captureView: () => text.captureView, captureSaving: () => text.captureSaving, captureSaved: () => text.captureSaved,
  captureFailed: () => text.captureFailed, captureRetry: () => text.captureRetry, retryLoad: () => text.retryLoad,
  stopClip: () => text.clipStop,
  clipCancelFailed: () => text.clipCancelFailed,
}, common: { close: () => text.dismiss, loading: () => text.loading, cancel: () => text.cancel, cancelled: () => text.cancelled, retry: () => text.retry } }]))
// 独立视频窗口始终使用深色 HUD，包含 portal 到 body 的分部菜单。
document.documentElement.classList.add("dark")
createApp(DesktopPlayer).use(createI18n({ legacy: false, locale: "zh-CN", messages })).mount("#app")
