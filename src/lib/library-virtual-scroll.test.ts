import { describe, expect, it } from "vitest"
import {
  estimateVirtualMovieChunkHeight,
  estimateVirtualMovieCardHeight,
  getVirtualMovieFocusChunkIndex,
  resolveVirtualMoviePosterLoadPolicy,
} from "@/lib/library-virtual-scroll"

describe("estimateVirtualMovieCardHeight", () => {
  it("accounts for poster aspect ratio and card chrome", () => {
    expect(estimateVirtualMovieCardHeight(188)).toBeGreaterThan(360)
    expect(estimateVirtualMovieCardHeight(304)).toBeGreaterThan(540)
  })
})

describe("estimateVirtualMovieChunkHeight", () => {
  it("keeps chunk height safely above the old flat estimate on wide layouts", () => {
    expect(
      estimateVirtualMovieChunkHeight({
        containerWidth: 1440,
        columnCount: 5,
        rowsPerChunk: 4,
        gapPx: 20,
      }),
    ).toBeGreaterThan(1400)
  })
})

describe("getVirtualMovieFocusChunkIndex", () => {
  it("uses the viewport midpoint so scrolling prewarms upcoming chunks", () => {
    expect(
      getVirtualMovieFocusChunkIndex({
        scrollTop: 0,
        viewportHeight: 900,
        chunkHeight: 1500,
      }),
    ).toBe(0)

    expect(
      getVirtualMovieFocusChunkIndex({
        scrollTop: 1200,
        viewportHeight: 900,
        chunkHeight: 1500,
      }),
    ).toBe(1)
  })
})

describe("resolveVirtualMoviePosterLoadPolicy", () => {
  it("keeps the current chunk eager and nearby chunks warmer than far chunks", () => {
    expect(resolveVirtualMoviePosterLoadPolicy(4, 4)).toEqual({
      loading: "eager",
      fetchPriority: "high",
    })

    expect(resolveVirtualMoviePosterLoadPolicy(5, 4)).toEqual({
      loading: "eager",
      fetchPriority: "auto",
    })

    expect(resolveVirtualMoviePosterLoadPolicy(8, 4)).toEqual({
      loading: "lazy",
      fetchPriority: "low",
    })
  })
})
