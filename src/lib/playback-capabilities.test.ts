import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  clientVideoCodecsQueryParam,
  resetPlaybackCapabilitiesCacheForTest,
  resolvePlaybackCapabilities,
} from "./playback-capabilities"

function stubCanPlayType(supported: (source: string) => boolean) {
  return vi.spyOn(HTMLMediaElement.prototype, "canPlayType").mockImplementation((source: string) =>
    supported(source) ? "probably" : "",
  )
}

beforeEach(() => {
  resetPlaybackCapabilitiesCacheForTest()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("playback capabilities", () => {
  it("reports only codecs the browser says it can decode", () => {
    stubCanPlayType((source) => source.includes("avc1"))

    const capabilities = resolvePlaybackCapabilities()

    expect(capabilities.mp4VideoCodecs).toEqual(["h264"])
    expect(clientVideoCodecsQueryParam(capabilities)).toBe("h264")
  })

  it("maps codec strings to backend codec names", () => {
    stubCanPlayType((source) => source.includes("hvc1") || source.includes("av01"))

    const capabilities = resolvePlaybackCapabilities()

    expect(capabilities.mp4VideoCodecs).toEqual(["hevc", "av1"])
    expect(clientVideoCodecsQueryParam(capabilities)).toBe("hevc,av1")
  })

  it("omits the query param when nothing is decodable", () => {
    stubCanPlayType(() => false)

    const capabilities = resolvePlaybackCapabilities()

    expect(capabilities.mp4VideoCodecs).toEqual([])
    expect(clientVideoCodecsQueryParam(capabilities)).toBeNull()
  })

  it("memoizes the probe result across calls within the tab session", () => {
    const canPlayType = stubCanPlayType(() => true)
    resolvePlaybackCapabilities()
    vi.mocked(canPlayType).mockRestore()

    const second = vi.spyOn(HTMLMediaElement.prototype, "canPlayType")
    const capabilities = resolvePlaybackCapabilities()

    expect(second).not.toHaveBeenCalled()
    expect(capabilities.mp4VideoCodecs).toEqual(["h264", "hevc", "av1"])
  })
})
