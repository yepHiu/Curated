import { describe, expect, it } from "vitest"
import {
  clampAbsolutePlaybackTarget,
  formatPlaybackClock,
  getAbsolutePlaybackTimeSec,
  getDescriptorDurationSec,
  getPlaybackTimelineOffsetSec,
  normalizeProgressTargetSec,
  resolvePlaybackTotalDurationSec,
} from "@/lib/player-playback-timeline"

describe("player playback timeline helpers", () => {
  it("formats playback clocks with invalid values falling back to zero", () => {
    expect(formatPlaybackClock(Number.NaN)).toBe("00:00")
    expect(formatPlaybackClock(-1)).toBe("00:00")
    expect(formatPlaybackClock(65.8)).toBe("01:05")
    expect(formatPlaybackClock(3661)).toBe("01:01:01")
  })

  it("resolves descriptor and media durations defensively", () => {
    expect(getDescriptorDurationSec(null)).toBe(0)
    expect(getDescriptorDurationSec({ durationSec: -1 })).toBe(0)
    expect(getDescriptorDurationSec({ durationSec: Number.NaN })).toBe(0)
    expect(getDescriptorDurationSec({ durationSec: 120 })).toBe(120)
    expect(resolvePlaybackTotalDurationSec({ durationSec: 120 }, 90)).toBe(120)
    expect(resolvePlaybackTotalDurationSec({ durationSec: 120 }, 150)).toBe(150)
    expect(resolvePlaybackTotalDurationSec(null, Number.NaN)).toBe(0)
  })

  it("normalizes progress targets against the known duration", () => {
    expect(normalizeProgressTargetSec(Number.NaN, 100)).toBe(0)
    expect(normalizeProgressTargetSec(-5, 100)).toBe(0)
    expect(normalizeProgressTargetSec(120, 100)).toBe(100)
    expect(normalizeProgressTargetSec(12, 0)).toBe(12)
  })

  it("maps local HLS time to absolute playback time", () => {
    expect(getPlaybackTimelineOffsetSec(null)).toBe(0)
    expect(getPlaybackTimelineOffsetSec({ mode: "direct", startPositionSec: 50 })).toBe(0)
    expect(getPlaybackTimelineOffsetSec({ mode: "hls", startPositionSec: -1 })).toBe(0)
    expect(getPlaybackTimelineOffsetSec({ mode: "hls", startPositionSec: 50 })).toBe(50)
    expect(getAbsolutePlaybackTimeSec(12, { mode: "hls", startPositionSec: 50 })).toBe(62)
    expect(getAbsolutePlaybackTimeSec(Number.NaN, { mode: "hls", startPositionSec: 50 })).toBe(50)
  })

  it("clamps absolute seek targets to the available playback range", () => {
    expect(clampAbsolutePlaybackTarget(-5, 0)).toBe(0)
    expect(clampAbsolutePlaybackTarget(12, 0)).toBe(12)
    expect(clampAbsolutePlaybackTarget(-5, 100)).toBe(0)
    expect(clampAbsolutePlaybackTarget(120, 100)).toBe(99.75)
  })
})
