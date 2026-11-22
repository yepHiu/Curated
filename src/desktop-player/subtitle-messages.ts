import { subtitles as zh } from "@/locales/zh-CN.json"
import { subtitles as en } from "@/locales/en.json"
import { subtitles as ja } from "@/locales/ja.json"

const keys = ["displaySettings", "off", "chinese", "japanese", "bilingual", "loadFailed", "noSubtitles", "fontSize", "background", "position", "offset", "offsetHint"] as const

// The native window forbids unsafe-eval. Supply message functions so vue-i18n
// never compiles JSON strings at runtime, matching the other native messages.
function messages(source: typeof zh) {
  return Object.fromEntries(keys.map(key => [key, () => source[key]]))
}

export const subtitleMessages = { "zh-CN": messages(zh), en: messages(en), ja: messages(ja) }
