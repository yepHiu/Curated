import { describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import {
  listMoviesMatchingSavedView,
  SAVED_VIEW_CONFIRM_PREVIEW,
  splitSavedViewConfirmMovies,
} from "@/lib/saved-view-preview"

function movie(id: string, overrides: Partial<Movie> = {}): Movie {
  return {
    id,
    title: id,
    code: id,
    studio: "Studio",
    actors: ["Ada"],
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

describe("listMoviesMatchingSavedView", () => {
  it("returns movies that match saved-view filters", () => {
    const movies = [
      movie("ada-unwatched"),
      movie("other-unwatched", { actors: ["Mina"] }),
      movie("ada-favorite", { isFavorite: true }),
    ]
    const matched = listMoviesMatchingSavedView({
      movies,
      trashedMovies: [],
      filters: { schemaVersion: 1, actor: "Ada" },
      hasPlayedMovie: () => false,
      getProgress: () => undefined,
    })
    expect(matched.map((item) => item.id).sort()).toEqual(["ada-favorite", "ada-unwatched"])
  })
})

describe("splitSavedViewConfirmMovies", () => {
  it("folds titles beyond the preview count", () => {
    const movies = Array.from({ length: 6 }, (_, index) => movie(`m${index}`))
    const collapsed = splitSavedViewConfirmMovies(movies, false)
    expect(collapsed.visible).toHaveLength(SAVED_VIEW_CONFIRM_PREVIEW)
    expect(collapsed.foldedCount).toBe(2)
    expect(collapsed.canExpand).toBe(true)
    const expanded = splitSavedViewConfirmMovies(movies, true)
    expect(expanded.visible).toHaveLength(6)
    expect(expanded.canCollapse).toBe(true)
    expect(expanded.remainderAfterExpand).toBe(0)
  })
})
