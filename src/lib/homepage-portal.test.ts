import { describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"
import { buildHomepagePortalModel } from "@/lib/homepage-portal"

function makeMovie(
  id: string,
  overrides: Partial<Movie> = {},
): Movie {
  return {
    id,
    title: `Movie ${id}`,
    code: `CODE-${id}`,
    studio: "Studio A",
    actors: ["Actor A"],
    tags: ["tag-a"],
    userTags: [],
    runtimeMinutes: 120,
    rating: 4.0,
    metadataRating: 4.0,
    userRating: undefined,
    summary: `Summary ${id}`,
    isFavorite: false,
    addedAt: "2026-04-01T00:00:00.000Z",
    location: `D:/Library/${id}.mp4`,
    resolution: "1080p",
    year: 2026,
    releaseDate: "2026-04-01",
    tone: "from-primary/35 via-primary/10 to-card",
    coverClass: "aspect-[4/5.6]",
    ...overrides,
  }
}

function makeProgress(
  movieId: string,
  overrides: Partial<PlaybackProgressEntry> = {},
): PlaybackProgressEntry {
  return {
    movieId,
    positionSec: 300,
    durationSec: 1200,
    updatedAt: "2026-04-12T08:00:00.000Z",
    ...overrides,
  }
}

describe("buildHomepagePortalModel", () => {
  it("builds a deterministic 8-movie hero from the same day seed", () => {
    const movies = Array.from({ length: 12 }, (_, index) =>
      makeMovie(`m${index + 1}`, {
        addedAt: `2026-04-${String(index + 1).padStart(2, "0")}T00:00:00.000Z`,
      }),
    )

    const first = buildHomepagePortalModel({ movies, daySeed: "2026-04-12" })
    const second = buildHomepagePortalModel({ movies, daySeed: "2026-04-12" })

    expect(first.heroMovies).toHaveLength(8)
    expect(second.heroMovies).toHaveLength(8)
    expect(first.heroMovies.map((movie) => movie.id)).toEqual(
      second.heroMovies.map((movie) => movie.id),
    )
  })

  it("sorts recent imports by addedAt descending", () => {
    const movies = [
      makeMovie("old", { addedAt: "2026-04-01T00:00:00.000Z" }),
      makeMovie("newer", { addedAt: "2026-04-10T00:00:00.000Z" }),
      makeMovie("newest", { addedAt: "2026-04-12T00:00:00.000Z" }),
    ]

    const model = buildHomepagePortalModel({
      movies,
      daySeed: "2026-04-12",
      recentLimit: 3,
      heroLimit: 2,
    })

    expect(model.recentMovies.map((movie) => movie.id)).toEqual(["newest", "newer", "old"])
  })

  it("only includes unfinished progress rows in continue watching", () => {
    const movies = [
      makeMovie("continue-a"),
      makeMovie("continue-b"),
      makeMovie("finished"),
    ]

    const playbackEntries = [
      makeProgress("continue-a", { positionSec: 600, durationSec: 1200 }),
      makeProgress("continue-b", { positionSec: 60, durationSec: 1200 }),
      makeProgress("finished", { positionSec: 1190, durationSec: 1200 }),
    ]

    const model = buildHomepagePortalModel({
      movies,
      playbackEntries,
      daySeed: "2026-04-12",
      continueLimit: 4,
      heroLimit: 2,
    })

    expect(model.continueWatching.map((entry) => entry.movie.id)).toEqual([
      "continue-a",
      "continue-b",
    ])
    expect(model.continueWatching.every((entry) => entry.progressPercent < 95)).toBe(true)
  })

  it("favors taste-matched movies in recommendations over unrelated titles", () => {
    const seedMovie = makeMovie("seed", {
      isFavorite: true,
      userRating: 5,
      rating: 5,
      actors: ["Actor Shared"],
      studio: "Studio Shared",
      tags: ["tag-shared"],
      userTags: ["night"],
    })
    const matchedMovie = makeMovie("matched", {
      actors: ["Actor Shared"],
      studio: "Studio Shared",
      tags: ["tag-shared"],
      rating: 4.1,
      isFavorite: false,
    })
    const unrelatedMovie = makeMovie("unrelated", {
      actors: ["Actor Other"],
      studio: "Studio Other",
      tags: ["tag-other"],
      rating: 4.5,
      isFavorite: false,
    })

    const model = buildHomepagePortalModel({
      movies: [seedMovie, matchedMovie, unrelatedMovie],
      playbackEntries: [makeProgress("seed", { positionSec: 420, durationSec: 1200 })],
      daySeed: "2026-04-12",
      heroLimit: 1,
      recommendationLimit: 2,
    })

    expect(model.recommendations[0]?.movie.id).toBe("matched")
    expect(model.recommendations[0]?.reasons.some((reason) => reason.kind === "actor")).toBe(true)
  })
})
