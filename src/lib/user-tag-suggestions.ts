import { match } from "pinyin-pro"

/** 用于匹配的规范化：兼容等价字形 + 英文大小写 */
export function normalizeTagMatchText(s: string): string {
  return s.normalize("NFKC").trim().toLowerCase()
}

/** 含常见 CJK 统一汉字（不含扩展 B 等，标签场景足够） */
export function containsHanScript(s: string): boolean {
  return /[\u4E00-\u9FFF]/.test(s)
}

export interface FilterUserTagSuggestionsOptions {
  /** 默认 10 */
  limit?: number
  /** 默认 true：在查询串无汉字时启用 pinyin-pro `match` */
  enablePinyin?: boolean
}

/**
 * 从候选标签中筛选建议：NFKC 后子串匹配；查询无汉字时可拼音/简拼匹配（`pinyin-pro` match）。
 */
export function filterUserTagSuggestions(
  candidates: readonly string[],
  draft: string,
  exclude: ReadonlySet<string> | readonly string[],
  options: FilterUserTagSuggestionsOptions = {},
): string[] {
  const limit = options.limit ?? 10
  const enablePinyin = options.enablePinyin !== false
  const trimmed = draft.trim()
  if (!trimmed) {
    return []
  }

  const excludeSet = exclude instanceof Set ? exclude : new Set(exclude)
  const draftNorm = normalizeTagMatchText(trimmed)
  const tryPinyin = enablePinyin && !containsHanScript(trimmed)

  const hits: string[] = []
  const seen = new Set<string>()

  for (const raw of candidates) {
    if (!raw || excludeSet.has(raw) || seen.has(raw)) {
      continue
    }
    const textN = normalizeTagMatchText(raw)
    let ok = textN.includes(draftNorm)
    if (!ok && tryPinyin) {
      const m = match(raw, trimmed, { insensitive: true, continuous: true })
      ok = m !== null && m.length > 0
    }
    if (ok) {
      seen.add(raw)
      hits.push(raw)
    }
  }

  hits.sort((a, b) => {
    const na = normalizeTagMatchText(a)
    const nb = normalizeTagMatchText(b)
    const pa = na.startsWith(draftNorm) ? 0 : 1
    const pb = nb.startsWith(draftNorm) ? 0 : 1
    if (pa !== pb) {
      return pa - pb
    }
    return a.localeCompare(b, "zh-CN", { numeric: true })
  })

  return hits.slice(0, limit)
}

/**
 * 合并去重后按 zh-CN 排序，供详情页候选池。
 */
export function buildUserTagSuggestionPool(
  allMovies: readonly { tags: readonly string[]; userTags: readonly string[] }[],
  currentMovieTags: readonly string[],
): string[] {
  const set = new Set<string>()
  for (const m of allMovies) {
    for (const t of m.userTags) {
      if (t) {
        set.add(t)
      }
    }
  }
  for (const t of currentMovieTags) {
    if (t) {
      set.add(t)
    }
  }
  return [...set].sort((a, b) => a.localeCompare(b, "zh-CN", { numeric: true }))
}
