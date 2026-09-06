import { afterEach, describe, expect, it, vi } from "vitest"
import {
  buildHlsPlaybackConfig,
  startHlsLoadingAtSessionOrigin,
} from "@/lib/hls-player"

const hlsMock = vi.hoisted(() => {
  class FakeHls {
    static isSupported() {
      return true
    }
  }
  return { FakeHls }
})

vi.mock("hls.js/light", () => ({ default: hlsMock.FakeHls }))

afterEach(() => {
  vi.restoreAllMocks()
  document.querySelectorAll('script[data-curated-hls="true"]').forEach((script) => {
    script.remove()
  })
  delete window.Hls
})

describe("buildHlsPlaybackConfig", () => {
  it("starts event-style HLS sessions from the session timeline origin", () => {
    expect(buildHlsPlaybackConfig()).toMatchObject({
      autoStartLoad: false,
      startPosition: 0,
      startFragPrefetch: true,
      enableWorker: true,
      lowLatencyMode: false,
      maxBufferLength: 60,
      maxMaxBufferLength: 120,
      maxStarvationDelay: 12,
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

describe("loadHlsLibrary", () => {
  it("loads the bundled hls.js module without injecting a CDN script", async () => {
    vi.resetModules()
    const appendSpy = vi.spyOn(document.head, "appendChild")
    const { loadHlsLibrary } = await import("@/lib/hls-player")

    const Hls = await loadHlsLibrary()

    expect(Hls).toBe(hlsMock.FakeHls)
    expect(window.Hls).toBe(hlsMock.FakeHls)
    expect(appendSpy).not.toHaveBeenCalled()
    expect(document.querySelector('script[src*="cdn.jsdelivr.net"]')).toBeNull()
  })
})
