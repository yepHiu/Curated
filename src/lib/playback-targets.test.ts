import { describe, expect, it } from "vitest"
import {
  descriptorMatchesRequestedPlaybackTarget,
  isNearPlaybackEnd,
  resolveHlsLocalSeekTargetSec,
  resolvePreferredPlaybackTargetSec,
} from "@/lib/playback-targets"

describe("isNearPlaybackEnd", () => {
  it("treats progress at or above 95% as near the end", () => {
    expect(isNearPlaybackEnd(95, 100)).toBe(true)
    expect(isNearPlaybackEnd(100, 100)).toBe(true)
    expect(isNearPlaybackEnd(94.9, 100)).toBe(false)
  })

  it("ignores missing or invalid duration and position", () => {
    expect(isNearPlaybackEnd(100, 0)).toBe(false)
    expect(isNearPlaybackEnd(undefined, 100)).toBe(false)
    expect(isNearPlaybackEnd(100, undefined)).toBe(false)
  })
})

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

  it("rejects stale HLS descriptors outside the requested resume tolerance", () => {
    expect(
      descriptorMatchesRequestedPlaybackTarget(
        300,
        {
          movieId: "movie-1",
          mode: "hls",
          url: "/api/playback/sessions/demo/hls/index.m3u8",
          startPositionSec: 240,
          resumePositionSec: undefined,
          canDirectPlay: false,
        },
        1,
      ),
    ).toBe(false)
  })
})

describe("resolvePreferredPlaybackTargetSec", () => {
  it("uses the route resume query before descriptor and stored progress targets", () => {
    expect(
      resolvePreferredPlaybackTargetSec(
        300,
        {
          movieId: "movie-1",
          mode: "direct",
          url: "/api/library/movies/movie-1/play",
          startPositionSec: 120,
          resumePositionSec: 180,
          canDirectPlay: true,
        },
        90,
      ),
    ).toBe(300)
  })

  it("falls back through descriptor resume, descriptor start, then stored progress", () => {
    expect(
      resolvePreferredPlaybackTargetSec(undefined, {
        movieId: "movie-1",
        mode: "hls",
        url: "/api/playback/sessions/demo/hls/index.m3u8",
        startPositionSec: 120,
        resumePositionSec: 180,
        canDirectPlay: false,
      }, 90),
    ).toBe(180)

    expect(
      resolvePreferredPlaybackTargetSec(undefined, {
        movieId: "movie-1",
        mode: "hls",
        url: "/api/playback/sessions/demo/hls/index.m3u8",
        startPositionSec: 120,
        resumePositionSec: undefined,
        canDirectPlay: false,
      }, 90),
    ).toBe(120)

    expect(resolvePreferredPlaybackTargetSec(undefined, null, 90)).toBe(90)
  })

  it("ignores invalid resume candidates", () => {
    expect(
      resolvePreferredPlaybackTargetSec(Number.NaN, {
        movieId: "movie-1",
        mode: "direct",
        url: "/api/library/movies/movie-1/play",
        startPositionSec: -1,
        resumePositionSec: undefined,
        canDirectPlay: true,
      }, -10),
    ).toBeUndefined()
  })

  it("drops near-end resume targets so playback restarts from the beginning", () => {
    expect(
      resolvePreferredPlaybackTargetSec(
        undefined,
        {
          movieId: "movie-1",
          mode: "direct",
          url: "/api/library/movies/movie-1/play",
          resumePositionSec: 118,
          durationSec: 120,
          canDirectPlay: true,
        },
        118,
      ),
    ).toBeUndefined()

    expect(
      resolvePreferredPlaybackTargetSec(undefined, null, 99, 100),
    ).toBeUndefined()

    expect(
      resolvePreferredPlaybackTargetSec(50, null, 99, 100),
    ).toBe(50)
  })
})

describe("resolveHlsLocalSeekTargetSec", () => {
  it("seeks within the HLS session from the actual session start", () => {
    expect(resolveHlsLocalSeekTargetSec(123.456, 121.456)).toBeCloseTo(2, 6)
  })

  it("clamps targets before the HLS session start to the session origin", () => {
    expect(resolveHlsLocalSeekTargetSec(100, 120)).toBe(0)
  })

  it("returns undefined for invalid absolute targets", () => {
    expect(resolveHlsLocalSeekTargetSec(Number.NaN, 120)).toBeUndefined()
    expect(resolveHlsLocalSeekTargetSec(-1, 120)).toBeUndefined()
  })
})
