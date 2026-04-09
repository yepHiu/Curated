import { describe, expect, it } from "vitest"
import {
  descriptorMatchesRequestedPlaybackTarget,
  resolveHlsLocalSeekTargetSec,
} from "@/lib/playback-targets"

describe("descriptorMatchesRequestedPlaybackTarget", () => {
  it("prefers exact resumePositionSec over HLS session timeline origin", () => {
    expect(
      descriptorMatchesRequestedPlaybackTarget(123.456, {
        movieId: "movie-1",
        mode: "hls",
        url: "/api/playback/sessions/demo/hls/index.m3u8",
        startPositionSec: 121.456,
        resumePositionSec: 123.456,
        canDirectPlay: false,
      }),
    ).toBe(true)
  })
})

describe("resolveHlsLocalSeekTargetSec", () => {
  it("seeks within the HLS session from the actual session start", () => {
    expect(resolveHlsLocalSeekTargetSec(123.456, 121.456)).toBeCloseTo(2, 6)
  })
})
