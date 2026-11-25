import { describe, expect, it } from "vitest"
import { nativePlaybackAcceptance } from "./native-playback-acceptance"

describe("native playback acceptance report", () => {
  it("never records a real Windows case as covered on this machine", () => {
    const report = nativePlaybackAcceptance({
      platform: process.platform,
      mpvPath: process.env.CURATED_NATIVE_MPV,
      win32HostBuilt: false,
    })
    const real = report.filter(item => item.id !== "engine-missing" && item.id !== "proxy-and-progress")
    expect(real.every(item => item.status === "skipped" || item.status === "manual-windows")).toBe(true)
    expect(report.some(item => item.status === "unit-covered")).toBe(true)
    expect(JSON.stringify(report)).not.toContain("\"status\":\"passed\"")
  })

  it("keeps Linux CI from treating a configured mpv path as Windows acceptance", () => {
    const report = nativePlaybackAcceptance({ platform: "linux", mpvPath: "/usr/bin/mpv", win32HostBuilt: true })
    expect(report.find(item => item.id === "real-mpv-controls")?.status).toBe("skipped")
    expect(report.find(item => item.id === "win32-host-geometry")?.status).toBe("skipped")
    expect(report.filter(item => item.status === "manual-windows").map(item => item.id)).toEqual([
      "more-sources", "full-length-sync", "mixed-dpi", "installer-lifecycle",
    ])
  })
})
