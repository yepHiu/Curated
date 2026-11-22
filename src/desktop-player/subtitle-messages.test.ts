import { afterEach, describe, expect, it, vi } from "vitest"
import { createI18n } from "vue-i18n"
import { subtitleMessages } from "./subtitle-messages"

afterEach(() => vi.restoreAllMocks())

describe("native subtitle messages under the player CSP", () => {
  it.each(["zh-CN", "en", "ja"] as const)("renders every %s menu label without runtime compilation", locale => {
    const i18n = createI18n({ legacy: false, locale, messages: { [locale]: { subtitles: subtitleMessages[locale] } } })
    vi.spyOn(globalThis, "Function").mockImplementation(() => { throw new Error("unsafe-eval blocked") })
    for (const key of Object.keys(subtitleMessages[locale])) {
      const label = i18n.global.t(`subtitles.${key}`)
      expect(label).not.toBe(`subtitles.${key}`)
      expect(label.length).toBeGreaterThan(0)
    }
  })
})
