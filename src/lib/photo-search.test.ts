import { describe, expect, it } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import { filterPhotos } from "./photo-search"

function makePhoto(overrides: Partial<PhotoBook> = {}): PhotoBook {
  return {
    id: "photo-1",
    title: "Summer Frame",
    tags: ["portrait"],
    rating: null,
    isFavorite: false,
    pageCount: 8,
    currentPageIndex: 0,
    sourceFileName: "summer.cbz",
    location: "D:/Photos/summer.cbz",
    addedAt: "2026-07-01T00:00:00.000Z",
    updatedAt: "2026-07-01T00:00:00.000Z",
    ...overrides,
  }
}

describe("photo search", () => {
  it("matches title, tags, source filename, and source path", () => {
    const photos = [
      makePhoto({ id: "title", title: "Night Portrait" }),
      makePhoto({ id: "tag", title: "Quiet", tags: ["beach"] }),
      makePhoto({ id: "file", title: "Archive", sourceFileName: "special-volume-02.zip" }),
      makePhoto({ id: "path", title: "Path", location: "E:/Library/Hidden Shelf/path.cbz" }),
    ]

    expect(filterPhotos(photos, { q: "night" }).map((photo) => photo.id)).toEqual(["title"])
    expect(filterPhotos(photos, { q: "beach" }).map((photo) => photo.id)).toEqual(["tag"])
    expect(filterPhotos(photos, { q: "volume-02" }).map((photo) => photo.id)).toEqual(["file"])
    expect(filterPhotos(photos, { q: "hidden shelf" }).map((photo) => photo.id)).toEqual(["path"])
  })

  it("requires an exact tag match instead of a title substring", () => {
    const photos = [
      makePhoto({ id: "exact", tags: ["portrait"] }),
      makePhoto({ id: "title-hit", title: "portrait study", tags: ["studio"] }),
    ]

    expect(filterPhotos(photos, { tag: "portrait" }).map((photo) => photo.id)).toEqual(["exact"])
  })
})

it("browses confirmed credit names as exact tags alongside manual tags", () => {
  const metadata = { cleanTitle: "作品", translatedTitle: "作品", models: [{ name: "青井", evidence: "青井" }], photographers: [{ name: "Studio", evidence: "Studio" }] }
  const book = makePhoto({ tags: [], metadata })
  expect(filterPhotos([book], { tag: " 青井 " })).toEqual([book])
  expect(filterPhotos([book], { tag: "Studio" })).toEqual([book])
  expect(filterPhotos([book], { tag: "青" })).toEqual([])
  expect(filterPhotos([book], { tag: "青井", q: "missing" })).toEqual([])
  expect(filterPhotos([{ ...book, metadata: undefined }], { tag: "青井" })).toEqual([])
  const manual = { ...book, tags: ["青井"], metadata: undefined }
  expect(filterPhotos([manual], { tag: "青井" })).toEqual([manual])
  expect(book.tags).toEqual([])
})
