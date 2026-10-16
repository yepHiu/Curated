import { describe, expect, it } from "vitest"
import { playbackInfoText, webPlaybackInfoGroups } from "./web-playback-info"

describe("Web playback information", () => {
  it("groups actual Web diagnostics without inventing native engine measurements", () => {
    const rows = [
      { key: "state", label: "State", value: "Paused" },
      { key: "audio-tracks", label: "Audio tracks", value: "0" },
      { key: "fps", label: "Frame rate", value: "N/A" },
      { key: "video-current", label: "Bitrate", value: "2.50 Mbps" },
      { key: "transcode-profile", label: "Transcoder", value: "AMD AMF" },
      { key: "current", label: "Time", value: "01:02 / 10:00" },
    ]
    for (const lang of ["zh-CN", "en", "ja"] as const) {
      const groups = webPlaybackInfoGroups(rows, lang)
      expect(groups).toHaveLength(4)
      expect(groups[0]!.rows[0]!.value).toBe("paused")
      expect(groups[1]!.rows[0]!.metrics).toEqual([{ number: "—", unit: "", reserveUnit: true }])
      expect(groups[1]!.rows[1]!.metrics).toEqual([{ number: "2.50", unit: "Mbps", reserveUnit: true }])
      expect(groups[2]!.rows[0]!.value).toBe("0")
      expect(groups[3]!.rows[0]!.value).toBe("AMD AMF")
      expect(groups[0]!.rows[1]!.metrics?.map(metric => metric.number)).toEqual(["01:02", "10:00"])
      const exported = playbackInfoText("Playback info", groups)
      expect(exported).toContain("AMD AMF")
      expect(exported).toContain("2.50 Mbps")
      expect(exported).not.toContain("N/A")
      expect(exported).not.toContain("mpv")
    }
  })
})
