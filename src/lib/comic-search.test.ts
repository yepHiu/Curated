import { describe, expect, it } from "vitest"
import type { ComicBook } from "@/domain/comic/types"
import { filterComics } from "./comic-search"

function makeComic(overrides: Partial<ComicBook> = {}): ComicBook {
  return {
    id: "comic-1",
    title: "Rain Garden",
    tags: ["作者:青井", "series:rain"],
    rating: 4,
    isFavorite: false,
    readStatus: "unread",
    pageCount: 20,
    currentPageIndex: 0,
    sourceFileName: "rain-garden.cbz",
    location: "D:/Comics/Rain/rain-garden.cbz",
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
    ...overrides,
  }
}

describe("comic search", () => {
  const comics = [
    makeComic({ id: "title", title: "Glass City Notebook" }),
    makeComic({ id: "tag", title: "Quiet Pages", tags: ["社团:North Pier", "monochrome"] }),
    makeComic({ id: "file", title: "Archive", sourceFileName: "special-volume-02.zip" }),
    makeComic({ id: "path", title: "Path Match", location: "E:/Library/Hidden Shelf/path-match.cbz" }),
  ]

  it("matches title, tags, source filename, and source path", () => {
    expect(filterComics(comics, { q: "glass" }).map((comic) => comic.id)).toEqual(["title"])
    expect(filterComics(comics, { q: "north pier" }).map((comic) => comic.id)).toEqual(["tag"])
    expect(filterComics(comics, { q: "volume-02" }).map((comic) => comic.id)).toEqual(["file"])
    expect(filterComics(comics, { q: "hidden shelf" }).map((comic) => comic.id)).toEqual(["path"])
  })

  it("filters all, favorite, unread, reading, and read comics", () => {
    const list = [
      makeComic({ id: "favorite", isFavorite: true, readStatus: "reading" }),
      makeComic({ id: "unread", readStatus: "unread" }),
      makeComic({ id: "read", readStatus: "read" }),
    ]

    expect(filterComics(list, { readStatus: "all" }).map((comic) => comic.id)).toEqual([
      "favorite",
      "unread",
      "read",
    ])
    expect(filterComics(list, { favorite: true }).map((comic) => comic.id)).toEqual(["favorite"])
    expect(filterComics(list, { readStatus: "unread" }).map((comic) => comic.id)).toEqual(["unread"])
    expect(filterComics(list, { readStatus: "reading" }).map((comic) => comic.id)).toEqual(["favorite"])
    expect(filterComics(list, { readStatus: "read" }).map((comic) => comic.id)).toEqual(["read"])
  })
})
