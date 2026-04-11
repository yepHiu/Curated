import { describe, expect, it, vi } from "vitest"

vi.mock("@/lib/playback-progress-storage", () => ({
  getResumeSecondsForOpenPlayer: () => undefined,
}))

import { buildPlayerRouteFromCuratedFrame } from "@/lib/player-route"

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
