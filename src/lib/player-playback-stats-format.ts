export type PlaybackSourceFormatFields = {
  sourceContainer?: string | null
  sourceVideoCodec?: string | null
  sourceAudioCodec?: string | null
}

export function formatBitrateLabel(kbps: number | null): string {
  if (!kbps || kbps <= 0) return "N/A"
  if (kbps >= 1000) return `${(kbps / 1000).toFixed(2)} Mbps`
  return `${Math.round(kbps)} kbps`
}

export function formatResolutionLabel(width: number | null, height: number | null): string {
  if (!width || !height) return "N/A"
  return `${width} \u00d7 ${height}`
}

export function formatFpsLabel(fps: number | null): string {
  if (!fps || fps <= 0) return "N/A"
  return `${fps.toFixed(fps >= 100 ? 0 : 2)} fps`
}

export function formatTimecodeLabel(seconds: number | null | undefined): string {
  if (seconds == null || !Number.isFinite(seconds) || seconds < 0) return "N/A"
  return formatClockLabel(seconds)
}

export function formatPercentLabel(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return "N/A"
  return `${Math.max(0, Math.min(100, value)).toFixed(1)}%`
}

export function formatCountLabel(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return "N/A"
  return String(Math.max(0, Math.trunc(value)))
}

export function formatReasonLabel(reason: string | null | undefined): string {
  const text = reason?.trim() || ""
  if (!text) return "N/A"
  return text
}

export function formatSessionKindLabel(sessionKind: string | null | undefined): string {
  switch ((sessionKind ?? "").trim().toLowerCase()) {
    case "direct-file":
      return "Direct File"
    case "remux-hls":
      return "Remux HLS"
    case "transcode-hls":
      return "Transcode HLS"
    default:
      return "N/A"
  }
}

export function formatSourceFormatLabel(
  descriptor: PlaybackSourceFormatFields | null | undefined,
): string {
  const container = descriptor?.sourceContainer?.trim()
  const videoCodec = descriptor?.sourceVideoCodec?.trim()
  const audioCodec = descriptor?.sourceAudioCodec?.trim()
  const parts = [container, videoCodec, audioCodec].filter((value) => Boolean(value))
  if (parts.length === 0) return "N/A"
  return parts.join(" / ")
}

export function formatTranscodeProfileLabel(profile: string | null | undefined): string {
  switch ((profile ?? "").trim().toLowerCase()) {
    case "remux_copy":
      return "FFmpeg Stream Copy"
    case "h264_amf":
      return "AMD AMF"
    case "h264_qsv":
      return "Intel QSV"
    case "h264_nvenc":
      return "NVIDIA NVENC"
    case "h264_videotoolbox":
      return "VideoToolbox"
    case "libx264":
      return "libx264"
    default:
      return "N/A"
  }
}

export function isPlaybackStatUnavailable(value: string): boolean {
  return value === "N/A"
}

function formatClockLabel(seconds: number): string {
  const s = Math.floor(seconds % 60)
  const m = Math.floor(seconds / 60) % 60
  const h = Math.floor(seconds / 3600)
  const pad = (n: number) => String(n).padStart(2, "0")
  if (h > 0) return `${pad(h)}:${pad(m)}:${pad(s)}`
  return `${pad(m)}:${pad(s)}`
}
