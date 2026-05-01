export type PlaybackTimelineDescriptorFields = {
  mode?: string | null
  startPositionSec?: number | null
  durationSec?: number | null
}

export function formatPlaybackClock(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "00:00"
  const s = Math.floor(seconds % 60)
  const m = Math.floor(seconds / 60) % 60
  const h = Math.floor(seconds / 3600)
  const pad = (n: number) => String(n).padStart(2, "0")
  if (h > 0) return `${pad(h)}:${pad(m)}:${pad(s)}`
  return `${pad(m)}:${pad(s)}`
}

export function getDescriptorDurationSec(
  descriptor: PlaybackTimelineDescriptorFields | null | undefined,
): number {
  const raw = descriptor?.durationSec
  return raw != null && Number.isFinite(raw) && raw > 0 ? raw : 0
}

export function resolvePlaybackTotalDurationSec(
  descriptor: PlaybackTimelineDescriptorFields | null | undefined,
  mediaDurationSec: number,
): number {
  const finiteMediaDuration =
    Number.isFinite(mediaDurationSec) && mediaDurationSec > 0 ? mediaDurationSec : 0
  return Math.max(finiteMediaDuration, getDescriptorDurationSec(descriptor))
}

export function normalizeProgressTargetSec(rawValue: number, totalDurationSec: number): number {
  const normalized = Number.isFinite(rawValue) ? rawValue : 0
  if (totalDurationSec <= 0) {
    return Math.max(0, normalized)
  }
  return Math.min(Math.max(0, normalized), totalDurationSec)
}

export function getPlaybackTimelineOffsetSec(
  descriptor: PlaybackTimelineDescriptorFields | null | undefined,
): number {
  if (!descriptor || descriptor.mode !== "hls") {
    return 0
  }
  const offset = Number(descriptor.startPositionSec ?? 0)
  return Number.isFinite(offset) && offset > 0 ? offset : 0
}

export function getAbsolutePlaybackTimeSec(
  localTimeSec: number,
  descriptor: PlaybackTimelineDescriptorFields | null | undefined,
): number {
  const local = Number.isFinite(localTimeSec) && localTimeSec > 0 ? localTimeSec : 0
  return local + getPlaybackTimelineOffsetSec(descriptor)
}

export function clampAbsolutePlaybackTarget(targetSec: number, totalDurationSec: number): number {
  const normalized = Number.isFinite(targetSec) ? targetSec : 0
  if (totalDurationSec <= 0) {
    return Math.max(0, normalized)
  }
  return Math.min(Math.max(0, normalized), Math.max(0, totalDurationSec - 0.25))
}
