export type PlaybackStatsSnapshot = {
  audioBitrateKbps: number | null
  videoBitrateKbps: number | null
  currentBitrateKbps: number | null
  bandwidthEstimateKbps: number | null
  width: number | null
  height: number | null
  fps: number | null
}

export type HlsLevelStatsFields = {
  bitrate?: number
  width?: number
  height?: number
  frameRate?: number | string
  attrs?: Record<string, unknown>
}

export function createEmptyPlaybackStats(): PlaybackStatsSnapshot {
  return {
    audioBitrateKbps: null,
    videoBitrateKbps: null,
    currentBitrateKbps: null,
    bandwidthEstimateKbps: null,
    width: null,
    height: null,
    fps: null,
  }
}

export function toFiniteNumber(value: unknown): number | null {
  const num =
    typeof value === "number"
      ? value
      : typeof value === "string" && value.trim()
        ? Number(value)
        : NaN
  return Number.isFinite(num) ? num : null
}

export function applyVideoDimensionsToPlaybackStats(
  current: PlaybackStatsSnapshot,
  videoWidth: number,
  videoHeight: number,
): PlaybackStatsSnapshot {
  return {
    ...current,
    width: Number.isFinite(videoWidth) && videoWidth > 0 ? videoWidth : null,
    height: Number.isFinite(videoHeight) && videoHeight > 0 ? videoHeight : null,
  }
}

export function applyHlsLevelToPlaybackStats(
  current: PlaybackStatsSnapshot,
  level?: HlsLevelStatsFields | null,
): PlaybackStatsSnapshot {
  if (!level) return current
  const width = toFiniteNumber(level.width)
  const height = toFiniteNumber(level.height)
  const frameRate =
    toFiniteNumber(level.frameRate) ??
    toFiniteNumber(level.attrs?.["FRAME-RATE"]) ??
    toFiniteNumber(level.attrs?.FRAME_RATE)
  const videoBitrate = toFiniteNumber(level.bitrate)

  return {
    ...current,
    width: width && width > 0 ? width : current.width,
    height: height && height > 0 ? height : current.height,
    fps: frameRate && frameRate > 0 ? frameRate : current.fps,
    videoBitrateKbps:
      videoBitrate && videoBitrate > 0
        ? Math.round(videoBitrate / 1000)
        : current.videoBitrateKbps,
  }
}

export function applyHlsBandwidthEstimateToPlaybackStats(
  current: PlaybackStatsSnapshot,
  value: unknown,
): PlaybackStatsSnapshot {
  const bandwidthEstimate = toFiniteNumber(value)
  if (!bandwidthEstimate || bandwidthEstimate <= 0) {
    return current
  }

  return {
    ...current,
    bandwidthEstimateKbps: Math.round(bandwidthEstimate / 1000),
  }
}

export function applyHlsFragmentToPlaybackStats(
  current: PlaybackStatsSnapshot,
  data?: unknown,
): PlaybackStatsSnapshot {
  if (typeof data !== "object" || data === null) return current

  const stats =
    "stats" in data && typeof (data as { stats?: unknown }).stats === "object"
      ? ((data as { stats?: Record<string, unknown> }).stats ?? null)
      : null
  const frag =
    "frag" in data && typeof (data as { frag?: unknown }).frag === "object"
      ? ((data as { frag?: Record<string, unknown> }).frag ?? null)
      : null

  const loadedBytes =
    toFiniteNumber(stats?.loaded) ??
    toFiniteNumber(stats?.total) ??
    toFiniteNumber(frag?.loaded)
  const durationSec =
    toFiniteNumber(frag?.duration) ??
    toFiniteNumber(frag?.maxStartPts) ??
    null
  const bandwidthEstimate =
    toFiniteNumber(stats?.bwEstimate) ??
    toFiniteNumber(stats?.bandwidthEstimate) ??
    null

  let currentBitrateKbps = current.currentBitrateKbps
  if (loadedBytes && loadedBytes > 0 && durationSec && durationSec > 0) {
    const measuredBitrateKbps = Math.round((loadedBytes * 8) / durationSec / 1000)
    if (Number.isFinite(measuredBitrateKbps) && measuredBitrateKbps > 0) {
      currentBitrateKbps =
        currentBitrateKbps && currentBitrateKbps > 0
          ? Math.round(currentBitrateKbps * 0.65 + measuredBitrateKbps * 0.35)
          : measuredBitrateKbps
    }
  }

  return {
    ...current,
    currentBitrateKbps,
    bandwidthEstimateKbps:
      bandwidthEstimate && bandwidthEstimate > 0
        ? Math.round(bandwidthEstimate / 1000)
        : current.bandwidthEstimateKbps,
  }
}
