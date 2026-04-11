import type { CuratedFrameFacetItemDTO } from "@/api/types"

export function buildCuratedFrameTagFacets(
  rows: ReadonlyArray<{ tags: readonly string[] }>,
  locale: string,
): CuratedFrameFacetItemDTO[] {
  const counts = new Map<string, number>()
  for (const row of rows) {
    for (const raw of row.tags) {
      const tag = raw.trim()
      if (!tag) {
        continue
      }
      counts.set(tag, (counts.get(tag) ?? 0) + 1)
    }
  }
  return [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, locale, { numeric: true }))
}

export function visibleCuratedFrameTagFacets<T>(
  facets: readonly T[],
  limit: number,
  expanded: boolean,
): T[] {
  return expanded ? [...facets] : facets.slice(0, limit)
}
