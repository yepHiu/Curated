import { describe, expect, it } from "vitest"
import { projectPlaybackDiagnostics, playbackDiagnosticsReport } from "./native-playback-diagnostics"
import { emptyNativePlayerState } from "./native-player-contract"

describe("native diagnostic projection", () => {
  it("preserves units, zero metrics and signed A/V offset while whitelisting node fields", () => {
    const result = projectPlaybackDiagnostics({ "mpv-version": "mpv 0.41", "video-params": { w: 1920, h: 1080, pixelformat: "yuv420p", secret: "hidden" },
      "container-fps": 30000 / 1001, "video-bitrate": 2500000, "audio-bitrate": 128000,
      "audio-params": { samplerate: 48000, "channel-count": 2, "hr-channels": "stereo" }, avsync: -0.012,
      "demuxer-cache-state": { "fw-bytes": 0, "reader-pts": 999 }, "paused-for-cache": false, "cache-buffering-state": 0 })
    expect(result).toMatchObject({ width: 1920, height: 1080, sourceFps: 30000 / 1001, videoBitrate: 2500000, audioBitrate: 128000,
      audioSampleRate: 48000, audioChannels: "stereo", avSyncSec: -0.012, cacheBytes: 0, bufferingPercent: 0, pausedForCache: false })
    expect(JSON.stringify(result)).not.toContain("hidden")
    expect(JSON.stringify(result)).not.toContain("reader-pts")
  })
  it("clears unsupported and invalid values instead of carrying values from another file", () => {
    const result = projectPlaybackDiagnostics({ "video-params": [], "audio-params": null, "video-bitrate": NaN,
      "audio-bitrate": Infinity, avsync: "0", "container-fps": -1, "paused-for-cache": "false", "file-format": "bad\nvalue" })
    expect(Object.values(result).every(value => value === null)).toBe(true)
    expect(projectPlaybackDiagnostics({ "audio-params": { "channel-count": 6 } }).audioChannels).toBe("6")
  })
  it("exports only approved playback fields and bounds rate history", () => {
    const state = { ...emptyNativePlayerState(), movieId: "private-movie", fileId: "private-file", error: "cookie=secret",
      diagnostics: { ...projectPlaybackDiagnostics({ "video-params": { w: 640, h: 360 } }),
        network: { sourceHost: "server.test:8080", protocol: "http" as const, mime: "video/mp4", bytesPerSec: 0,
          receivedBytes: 42, requests: 2, rangeRequests: 1, history: Array(90).fill(0), url: "http://server.test/secret-token", cookie: "auth-secret" },
        path: "C:/secret/mpv.exe", raw: { cookie: "auth-secret" } } }
    const report = playbackDiagnosticsReport(state, new Date("2026-10-06T12:00:00Z"))
    const json = JSON.parse(report)
    expect(json.metrics.width).toBe(640)
    expect(json.network).toMatchObject({ sourceHost: "server.test:8080", bytesPerSec: 0, receivedBytes: 42 })
    expect(json.network.history).toHaveLength(30)
    expect(report).not.toMatch(/secret|private|mpv\.exe|url|cookie/i)
  })
})
