import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import type { ComicBook, ComicPage, ComicReaderSettings } from "@/domain/comic/types"
import { getTemporaryStitch } from "@/lib/comic-reader-controls"
import ComicReader from "./ComicReader.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("./ComicReaderChrome.vue", () => ({
  default: {
    name: "ComicReaderChrome",
    props: ["pageIndex", "pageCount", "mode", "stitched"],
    emits: ["previous", "next", "toggleMode", "stitchPrevious", "stitchNext", "clearStitch"],
    template: `
      <div data-reader-chrome>
        <span data-reader-chrome-mode>{{ mode }}</span>
        <button data-reader-prev @click="$emit('previous')" />
        <button data-reader-next @click="$emit('next')" />
        <button data-reader-toggle-mode @click="$emit('toggleMode')" />
        <button data-reader-stitch-prev @click="$emit('stitchPrevious')" />
        <button data-reader-stitch-next @click="$emit('stitchNext')" />
        <button data-reader-clear-stitch @click="$emit('clearStitch')" />
      </div>
    `,
  },
}))

vi.mock("./ComicReaderSettingsMenu.vue", () => ({
  default: { name: "ComicReaderSettingsMenu", template: "<div data-reader-settings />" },
}))

function pages(comicId: string): ComicPage[] {
  return Array.from({ length: 4 }, (_, index) => ({
    comicId,
    index,
    entryPath: `${index + 1}.jpg`,
    fileName: `${index + 1}.jpg`,
    imageUrl: `https://example.com/page-${index + 1}.jpg`,
    thumbUrl: `https://example.com/thumb-${index + 1}.jpg`,
  }))
}

function makeComic(): ComicBook {
  return {
    id: "reader-comic",
    title: "Reader Comic",
    tags: [],
    rating: null,
    isFavorite: false,
    readStatus: "unread",
    pageCount: 4,
    currentPageIndex: 0,
    sourceFileName: "reader.cbz",
    location: "D:/Comics/reader.cbz",
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
    pages: pages("reader-comic"),
  }
}

