import { describe, expect, it } from "vitest"
import {
  buildHlsPlaybackConfig,
  startHlsLoadingAtSessionOrigin,
} from "@/lib/hls-player"

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
