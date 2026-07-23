import { describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"
import type { PlaybackWatchTimeMovieEntry } from "@/lib/playback-watch-time-storage"
import { buildPersonalInsightsBreakdown, buildPersonalInsightsOverview } from "./personal-insights"

const now = new Date("2026-07-21T16:30:00.000Z")

function movie(id: string, overrides: Partial<Movie> = {}): Movie {
  return {
    id,
    title: id,
    code: id.toUpperCase(),
    studio: "",
    actors: [],
    tags: [],
    userTags: [],
    runtimeMinutes: 120,
    rating: 4,
    summary: "",
    isFavorite: false,
    addedAt: "2026-01-01",
    location: `D:/media/${id}.mp4`,
    resolution: "1080p",
    year: 2026,
    tone: "",
    coverClass: "",
    ...overrides,
  }
}

const movies: Movie[] = [
  movie("movie-a", {
    studio: "Studio Alpha",
    actors: ["Alice", "Shared"],
    tags: ["Action", "Drama"],
    userTags: [" action "],
    userRating: 4.5,
  }),
  movie("movie-b", {
    studio: "Studio Beta",
    actors: ["Bob", "Shared"],
    tags: ["Action"],
  }),
  movie("movie-c", {
    studio: "Studio Alpha",
    actors: ["Carol"],
    tags: ["Classic"],
    userRating: 3.5,
  }),
]

const progressEntries: PlaybackProgressEntry[] = [
  { movieId: "movie-a", positionSec: 90, durationSec: 100, updatedAt: "2026-07-02T00:00:00Z" },
  { movieId: "movie-b", positionSec: 20, durationSec: 100, updatedAt: "2026-07-02T00:00:00Z" },
]

const watchTimeEntries: PlaybackWatchTimeMovieEntry[] = [
  { dayKey: "2026-07-01", movieId: "movie-a", watchedSec: 120 },
  { dayKey: "2026-07-02", movieId: "movie-a", watchedSec: 30 },
  { dayKey: "2026-07-02", movieId: "movie-b", watchedSec: 50 },
  { dayKey: "2026-06-01", movieId: "movie-c", watchedSec: 80 },
  { dayKey: "2026-07-01", movieId: "missing-movie", watchedSec: 999 },
  { dayKey: "2026-08-01", movieId: "movie-a", watchedSec: 999 },
]

describe("personal insights aggregates", () => {
  it("uses inclusive local calendar windows and explicit current-state denominators", () => {
    const overview = buildPersonalInsightsOverview({
      movies,
      progressEntries,
      watchTimeEntries,
      range: "30d",
      timezone: "Asia/Shanghai",
      now,
    })

    expect(overview).toEqual({
      range: "30d",
      from: "2026-06-23",
      to: "2026-07-22",
      timezone: "Asia/Shanghai",
      generatedAt: "2026-07-21T16:30:00.000Z",
      dataSince: "2026-06-01",
      watchedSeconds: 200,
      startedMovies: 2,
      completedMovies: 1,
      completionRate: 0.5,
      completionThreshold: 0.9,
      ratedMovies: 1,
      averageUserRating: 4.5,
    })
  })

  it("returns null rates for empty denominators instead of misleading zeroes", () => {
    const overview = buildPersonalInsightsOverview({
      movies,
      progressEntries: [],
      watchTimeEntries: [],
      range: "90d",
      timezone: "UTC",
      now,
    })

    expect(overview).toMatchObject({
      from: "2026-04-23",
      to: "2026-07-21",
      dataSince: null,
      watchedSeconds: 0,
      startedMovies: 0,
      completedMovies: 0,
      completionRate: null,
      ratedMovies: 0,
      averageUserRating: null,
    })
  })

  it("uses dataSince for all-time and includes every non-future library row", () => {
    const overview = buildPersonalInsightsOverview({
      movies,
      progressEntries,
      watchTimeEntries,
      range: "all",
      timezone: "UTC",
      now,
    })

    expect(overview).toMatchObject({
      from: "2026-06-01",
      to: "2026-07-21",
      watchedSeconds: 280,
      startedMovies: 3,
      ratedMovies: 2,
      averageUserRating: 4,
    })
  })

  it("applies full-per-entity attribution, per-movie tag dedupe, stable order, and limit", () => {
    const source = {
      movies,
      progressEntries,
      watchTimeEntries,
      range: "30d" as const,
      timezone: "Asia/Shanghai",
      now,
    }

    expect(buildPersonalInsightsBreakdown({ ...source, dimension: "actor" }).items).toEqual([
      { name: "Shared", watchedSeconds: 200, movieCount: 2, shareOfTotal: 1 },
      { name: "Alice", watchedSeconds: 150, movieCount: 1, shareOfTotal: 0.75 },
      { name: "Bob", watchedSeconds: 50, movieCount: 1, shareOfTotal: 0.25 },
    ])
    expect(buildPersonalInsightsBreakdown({ ...source, dimension: "studio" }).items).toEqual([
      { name: "Studio Alpha", watchedSeconds: 150, movieCount: 1, shareOfTotal: 0.75 },
      { name: "Studio Beta", watchedSeconds: 50, movieCount: 1, shareOfTotal: 0.25 },
    ])
    const tags = buildPersonalInsightsBreakdown({ ...source, dimension: "tag", limit: 1 })
    expect(tags).toMatchObject({
      totalWatchedSeconds: 200,
      attribution: "full-per-entity",
      limit: 1,
    })
    expect(tags.items).toEqual([
      { name: "Action", watchedSeconds: 200, movieCount: 2, shareOfTotal: 1 },
    ])
  })

  it("rejects invalid timezone and out-of-contract limits", () => {
    expect(() => buildPersonalInsightsOverview({
      movies,
      progressEntries,
      watchTimeEntries,
      range: "30d",
      timezone: "Mars/Olympus",
      now,
    })).toThrow("timezone is invalid")
    expect(() => buildPersonalInsightsBreakdown({
      movies,
      progressEntries,
      watchTimeEntries,
      range: "30d",
      timezone: "UTC",
      now,
      dimension: "tag",
      limit: 26,
    })).toThrow("between 1 and 25")
  })

  it("uses movie count then stable case-insensitive name ordering before applying the limit", () => {
    const tieMovies = [
      movie("two-a", { studio: "Two" }),
      movie("two-b", { studio: "Two" }),
      movie("alpha", { studio: "alpha" }),
      movie("zeta", { studio: "Zeta" }),
    ]
    const tieRows = [
      { dayKey: "2026-07-01", movieId: "two-a", watchedSec: 30 },
      { dayKey: "2026-07-01", movieId: "two-b", watchedSec: 30 },
      { dayKey: "2026-07-01", movieId: "alpha", watchedSec: 60 },
      { dayKey: "2026-07-01", movieId: "zeta", watchedSec: 60 },
    ]

    const result = buildPersonalInsightsBreakdown({
      movies: tieMovies,
      progressEntries: [],
      watchTimeEntries: tieRows,
      range: "30d",
      timezone: "UTC",
      now,
      dimension: "studio",
      limit: 2,
    })

    expect(result.items.map((item) => [item.name, item.movieCount])).toEqual([
      ["Two", 2],
      ["alpha", 1],
    ])
  })
})
