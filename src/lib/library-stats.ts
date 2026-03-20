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

export function countUniqueTags(movies: readonly Movie[]): number {
  const set = new Set<string>()
  for (const m of movies) {
    for (const tag of m.tags) {
      const s = tag.trim()
      if (s) set.add(s)
    }
  }
  return set.size
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
