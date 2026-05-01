import { describe, expect, test } from "vitest"
import {
  defaultNativePlayerBackendCommand,
  defaultNativePlayerBrowserTemplate,
  looksLikeBrowserProtocolLaunchTarget,
  normalizeNativePlayerPresetForBrowserLaunch,
} from "./native-player-launch"

describe("native player launch defaults", () => {
  test("does not infer mpv or PotPlayer when no native player is configured", () => {
    expect(normalizeNativePlayerPresetForBrowserLaunch(undefined, "")).toBe("custom")
    expect(defaultNativePlayerBackendCommand(undefined)).toBe("")
    expect(defaultNativePlayerBrowserTemplate(undefined)).toBe("")
  })

  test("rejects browser-executable protocols as launch targets", () => {
    expect(looksLikeBrowserProtocolLaunchTarget("javascript:alert(1)")).toBe(false)
    expect(looksLikeBrowserProtocolLaunchTarget("data:text/html,<script>alert(1)</script>")).toBe(false)
    expect(looksLikeBrowserProtocolLaunchTarget("vbscript:msgbox(1)")).toBe(false)
    expect(looksLikeBrowserProtocolLaunchTarget("potplayer:http://127.0.0.1/video.mp4")).toBe(true)
  })
})
