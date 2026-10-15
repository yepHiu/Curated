import { defaultSubtitleDisplay, validateSubtitleDisplay, type SubtitleDisplay } from "../../electron/subtitle-format"
const key = "curated-subtitle-display-v1"
export function readSubtitlePreferences(scope?: string): SubtitleDisplay {
  try {
    const display = validateSubtitleDisplay(JSON.parse(localStorage.getItem(key) ?? "null"))
    const track = scope ? JSON.parse(localStorage.getItem(`${key}:${scope}`) ?? "null") : null
    return validateSubtitleDisplay({ ...display, offset: 0, ...(track ?? {}) })
  } catch { return { ...defaultSubtitleDisplay } }
}
export function saveSubtitlePreferences(value: SubtitleDisplay, scope?: string) {
  try {
    const display = validateSubtitleDisplay(value)
    localStorage.setItem(key, JSON.stringify({ ...display, offset: 0 }))
    if (scope) localStorage.setItem(`${key}:${scope}`, JSON.stringify({ mode: display.mode, offset: display.offset }))
  } catch { /* Playback also works without local storage. */ }
}
