import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { ComicBook, ComicPage } from "@/domain/comic/types"
import ComicPagePreviewGrid from "./ComicPagePreviewGrid.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

function pages(comicId: string): ComicPage[] {
  return Array.from({ length: 15 }, (_, index) => ({
    comicId,
    index,
    entryPath: `${String(index + 1).padStart(3, "0")}.jpg`,
    fileName: `${String(index + 1).padStart(3, "0")}.jpg`,
    thumbUrl: `https://example.com/thumb-${index}.jpg`,
  }))
}

function makeComic(): ComicBook {
  return {
    id: "comic-preview-1",
    title: "Preview Comic",
    tags: [],
    rating: null,
    isFavorite: false,
    readStatus: "unread",
    pageCount: 15,
    currentPageIndex: 0,
    sourceFileName: "preview.cbz",
    location: "D:/Comics/preview.cbz",
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
    pages: pages("comic-preview-1"),
  }
}

describe("ComicPagePreviewGrid", () => {
  it("starts with a bounded batch and opens the reader at the clicked page", async () => {
    const wrapper = mount(ComicPagePreviewGrid, {
      props: {
        comic: makeComic(),
      },
    })

    const previews = wrapper.findAll("[data-comic-page-preview]")
    expect(previews).toHaveLength(4)
    expect(wrapper.text()).not.toContain("comics.previewPage")
    expect(previews[0]!.find("img").exists()).toBe(true)
    expect(previews[0]!.classes()).not.toEqual(
      expect.arrayContaining(["border", "bg-card/70", "p-1.5"]),
    )

    await previews[2]!.trigger("click")

    expect(wrapper.emitted("openReader")?.[0]).toEqual([2])
  })
})
