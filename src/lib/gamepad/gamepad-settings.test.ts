import { describe, expect, it } from "vitest"
import {
  GAMEPAD_CONTROLS_STORAGE_KEY,
  getStoredGamepadControlsEnabled,
  setStoredGamepadControlsEnabled,
} from "@/lib/gamepad/gamepad-settings"

describe("gamepad settings storage", () => {
  it("enables gamepad controls by default when no preference is stored", () => {
    const storage = new Map<string, string>()

    expect(getStoredGamepadControlsEnabled(storage)).toBe(true)
  })

  it("persists disabled and enabled states", () => {
    const storage = new Map<string, string>()

    setStoredGamepadControlsEnabled(false, storage)
    expect(storage.get(GAMEPAD_CONTROLS_STORAGE_KEY)).toBe("false")
    expect(getStoredGamepadControlsEnabled(storage)).toBe(false)

    setStoredGamepadControlsEnabled(true, storage)
    expect(storage.get(GAMEPAD_CONTROLS_STORAGE_KEY)).toBe("true")
    expect(getStoredGamepadControlsEnabled(storage)).toBe(true)
  })

  it("falls back to enabled for malformed stored values", () => {
    const storage = new Map<string, string>([[GAMEPAD_CONTROLS_STORAGE_KEY, "maybe"]])

    expect(getStoredGamepadControlsEnabled(storage)).toBe(true)
  })
})
