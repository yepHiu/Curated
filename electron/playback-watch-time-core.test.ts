import { describe, expect, it, vi } from "vitest"
import { createPlaybackWatchTimeTracker } from "./playback-watch-time-core"
describe("shared watch time calculation", () => {
  it("splits actual wall time at local midnight and does not replay already saved days after a failed flush", async () => {
    let now = new Date(2026, 9, 5, 23, 59, 58).getTime()
    const sink = vi.fn().mockResolvedValueOnce(undefined).mockRejectedValueOnce(new Error("offline")).mockResolvedValue(undefined)
    const tracker = createPlaybackWatchTimeTracker({ movieId: "movie", now: () => now, addDelta: sink })
    tracker.onPlay(10)
    now += 5000
    tracker.onTimeUpdate(15)
    await expect(tracker.flush()).rejects.toThrow("offline")
    await tracker.flush()
    expect(sink.mock.calls).toEqual([["movie", "2026-10-05", 2], ["movie", "2026-10-06", 3], ["movie", "2026-10-06", 3]])
  })
  it("counts wall time at increased speed without counting seek or paused time", async () => {
    let now = new Date(2026, 9, 6, 12).getTime()
    const sink = vi.fn()
    const tracker = createPlaybackWatchTimeTracker({ movieId: "movie", now: () => now, addDelta: sink })
    tracker.onPlay(0); now += 10000; tracker.onTimeUpdate(20)
    tracker.onSeeking(80); now += 5000; tracker.onTimeUpdate(90)
    tracker.onPause(90); now += 10000; tracker.onTimeUpdate(100)
    await tracker.flush()
    expect(sink).toHaveBeenCalledWith("movie", "2026-10-06", 15)
  })
})
