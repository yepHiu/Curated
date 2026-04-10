import type { CuratedFrameRecord } from "@/domain/curated-frame/types"

type CuratedFrameLike = Pick<CuratedFrameRecord, "id" | "movieId" | "positionSec" | "capturedAt">

export type CuratedFrameNearDuplicateGroup<T extends CuratedFrameLike = CuratedFrameLike> = {
  movieId: string
  items: T[]
}

export function findCuratedFrameNearDuplicateGroups<T extends CuratedFrameLike>(
  rows: readonly T[],
  thresholdSec: number,
): CuratedFrameNearDuplicateGroup<T>[] {
  if (!(thresholdSec > 0)) {
    return []
  }

  const byMovie = new Map<string, T[]>()
  for (const row of rows) {
    const movieId = row.movieId.trim()
    if (!movieId) {
      continue
    }
    const bucket = byMovie.get(movieId)
    if (bucket) {
      bucket.push(row)
    } else {
      byMovie.set(movieId, [row])
    }
  }

  const groups: CuratedFrameNearDuplicateGroup<T>[] = []
  for (const [movieId, items] of byMovie) {
    const ordered = [...items].sort((a, b) => {
      if (a.positionSec !== b.positionSec) {
        return a.positionSec - b.positionSec
      }
      return b.capturedAt.localeCompare(a.capturedAt)
    })
    let current: T[] = []
    for (const item of ordered) {
      const previous = current[current.length - 1]
      if (!previous || Math.abs(item.positionSec - previous.positionSec) <= thresholdSec) {
        current.push(item)
        continue
      }
      if (current.length > 1) {
        groups.push({ movieId, items: current })
      }
      current = [item]
    }
    if (current.length > 1) {
      groups.push({ movieId, items: current })
    }
  }

  return groups.sort((a, b) => {
    const left = a.items[0]?.capturedAt ?? ""
    const right = b.items[0]?.capturedAt ?? ""
    return right.localeCompare(left)
  })
}

export function buildCuratedFrameNearDuplicateIndex<T extends CuratedFrameLike>(
  groups: readonly CuratedFrameNearDuplicateGroup<T>[],
): Set<string> {
  const ids = new Set<string>()
  for (const group of groups) {
    for (const item of group.items) {
      ids.add(item.id)
    }
  }
  return ids
}
