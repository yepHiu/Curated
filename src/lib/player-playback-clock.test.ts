import { describe, expect, it } from "vitest"
import {
  clearOptimisticSeekTargetIfSettled,
  hasAuthoritativeClockMoved,
  shouldEnterSeekWaitingState,
  shouldReconcileDisplayedPlaybackTime,
} from "@/lib/player-playback-clock"

describe("clearOptimisticSeekTargetIfSettled", () => {
  it("clears the optimistic seek once the authoritative clock reaches the target", () => {
    expect(clearOptimisticSeekTargetIfSettled(120, 120.4)).toBeNull()
  })

  it("clears stale optimistic seek state when playback has progressed away from the target", () => {
    expect(clearOptimisticSeekTargetIfSettled(120, 126)).toBeNull()
  })

  it("keeps the optimistic seek while the authoritative clock is still near the pending target", () => {
    expect(clearOptimisticSeekTargetIfSettled(120, 118.8)).toBe(120)
  })
})

describe("shouldReconcileDisplayedPlaybackTime", () => {
  it("reconciles when the displayed time drifts behind the authoritative media clock", () => {
    expect(
      shouldReconcileDisplayedPlaybackTime({
        displayedTimeSec: 120,
        authoritativeTimeSec: 126,
        optimisticSeekTargetSec: 120,
        isScrubbingProgress: false,
        isPlaybackWaiting: false,
      }),
    ).toBe(true)
  })

  it("does not reconcile while the user is actively scrubbing the progress slider", () => {
    expect(
      shouldReconcileDisplayedPlaybackTime({
        displayedTimeSec: 120,
        authoritativeTimeSec: 126,
        optimisticSeekTargetSec: 120,
        isScrubbingProgress: true,
        isPlaybackWaiting: false,
      }),
    ).toBe(false)
  })
})

describe("shouldEnterSeekWaitingState", () => {
  it("does not put direct playback into seek-buffering state before the media element reports real buffering", () => {
    expect(shouldEnterSeekWaitingState("direct")).toBe(false)
  })

  it("keeps HLS seeks in seek-buffering state because they may require stream window refill", () => {
    expect(shouldEnterSeekWaitingState("hls")).toBe(true)
  })
})

describe("hasAuthoritativeClockMoved", () => {
  it("treats normal playback-sized clock movement as movement for reconciliation", () => {
    expect(hasAuthoritativeClockMoved(20, 20.1)).toBe(true)
  })

  it("ignores tiny clock jitter", () => {
    expect(hasAuthoritativeClockMoved(20, 20.01)).toBe(false)
  })
})
