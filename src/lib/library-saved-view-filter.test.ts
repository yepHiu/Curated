import { describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import { filterMoviesBySavedView } from "@/lib/library-saved-view-filter"

function movie(id: string, overrides: Partial<Movie> = {}): Movie {
  return {
    id,
    title: id,
    code: id,
    studio: "Studio",
    actors: [],
    tags: [],
    userTags: [],
    runtimeMinutes: 100,
    rating: 4,
    summary: "",
    isFavorite: false,
    addedAt: "2026-07-01T00:00:00Z",
    location: `D:/${id}.mp4`,
    resolution: "1080p",
    year: 2026,
    tone: "",
    coverClass: "",
    ...overrides,
  }
}

describe("filterMoviesBySavedView", () => {
  const movies = [
    movie("unwatched-4k", {
      userRating: 5,
      rating: 5,
      resolution: "2160p",
      addedAt: "2026-07-19T00:00:00Z",
    }),
    movie("in-progress", { userRating: 4 }),
    movie("completed", { userRating: 3, addedAt: "2025-01-01T00:00:00Z" }),
    movie("metadata-five", { userRating: undefined, rating: 5 }),
  ]
  const progress = new Map([
    ["in-progress", { movieId: "in-progress", positionSec: 100, durationSec: 1000, updatedAt: "2026-07-20T00:00:00Z" }],
    ["completed", { movieId: "completed", positionSec: 960, durationSec: 1000, updatedAt: "2026-07-20T00:00:00Z" }],
  ])
  const runtime = {
    now: new Date("2026-07-20T00:00:00Z"),
    hasPlayedMovie: (id: string) => id === "in-progress" || id === "completed",
    getProgress: (id: string) => progress.get(id),
  }

  it("distinguishes unwatched in-progress and completed", () => {
    expect(filterMoviesBySavedView(movies, { schemaVersion: 1, playState: "unwatched" }, runtime).map((item) => item.id)).toEqual([
      "unwatched-4k",
      "metadata-five",
    ])
    expect(filterMoviesBySavedView(movies, { schemaVersion: 1, playState: "in-progress" }, runtime).map((item) => item.id)).toEqual(["in-progress"])
    expect(filterMoviesBySavedView(movies, { schemaVersion: 1, playState: "completed" }, runtime).map((item) => item.id)).toEqual(["completed"])
  })

  it("uses explicit user rating and canonical 4K resolution", () => {
    expect(
      filterMoviesBySavedView(
        movies,
        { schemaVersion: 1, userRating: 5, resolution: "4k" },
        runtime,
      ).map((item) => item.id),
    ).toEqual(["unwatched-4k"])
  })

  it("applies a relative added window from the evaluation time", () => {
    expect(
      filterMoviesBySavedView(
        movies,
        { schemaVersion: 1, addedWithinDays: 30 },
        runtime,
      ).map((item) => item.id),
    ).toEqual(["unwatched-4k", "in-progress", "metadata-five"])
  })
})
