export const DEFAULT_PLAYBACK_FRAME_RATE = 30

const MIN_PLAYBACK_FRAME_RATE = 1
const MAX_PLAYBACK_FRAME_RATE = 240

function validFrameRate(value: number | null | undefined): value is number {
  return typeof value === "number" && Number.isFinite(value) &&
    value >= MIN_PLAYBACK_FRAME_RATE && value <= MAX_PLAYBACK_FRAME_RATE
}

/**
 * Browsers do not expose a reliable source-frame stepping API. Seek by the
 * stream's known/measured frame duration, with a conventional 30 fps fallback
 * until that information is available.
 */
export function getPlaybackFrameStepSec(frameRate: number | null | undefined): number {
  return 1 / (validFrameRate(frameRate) ? frameRate : DEFAULT_PLAYBACK_FRAME_RATE)
}

type FrameSample = { mediaTime: number; presentedFrames: number }

/** Source-frame estimate uses media timestamps, never elapsed wall time.
 * Pause/seek interrupts discard the sample boundary but retain the estimate.
 * A manifest frame rate takes precedence over measured presentation cadence.
 */
export function createPlaybackFrameStepper() {
  let previous: FrameSample | null = null
  let knownFrameRate: number | null = null
  const durations: number[] = []
  return {
    setFrameRate(value: number | null | undefined) {
      knownFrameRate = validFrameRate(value) ? value : null
    },
    observe(sample: FrameSample, active: boolean) {
      if (!active || !Number.isFinite(sample.mediaTime) || !Number.isFinite(sample.presentedFrames)) {
        previous = null
        return
      }
      const last = previous
      previous = { ...sample }
      if (!last) return
      const frames = sample.presentedFrames - last.presentedFrames
      const elapsed = sample.mediaTime - last.mediaTime
      if (frames <= 0 || elapsed <= 0) return
      const duration = elapsed / frames
      if (!validFrameRate(1 / duration)) return
      durations.push(duration)
      if (durations.length > 12) durations.shift()
    },
    stepSec() {
      if (knownFrameRate != null) return getPlaybackFrameStepSec(knownFrameRate)
      if (!durations.length) return getPlaybackFrameStepSec(null)
      const sorted = [...durations].sort((a, b) => a - b)
      return sorted[Math.floor(sorted.length / 2)]!
    },
    interrupt() { previous = null },
    reset() {
      previous = null
      knownFrameRate = null
      durations.length = 0
    },
  }
}
