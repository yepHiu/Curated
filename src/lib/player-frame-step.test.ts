import { describe, expect, it } from "vitest"
import {
  DEFAULT_PLAYBACK_FRAME_RATE,
  createPlaybackFrameStepper,
  getPlaybackFrameStepSec,
} from "@/lib/player-frame-step"

describe("player frame stepping", () => {
  it("uses the stream frame rate when it is available", () => {
    expect(getPlaybackFrameStepSec(24)).toBeCloseTo(1 / 24)
    expect(getPlaybackFrameStepSec(59.94)).toBeCloseTo(1 / 59.94)
  })

  it("falls back to 30 fps when the frame rate is unavailable or implausible", () => {
    const fallbackStep = 1 / DEFAULT_PLAYBACK_FRAME_RATE

    expect(getPlaybackFrameStepSec(null)).toBe(fallbackStep)
    expect(getPlaybackFrameStepSec(undefined)).toBe(fallbackStep)
    expect(getPlaybackFrameStepSec(0)).toBe(fallbackStep)
    expect(getPlaybackFrameStepSec(300)).toBe(fallbackStep)
  })
})

describe("source frame duration estimation", () => {
  it("uses media timestamps across playback rates and missed callbacks", () => {
    const stepper = createPlaybackFrameStepper()
    stepper.observe({ mediaTime: 10, presentedFrames: 1 }, true)
    stepper.observe({ mediaTime: 10 + 3 / 24, presentedFrames: 4 }, true)
    expect(stepper.stepSec()).toBeCloseTo(1 / 24)
  })

  it("retains the source estimate across pause and seek frames", () => {
    const stepper = createPlaybackFrameStepper()
    stepper.observe({ mediaTime: 10, presentedFrames: 1 }, true)
    stepper.observe({ mediaTime: 10 + 1 / 24, presentedFrames: 2 }, true)
    stepper.observe({ mediaTime: 50, presentedFrames: 3 }, false)
    stepper.observe({ mediaTime: 50 + 1 / 24, presentedFrames: 4 }, false)
    expect(stepper.stepSec()).toBeCloseTo(1 / 24)
    stepper.interrupt()
    stepper.observe({ mediaTime: 80, presentedFrames: 5 }, true)
    expect(stepper.stepSec()).toBeCloseTo(1 / 24)
  })

  it("prefers manifest fps and resets the estimate for a new source", () => {
    const stepper = createPlaybackFrameStepper()
    stepper.observe({ mediaTime: 1, presentedFrames: 1 }, true)
    stepper.observe({ mediaTime: 1.1, presentedFrames: 2 }, true)
    stepper.setFrameRate(59.94)
    expect(stepper.stepSec()).toBeCloseTo(1 / 59.94)
    stepper.reset()
    expect(stepper.stepSec()).toBe(1 / 30)
  })

  it("ignores invalid timestamps and one isolated dropped frame", () => {
    const stepper = createPlaybackFrameStepper()
    stepper.observe({ mediaTime: 0, presentedFrames: 0 }, true)
    stepper.observe({ mediaTime: 1 / 24, presentedFrames: 1 }, true)
    stepper.observe({ mediaTime: 2 / 24, presentedFrames: 2 }, true)
    stepper.observe({ mediaTime: 4 / 24, presentedFrames: 3 }, true)
    stepper.observe({ mediaTime: NaN, presentedFrames: 4 }, true)
    expect(stepper.stepSec()).toBeCloseTo(1 / 24)
  })
})

