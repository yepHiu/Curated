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

/** 设置页顶部三张统计卡（入库数、标签种类、已播放去重部数）；文案 key 由 i18n 解析 */
export function buildSettingsDashboardStats(
  movies: readonly Movie[],
  playedUniqueCount: number,
  locale: string,
): LibraryStat[] {
  const movieCount = movies.length
  const tagKinds = countUniqueTags(movies)
  const fmt = (n: number) => n.toLocaleString(locale)
  return [
    {
      labelKey: "stats.moviesInLibrary",
      value: fmt(movieCount),
      detailKey: "stats.moviesInLibraryDetail",
    },
    {
      labelKey: "stats.tagKinds",
      value: fmt(tagKinds),
      detailKey: "stats.tagKindsDetail",
    },
    {
      labelKey: "stats.playedMovies",
      value: fmt(playedUniqueCount),
      detailKey: "stats.playedMoviesDetail",
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
