import { describe, expect, it } from "vitest"
import en from "@/locales/en.json"
import ja from "@/locales/ja.json"
import zhCN from "@/locales/zh-CN.json"

const locales = {
  en,
  ja,
  "zh-CN": zhCN,
} satisfies Record<string, Record<string, unknown>>

const requiredLocaleKeys = [
  "curated.tagFilterTitle",
  "curated.tagFilterAll",
  "curated.tagFilterEmpty",
  "curated.tagFilterNoMatches",
  "curated.tagFilterShowMore",
  "curated.tagFilterShowLess",
  "curated.ariaFilterFrameTag",
  "curated.ariaClearFrameTagFilter",
  "settings.curatedExportFormatSaving",
]

function readLocaleKey(messages: Record<string, unknown>, key: string): unknown {
  let cursor: unknown = messages
  for (const segment of key.split(".")) {
    if (!cursor || typeof cursor !== "object" || Array.isArray(cursor)) {
      return undefined
    }
    cursor = (cursor as Record<string, unknown>)[segment]
  }
  return cursor
}

describe("locale key parity", () => {
  it.each(Object.entries(locales))("%s has curated tag filter and saving keys", (_locale, messages) => {
    const missing = requiredLocaleKeys.filter((key) => {
      const value = readLocaleKey(messages, key)
      return typeof value !== "string" || value.trim() === ""
    })

    expect(missing).toEqual([])
  })
})
