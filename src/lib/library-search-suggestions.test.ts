import { describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import {
  buildLibrarySearchSuggestions,
  librarySearchSuggestionsHasAny,
} from "@/lib/library-search-suggestions"

function m(partial: Partial<Movie> & Pick<Movie, "id" | "title" | "code">): Movie {
  return {
    studio: "",
    actors: [],
    tags: [],
    userTags: [],
    runtimeMinutes: 0,
    rating: 0,
    summary: "",
    isFavorite: false,
    addedAt: "",
    location: "",
    resolution: "",
    year: 0,
    tone: "",
    coverClass: "",
    ...partial,
  }
}

describe("buildLibrarySearchSuggestions", () => {
  it("returns empty groups for blank needle", () => {
    const movies = [m({ id: "1", title: "T", code: "ABC-1", actors: ["A"] })]
    expect(buildLibrarySearchSuggestions("", movies)).toEqual({
      actors: [],
      tags: [],
      codes: [],
    })
    expect(buildLibrarySearchSuggestions("   ", movies)).toEqual({
      actors: [],
      tags: [],
      codes: [],
    })
  })

  it("suggests actor, tag, and code with caps", () => {
    const movies: Movie[] = [
      m({
        id: "1",
        title: "One",
        code: "ZZZ-9",
        actors: ["Mina Kaze", "Rin"],
        tags: ["Drama"],
        userTags: ["Fav"],
      }),
      m({
        id: "2",
        title: "Two",
        code: "MKB-100",
        actors: ["Mina Test"],
        tags: ["DramaX"],
      }),
    ]
    const g = buildLibrarySearchSuggestions("mina", movies, {
      actor: 5,
      tag: 5,
      code: 5,
    } satisfies Partial<Record<"actor" | "tag" | "code", number>>)
    expect(g.actors.map((x) => x.canonical)).toContain("Mina Kaze")
    expect(g.actors.map((x) => x.canonical)).toContain("Mina Test")
    expect(g.codes).toEqual([])
  })

  it("matches code by loose normalization", () => {
    const movies = [m({ id: "x", title: "T", code: "MKB-100" })]
    const g = buildLibrarySearchSuggestions("mkb100", movies)
    expect(g.codes.some((c) => c.code === "MKB-100")).toBe(true)
    expect(librarySearchSuggestionsHasAny(g)).toBe(true)
  })

  it("dedupes actors case-insensitively keeping first canonical spelling", () => {
    const movies: Movie[] = [
      m({ id: "1", title: "A", code: "A-1", actors: ["Mina Kaze"] }),
      m({ id: "2", title: "B", code: "B-1", actors: ["mina kaze"] }),
    ]
    const g = buildLibrarySearchSuggestions("mina", movies)
    expect(g.actors.filter((a) => a.kind === "actor")).toHaveLength(1)
    expect(g.actors[0].canonical).toBe("Mina Kaze")
  })
})
