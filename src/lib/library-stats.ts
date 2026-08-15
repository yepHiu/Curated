import type { LibraryStat } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"

/** 与「Recently Added」浏览模式一致：按入库日期在窗口内的影片 */
export const RECENT_ADDED_LOOKBACK_DAYS = 30

export function isMovieRecentlyAdded(addedAt: string, days = RECENT_ADDED_LOOKBACK_DAYS): boolean {
  const t = Date.parse(addedAt)
  if (Number.isNaN(t)) {
    return false
  }
  const cutoff = Date.now() - days * 86_400_000
  return t >= cutoff
}

/** 元数据 + 用户标签去重后的种类数（侧边栏 Tags 提示与「标签浏览」一致） */
export function countUniqueTags(movies: readonly Movie[]): number {
  const set = new Set<string>()
  for (const m of movies) {
    for (const tag of m.tags) {
      const s = tag.trim()
      if (s) set.add(s)
    }
    for (const tag of m.userTags) {
      const s = tag.trim()
      if (s) set.add(s)
    }
  }
  return set.size
}

export interface TagCountEntry {
  tag: string
  count: number
}

export interface NamedCountEntry {
  name: string
  count: number
}

function sortNamedCounts(entries: NamedCountEntry[], locale: string): NamedCountEntry[] {
  return entries.sort((a, b) => {
    if (b.count !== a.count) return b.count - a.count
    return a.name.localeCompare(b.name, locale, { numeric: true })
  })
}

/** Count how often each trimmed name appears; first-seen casing wins. */
export function aggregateNamedCounts(values: readonly string[], locale: string): NamedCountEntry[] {
  const map = new Map<string, NamedCountEntry>()
  for (const raw of values) {
    const name = raw.trim()
    if (!name) continue
    const key = name.toLocaleLowerCase()
    const existing = map.get(key)
    if (existing) {
      existing.count += 1
    } else {
      map.set(key, { name, count: 1 })
    }
  }
  return sortNamedCounts([...map.values()], locale)
}

/** Keep an active filter visible even if it is no longer in the current inventory. */
export function withCurrentNamedCounts(
  rows: readonly NamedCountEntry[],
  current: readonly string[],
  locale: string,
): NamedCountEntry[] {
  const map = new Map<string, NamedCountEntry>()
  for (const row of rows) {
    map.set(row.name.toLocaleLowerCase(), { ...row })
  }
  for (const value of current) {
    const name = value.trim()
    if (!name) continue
    const key = name.toLocaleLowerCase()
    if (!map.has(key)) {
      map.set(key, { name, count: 0 })
    }
  }
  return sortNamedCounts([...map.values()], locale)
}

/** 元数据/NFO 标签：按影片命中次数聚合，次数降序 */
export function aggregateMetadataTagCounts(movies: readonly Movie[], locale: string): TagCountEntry[] {
  const map = new Map<string, number>()
  for (const m of movies) {
    for (const tag of m.tags) {
      const s = tag.trim()
      if (!s) continue
      map.set(s, (map.get(s) ?? 0) + 1)
    }
  }
  return [...map.entries()]
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => {
      if (b.count !== a.count) return b.count - a.count
      return a.tag.localeCompare(b.tag, locale, { numeric: true })
    })
}

/** 用户标签：按影片命中次数聚合，次数降序 */
export function aggregateUserTagCounts(movies: readonly Movie[], locale: string): TagCountEntry[] {
  const map = new Map<string, number>()
  for (const m of movies) {
    for (const tag of m.userTags) {
      const s = tag.trim()
      if (!s) continue
      map.set(s, (map.get(s) ?? 0) + 1)
    }
  }
  return [...map.entries()]
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => {
      if (b.count !== a.count) return b.count - a.count
      return a.tag.localeCompare(b.tag, locale, { numeric: true })
    })
}

/** 设置页顶部三张统计卡（入库数、标签种类、萃取帧条数）；仅标题 + 数值，无副说明 */
export function buildSettingsDashboardStats(
  movies: readonly Movie[],
  curatedFramesCount: number,
  locale: string,
): LibraryStat[] {
  const movieCount = movies.length
  const tagKinds = countUniqueTags(movies)
  const fmt = (n: number) => n.toLocaleString(locale)
  return [
    {
      labelKey: "stats.moviesInLibrary",
      value: fmt(movieCount),
    },
    {
      labelKey: "stats.tagKinds",
      value: fmt(tagKinds),
    },
    {
      labelKey: "stats.curatedFramesCount",
      value: fmt(curatedFramesCount),
    },
  ]
}

/** 侧边栏数字：大数缩写为 2.1k 形式 */
export function formatSidebarCount(n: number): string {
  if (n <= 0) return "0"
  if (n < 1000) return String(n)
  const k = n / 1000
  if (k < 10) {
    const rounded = Math.round(k * 10) / 10
    return Number.isInteger(rounded) ? `${rounded}k` : `${rounded.toFixed(1)}k`
  }
  return `${Math.round(k)}k`
}
