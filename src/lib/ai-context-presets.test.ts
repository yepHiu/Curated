import { describe, expect, it } from "vitest"
import { AI_CONTEXT_PRESETS, validAIContextWindow } from "./ai-context-presets"

describe("model context capacities", () => {
  it("keeps every preset within the supported integer range with an official reference", () => {
    expect(new Set(AI_CONTEXT_PRESETS.map(preset => preset.id)).size).toBe(AI_CONTEXT_PRESETS.length)
    for (const preset of AI_CONTEXT_PRESETS) {
      expect(validAIContextWindow(preset.tokens)).toBe(true)
      expect(new URL(preset.source).protocol).toBe("https:")
    }
  })
  it("rejects non-finite and fractional capacities", () => {
    for (const value of [NaN, Infinity, 65536.5, -1, 0, 2097153]) expect(validAIContextWindow(value)).toBe(false)
  })
})
