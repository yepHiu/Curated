import { describe, expect, it, vi } from "vitest"
import {
  resetVideoElementPlaybackPipeline,
  shouldDeferPlaybackStartUntilCurrentData,
  shouldResetVideoElementBeforeModeAttach,
} from "@/lib/player-media-source"

describe("shouldResetVideoElementBeforeModeAttach", () => {
  it("resets the existing video pipeline when switching from direct playback into HLS", () => {
    expect(shouldResetVideoElementBeforeModeAttach(undefined, "hls")).toBe(true)
    expect(shouldResetVideoElementBeforeModeAttach("direct", "hls")).toBe(true)
    expect(shouldResetVideoElementBeforeModeAttach("hls", "hls")).toBe(false)
    expect(shouldResetVideoElementBeforeModeAttach("hls", "direct")).toBe(false)
  })
})

describe("resetVideoElementPlaybackPipeline", () => {
  it("pauses, clears the src attribute, and reloads the video element before HLS takeover", () => {
    const pause = vi.fn()
    const removeAttribute = vi.fn()
    const load = vi.fn()

    resetVideoElementPlaybackPipeline({
      pause,
      removeAttribute,
      load,
    })

    expect(pause).toHaveBeenCalledTimes(1)
    expect(removeAttribute).toHaveBeenCalledWith("src")
    expect(load).toHaveBeenCalledTimes(1)
  })
})

describe("shouldDeferPlaybackStartUntilCurrentData", () => {
  it("does not defer HLS resume playback while the media source is still attaching", () => {
    expect(
      shouldDeferPlaybackStartUntilCurrentData({
        readyState: 0,
        haveCurrentData: 2,
        playbackMode: "hls",
        resumeRequested: true,
      }),
    ).toBe(false)
  })

  it("keeps direct playback and route autoplay gated on current media data", () => {
    expect(
      shouldDeferPlaybackStartUntilCurrentData({
        readyState: 0,
        haveCurrentData: 2,
        playbackMode: "direct",
        resumeRequested: true,
      }),
    ).toBe(true)

    expect(
      shouldDeferPlaybackStartUntilCurrentData({
        readyState: 0,
        haveCurrentData: 2,
        playbackMode: "hls",
        resumeRequested: false,
      }),
    ).toBe(true)
  })
})
