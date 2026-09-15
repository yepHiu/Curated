import { afterEach, describe, expect, it, vi } from "vitest"
import { collectAdjacentBookPageUrls, prefetchBookPageUrls } from "./book-page-prefetch"

describe("collectAdjacentBookPageUrls", () => {
  it("includes current and neighbor original image URLs", () => {
    const pages = [
      { index: 0, imageUrl: "https://example.com/0.jpg" },
      { index: 1, imageUrl: "https://example.com/1.jpg" },
      { index: 2, imageUrl: "https://example.com/2.jpg" },
      { index: 3, imageUrl: "https://example.com/3.jpg" },
    ]

    expect(collectAdjacentBookPageUrls(pages, [1])).toEqual([
      "https://example.com/0.jpg",
      "https://example.com/1.jpg",
      "https://example.com/2.jpg",
    ])
  })
})

describe("prefetchBookPageUrls", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("assigns unique srcs and clears them on cancel", () => {
    const created: { src: string; decoding: string }[] = []
    class FakeImage {
      decoding = ""
      src = ""
      constructor() {
        created.push(this)
      }
    }
    vi.stubGlobal("Image", FakeImage)

    const cancel = prefetchBookPageUrls([
      " https://example.com/a.jpg ",
      "https://example.com/a.jpg",
      "https://example.com/b.jpg",
    ])

    expect(created.map((image) => image.src)).toEqual([
      "https://example.com/a.jpg",
      "https://example.com/b.jpg",
    ])
    cancel()
    expect(created.every((image) => image.src === "")).toBe(true)
  })
})
