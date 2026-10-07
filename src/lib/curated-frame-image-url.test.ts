import { afterEach, describe, expect, it, vi } from "vitest"
import { curatedFrameImageUrl, curatedFrameThumbnailUrl } from "@/lib/curated-frame-image-url"

describe("curated frame image urls", () => {
  afterEach(() => vi.unstubAllEnvs())

  it.each([undefined, "", "   "])("uses same-origin /api for a production build with base %j", (base) => {
    vi.stubEnv("DEV", false)
    vi.stubEnv("VITE_USE_WEB_API", "true")
    vi.stubEnv("VITE_API_BASE_URL", base)
    expect(curatedFrameImageUrl("frame 1")).toBe(`${window.location.origin}/api/curated-frames/frame%201/image`)
    expect(curatedFrameThumbnailUrl("frame 1")).toBe(`${window.location.origin}/api/curated-frames/frame%201/thumbnail`)
  })

  it("uses the same direct backend as list requests in local Web API development", () => {
    vi.stubEnv("DEV", true)
    vi.stubEnv("VITE_USE_WEB_API", "true")
    vi.stubEnv("VITE_API_BASE_URL", "")
    const origin = new URL(window.location.origin)
    const base = `${origin.protocol}//${origin.hostname}:8080/api/curated-frames/frame%2F1`
    expect(curatedFrameImageUrl("frame/1")).toBe(`${base}/image`)
    expect(curatedFrameThumbnailUrl("frame/1")).toBe(`${base}/thumbnail`)
  })

  it.each([
    [" https://server.example/custom/api/ ", "https://server.example/custom/api"],
    ["/custom/api/", `${window.location.origin}/custom/api`],
  ])("honors the configured API base %j", (base, expected) => {
    vi.stubEnv("VITE_API_BASE_URL", base)
    expect(curatedFrameImageUrl("frame 1")).toBe(`${expected}/curated-frames/frame%201/image`)
    expect(curatedFrameThumbnailUrl("frame 1")).toBe(`${expected}/curated-frames/frame%201/thumbnail`)
  })
})
