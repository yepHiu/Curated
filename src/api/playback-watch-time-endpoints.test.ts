import { afterEach, describe, expect, it, vi } from "vitest"
import { api } from "./endpoints"

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("playback watch time endpoints", () => {
  it("requests the daily watch time list with the requested day window", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      jsonResponse({
        items: [{ dayKey: "2026-05-01", watchedSec: 3600 }],
        totalWatchedSec: 3600,
        activeDays: 1,
        maxDayWatchedSec: 3600,
        longestStreakDays: 1,
      }),
    )
    vi.stubGlobal("fetch", fetchMock)

    await expect(api.listPlaybackWatchTimeDaily(91)).resolves.toMatchObject({
      totalWatchedSec: 3600,
      activeDays: 1,
    })

    const [url, init] = fetchMock.mock.calls[0] ?? []
    const parsed = new URL(String(url))
    expect(parsed.pathname).toBe("/api/playback/watch-time/daily")
    expect(parsed.searchParams.get("days")).toBe("91")
    expect(init).toMatchObject({ method: "GET" })
  })

  it("posts bounded daily watch time increments", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response(null, { status: 204 }))
    vi.stubGlobal("fetch", fetchMock)

    await expect(
      api.addPlaybackWatchTimeDaily({
        movieId: "movie-1",
        dayKey: "2026-05-01",
        watchedSec: 42,
      }),
    ).resolves.toBeUndefined()

    const [url, init] = fetchMock.mock.calls[0] ?? []
    const parsed = new URL(String(url))
    expect(parsed.pathname).toBe("/api/playback/watch-time/daily")
    expect(init).toMatchObject({ method: "POST" })
    expect(JSON.parse(String(init?.body))).toEqual({
      movieId: "movie-1",
      dayKey: "2026-05-01",
      watchedSec: 42,
    })
  })
})
