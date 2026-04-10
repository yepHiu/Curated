import { describe, expect, it } from "vitest"
import { buildCuratedFrameNearDuplicateIndex, findCuratedFrameNearDuplicateGroups } from "@/lib/curated-frames/near-duplicates"
import type { CuratedFrameRecord } from "@/domain/curated-frame/types"

function frame(id: string, movieId: string, positionSec: number): CuratedFrameRecord {
  return {
    id,
    movieId,
    title: movieId,
    code: movieId,
    actors: [],
    positionSec,
    capturedAt: `2026-04-11T00:00:0${id.length}Z`,
    tags: [],
  }
}

describe("curated frame near duplicate detection", () => {
  it("groups frames from the same movie when their timestamps are within the threshold", () => {
    const groups = findCuratedFrameNearDuplicateGroups([
      frame("a", "movie-1", 12),
      frame("b", "movie-1", 14.9),
      frame("c", "movie-1", 18.1),
      frame("d", "movie-2", 12),
    ], 3)

    expect(groups).toHaveLength(1)
    expect(groups[0]?.items.map((item) => item.id)).toEqual(["a", "b"])
    expect(buildCuratedFrameNearDuplicateIndex(groups)).toEqual(new Set(["a", "b"]))
  })

  it("does not treat frames from different movies as duplicates", () => {
    const groups = findCuratedFrameNearDuplicateGroups([
      frame("a", "movie-1", 12),
      frame("b", "movie-2", 12.1),
    ], 3)

    expect(groups).toEqual([])
  })
})
