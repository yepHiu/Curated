import { describe, expect, it, vi } from "vitest"
import { createPlaybackWatchTimeTracker } from "./playback-watch-time-tracker"

function localNoonMs(day: string) {
  const [year, month, date] = day.split("-").map(Number)
  return new Date(year, (month ?? 1) - 1, date ?? 1, 12, 0, 0, 0).getTime()
}

describe("playback watch time tracker", () => {
  it("records wall-clock playback time while media time advances", async () => {
    let now = localNoonMs("2026-05-01")
    const addDelta = vi.fn().mockResolvedValue(undefined)
    const tracker = createPlaybackWatchTimeTracker({
      movieId: "movie-1",
      now: () => now,
      addDelta,
    })

    tracker.onPlay(10)
    now += 10_000
    tracker.onTimeUpdate(20)
    now += 12_000
    tracker.onTimeUpdate(32)
    await tracker.flush(32)

    expect(addDelta).toHaveBeenCalledTimes(1)
    expect(addDelta).toHaveBeenCalledWith("movie-1", "2026-05-01", 22)
  })

  it("does not count paused or buffered wall time", async () => {
    let now = localNoonMs("2026-05-01")
    const addDelta = vi.fn().mockResolvedValue(undefined)
    const tracker = createPlaybackWatchTimeTracker({
      movieId: "movie-1",
      now: () => now,
      addDelta,
    })

    tracker.onPlay(10)
    now += 15_000
    tracker.onTimeUpdate(10)
    tracker.onPause(10)
    now += 15_000
    tracker.onTimeUpdate(25)
    await tracker.flush(25)

    expect(addDelta).not.toHaveBeenCalled()
  })

  it("ignores seek jumps and resumes counting from the new media position", async () => {
    let now = localNoonMs("2026-05-01")
    const addDelta = vi.fn().mockResolvedValue(undefined)
    const tracker = createPlaybackWatchTimeTracker({
      movieId: "movie-1",
      now: () => now,
      addDelta,
    })

    tracker.onPlay(10)
    now += 5_000
    tracker.onSeeking(130)
    now += 5_000
    tracker.onTimeUpdate(135)
    await tracker.flush(135)

    expect(addDelta).toHaveBeenCalledTimes(1)
    expect(addDelta).toHaveBeenCalledWith("movie-1", "2026-05-01", 5)
  })

  it("bounds a single sample so background stalls cannot inflate watch time", async () => {
    let now = localNoonMs("2026-05-01")
    const addDelta = vi.fn().mockResolvedValue(undefined)
    const tracker = createPlaybackWatchTimeTracker({
      movieId: "movie-1",
      now: () => now,
      addDelta,
    })

    tracker.onPlay(0)
    now += 120_000
    tracker.onTimeUpdate(120)
    await tracker.flush(120)

    expect(addDelta).toHaveBeenCalledWith("movie-1", "2026-05-01", 30)
  })
})
