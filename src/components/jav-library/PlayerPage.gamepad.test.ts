import { describe, expect, it } from "vitest"
import source from "./PlayerPage.vue?raw"

describe("PlayerPage gamepad integration", () => {
  it("wires gamepad controls to existing player actions", () => {
    expect(source).toContain("usePlayerGamepadControls")
    expect(source).toContain("togglePlayPause")
    expect(source).toContain("seekDelta")
    expect(source).toContain("adjustVolume")
    expect(source).toContain("showPlaybackFeedback")
    expect(source).toContain("showVolumeFeedback")
    expect(source).toContain("runCuratedCapture")
    expect(source).toContain("resolveNavigationBackLink")
    expect(source).toContain("useGamepadControlsPreference")
    expect(source).toContain("enabled: gamepadControlsEnabled")
  })
})
