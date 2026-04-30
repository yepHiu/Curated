import { afterEach, describe, expect, it, vi } from "vitest"

const STORAGE_KEY = "jav-library-playback-progress-v1"

function makeApiMock() {
  return {
    listPlaybackProgress: vi.fn(),
    putPlaybackProgress: vi.fn(),
    deletePlaybackProgress: vi.fn(),
  }
}

async function importStorage(opts?: {
  useWebApi?: boolean
  api?: ReturnType<typeof makeApiMock>
}) {
  vi.resetModules()
  vi.stubEnv("VITE_USE_WEB_API", opts?.useWebApi ? "true" : "false")
  const api = opts?.api ?? makeApiMock()
  vi.doMock("@/api/endpoints", () => ({ api }))
  return import("@/lib/playback-progress-storage")
}

afterEach(() => {
  localStorage.clear()
  vi.resetModules()
  vi.clearAllMocks()
  vi.unstubAllEnvs()
})

describe("playback progress storage", () => {
  it("parses fractional second route queries", () => {
    return importStorage().then(({ parseResumeSecondsFromQuery }) => {
      expect(parseResumeSecondsFromQuery("123.456")).toBe(123.456)
    })
  })

  it("ignores malformed localStorage rows when hydrating mock cache", async () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        valid: {
          movieId: "valid",
          positionSec: 12,
          durationSec: 120,
          updatedAt: "2026-04-30T00:00:00.000Z",
        },
        badPosition: {
          movieId: "badPosition",
          positionSec: "12",
          durationSec: 120,
          updatedAt: "2026-04-30T00:00:00.000Z",
        },
        badUpdatedAt: {
          movieId: "badUpdatedAt",
          positionSec: 12,
          durationSec: 120,
          updatedAt: null,
        },
      }),
    )

    const { getProgress, listSortedByUpdatedDesc } = await importStorage()

    expect(getProgress("valid")?.positionSec).toBe(12)
    expect(getProgress("badPosition")).toBeUndefined()
    expect(getProgress("badUpdatedAt")).toBeUndefined()
    expect(listSortedByUpdatedDesc().map((row) => row.movieId)).toEqual(["valid"])
  })

  it("saves local progress with clamped positions and resume thresholds", async () => {
    const {
      getProgress,
      getResumeSecondsForOpenPlayer,
      saveProgress,
    } = await importStorage()

    saveProgress(" movie-1 ", 125.5, 100)
    expect(getProgress("movie-1")).toMatchObject({
      movieId: "movie-1",
      positionSec: 100,
      durationSec: 100,
    })
    expect(getResumeSecondsForOpenPlayer("movie-1")).toBeUndefined()

    saveProgress("movie-1", 10.8, 100)
    expect(getResumeSecondsForOpenPlayer("movie-1")).toBe(10)

    saveProgress("movie-1", 4.9, 100)
    expect(getResumeSecondsForOpenPlayer("movie-1")).toBeUndefined()
  })

  it("lists local progress by newest update and removes entries", async () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        old: {
          movieId: "old",
          positionSec: 10,
          durationSec: 100,
          updatedAt: "2026-04-29T00:00:00.000Z",
        },
        newest: {
          movieId: "newest",
          positionSec: 20,
          durationSec: 100,
          updatedAt: "2026-04-30T00:00:00.000Z",
        },
      }),
    )
    const { getProgress, listSortedByUpdatedDesc, removeProgress } = await importStorage()

    expect(listSortedByUpdatedDesc().map((row) => row.movieId)).toEqual(["newest", "old"])

    removeProgress("newest")

    expect(getProgress("newest")).toBeUndefined()
    expect(listSortedByUpdatedDesc().map((row) => row.movieId)).toEqual(["old"])
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "{}")).not.toHaveProperty("newest")
  })

  it("hydrates web progress and keeps cache on later API failure", async () => {
    const api = makeApiMock()
    api.listPlaybackProgress.mockResolvedValueOnce({
      items: [
        {
          movieId: "web-1",
          positionSec: 30,
          durationSec: 300,
          updatedAt: "2026-04-30T00:00:00.000Z",
        },
      ],
    })
    const { getProgress, hydratePlaybackProgress } = await importStorage({
      useWebApi: true,
      api,
    })

    await hydratePlaybackProgress()
    expect(getProgress("web-1")?.positionSec).toBe(30)

    api.listPlaybackProgress.mockRejectedValueOnce(new Error("offline"))
    await hydratePlaybackProgress()

    expect(getProgress("web-1")?.positionSec).toBe(30)
  })

  it("writes and deletes web progress through the API while updating memory immediately", async () => {
    const api = makeApiMock()
    api.putPlaybackProgress.mockResolvedValue(undefined)
    api.deletePlaybackProgress.mockResolvedValue(undefined)
    const { getProgress, removeProgress, saveProgress } = await importStorage({
      useWebApi: true,
      api,
    })

    saveProgress("web-2", 20, 200)

    expect(getProgress("web-2")?.positionSec).toBe(20)
    expect(api.putPlaybackProgress).toHaveBeenCalledWith("web-2", {
      positionSec: 20,
      durationSec: 200,
    })

    removeProgress("web-2")

    expect(getProgress("web-2")).toBeUndefined()
    expect(api.deletePlaybackProgress).toHaveBeenCalledWith("web-2")
  })
})
