import { afterEach, describe, expect, it, vi } from "vitest"

const STORAGE_KEY = "curated-playback-watch-time-daily-v1"

function makeApiMock() {
  return {
    listPlaybackWatchTimeDaily: vi.fn(),
    addPlaybackWatchTimeDaily: vi.fn(),
  }
}

function localDayKey(date: Date) {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, "0")
  const d = String(date.getDate()).padStart(2, "0")
  return `${y}-${m}-${d}`
}

function daysAgo(n: number) {
  const date = new Date()
  date.setHours(12, 0, 0, 0)
  date.setDate(date.getDate() - n)
  return localDayKey(date)
}

async function importStorage(opts?: {
  useWebApi?: boolean
  api?: ReturnType<typeof makeApiMock>
}) {
  vi.resetModules()
  vi.stubEnv("VITE_USE_WEB_API", opts?.useWebApi ? "true" : "false")
  const api = opts?.api ?? makeApiMock()
  vi.doMock("@/api/endpoints", () => ({ api }))
  return import("@/lib/playback-watch-time-storage")
}

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
  vi.resetModules()
  vi.clearAllMocks()
  vi.unstubAllEnvs()
})

describe("playback watch time storage", () => {
  it("accumulates mock watch time by day and movie in localStorage", async () => {
    const today = daysAgo(0)
    const yesterday = daysAgo(1)
    const outsideDefaultWindow = daysAgo(92)
    const {
      addWatchTimeDelta,
      listDailyWatchTime,
      watchTimeRevision,
    } = await importStorage()
    const initialRevision = watchTimeRevision.value

    await addWatchTimeDelta(" movie-1 ", today, 120)
    await addWatchTimeDelta("movie-1", today, 30)
    await addWatchTimeDelta("movie-2", today, 50)
    await addWatchTimeDelta("movie-1", yesterday, 70)
    await addWatchTimeDelta("movie-3", outsideDefaultWindow, 999)

    const summary = await listDailyWatchTime()

    expect(watchTimeRevision.value).toBe(initialRevision + 5)
    expect(summary.items).toEqual([
      { dayKey: today, watchedSec: 200 },
      { dayKey: yesterday, watchedSec: 70 },
    ])
    expect(summary.totalWatchedSec).toBe(270)
    expect(summary.activeDays).toBe(2)
    expect(summary.maxDayWatchedSec).toBe(200)
    expect(summary.longestStreakDays).toBe(2)
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "{}")).toMatchObject({
      [today]: {
        "movie-1": 150,
        "movie-2": 50,
      },
      [yesterday]: {
        "movie-1": 70,
      },
    })
  })

  it("ignores invalid mock increments without bumping revision", async () => {
    const today = daysAgo(0)
    const { addWatchTimeDelta, listDailyWatchTime, watchTimeRevision } =
      await importStorage()
    const initialRevision = watchTimeRevision.value

    await addWatchTimeDelta("", today, 120)
    await addWatchTimeDelta("movie-1", "2026-5-1", 120)
    await addWatchTimeDelta("movie-1", today, 0)
    await addWatchTimeDelta("movie-1", today, Number.NaN)

    expect(watchTimeRevision.value).toBe(initialRevision)
    await expect(listDailyWatchTime()).resolves.toMatchObject({
      items: [],
      totalWatchedSec: 0,
      activeDays: 0,
    })
  })

  it("uses the Web API for listing and adding daily watch time", async () => {
    const api = makeApiMock()
    api.listPlaybackWatchTimeDaily.mockResolvedValueOnce({
      items: [{ dayKey: "2026-05-01", watchedSec: 600 }],
      totalWatchedSec: 600,
      activeDays: 1,
      maxDayWatchedSec: 600,
      longestStreakDays: 1,
    })
    api.addPlaybackWatchTimeDaily.mockResolvedValue(undefined)
    const {
      addWatchTimeDelta,
      listDailyWatchTime,
      watchTimeRevision,
    } = await importStorage({ useWebApi: true, api })
    const initialRevision = watchTimeRevision.value

    await addWatchTimeDelta(" movie-1 ", "2026-05-01", 480)
    const summary = await listDailyWatchTime()

    expect(api.addPlaybackWatchTimeDaily).toHaveBeenCalledWith({
      movieId: "movie-1",
      dayKey: "2026-05-01",
      watchedSec: 300,
    })
    expect(api.listPlaybackWatchTimeDaily).toHaveBeenCalledWith(91)
    expect(summary.totalWatchedSec).toBe(600)
    expect(watchTimeRevision.value).toBe(initialRevision + 1)
  })
})
