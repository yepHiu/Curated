import type { CuratedFrameDbRow } from "@/lib/curated-frames/db"

function pad2(n: number) {
  return String(n).padStart(2, "0")
}

function formatTimeForHaystack(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return ""
  const s = Math.floor(seconds % 60)
  const m = Math.floor(seconds / 60) % 60
  const h = Math.floor(seconds / 3600)
  if (h > 0) return `${h}:${pad2(m)}:${pad2(s)} ${pad2(m)}:${pad2(s)}`
  return `${m}:${pad2(s)}`
}

export function curatedFrameHaystack(row: CuratedFrameDbRow): string {
  const parts = [
    row.title,
    row.code,
    row.actors.join(" "),
    row.tags.join(" "),
    String(row.positionSec),
    formatTimeForHaystack(row.positionSec),
    row.capturedAt,
    row.movieId,
  ]
  return parts.join(" ").toLowerCase()
}

export function filterCuratedFramesByQuery(
  rows: CuratedFrameDbRow[],
  query: string,
): CuratedFrameDbRow[] {
  const q = query.trim().toLowerCase()
  if (!q) return rows
  return rows.filter((row) => curatedFrameHaystack(row).includes(q))
}

/**
 * 仅从萃取帧库收集已出现过的标签，用于联想；与影片元数据 / 用户标签完全隔离。
 */
export function buildCuratedFrameTagSuggestionPool(rows: readonly CuratedFrameDbRow[]): string[] {
  const set = new Set<string>()
  for (const row of rows) {
    for (const t of row.tags) {
      const s = t.trim()
      if (s) set.add(s)
    }
  }
  return [...set].sort((a, b) => a.localeCompare(b, "zh-CN", { numeric: true }))
}
