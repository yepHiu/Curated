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

  it("treats userRating as a minimum and can isolate unrated movies", () => {
    expect(
      filterMoviesBySavedView(movies, { schemaVersion: 1, userRating: 4 }, runtime).map((item) => item.id),
    ).toEqual(["unwatched-4k", "in-progress"])
    expect(
      filterMoviesBySavedView(movies, { schemaVersion: 1, unrated: true, userRating: 5 }, runtime).map(
        (item) => item.id,
      ),
    ).toEqual(["metadata-five"])
  })

  it("filters by year runtime and catalog gaps", () => {
    const extra = [
      movie("old-short", {
        year: 2018,
        runtimeMinutes: 60,
        actors: ["A"],
        tags: ["Featured"],
        coverUrl: "https://example/cover.jpg",
        thumbUrl: "https://example/thumb.jpg",
      }),
      movie("unknown-year", {
        year: 0,
        runtimeMinutes: 200,
        actors: [],
        tags: [],
      }),
    ]
    expect(
      filterMoviesBySavedView([...movies, ...extra], { schemaVersion: 1, year: "2018" }, runtime).map(
        (item) => item.id,
      ),
    ).toEqual(["old-short"])
    expect(
      filterMoviesBySavedView([...movies, ...extra], { schemaVersion: 1, year: "unknown" }, runtime).map(
        (item) => item.id,
      ),
    ).toEqual(["unknown-year"])
    expect(
      filterMoviesBySavedView([...movies, ...extra], { schemaVersion: 1, runtime: "short" }, runtime).map(
        (item) => item.id,
      ),
    ).toEqual(["old-short"])
    expect(
      filterMoviesBySavedView([...movies, ...extra], { schemaVersion: 1, catalog: "unscraped" }, runtime).map(
        (item) => item.id,
      ),
    ).toEqual(["unwatched-4k", "in-progress", "completed", "metadata-five", "unknown-year"])
  })

  it("requires every selected tag across user tags and metadata tags", () => {
    const tagged = [
      movie("both", { tags: ["Featured"], userTags: ["mine"] }),
      movie("meta-only", { tags: ["Featured"], userTags: [] }),
      movie("user-only", { tags: ["Drama"], userTags: ["mine"] }),
    ]
    expect(
      filterMoviesBySavedView(tagged, { schemaVersion: 1, tag: "Featured,mine" }, runtime).map((item) => item.id),
    ).toEqual(["both"])
    expect(
      filterMoviesBySavedView(tagged, { schemaVersion: 1, tag: "mine" }, runtime).map((item) => item.id),
    ).toEqual(["both", "user-only"])
  })

  it("requires every selected actor and any selected studio", () => {
    const cast = [
      movie("both", { actors: ["Actor A", "Actor B"], studio: "Studio A" }),
      movie("one-actor", { actors: ["Actor A"], studio: "Studio B" }),
      movie("other-studio", { actors: ["Actor B"], studio: "Studio C" }),
    ]
    expect(
      filterMoviesBySavedView(cast, { schemaVersion: 1, actor: "Actor A,Actor B" }, runtime).map(
        (item) => item.id,
      ),
    ).toEqual(["both"])
    expect(
      filterMoviesBySavedView(cast, { schemaVersion: 1, studio: "Studio A,Studio C" }, runtime).map(
        (item) => item.id,
      ),
    ).toEqual(["both", "other-studio"])
  })
})
