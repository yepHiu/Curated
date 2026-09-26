import { describe, expect, it, vi } from "vitest"
import { checkDesktopUpdate, isTrustedDesktopSender, selectDesktopUpdate } from "./desktop-updates"
import type { DesktopInfo } from "./desktop-contract"

const info: DesktopInfo = { version: "0.1.0", buildStamp: "20260926.123433", development: false, distribution: "desktop", platform: "macos", arch: "arm64" }
const asset = { component: "desktop", variant: "standalone", channel: "stable", version: "0.2.0", platform: "macos", arch: "arm64", format: "dmg", url: "https://downloads.example.com/desktop.dmg", sha256: "a".repeat(64) }
const feed = "https://updates.example.com/desktop.json"

describe("Desktop updates", () => {
  it("selects by component, variant, channel, platform and architecture before comparing numeric versions", () => {
    const artifacts = [asset, { ...asset, version: "0.10.0" }, ...[
      { component: "full" }, { component: "server" }, { variant: "bundle" }, { channel: "dev" },
      { platform: "windows", format: "exe" }, { arch: "x64" }, { format: "zip" },
    ].map((change) => ({ ...asset, version: "9.0.0", ...change }))]
    expect(selectDesktopUpdate({ schema: 1, artifacts }, info)).toEqual({ status: "update-available", latestVersion: "0.10.0", downloadUrl: asset.url })
  })

  it("does not claim current when no compatible artifact exists or offer a downgrade", () => {
    expect(selectDesktopUpdate({ schema: 1, artifacts: [{ ...asset, arch: "x64" }] }, info).status).toBe("no-artifact")
    expect(selectDesktopUpdate({ schema: 1, artifacts: [asset] }, { ...info, version: "0.3.0" }).status).toBe("up-to-date")
  })

  it.each([{ version: "0.2.0-beta" }, { url: "javascript:alert(1)" }, { url: "https://user:pass@example.com/app" }, { sha256: "" }])("rejects malformed matching artifacts: %j", (change) => {
    expect(() => selectDesktopUpdate({ schema: 1, artifacts: [{ ...asset, ...change }] }, info)).toThrow()
  })

  it("never requests a production feed for development, legacy or unconfigured builds", async () => {
    const fetcher = vi.fn()
    expect((await checkDesktopUpdate({ ...info, development: true }, feed, fetcher)).status).toBe("development")
    expect((await checkDesktopUpdate({ ...info, distribution: "legacy" }, feed, fetcher)).status).toBe("bundled")
    expect((await checkDesktopUpdate(info, null, fetcher)).status).toBe("not-configured")
    expect(fetcher).not.toHaveBeenCalled()
  })

  it("checks the configured HTTPS manifest and turns offline, invalid and oversized feeds into errors", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ schema: 1, artifacts: [asset] })))
    expect((await checkDesktopUpdate(info, feed, fetcher)).status).toBe("update-available")
    expect(fetcher).toHaveBeenCalledWith(feed, expect.objectContaining({ redirect: "error", cache: "no-store" }))
    for (const response of [new Response("offline", { status: 503 }), new Response("{}"), new Response("x".repeat(1024 * 1024 + 1))]) {
      expect((await checkDesktopUpdate(info, feed, vi.fn().mockResolvedValue(response))).status).toBe("error")
    }
    expect((await checkDesktopUpdate(info, feed, vi.fn().mockRejectedValue(new Error("offline")))).status).toBe("error")
  })

  it("only authorizes the current main frame at the renderer origin", () => {
    const origin = "http://127.0.0.1:5173"
    expect(isTrustedDesktopSender(1, 1, origin + "/settings", true, origin)).toBe(true)
    expect(isTrustedDesktopSender(1, 1, origin, false, origin)).toBe(false)
    expect(isTrustedDesktopSender(2, 1, origin, true, origin)).toBe(false)
    expect(isTrustedDesktopSender(1, 1, "https://example.com", true, origin)).toBe(false)
    expect(isTrustedDesktopSender(1, undefined, origin, true, origin)).toBe(false)
  })
})
