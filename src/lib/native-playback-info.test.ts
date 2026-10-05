import { describe, expect, it } from "vitest"
import { avOffset, bitrate, bytes, metric, playbackInfoGroups, transferPolyline } from "./native-playback-info"
import { emptyNativePlayerState } from "../../electron/native-player-contract"
import { projectPlaybackDiagnostics } from "../../electron/native-playback-diagnostics"

describe("native playback information", () => {
  it("distinguishes unknown, zero, binary bytes, decimal bitrates and signed synchronization", () => {
    expect(bytes(null)).toBe("—")
    expect(bytes(0)).toBe("0 B")
    expect(bytes(1048576)).toBe("1 MiB")
    expect(bytes(0.1)).toBe("0.1 B")
    expect(bitrate(2500000)).toBe("2.5 Mbps")
    expect(bitrate(128000)).toBe("128 Kbps")
    expect(metric(NaN)).toBe("—")
    expect(avOffset(-0.012)).toBe("-12 ms")
    expect(avOffset(0.001)).toBe("+1 ms")
    expect(avOffset(0)).toBe("0 ms")
  })
  it("shows unavailable audio and real idle network metrics with localized labels", () => {
    const state = { ...emptyNativePlayerState(), status: "paused" as const, diagnostics: { ...projectPlaybackDiagnostics({}),
      network: { sourceHost: "server:8080", protocol: "http" as const, mime: "video/mp4", bytesPerSec: 0, receivedBytes: 1024, requests: 2, rangeRequests: 1, history: [] } } }
    const groups = playbackInfoGroups(state, "zh-CN")
    expect(groups[2]!.rows.every(row => row.value === "—")).toBe(true)
    expect(groups[3]!.rows.map(row => row.value)).toContain("0 B/s")
    expect(groups[3]!.rows.map(row => row.value)).toContain("1 KiB")
    expect(groups[1]!.rows.map(row => row.value)).toContain("0 / 0")
    expect(playbackInfoGroups(state, "en")[0]!.rows.map(row => row.value)).toContain("Paused")
    expect(playbackInfoGroups(undefined, "ja")[0]!.title).toBe("再生")
  })
  it("bounds and right-aligns transfer history with zero at baseline", () => {
    expect(transferPolyline([0, 10])).toBe("231.7,34.0 240.0,2.0")
    expect(transferPolyline(Array(100).fill(0)).split(" ")).toHaveLength(30)
    expect(transferPolyline([NaN, -1])).not.toMatch(/NaN|Infinity/)
  })
})
