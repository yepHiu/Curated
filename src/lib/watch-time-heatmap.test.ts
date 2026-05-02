import { describe, expect, it } from "vitest"
import {
  WATCH_TIME_HEATMAP_DAYS,
  WATCH_TIME_HEATMAP_WEEKS,
  buildWatchTimeHeatmap,
  buildWatchTimeSummary,
  formatWatchTimeDuration,
  getWatchTimeLevel,
} from "./watch-time-heatmap"

describe("watch time heatmap", () => {
  it("builds a Sunday-aligned 13 week grid around the current week", () => {
    const heatmap = buildWatchTimeHeatmap([], {
      today: new Date(2026, 4, 1),
    })

    expect(WATCH_TIME_HEATMAP_WEEKS).toBe(13)
    expect(WATCH_TIME_HEATMAP_DAYS).toBe(91)
    expect(heatmap.weeks).toHaveLength(13)
    expect(heatmap.cells).toHaveLength(91)
    expect(heatmap.cells[0]?.dayKey).toBe("2026-02-01")
    expect(heatmap.cells.at(-1)?.dayKey).toBe("2026-05-02")
    expect(heatmap.cells.filter((cell) => cell.isFuture).map((cell) => cell.dayKey)).toEqual([
      "2026-05-02",
    ])
  })

  it("maps watch seconds to five visual levels", () => {
    expect(getWatchTimeLevel(0)).toBe(0)
    expect(getWatchTimeLevel(60)).toBe(1)
    expect(getWatchTimeLevel(1_799)).toBe(1)
    expect(getWatchTimeLevel(1_800)).toBe(2)
    expect(getWatchTimeLevel(5_399)).toBe(2)
    expect(getWatchTimeLevel(5_400)).toBe(3)
    expect(getWatchTimeLevel(8_999)).toBe(3)
    expect(getWatchTimeLevel(9_000)).toBe(4)
  })

  it("summarizes this week, past three months, max day, and longest streak", () => {
    const summary = buildWatchTimeSummary(
      [
        { dayKey: "2026-01-31", watchedSec: 999 },
        { dayKey: "2026-04-20", watchedSec: 60 },
        { dayKey: "2026-04-21", watchedSec: 60 },
        { dayKey: "2026-04-22", watchedSec: 60 },
        { dayKey: "2026-04-27", watchedSec: 600 },
        { dayKey: "2026-04-28", watchedSec: 3_600 },
        { dayKey: "2026-04-30", watchedSec: 120 },
        { dayKey: "2026-05-01", watchedSec: 7_200 },
      ],
      { today: new Date(2026, 4, 1) },
    )

    expect(summary.thisWeekWatchedSec).toBe(11_520)
    expect(summary.totalWatchedSec).toBe(11_700)
    expect(summary.activeDays).toBe(7)
    expect(summary.maxDayWatchedSec).toBe(7_200)
    expect(summary.longestStreakDays).toBe(3)
  })

  it("formats compact watch time labels", () => {
    expect(formatWatchTimeDuration(0)).toBe("0m")
    expect(formatWatchTimeDuration(45)).toBe("1m")
    expect(formatWatchTimeDuration(45 * 60)).toBe("45m")
    expect(formatWatchTimeDuration(90 * 60)).toBe("1h 30m")
  })
})
