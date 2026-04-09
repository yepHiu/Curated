import type { PlaybackDescriptorDTO } from "@/api/types"

function normalizePlaybackSecond(value: number | undefined): number | undefined {
  if (!Number.isFinite(value) || value == null || value < 0) {
    return undefined
  }
  return value
}

export function resolveDescriptorPlaybackTargetSec(
  descriptor: PlaybackDescriptorDTO | null | undefined,
): number | undefined {
  if (!descriptor) return undefined
  return (
    normalizePlaybackSecond(descriptor.resumePositionSec) ??
    normalizePlaybackSecond(descriptor.startPositionSec)
  )
}

export function descriptorMatchesRequestedPlaybackTarget(
  requestedTargetSec: number | undefined,
  descriptor: PlaybackDescriptorDTO | null | undefined,
  toleranceSec: number = 1,
): boolean {
  const normalizedRequested = normalizePlaybackSecond(requestedTargetSec)
  if (normalizedRequested === undefined) return false
  const descriptorTarget = resolveDescriptorPlaybackTargetSec(descriptor)
  if (descriptorTarget === undefined) return false
  return Math.abs(descriptorTarget - normalizedRequested) <= Math.max(0, toleranceSec)
}

export function resolvePreferredPlaybackTargetSec(
  requestedTargetSec: number | undefined,
  descriptor: PlaybackDescriptorDTO | null | undefined,
  storedProgressSec: number | undefined,
): number | undefined {
  return (
    normalizePlaybackSecond(requestedTargetSec) ??
    resolveDescriptorPlaybackTargetSec(descriptor) ??
    normalizePlaybackSecond(storedProgressSec)
  )
}

export function resolveHlsLocalSeekTargetSec(
  targetAbsoluteSec: number | undefined,
  sessionStartSec: number | undefined,
): number | undefined {
  const normalizedTarget = normalizePlaybackSecond(targetAbsoluteSec)
  if (normalizedTarget === undefined) return undefined
  const normalizedStart = normalizePlaybackSecond(sessionStartSec) ?? 0
  return Math.max(0, normalizedTarget - normalizedStart)
}
