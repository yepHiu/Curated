import { afterEach, describe, expect, it, vi } from "vitest"
import {
  buildHlsPlaybackConfig,
  prewarmHlsResources,
  startHlsLoadingAtSessionOrigin,
} from "@/lib/hls-player"

afterEach(() => {
  vi.restoreAllMocks()
})

describe("buildHlsPlaybackConfig", () => {
  it("starts event-style HLS sessions from the session timeline origin", () => {
    expect(buildHlsPlaybackConfig()).toMatchObject({
      autoStartLoad: false,
      startPosition: 0,
      startFragPrefetch: true,
      enableWorker: true,
      lowLatencyMode: false,
    })
  })

  it("loads HLS session resources with credentials for authenticated media endpoints", () => {
    const config = buildHlsPlaybackConfig()
    const xhr = { withCredentials: false }

    expect(config.xhrSetup).toBeTypeOf("function")
    ;(config.xhrSetup as (xhr: { withCredentials: boolean }) => void)(xhr)

    expect(xhr.withCredentials).toBe(true)
  })
})

describe("startHlsLoadingAtSessionOrigin", () => {
  it("starts fragment loading from the beginning of the generated session timeline", () => {
    const calls: number[] = []

    startHlsLoadingAtSessionOrigin({
      startLoad: (position) => {
        calls.push(position ?? -1)
      },
    })

    expect(calls).toEqual([0])
  })
})

describe("prewarmHlsResources", () => {
  it("fetches HLS playlists and media resources with credentials", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      if (url.endsWith("index.m3u8")) {
        return new Response("#EXTM3U\n#EXTINF:1,\nseg0.ts\n", { status: 200 })
      }
      return new Response(new Uint8Array([1, 2, 3]), { status: 200 })
    })
    vi.stubGlobal("fetch", fetchMock)

    await prewarmHlsResources("http://127.0.0.1:8080/api/playback/sessions/s1/hls/index.m3u8", {
      resourceCount: 1,
      timeoutMs: 1000,
    })

    expect(fetchMock).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/api/playback/sessions/s1/hls/index.m3u8",
      expect.objectContaining({ credentials: "include" }),
    )
    expect(fetchMock).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/api/playback/sessions/s1/hls/seg0.ts",
      expect.objectContaining({ credentials: "include" }),
    )
  })
})
