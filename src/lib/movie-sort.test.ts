import { describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import {
  compareMoviesByLibrarySort,
  librarySortKeyFromTab,
  libraryTabFromSortKey,
} from "@/lib/movie-sort"

function movie(id: string, overrides: Partial<Movie> = {}): Movie {
  return {
    id,
    title: id,
    code: id,
    studio: "",
    actors: [],
    tags: [],
    userTags: [],
    runtimeMinutes: 100,
    rating: 0,
    summary: "",
    isFavorite: false,
    addedAt: "2026-01-01T00:00:00Z",
    location: `D:/${id}.mp4`,
    resolution: "1080p",
    year: 0,
    tone: "",
    coverClass: "",
    ...overrides,
  }
}

describe("movie-sort", () => {
  it("maps tabs to and from the shared library sort keys", () => {
    expect(librarySortKeyFromTab("all")).toBe("added")
    expect(librarySortKeyFromTab("new")).toBe("release")
    expect(librarySortKeyFromTab("top-rated")).toBe("rating")
    expect(libraryTabFromSortKey("code")).toBe("none")
    expect(libraryTabFromSortKey("release")).toBe("new")
  })

  it("sorts by code, actor, studio, and year with empty values last", () => {
    const items = [
      movie("b", { code: "DEF-2", actors: ["Zoe"], studio: "Beta", year: 2020, addedAt: "2026-02-01T00:00:00Z" }),
      movie("a", { code: "ABC-10", actors: ["Amy"], studio: "Alpha", year: 2024, addedAt: "2026-01-01T00:00:00Z" }),
      movie("empty", { code: "", actors: [], studio: "", year: 0, addedAt: "2026-03-01T00:00:00Z" }),
    ]

    expect([...items].sort((left, right) => compareMoviesByLibrarySort(left, right, "code")).map((item) => item.id)).toEqual([
      "a",
      "b",
      "empty",
    ])
    expect([...items].sort((left, right) => compareMoviesByLibrarySort(left, right, "actor")).map((item) => item.id)).toEqual([
      "a",
      "b",
      "empty",
    ])
    expect([...items].sort((left, right) => compareMoviesByLibrarySort(left, right, "studio")).map((item) => item.id)).toEqual([
      "a",
      "b",
      "empty",
    ])
    expect([...items].sort((left, right) => compareMoviesByLibrarySort(left, right, "year")).map((item) => item.id)).toEqual([
      "a",
      "b",
      "empty",
    ])
  })

  it("sorts by added time without using catalog-code id", () => {
    // 同一天应按入库时刻排序；日期相同且无法区分时刻时保持原次序，不用番号。
    const later = movie("aaa-001", { code: "AAA-001", addedAt: "2026-09-15T18:00:00Z" })
    const earlier = movie("zzz-999", { code: "ZZZ-999", addedAt: "2026-09-15T10:00:00Z" })
    expect(
      [earlier, later].sort((left, right) => compareMoviesByLibrarySort(left, right, "added")).map((item) => item.id),
    ).toEqual(["aaa-001", "zzz-999"])

    const sameDayLaterId = movie("zzz-999", { code: "ZZZ-999", addedAt: "2026-09-15" })
    const sameDayEarlierId = movie("aaa-001", { code: "AAA-001", addedAt: "2026-09-15" })
    expect(
      [sameDayLaterId, sameDayEarlierId]
        .sort((left, right) => compareMoviesByLibrarySort(left, right, "added"))
        .map((item) => item.id),
    ).toEqual(["zzz-999", "aaa-001"])
  })
})
