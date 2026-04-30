import { describe, expect, it } from "vitest"
import {
  formatBitrateLabel,
  formatCountLabel,
  formatFpsLabel,
  formatPercentLabel,
  formatReasonLabel,
  formatResolutionLabel,
  formatSessionKindLabel,
  formatSourceFormatLabel,
  formatTimecodeLabel,
  formatTranscodeProfileLabel,
  isPlaybackStatUnavailable,
} from "@/lib/player-playback-stats-format"

describe("player playback stats formatting", () => {
  it("formats bitrate labels and rejects missing values", () => {
    expect(formatBitrateLabel(null)).toBe("N/A")
    expect(formatBitrateLabel(0)).toBe("N/A")
    expect(formatBitrateLabel(-1)).toBe("N/A")
    expect(formatBitrateLabel(950)).toBe("950 kbps")
    expect(formatBitrateLabel(2500)).toBe("2.50 Mbps")
  })

  it("formats resolution and fps labels", () => {
    expect(formatResolutionLabel(null, 1080)).toBe("N/A")
    expect(formatResolutionLabel(1920, 1080)).toBe("1920 \u00d7 1080")
    expect(formatFpsLabel(null)).toBe("N/A")
    expect(formatFpsLabel(0)).toBe("N/A")
    expect(formatFpsLabel(23.976)).toBe("23.98 fps")
    expect(formatFpsLabel(120)).toBe("120 fps")
  })

  it("formats timecode, percent, and count values with clamps", () => {
    expect(formatTimecodeLabel(undefined)).toBe("N/A")
    expect(formatTimecodeLabel(-1)).toBe("N/A")
    expect(formatTimecodeLabel(Number.NaN)).toBe("N/A")
    expect(formatTimecodeLabel(3661.9)).toBe("01:01:01")
    expect(formatPercentLabel(undefined)).toBe("N/A")
    expect(formatPercentLabel(-12)).toBe("0.0%")
    expect(formatPercentLabel(120)).toBe("100.0%")
    expect(formatCountLabel(undefined)).toBe("N/A")
    expect(formatCountLabel(-3)).toBe("0")
    expect(formatCountLabel(3.9)).toBe("3")
  })

  it("formats reason, session kind, source format, and transcode profile labels", () => {
    expect(formatReasonLabel("  network-window  ")).toBe("network-window")
    expect(formatReasonLabel(" ")).toBe("N/A")
    expect(formatSessionKindLabel("direct-file")).toBe("Direct File")
    expect(formatSessionKindLabel("remux-hls")).toBe("Remux HLS")
    expect(formatSessionKindLabel("transcode-hls")).toBe("Transcode HLS")
    expect(formatSessionKindLabel("unknown")).toBe("N/A")
    expect(formatSourceFormatLabel(null)).toBe("N/A")
    expect(
      formatSourceFormatLabel({
        sourceContainer: "mp4",
        sourceVideoCodec: "h264",
        sourceAudioCodec: "aac",
      }),
    ).toBe("mp4 / h264 / aac")
    expect(formatTranscodeProfileLabel("remux_copy")).toBe("FFmpeg Stream Copy")
    expect(formatTranscodeProfileLabel("h264_nvenc")).toBe("NVIDIA NVENC")
    expect(formatTranscodeProfileLabel("libx264")).toBe("libx264")
    expect(formatTranscodeProfileLabel("unknown")).toBe("N/A")
  })

  it("detects unavailable stat labels", () => {
    expect(isPlaybackStatUnavailable("N/A")).toBe(true)
    expect(isPlaybackStatUnavailable("0.0%")).toBe(false)
  })
})
