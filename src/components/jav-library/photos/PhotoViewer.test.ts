import { flushPromises, mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { PhotoBook, PhotoPage, PhotoViewerSettings } from "@/domain/photo/types"
import PhotoViewer from "./PhotoViewer.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

function pages(photoId: string): PhotoPage[] {
  return Array.from({ length: 4 }, (_, index) => ({
    photoId,
    index,
    entryPath: `${index + 1}.jpg`,
    fileName: `${index + 1}.jpg`,
    imageUrl: `https://example.com/photo-page-${index + 1}.jpg`,
    thumbUrl: `https://example.com/photo-thumb-${index + 1}.jpg`,
  }))
}

function makePhoto(): PhotoBook {
  return {
    id: "viewer-photo",
    title: "Viewer Photo",
    tags: [],
    rating: null,
    isFavorite: false,
    pageCount: 4,
    currentPageIndex: 0,
    sourceFileName: "viewer.cbz",
    location: "D:/Photos/viewer.cbz",
    addedAt: "2026-07-01T00:00:00.000Z",
    updatedAt: "2026-07-01T00:00:00.000Z",
    pages: pages("viewer-photo"),
  }
}

const defaults: PhotoViewerSettings = {
  mode: "page",
  fit: "contain",
  direction: "ltr",
}

describe("PhotoViewer", () => {
  it("uses photo browsing chrome and does not render comic stitch controls", async () => {
    const wrapper = mount(PhotoViewer, {
      props: {
        photo: makePhoto(),
        viewerDefaults: defaults,
      },
    })
    await flushPromises()

    expect(wrapper.find("[data-photo-viewer-chrome]").exists()).toBe(true)
    expect(wrapper.text()).toContain("photos.viewerBrowse")
    expect(wrapper.find("[data-reader-stitch-previous]").exists()).toBe(false)
    expect(wrapper.find("[data-reader-stitch-next]").exists()).toBe(false)
    expect(wrapper.find("[data-reader-clear-stitch]").exists()).toBe(false)
    expect(wrapper.text()).not.toContain("comics.readerStitchPrevious")
    expect(wrapper.text()).not.toContain("comics.readerStitchNext")
  })

  it("lets page mode images use the full viewer surface behind a hideable toolbar", async () => {
    const wrapper = mount(PhotoViewer, {
      props: {
        photo: makePhoto(),
        viewerDefaults: defaults,
      },
    })
    await flushPromises()

    expect(wrapper.get("[data-photo-viewer-root]").classes()).toContain("h-full")
    expect(wrapper.get("[data-photo-viewer-surface]").classes()).toEqual(
      expect.arrayContaining(["h-full", "min-h-0"]),
    )
    expect(wrapper.get("[data-photo-viewer-page-track]").classes()).toEqual(
      expect.arrayContaining(["h-full", "min-h-0", "w-full", "max-h-full"]),
    )
    expect(wrapper.get("[data-photo-viewer-page-image]").classes()).toEqual(
      expect.arrayContaining(["h-full", "w-full", "max-h-full", "max-w-full", "object-contain"]),
    )

    const overlay = () => wrapper.get("[data-photo-viewer-chrome-overlay]")

    expect(overlay().attributes("data-photo-viewer-chrome-visible")).toBe("true")
    await wrapper.get("[data-photo-viewer-surface]").trigger("click")
    expect(overlay().attributes("data-photo-viewer-chrome-visible")).toBe("false")
    await wrapper.get("[data-photo-viewer-chrome]").trigger("click")
    expect(overlay().attributes("data-photo-viewer-chrome-visible")).toBe("false")
  })

  it("advances with arrow keys and saves no page captions under the image", async () => {
    const wrapper = mount(PhotoViewer, {
      props: {
        photo: makePhoto(),
        viewerDefaults: defaults,
      },
    })
    await flushPromises()

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight" }))
    await flushPromises()

    expect(wrapper.get("[data-photo-viewer-page-index]").text()).toBe("1")
    expect(wrapper.get("[data-photo-viewer-page-image]").attributes("src")).toBe(
      "https://example.com/photo-page-2.jpg",
    )
    expect(wrapper.find("figcaption").exists()).toBe(false)
    expect(wrapper.get("[data-photo-viewer-visible-page]").text()).toBe("")
  })

  it("prefetches the current and next original page images", async () => {
    const created: { src: string }[] = []
    class FakeImage {
      decoding = ""
      src = ""
      constructor() {
        created.push(this)
      }
    }
    vi.stubGlobal("Image", FakeImage)

    const wrapper = mount(PhotoViewer, {
      props: {
        photo: makePhoto(),
        viewerDefaults: defaults,
      },
    })
    await flushPromises()

    expect(created.map((image) => image.src)).toEqual([
      "https://example.com/photo-page-1.jpg",
      "https://example.com/photo-page-2.jpg",
    ])
    wrapper.unmount()
    vi.unstubAllGlobals()
  })
})
