import { describe, expect, it } from "vitest"
import { canDirectPlaySource, createHlsRecoveryPolicy } from "./player-hls-recovery"

describe("HLS recovery", () => {
  it("limits retries and permits another attempt only after explicit reset", () => {
    const policy = createHlsRecoveryPolicy()
    expect(policy.next("networkError")).toBe("network")
    expect(policy.next("mediaError")).toBe("media")
    expect(policy.next("networkError")).toBe("session")
    expect(policy.next("mediaError")).toBe("failed")
    policy.reset()
    expect(policy.next("session")).toBe("session")
    expect(policy.next("session")).toBe("failed")
  })
  it("does not let an mp4 extension override codec incompatibility", () => {
    const descriptor = { movieId: "m", mode: "hls" as const, url: "/hls", fileName: "m.mp4", sourceVideoCodec: "hevc", sourceAudioCodec: "aac", canDirectPlay: false }
    expect(canDirectPlaySource(descriptor, ["h264"])).toBe(false)
    expect(canDirectPlaySource(descriptor, ["hevc"])).toBe(true)
    expect(canDirectPlaySource({ ...descriptor, sourceAudioCodec: "dts" }, ["hevc"])).toBe(false)
    expect(canDirectPlaySource({ ...descriptor, fileName: "m.mkv" }, ["hevc"])).toBe(false)
  })
})
