type PlaybackTimeReconciliationInput = {
  displayedTimeSec: number
  authoritativeTimeSec: number
  optimisticSeekTargetSec: number | null
  isScrubbingProgress: boolean
  isPlaybackWaiting: boolean
  authoritativeClockAdvanced?: boolean
  toleranceSec?: number
}

type PlaybackMode = "direct" | "hls" | (string & {})

export function shouldEnterSeekWaitingState(mode: PlaybackMode | null | undefined): boolean {
  return mode === "hls"
}

export function hasAuthoritativeClockMoved(
  previousTimeSec: number | null | undefined,
  currentTimeSec: number,
  movementThresholdSec: number = 0.05,
): boolean {
  if (previousTimeSec == null || !Number.isFinite(previousTimeSec)) return false
  if (!Number.isFinite(currentTimeSec)) return false
  return Math.abs(currentTimeSec - previousTimeSec) >= Math.max(0, movementThresholdSec)
}

export function clearOptimisticSeekTargetIfSettled(
  optimisticSeekTargetSec: number | null,
  authoritativeTimeSec: number,
  toleranceSec: number = 1,
  staleDriftSec: number = 4,
): number | null {
  if (optimisticSeekTargetSec == null) return null
  const authoritative = Number.isFinite(authoritativeTimeSec) ? authoritativeTimeSec : 0
  const driftSec = Math.abs(authoritative - optimisticSeekTargetSec)
  if (driftSec <= Math.max(0, toleranceSec)) return null
  if (driftSec >= Math.max(staleDriftSec, toleranceSec * 2)) return null
  return optimisticSeekTargetSec
}

export function shouldReconcileDisplayedPlaybackTime(
  input: PlaybackTimeReconciliationInput,
): boolean {
  if (input.isScrubbingProgress) return false

  const toleranceSec = Math.max(0, input.toleranceSec ?? 0.75)
  const displayed = Number.isFinite(input.displayedTimeSec) ? input.displayedTimeSec : 0
  const authoritative = Number.isFinite(input.authoritativeTimeSec) ? input.authoritativeTimeSec : 0
  if (Math.abs(authoritative - displayed) <= toleranceSec) return false

  if (
    input.isPlaybackWaiting &&
    input.optimisticSeekTargetSec != null &&
    input.authoritativeClockAdvanced !== true
  ) {
    return false
  }

  return true
}
