import { describe, expect, it, vi } from "vitest"

vi.mock("@/lib/playback-progress-storage", () => ({
  getResumeSecondsForOpenPlayer: () => undefined,
}))

import { buildPlayerRouteFromCuratedFrame, buildPlayerRouteFromHistory } from "@/lib/player-route"

describe("buildPlayerRouteFromHistory", () => {
  it.each(["library", "fc2"] as const)("keeps file and resume time with a shared history return for %s", (sourceMode) => {
    // 历史入口恢复具体分部与时间，作品类别只决定播放器队列。
    expect(buildPlayerRouteFromHistory("movie-1", 240.8, "part-2", sourceMode)).toEqual({
      name: "player",
      params: { id: "movie-1" },
      query: {
        autoplay: "1",
        back: "history",
        ...(sourceMode === "fc2" ? { browse: "fc2" } : {}),
        fileId: "part-2",
        t: "240",
      },
    })
  })
})

describe("buildPlayerRouteFromCuratedFrame", () => {
  it("preserves fractional seconds for frame-based playback jumps", () => {
    expect(buildPlayerRouteFromCuratedFrame("movie-1", 123.456)).toEqual({
      name: "player",
      params: { id: "movie-1" },
      query: {
        autoplay: "1",
        back: "curated-frames",
        t: "123.456",
      },
    })
  })
})
