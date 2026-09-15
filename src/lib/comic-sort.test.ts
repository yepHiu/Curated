import { describe, expect, it } from "vitest"
import type { ComicBook } from "@/domain/comic/types"
import { sortComics, type ComicLibrarySortValue } from "./comic-sort"

function makeComic(
  id: string,
  overrides: Partial<ComicBook> = {},
): ComicBook {
  return {
    id,
    title: id,
    tags: [],
    rating: null,
    isFavorite: false,
    readStatus: "unread",
    pageCount: 10,
    currentPageIndex: 0,
    sourceFileName: `${id}.cbz`,
    location: `D:/Comics/${id}.cbz`,
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
    ...overrides,
  }
}

function ids(comics: ComicBook[]) {
  return comics.map((comic) => comic.id)
}

describe("sortComics", () => {
  it("sorts comics by imported time from newest to oldest by default", () => {
    const comics = [
      makeComic("old", { addedAt: "2026-01-01T00:00:00.000Z" }),
      makeComic("new", { addedAt: "2026-06-01T00:00:00.000Z" }),
      makeComic("middle", { addedAt: "2026-03-01T00:00:00.000Z" }),
    ]

    expect(ids(sortComics(comics, "addedAt"))).toEqual(["new", "middle", "old"])
  })

  it("sorts comics by source filename with numeric collation", () => {
    const comics = [
      makeComic("volume-10", { sourceFileName: "Volume 10.cbz" }),
      makeComic("volume-2", { sourceFileName: "Volume 2.cbz" }),
      makeComic("volume-1", { sourceFileName: "Volume 1.cbz" }),
    ]

    expect(ids(sortComics(comics, "fileName"))).toEqual(["volume-1", "volume-2", "volume-10"])
  })

  it("falls back to imported-time sorting for unknown sort values", () => {
    const comics = [
      makeComic("old", { addedAt: "2026-01-01T00:00:00.000Z" }),
      makeComic("new", { addedAt: "2026-06-01T00:00:00.000Z" }),
    ]

    expect(ids(sortComics(comics, "unexpected" as ComicLibrarySortValue))).toEqual(["new", "old"])
  })
})
