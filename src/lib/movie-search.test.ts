import { describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import { movieSearchHaystack, normalizeLooseCode } from "@/lib/movie-search"

function minimalMovie(partial: Partial<Movie> & Pick<Movie, "id" | "title" | "code">): Movie {
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

describe("normalizeLooseCode", () => {
  it("folds case and removes separators", () => {
    expect(normalizeLooseCode("MKB-100")).toBe("mkb100")
    expect(normalizeLooseCode("abc_123")).toBe("abc123")
    expect(normalizeLooseCode("FC2 PPV 123")).toBe("fc2ppv123")
  })
})

describe("movieSearchHaystack", () => {
  it("matches loose code query against hyphenated code", () => {
    const m = minimalMovie({
      id: "x",
      title: "T",
      code: "MKB-100",
    })
    const hay = movieSearchHaystack(m)
    expect(hay.includes("mkb100")).toBe(true)
    expect(hay.includes("mkb-100")).toBe(true)
  })

  it("still matches title and actors", () => {
    const m = minimalMovie({
      id: "x",
      title: "Midnight",
      code: "X-1",
      actors: ["Mina Kaze"],
    })
    const hay = movieSearchHaystack(m)
    expect(hay.includes("midnight")).toBe(true)
    expect(hay.includes("mina")).toBe(true)
  })
})
