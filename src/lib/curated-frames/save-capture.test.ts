import { describe, expect, it } from "vitest"
import { resolveCuratedCapturePositionSec } from "@/lib/curated-frames/save-capture"

describe("resolveCuratedCapturePositionSec", () => {
  it("prefers an explicit absolute playback position when provided", () => {
    expect(resolveCuratedCapturePositionSec(12.25, 145.875)).toBe(145.875)
  })

  it("falls back to the video local currentTime when no override is provided", () => {
    expect(resolveCuratedCapturePositionSec(12.25)).toBe(12.25)
  })
})
