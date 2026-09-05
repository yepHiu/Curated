import { afterEach, describe, expect, it, vi } from "vitest"
import {
  HLS_SEEK_REUSE_LEAD_SEC,
  HLS_TRANSCODE_SEEK_REUSE_LEAD_SEC,
  getMediaWrittenEndSec,
  hlsSeekReuseLeadSec,
  isPrematureHlsEndedEvent,
  shouldReuseHlsSessionForSeek,
  waitForMediaWrittenEnd,
  bufferedPlaybackSeconds,
  waitForPlaybackBuffer,
} from "@/lib/player-hls-seek"

function fakeMedia(duration: number, bufferedEnd?: number) {
  const listeners = new Map<string, Set<EventListener>>()
  return {
    duration,
    buffered:
      bufferedEnd == null
        ? { length: 0, end: () => 0 }
        : {
            length: 1,
            end: () => bufferedEnd,
          },
    addEventListener(type: string, listener: EventListener) {
      const bucket = listeners.get(type) ?? new Set()
      bucket.add(listener)
      listeners.set(type, bucket)
    },
    removeEventListener(type: string, listener: EventListener) {
      listeners.get(type)?.delete(listener)
    },
    emit(type: string) {
      for (const listener of listeners.get(type) ?? []) {
        listener(new Event(type))
      }
    },
    setDuration(next: number) {
      this.duration = next
    },
  }
}

describe("getMediaWrittenEndSec", () => {
  it("uses the later of finite duration and the last buffered end", () => {
    expect(getMediaWrittenEndSec({ duration: Number.NaN })).toBe(0)
    expect(getMediaWrittenEndSec({ duration: 4, buffered: { length: 0, end: () => 0 } })).toBe(4)
    expect(
      getMediaWrittenEndSec({
        duration: 4,
        buffered: { length: 1, end: () => 7.5 },
      }),
    ).toBe(7.5)
    expect(
      getMediaWrittenEndSec({
        duration: Number.POSITIVE_INFINITY,
        buffered: { length: 1, end: () => 6 },
      }),
    ).toBe(6)
  })
})

describe("shouldReuseHlsSessionForSeek", () => {
  it("reuses the session for skip-forward that is only slightly past the written edge", () => {
    expect(
      shouldReuseHlsSessionForSeek({
        localTargetSec: 14,
        writtenEndSec: 4,
        encoderSpeed: "3x",
      }),
    ).toBe(true)
    expect(
      shouldReuseHlsSessionForSeek({
        localTargetSec: 4.2,
        writtenEndSec: 4,
      }),
    ).toBe(true)
  })

  it("starts a new session for large jumps and forced swaps", () => {
    expect(
      shouldReuseHlsSessionForSeek({
        localTargetSec: 4 + HLS_SEEK_REUSE_LEAD_SEC + 1,
        writtenEndSec: 4,
      }),
    ).toBe(false)
    expect(
      shouldReuseHlsSessionForSeek({
        forceSessionSwap: true,
        localTargetSec: 2,
        writtenEndSec: 10,
      }),
    ).toBe(false)
    expect(
      shouldReuseHlsSessionForSeek({
        localTargetSec: -1,
        writtenEndSec: 10,
      }),
    ).toBe(false)
  })

  it("uses a longer reuse lead for transcode sessions", () => {
    expect(hlsSeekReuseLeadSec("transcode-hls")).toBe(HLS_TRANSCODE_SEEK_REUSE_LEAD_SEC)
    expect(hlsSeekReuseLeadSec("remux-hls")).toBe(HLS_SEEK_REUSE_LEAD_SEC)
    expect(
      shouldReuseHlsSessionForSeek({
        localTargetSec: 4 + HLS_SEEK_REUSE_LEAD_SEC + 1,
        writtenEndSec: 4,
        reuseLeadSec: hlsSeekReuseLeadSec("transcode-hls"),
        encoderSpeed: "10x",
      }),
    ).toBe(true)
    expect(
      shouldReuseHlsSessionForSeek({
        localTargetSec: 4 + HLS_TRANSCODE_SEEK_REUSE_LEAD_SEC + 1,
        writtenEndSec: 4,
        reuseLeadSec: hlsSeekReuseLeadSec("transcode-hls"),
      }),
    ).toBe(false)
  })

  it("only reuses an unknown window near the session origin", () => {
    expect(
      shouldReuseHlsSessionForSeek({
        localTargetSec: 3,
        writtenEndSec: 0,
      }),
    ).toBe(true)
    expect(
      shouldReuseHlsSessionForSeek({
        localTargetSec: HLS_SEEK_REUSE_LEAD_SEC + 5,
        writtenEndSec: 0,
      }),
    ).toBe(false)
  })
})

describe("isPrematureHlsEndedEvent", () => {
  it("treats ended as window exhaustion when the movie timeline still has remaining time", () => {
    expect(
      isPrematureHlsEndedEvent({
        absoluteTimeSec: 304,
        totalDurationSec: 7200,
      }),
    ).toBe(true)
    expect(
      isPrematureHlsEndedEvent({
        absoluteTimeSec: 7199.6,
        totalDurationSec: 7200,
      }),
    ).toBe(false)
    expect(
      isPrematureHlsEndedEvent({
        absoluteTimeSec: 12,
        totalDurationSec: 0,
      }),
    ).toBe(false)
  })
})

describe("waitForMediaWrittenEnd", () => {
  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it("swaps instead of waiting when a slow encoder cannot catch up within budget", () => {
    expect(shouldReuseHlsSessionForSeek({ localTargetSec: 70, writtenEndSec: 20, encoderSpeed: "1x", reuseLeadSec: 60 })).toBe(false)
    expect(shouldReuseHlsSessionForSeek({ localTargetSec: 70, writtenEndSec: 20, encoderSpeed: "20x", reuseLeadSec: 60 })).toBe(true)
  })

  it("requires contiguous downloaded data after the current time, adjusted for speed", async () => {
    vi.useFakeTimers()
    const media = { ...fakeMedia(7200), currentTime: 30, playbackRate: 2,
      buffered: { length: 2, start: (i: number) => i === 0 ? 0 : 100, end: (i: number): number => i === 0 ? 34 : 200 },
    }
    expect(bufferedPlaybackSeconds(media)).toBe(2)
    const pending = waitForPlaybackBuffer(media, 8, { timeoutMs: 200 })
    await vi.advanceTimersByTimeAsync(400)
    expect(await pending).toBe(false)
    media.buffered.end = (i: number) => i === 0 ? 48 : 200
    expect(await waitForPlaybackBuffer(media, 8)).toBe(true)
    media.currentTime = 50
    expect(bufferedPlaybackSeconds(media)).toBe(0)
  })

  it("resolves immediately when the written window already covers the target", async () => {
    await expect(waitForMediaWrittenEnd(fakeMedia(12), 10)).resolves.toBe(true)
  })

  it("resolves after duration grows to cover the seek target", async () => {
    const media = fakeMedia(4)
    const pending = waitForMediaWrittenEnd(media, 10, { timeoutMs: 2000, intervalMs: 50 })
    media.setDuration(10.2)
    media.emit("durationchange")
    await expect(pending).resolves.toBe(true)
  })

  it("gives up when the written window never catches the target", async () => {
    vi.useFakeTimers()
    const media = fakeMedia(4)
    const pending = waitForMediaWrittenEnd(media, 30, { timeoutMs: 200, intervalMs: 50 })
    await vi.advanceTimersByTimeAsync(250)
    await expect(pending).resolves.toBe(false)
  })
})
