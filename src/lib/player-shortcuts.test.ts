import { describe, expect, it } from "vitest"
import {
  DEFAULT_CURATED_CAPTURE_KEY_CODE,
  formatCuratedCaptureKeyLabel,
  isCuratedCaptureKeyReserved,
  isSupportedCuratedCaptureKeyCode,
  normalizeCuratedCaptureKeyCode,
  shouldIgnoreGlobalPlaybackHotkeysForTarget,
} from "@/lib/player-shortcuts"

describe("player shortcut utilities", () => {
  it("falls back to the default curated capture key for invalid persisted values", () => {
    expect(normalizeCuratedCaptureKeyCode("")).toBe(DEFAULT_CURATED_CAPTURE_KEY_CODE)
    expect(normalizeCuratedCaptureKeyCode("ArrowUp")).toBe(DEFAULT_CURATED_CAPTURE_KEY_CODE)
    expect(normalizeCuratedCaptureKeyCode("Minus")).toBe(DEFAULT_CURATED_CAPTURE_KEY_CODE)
  })

  it("accepts supported single-key curated capture codes", () => {
    expect(isSupportedCuratedCaptureKeyCode("KeyX")).toBe(true)
    expect(isSupportedCuratedCaptureKeyCode("Digit7")).toBe(true)
    expect(isSupportedCuratedCaptureKeyCode("F8")).toBe(true)
    expect(isSupportedCuratedCaptureKeyCode("PageUp")).toBe(true)
    expect(isSupportedCuratedCaptureKeyCode("PageDown")).toBe(true)
    expect(isSupportedCuratedCaptureKeyCode("ArrowDown")).toBe(false)
    expect(isSupportedCuratedCaptureKeyCode("Escape")).toBe(false)
  })

  it("marks player-owned shortcut keys as reserved for curated capture", () => {
    expect(isCuratedCaptureKeyReserved("Space")).toBe(true)
    expect(isCuratedCaptureKeyReserved("ArrowUp")).toBe(true)
    expect(isCuratedCaptureKeyReserved("KeyK")).toBe(true)
    expect(isCuratedCaptureKeyReserved("KeyC")).toBe(false)
    expect(isCuratedCaptureKeyReserved("F8")).toBe(false)
  })

  it("formats capture key labels for settings display", () => {
    expect(formatCuratedCaptureKeyLabel("KeyC")).toBe("C")
    expect(formatCuratedCaptureKeyLabel("Digit7")).toBe("7")
    expect(formatCuratedCaptureKeyLabel("F8")).toBe("F8")
    expect(formatCuratedCaptureKeyLabel("PageDown")).toBe("PageDown")
  })

  it("ignores global playback hotkeys for typing targets and slider-focused descendants", () => {
    const input = document.createElement("input")
    expect(shouldIgnoreGlobalPlaybackHotkeysForTarget(input)).toBe(true)

    const editable = document.createElement("div")
    editable.contentEditable = "true"
    expect(shouldIgnoreGlobalPlaybackHotkeysForTarget(editable)).toBe(true)

    const slider = document.createElement("div")
    slider.setAttribute("data-slot", "slider")
    const sliderThumb = document.createElement("button")
    slider.appendChild(sliderThumb)
    expect(shouldIgnoreGlobalPlaybackHotkeysForTarget(sliderThumb)).toBe(true)

    const plain = document.createElement("div")
    expect(shouldIgnoreGlobalPlaybackHotkeysForTarget(plain)).toBe(false)
  })
})
