import { describe, expect, it } from "vitest"
import { avOffset, bitrate, bytes, metric, playbackInfoGroups, transferPolyline } from "./native-playback-info"
import { emptyNativePlayerState } from "../../electron/native-player-contract"
import { projectPlaybackDiagnostics } from "../../electron/native-playback-diagnostics"

describe("native playback information", () => {
  it("distinguishes unknown, zero, binary bytes, decimal bitrates and signed synchronization", () => {
    expect(bytes(null)).toBe("—")
    expect(bytes(0)).toBe("0.00 B")
    expect(bytes(1048576)).toBe("1.00 MiB")
    expect(bytes(0.1)).toBe("0.10 B")
    expect(bitrate(2500000)).toBe("2.50 Mbps")
    expect(bitrate(128000)).toBe("128.00 Kbps")
    expect(metric(NaN)).toBe("—")
    expect(avOffset(-0.012)).toBe("-12.00 ms")
    expect(avOffset(0.001)).toBe("+1.00 ms")
    expect(avOffset(0)).toBe("+0.00 ms")
  })
  it("shows unavailable audio and real idle network metrics with localized labels", () => {
    const state = { ...emptyNativePlayerState(), status: "paused" as const, diagnostics: { ...projectPlaybackDiagnostics({}),
      network: { sourceHost: "server:8080", protocol: "http" as const, mime: "video/mp4", bytesPerSec: 0, receivedBytes: 1024, requests: 2, rangeRequests: 1, history: [] } } }
    const groups = playbackInfoGroups(state, "zh-CN")
    expect(groups[2]!.rows.every(row => row.value === "—")).toBe(true)
    expect(groups[3]!.rows.map(row => row.value)).toContain("0.00 B/s")
    expect(groups[3]!.rows.map(row => row.value)).toContain("1.00 KiB")
    expect(groups[1]!.rows.map(row => row.value)).toContain("0 / 0")
    expect(playbackInfoGroups(state, "en")[0]!.rows.map(row => row.value)).toContain("Paused")
    expect(playbackInfoGroups(undefined, "ja")[0]!.title).toBe("再生")
  })
  it("keeps numeric and unit slots through missing values, sign changes and unit changes", () => {
    const state = { ...emptyNativePlayerState(), diagnostics: { ...projectPlaybackDiagnostics({}),
      network: { sourceHost: "server", protocol: "http" as const, mime: null, bytesPerSec: 0, receivedBytes: 0, requests: 9, rangeRequests: 0, history: [] } } }
    const audio = () => playbackInfoGroups(state, "zh-CN")[2]!.rows.at(-1)!.metrics
    expect(audio()).toEqual([{ number: "—", unit: "", reserveUnit: true }])
    state.diagnostics.avSyncSec = -0.0001
    expect(audio()).toEqual([{ number: "-0.10", unit: "ms", reserveUnit: true }])
    state.diagnostics.avSyncSec = 0
    expect(audio()).toEqual([{ number: "+0.00", unit: "ms", reserveUnit: true }])
    const network = () => playbackInfoGroups(state, "zh-CN")[3]!.rows
    expect(network()[1]!.metrics).toEqual([{ number: "0.00", unit: "B/s", reserveUnit: true }])
    state.diagnostics.network.bytesPerSec = 1024 ** 2
    expect(network()[1]!.metrics).toEqual([{ number: "1.00", unit: "MiB/s", reserveUnit: true }])
    expect(network()[3]!.metrics).toEqual([{ number: "9", unit: "", reserveUnit: false }, { number: "0", unit: "", reserveUnit: false }])
    expect(network()[4]!.metrics).toHaveLength(2)
  })
  it("bounds and right-aligns transfer history with zero at baseline", () => {
    expect(transferPolyline([0, 10])).toBe("231.7,34.0 240.0,2.0")
    expect(transferPolyline(Array(100).fill(0)).split(" ")).toHaveLength(30)
    expect(transferPolyline([NaN, -1])).not.toMatch(/NaN|Infinity/)
  })
})
