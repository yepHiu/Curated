import { describe, expect, it } from "vitest"
import {
  buildUserTagSuggestionPool,
  containsHanScript,
  filterUserTagSuggestions,
  normalizeTagMatchText,
} from "@/lib/user-tag-suggestions"

describe("normalizeTagMatchText", () => {
  it("NFKC 全角与半角拉丁统一", () => {
    expect(normalizeTagMatchText("４Ｋ")).toBe("4k")
  })

  it("英文小写", () => {
    expect(normalizeTagMatchText("Romance")).toBe("romance")
  })
})

describe("filterUserTagSuggestions", () => {
  const pool = ["中文字幕", "字幕", "4K", "Office", "office-hr"]

  it("中文子串", () => {
    expect(filterUserTagSuggestions(pool, "幕", new Set(), { limit: 20 })).toEqual([
      "中文字幕",
      "字幕",
    ])
  })

  it("排除已选 userTags", () => {
    expect(filterUserTagSuggestions(pool, "字幕", new Set(["字幕"]))).toEqual(["中文字幕"])
  })

  it("英文大小写不敏感", () => {
    expect(filterUserTagSuggestions(pool, "office", new Set())).toContain("Office")
  })

  it("拼音全拼匹配中文标签", () => {
    const r = filterUserTagSuggestions(["字幕"], "zimu", new Set(), { limit: 5 })
    expect(r).toContain("字幕")
  })

  it("拼音首字母匹配", () => {
    const r = filterUserTagSuggestions(["中文字幕"], "zwzm", new Set(), { limit: 5 })
    expect(r).toContain("中文字幕")
  })

  it("查询含汉字时不走纯拼音歧义（仍用子串）", () => {
    const r = filterUserTagSuggestions(["han字"], "han", new Set(), { limit: 5 })
    expect(r).toContain("han字")
  })

  it("enablePinyin false 时仅用子串", () => {
    const r = filterUserTagSuggestions(["字幕"], "zimu", new Set(), {
      limit: 5,
      enablePinyin: false,
    })
    expect(r).toEqual([])
  })

  it("limit 截断", () => {
    const many = ["标签1", "标签2", "标签3", "标签4"]
    expect(filterUserTagSuggestions(many, "标签", new Set(), { limit: 2 })).toHaveLength(2)
  })
})

describe("containsHanScript", () => {
  it("识别汉字", () => {
    expect(containsHanScript("ab字")).toBe(true)
  })
  it("纯拉丁", () => {
    expect(containsHanScript("zimu")).toBe(false)
  })
})

describe("buildUserTagSuggestionPool", () => {
  it("合并 userTags 与当前片 tags 并去重排序", () => {
    const movies = [
      { tags: ["A"], userTags: ["乙", "甲"] },
      { tags: ["B"], userTags: ["甲", "丙"] },
    ]
    expect(buildUserTagSuggestionPool(movies, ["乙", "元数据"])).toEqual([
      "丙",
      "甲",
      "乙",
      "元数据",
    ])
  })
})
