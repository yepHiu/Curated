import { describe, expect, it } from "vitest"
import {
  applyHlsBandwidthEstimateToPlaybackStats,
  applyHlsFragmentToPlaybackStats,
  applyHlsLevelToPlaybackStats,
  applyVideoDimensionsToPlaybackStats,
  createEmptyPlaybackStats,
  toFiniteNumber,
} from "@/lib/player-playback-stats"

const baseStats = {
  audioBitrateKbps: null,
  videoBitrateKbps: 1200,
  currentBitrateKbps: null,
  bandwidthEstimateKbps: null,
  width: 1280,
  height: 720,
  fps: 24,
}

describe("player playback stats reducers", () => {
  it("creates an empty stats snapshot and parses finite numbers", () => {
    expect(createEmptyPlaybackStats()).toEqual({
      audioBitrateKbps: null,
      videoBitrateKbps: null,
      currentBitrateKbps: null,
      bandwidthEstimateKbps: null,
      width: null,
      height: null,
      fps: null,
    })
    expect(toFiniteNumber(12)).toBe(12)
    expect(toFiniteNumber(" 12.5 ")).toBe(12.5)
    expect(toFiniteNumber(" ")).toBeNull()
    expect(toFiniteNumber(Number.NaN)).toBeNull()
  })

  it("updates video dimensions from the media element dimensions", () => {
    expect(applyVideoDimensionsToPlaybackStats(baseStats, 1920, 1080)).toMatchObject({
      width: 1920,
      height: 1080,
    })
    expect(applyVideoDimensionsToPlaybackStats(baseStats, 0, Number.NaN)).toMatchObject({
      width: null,
      height: null,
    })
  })

  it("updates HLS level stats while preserving existing values for invalid fields", () => {
    expect(
      applyHlsLevelToPlaybackStats(baseStats, {
        width: 1920,
        height: 1080,
        frameRate: "29.97",
        bitrate: 3_500_000,
      }),
    ).toMatchObject({
      width: 1920,
      height: 1080,
      fps: 29.97,
      videoBitrateKbps: 3500,
    })
    expect(
      applyHlsLevelToPlaybackStats(baseStats, {
        width: 0,
        height: undefined,
        attrs: { "FRAME-RATE": "59.94" },
        bitrate: -1,
      }),
    ).toMatchObject({
      width: 1280,
      height: 720,
      fps: 59.94,
      videoBitrateKbps: 1200,
    })
  })

  it("updates HLS bandwidth estimates only when finite and positive", () => {
    expect(applyHlsBandwidthEstimateToPlaybackStats(baseStats, 4_200_000)).toMatchObject({
      bandwidthEstimateKbps: 4200,
    })
    expect(applyHlsBandwidthEstimateToPlaybackStats(baseStats, "bad")).toBe(baseStats)
    expect(applyHlsBandwidthEstimateToPlaybackStats(baseStats, 0)).toBe(baseStats)
  })

  it("derives smoothed fragment bitrate and bandwidth estimate", () => {
    const first = applyHlsFragmentToPlaybackStats(baseStats, {
      stats: { loaded: 1_000_000, bwEstimate: 5_500_000 },
      frag: { duration: 4 },
    })

    expect(first.currentBitrateKbps).toBe(2000)
    expect(first.bandwidthEstimateKbps).toBe(5500)

    const smoothed = applyHlsFragmentToPlaybackStats(
      { ...baseStats, currentBitrateKbps: 2000 },
      {
        stats: { total: 2_000_000 },
        frag: { duration: 4 },
      },
    )

    expect(smoothed.currentBitrateKbps).toBe(2700)
  })
})
