/**
 * Browser playback capability reporting.
 *
 * The backend's direct-play whitelist cannot know whether *this* browser can
 * decode HEVC or AV1 in an MP4 container (support varies by browser, OS, and
 * hardware). Probing `canPlayType` once per tab session and sending the result
 * as `clientVideoCodecs` on the playback descriptor request lets the backend
 * hand out HLS for sources the video element would fail to decode, instead of
 * recovering through the decode-error fallback path.
 */

const CAPABILITIES_STORAGE_KEY = "curated-playback-capabilities-v1"

export type PlaybackCapabilities = {
  /** mp4-family video codecs this browser reports it can decode. */
  mp4VideoCodecs: string[]
}

type CodecProbe = {
  /** Normalized codec name sent to the backend. */
  name: string
  /** Full MIME type with codecs parameter for canPlayType. */
  canPlayTypeSource: string
}

const MP4_VIDEO_CODEC_PROBES: CodecProbe[] = [
  { name: "h264", canPlayTypeSource: 'video/mp4; codecs="avc1.42E01E"' },
  { name: "hevc", canPlayTypeSource: 'video/mp4; codecs="hvc1.1.6.L93.B0"' },
  { name: "av1", canPlayTypeSource: 'video/mp4; codecs="av01.0.08M.08"' },
]

let cachedCapabilities: PlaybackCapabilities | null = null

export function resolvePlaybackCapabilities(): PlaybackCapabilities {
  if (cachedCapabilities) {
    return cachedCapabilities
  }
  cachedCapabilities = readStoredCapabilities() ?? probePlaybackCapabilities()
  persistCapabilities(cachedCapabilities)
  return cachedCapabilities
}

/** Serialized query value for `clientVideoCodecs`, or null when nothing to send. */
export function clientVideoCodecsQueryParam(capabilities: PlaybackCapabilities): string | null {
  if (capabilities.mp4VideoCodecs.length === 0) {
    return null
  }
  return capabilities.mp4VideoCodecs.join(",")
}

function probePlaybackCapabilities(): PlaybackCapabilities {
  if (typeof document === "undefined") {
    return { mp4VideoCodecs: [] }
  }
  const video = document.createElement("video")
  const mp4VideoCodecs = MP4_VIDEO_CODEC_PROBES.filter(
    (probe) => typeof video.canPlayType === "function" && video.canPlayType(probe.canPlayTypeSource) !== "",
  ).map((probe) => probe.name)
  return { mp4VideoCodecs }
}

function readStoredCapabilities(): PlaybackCapabilities | null {
  try {
    const raw = sessionStorage.getItem(CAPABILITIES_STORAGE_KEY)
    if (!raw) {
      return null
    }
    const parsed = JSON.parse(raw) as Partial<PlaybackCapabilities>
    if (!Array.isArray(parsed.mp4VideoCodecs)) {
      return null
    }
    return {
      mp4VideoCodecs: parsed.mp4VideoCodecs.filter(
        (codec): codec is string => typeof codec === "string" && MP4_VIDEO_CODEC_PROBES.some((probe) => probe.name === codec),
      ),
    }
  } catch {
    return null
  }
}

function persistCapabilities(capabilities: PlaybackCapabilities): void {
  try {
    sessionStorage.setItem(CAPABILITIES_STORAGE_KEY, JSON.stringify(capabilities))
  } catch {
    // Storage can be unavailable (private mode); probing again is cheap.
  }
}

/** Test-only reset of the memoized capability snapshot. */
export function resetPlaybackCapabilitiesCacheForTest(): void {
  cachedCapabilities = null
  try {
    sessionStorage.removeItem(CAPABILITIES_STORAGE_KEY)
  } catch {
    // Ignore storage failures in tests.
  }
}
