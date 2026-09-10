import { describe, expect, it } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import { sortPhotos } from "./photo-sort"

function photo(id: string, overrides: Partial<PhotoBook> = {}): PhotoBook {
  return {
    id,
    title: `Photo ${id}`,
    tags: [],
    rating: null,
    isFavorite: false,
    pageCount: 10,
    currentPageIndex: 0,
    sourceFileName: `${id}.cbz`,
    location: `D:/Photos/${id}.cbz`,
    addedAt: "2026-07-01T00:00:00.000Z",
    updatedAt: "2026-07-01T00:00:00.000Z",
    ...overrides,
  }
}

describe("sortPhotos", () => {
  it("sorts by imported time descending by default", () => {
    const got = sortPhotos(
      [
        photo("old", { addedAt: "2026-07-01T00:00:00.000Z" }),
        photo("new", { addedAt: "2026-07-02T00:00:00.000Z" }),
      ],
      "addedAt",
    )

    expect(got.map((item) => item.id)).toEqual(["new", "old"])
  })

  it("sorts naturally by source filename", () => {
    const got = sortPhotos(
      [
        photo("10", { sourceFileName: "photo-10.cbz" }),
        photo("2", { sourceFileName: "photo-2.cbz" }),
      ],
      "fileName",
    )

    expect(got.map((item) => item.id)).toEqual(["2", "10"])
  })

  it("puts favorites first before imported time", () => {
    const got = sortPhotos(
      [
        photo("new", { isFavorite: false, addedAt: "2026-07-03T00:00:00.000Z" }),
        photo("favorite", { isFavorite: true, addedAt: "2026-07-01T00:00:00.000Z" }),
      ],
      "favorite",
    )

    expect(got.map((item) => item.id)).toEqual(["favorite", "new"])
  })
})
