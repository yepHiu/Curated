import { describe, expect, it } from "vitest"

import { parsePresentBooksContent } from "./agent-book-cards"

describe("parsePresentBooksContent", () => {
  it("keeps comic and photo cards that include kind and the matching id", () => {
    const content = JSON.stringify({
      books: [
        { kind: "comic", comicId: "c1", title: "Summer", tags: ["恋爱"] },
        { kind: "photo", photoId: "p1", title: "Studio" },
        { kind: "comic", title: "missing-id" },
        { kind: "movie", id: "m1", title: "not a book" },
      ],
    })
    expect(parsePresentBooksContent(content)).toEqual([
      { kind: "comic", comicId: "c1", title: "Summer", coverUrl: undefined, photoId: undefined, tags: ["恋爱"] },
      { kind: "photo", photoId: "p1", title: "Studio", coverUrl: undefined, comicId: undefined, tags: undefined },
    ])
  })

  it("returns an empty list for invalid payloads", () => {
    expect(parsePresentBooksContent("not-json")).toEqual([])
    expect(parsePresentBooksContent("{}")).toEqual([])
  })
})
