import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import PhotoPagePreviewGrid from "./PhotoPagePreviewGrid.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

function makePhoto(): PhotoBook {
  return {
    id: "photo-preview-1",
    title: "Summer Frame",
    tags: ["portrait"],
    rating: 4.5,
    isFavorite: false,
    pageCount: 3,
    currentPageIndex: 0,
    coverUrl: "https://example.com/cover.jpg",
    sourceFileName: "summer-frame.cbz",
    location: "D:/Photos/summer-frame.cbz",
    addedAt: "2026-07-01T00:00:00.000Z",
    updatedAt: "2026-07-01T00:00:00.000Z",
    pages: [
      {
        photoId: "photo-preview-1",
        index: 0,
        entryPath: "001.jpg",
        fileName: "001.jpg",
        imageUrl: "https://example.com/page-1.jpg",
        thumbUrl: "https://example.com/thumb-1.jpg",
      },
      {
        photoId: "photo-preview-1",
        index: 1,
        entryPath: "002.jpg",
        fileName: "002.jpg",
        imageUrl: "https://example.com/page-2.jpg",
        thumbUrl: "https://example.com/thumb-2.jpg",
      },
    ],
  }
}

describe("PhotoPagePreviewGrid", () => {
  it("renders image-only preview tiles without visible page labels", async () => {
    const wrapper = mount(PhotoPagePreviewGrid, {
      props: {
        photo: makePhoto(),
      },
    })

    const previews = wrapper.findAll("[data-photo-page-preview]")

    expect(previews).toHaveLength(2)
    expect(previews[0]?.text()).toBe("")
    expect(previews[1]?.text()).toBe("")
    expect(wrapper.text()).not.toContain("第 1 页")
    expect(wrapper.text()).not.toContain("Page 1")
    expect(wrapper.get("[data-photo-page-preview] img").attributes("src")).toBe(
      "https://example.com/thumb-1.jpg",
    )

    await previews[1]?.trigger("click")

    expect(wrapper.emitted("openViewer")?.[0]).toEqual([1])
  })
})
