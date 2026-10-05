import type { NativePlaybackDiagnostics, NativePlayerState } from "./native-player-contract.js"

export const diagnosticProperties = ["mpv-version", "file-format", "video-params", "container-fps", "display-fps",
  "video-bitrate", "audio-codec-name", "audio-params", "audio-bitrate", "current-vo", "avsync",
  "estimated-frame-count", "demuxer-cache-duration", "demuxer-cache-state", "cache-buffering-state", "paused-for-cache"] as const

function number(value: unknown, signed = false): number | null {
  return typeof value === "number" && Number.isFinite(value) && (signed || value >= 0) ? value : null
}
function text(value: unknown): string | null {
  return typeof value === "string" && value.length > 0 && value.length <= 200 && !/[\x00-\x1f\x7f]/.test(value) ? value : null
}
function node(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : {}
}
/** 仅投影已知属性；不可用值明确清空，保留零值与有符号 A/V 偏差。 */
export function projectPlaybackDiagnostics(values: Record<string, unknown>): NativePlaybackDiagnostics {
  const video = node(values["video-params"]), audio = node(values["audio-params"]), cache = node(values["demuxer-cache-state"])
  return {
    engineVersion: text(values["mpv-version"]), container: text(values["file-format"]), width: number(video.w), height: number(video.h),
    sourceFps: number(values["container-fps"]), displayFps: number(values["display-fps"]), pixelFormat: text(video.pixelformat),
    videoBitrate: number(values["video-bitrate"]), audioCodec: text(values["audio-codec-name"]),
    audioSampleRate: number(audio.samplerate), audioChannels: text(audio["hr-channels"]) ?? (number(audio["channel-count"]) === null ? null : String(audio["channel-count"])),
    audioBitrate: number(values["audio-bitrate"]), videoOutput: text(values["current-vo"]), avSyncSec: number(values.avsync, true),
    estimatedFrames: number(values["estimated-frame-count"]), cacheDurationSec: number(values["demuxer-cache-duration"]), cacheBytes: number(cache["fw-bytes"]),
    bufferingPercent: number(values["cache-buffering-state"]) === null ? null : Math.min(100, number(values["cache-buffering-state"])!),
    pausedForCache: typeof values["paused-for-cache"] === "boolean" ? values["paused-for-cache"] : null,
  }
}
/** 报告只包含播放指标，不能直接序列化完整快照或 mpv 的原始属性树。 */
export function playbackDiagnosticsReport(state: NativePlayerState, now = new Date()): string {
  const d = state.diagnostics
  const metrics = d ? projectPlaybackDiagnostics({
    "mpv-version": d.engineVersion, "file-format": d.container, "video-params": { w: d.width, h: d.height, pixelformat: d.pixelFormat },
    "container-fps": d.sourceFps, "display-fps": d.displayFps, "video-bitrate": d.videoBitrate, "audio-codec-name": d.audioCodec,
    "audio-params": { samplerate: d.audioSampleRate, "hr-channels": d.audioChannels }, "audio-bitrate": d.audioBitrate, "current-vo": d.videoOutput,
    avsync: d.avSyncSec, "estimated-frame-count": d.estimatedFrames, "demuxer-cache-duration": d.cacheDurationSec,
    "demuxer-cache-state": { "fw-bytes": d.cacheBytes }, "cache-buffering-state": d.bufferingPercent, "paused-for-cache": d.pausedForCache,
  }) : null
  const n = d?.network
  return JSON.stringify({ schema: 1, capturedAt: now.toISOString(), engine: "mpv", status: state.status,
    positionSec: number(state.positionSec), durationSec: number(state.durationSec), speed: number(state.speed),
    videoCodec: text(state.codec), hardwareDecoder: text(state.hwdec), presentationDrops: number(state.droppedFrames), decoderDrops: number(state.decoderDroppedFrames),
    metrics, network: n ? { sourceHost: n.sourceHost, protocol: n.protocol, mime: n.mime, bytesPerSec: n.bytesPerSec,
      receivedBytes: n.receivedBytes, requests: n.requests, rangeRequests: n.rangeRequests, history: n.history.slice(-30) } : null,
    units: { bitrate: "bits/s (estimated)", transferRate: "bytes/s", cacheDuration: "seconds", avSync: "seconds (signed)",
      estimatedFrames: "source FPS × duration estimate; not presented frames", receivedBytes: "session media bytes, including repeated ranges" },
  }, null, 2)
}