const defaults: ComicReaderSettings = {
  mode: "page",
  fit: "contain",
  direction: "ltr",
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe("ComicReader", () => {
  it("advances and goes back with LTR arrow keys", async () => {
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        readerDefaults: defaults,
      },
    })
    await flushPromises()

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight" }))
    await flushPromises()
    expect(wrapper.get("[data-reader-page-index]").text()).toBe("1")

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowLeft" }))
    await flushPromises()
    expect(wrapper.get("[data-reader-page-index]").text()).toBe("0")
  })

  it("reverses horizontal arrow keys in RTL and still lets space advance", async () => {
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        initialPageIndex: 1,
        readerDefaults: {
          ...defaults,
          direction: "rtl",
        },
      },
    })
    await flushPromises()

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight" }))
    await flushPromises()
    expect(wrapper.get("[data-reader-page-index]").text()).toBe("0")

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowLeft" }))
    await flushPromises()
    expect(wrapper.get("[data-reader-page-index]").text()).toBe("1")

    window.dispatchEvent(new KeyboardEvent("keydown", { key: " " }))
    await flushPromises()
    expect(wrapper.get("[data-reader-page-index]").text()).toBe("2")
  })

  it("uses per-book preferences over global defaults", async () => {
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        readerDefaults: defaults,
        loadPreferences: vi.fn().mockResolvedValue({
          comicId: "reader-comic",
          mode: "scroll",
          fit: "width",
          direction: "rtl",
        }),
      },
    })
    await flushPromises()

    expect(wrapper.get("[data-reader-direction]").text()).toBe("rtl")
    expect(wrapper.get("[data-reader-mode]").text()).toBe("scroll")
    expect(wrapper.get("[data-reader-fit]").text()).toBe("width")
  })

  it("lets page mode images use the full reader surface behind the overlay toolbar", async () => {
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        readerDefaults: defaults,
      },
    })
    await flushPromises()

    expect(wrapper.get("[data-reader-root]").classes()).toContain("h-full")
    expect(wrapper.get("[data-reader-surface]").classes()).toEqual(
      expect.arrayContaining(["h-full", "min-h-0"]),
    )
    expect(wrapper.get("[data-reader-scrollport]").classes()).not.toEqual(
      expect.arrayContaining(["pb-28", "sm:pb-32"]),
    )
    expect(wrapper.get("[data-reader-page-track]").classes()).toEqual(
      expect.arrayContaining(["h-full", "min-h-0", "w-full", "max-h-full"]),
    )
    expect(wrapper.get("[data-reader-visible-page]").classes()).toEqual(
      expect.arrayContaining(["grid", "h-full", "min-h-0", "w-full", "max-h-full"]),
    )
    expect(wrapper.get("[data-reader-image-frame]").classes()).toEqual(
      expect.arrayContaining(["h-full", "min-h-0", "w-full", "max-h-full"]),
    )
    expect(wrapper.get("[data-reader-page-image]").classes()).toEqual(
      expect.arrayContaining(["h-full", "w-full", "max-h-full", "max-w-full", "object-contain"]),
    )
    expect(wrapper.get("[data-reader-page-image]").classes()).not.toEqual(
      expect.arrayContaining(["rounded-lg", "bg-muted", "shadow-xl"]),
    )
  })

  it("toggles the immersive toolbar from the reader surface without reacting to toolbar clicks", async () => {
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        readerDefaults: defaults,
      },
    })
    await flushPromises()

    const overlay = () => wrapper.get("[data-reader-chrome-overlay]")

    expect(overlay().attributes("data-reader-chrome-visible")).toBe("true")
    expect(overlay().classes()).toEqual(expect.arrayContaining(["opacity-100", "translate-y-0"]))

    await wrapper.get("[data-reader-surface]").trigger("click")

    expect(overlay().attributes("data-reader-chrome-visible")).toBe("false")
    expect(overlay().classes()).toEqual(
      expect.arrayContaining(["pointer-events-none", "translate-y-3", "opacity-0"]),
    )

    await wrapper.get("[data-reader-chrome]").trigger("click")

    expect(overlay().attributes("data-reader-chrome-visible")).toBe("false")

    await wrapper.get("[data-reader-surface]").trigger("click")

    expect(overlay().attributes("data-reader-chrome-visible")).toBe("true")
    expect(overlay().classes()).toEqual(expect.arrayContaining(["opacity-100", "translate-y-0"]))
  })

  it("does not render page-number captions under reader images", async () => {
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        readerDefaults: defaults,
      },
    })
    await flushPromises()

    expect(wrapper.find("figcaption").exists()).toBe(false)
    expect(wrapper.get("[data-reader-visible-page]").text()).toBe("")
  })

  it("toggles reader mode from the toolbar and saves per-comic preferences", async () => {
    const savePreferences = vi.fn().mockResolvedValue({
      comicId: "reader-comic",
      mode: "scroll",
      fit: "contain",
      direction: "ltr",
      updatedAt: "2026-06-01T00:00:01.000Z",
    })
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        readerDefaults: defaults,
        savePreferences,
      },
    })
    await flushPromises()

    expect(wrapper.get("[data-reader-chrome-mode]").text()).toBe("page")

    await wrapper.get("[data-reader-toggle-mode]").trigger("click")
    await flushPromises()

    expect(wrapper.get("[data-reader-mode]").text()).toBe("scroll")
    expect(wrapper.get("[data-reader-chrome-mode]").text()).toBe("scroll")
    expect(wrapper.get("[data-reader-scrollport]").classes()).toEqual(
      expect.arrayContaining(["overflow-auto"]),
    )
    expect(savePreferences).toHaveBeenCalledWith("reader-comic", {
      mode: "scroll",
      fit: "contain",
      direction: "ltr",
    })
  })

  it("keeps temporary stitch session-only and clears it on unmount", async () => {
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        initialPageIndex: 1,
        readerDefaults: defaults,
      },
    })
    await flushPromises()

    await wrapper.get("[data-reader-stitch-next]").trigger("click")

    expect(
      wrapper
        .findAll("[data-reader-visible-page]")
        .map((node) => node.get("[data-reader-page-image]").attributes("src")),
    ).toEqual(["https://example.com/page-2.jpg", "https://example.com/page-3.jpg"])
    expect(getTemporaryStitch("reader-comic")).toEqual({
      anchorPageIndex: 1,
      adjacentPageIndex: 2,
    })

    wrapper.unmount()

    expect(getTemporaryStitch("reader-comic")).toBeUndefined()
  })

  it("removes the track gap while temporarily stitching two pages", async () => {
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        initialPageIndex: 1,
        readerDefaults: defaults,
      },
    })
    await flushPromises()

    await wrapper.get("[data-reader-stitch-next]").trigger("click")

    expect(wrapper.get("[data-reader-page-track]").classes()).toContain("gap-0")
    expect(wrapper.get("[data-reader-page-track]").classes()).not.toContain("gap-3")
  })

  it("aligns stitched page images toward the shared seam", async () => {
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        initialPageIndex: 1,
        readerDefaults: defaults,
      },
    })
    await flushPromises()

    await wrapper.get("[data-reader-stitch-next]").trigger("click")

    const images = wrapper.findAll("[data-reader-page-image]")

    expect(images).toHaveLength(2)
    expect(images[0]?.classes()).toContain("object-right")
    expect(images[1]?.classes()).toContain("object-left")
  })

  it("throttles progress saves and marks the final page completed", async () => {
    const saveProgress = vi.fn().mockResolvedValue({})
    const wrapper = mount(ComicReader, {
      props: {
        comic: makeComic(),
        initialPageIndex: 2,
        readerDefaults: defaults,
        saveProgress,
      },
    })
    await flushPromises()

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight" }))
    await flushPromises()
    expect(wrapper.get("[data-reader-page-index]").text()).toBe("3")
    expect(saveProgress).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(350)

    expect(saveProgress).toHaveBeenCalledWith("reader-comic", 3, true)
  })
})
