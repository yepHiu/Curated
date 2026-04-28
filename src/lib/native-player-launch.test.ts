import { describe, expect, test } from "vitest"
import {
  defaultNativePlayerBackendCommand,
  defaultNativePlayerBrowserTemplate,
  normalizeNativePlayerPresetForBrowserLaunch,
} from "./native-player-launch"

describe("native player launch defaults", () => {
  test("does not infer mpv or PotPlayer when no native player is configured", () => {
    expect(normalizeNativePlayerPresetForBrowserLaunch(undefined, "")).toBe("custom")
    expect(defaultNativePlayerBackendCommand(undefined)).toBe("")
    expect(defaultNativePlayerBrowserTemplate(undefined)).toBe("")
  })
})
