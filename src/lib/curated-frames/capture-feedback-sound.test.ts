import { describe, expect, it, vi } from "vitest"
import {
  disposeCuratedCaptureFeedbackAudio,
  playCuratedCaptureTriggerCue,
} from "@/lib/curated-frames/capture-feedback-sound"

describe("curated capture feedback sound", () => {
  it("does nothing when disabled", async () => {
    await expect(playCuratedCaptureTriggerCue(false)).resolves.toBeUndefined()
  })

  it("degrades silently when AudioContext is unavailable", async () => {
    await expect(playCuratedCaptureTriggerCue()).resolves.toBeUndefined()
    disposeCuratedCaptureFeedbackAudio()
  })

  it("does not surface AudioContext failures", async () => {
    const oldAudioContext = window.AudioContext
    Object.defineProperty(window, "AudioContext", {
      configurable: true,
      value: vi.fn(() => {
        throw new Error("audio unavailable")
      }),
    })
    await expect(playCuratedCaptureTriggerCue()).resolves.toBeUndefined()
    Object.defineProperty(window, "AudioContext", {
      configurable: true,
      value: oldAudioContext,
    })
    disposeCuratedCaptureFeedbackAudio()
  })
})
