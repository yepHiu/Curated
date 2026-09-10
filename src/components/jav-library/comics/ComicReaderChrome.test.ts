import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import ComicReaderChrome from "./ComicReaderChrome.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("ComicReaderChrome", () => {
  it("explains each icon-only reader action with titles", () => {
    const wrapper = mount(ComicReaderChrome, {
      props: {
        pageIndex: 3,
        pageCount: 53,
        mode: "page",
        stitched: true,
      },
    })

    expect(wrapper.get("[data-reader-previous]").attributes("title")).toBe(
      "comics.readerPrevious",
    )
    expect(wrapper.get("[data-reader-next]").attributes("title")).toBe("comics.readerNext")
    expect(wrapper.get("[data-reader-mode-toggle]").attributes("title")).toContain(
      "settings.comicReaderModeScroll",
    )
    expect(wrapper.get("[data-reader-stitch-previous]").attributes("title")).toBe(
      "comics.readerStitchPrevious",
    )
    expect(wrapper.get("[data-reader-stitch-next]").attributes("title")).toBe(
      "comics.readerStitchNext",
    )
    expect(wrapper.get("[data-reader-clear-stitch]").attributes("title")).toBe(
      "comics.readerClearStitch",
    )
  })

  it("uses visible short labels for page stitching actions", () => {
    const wrapper = mount(ComicReaderChrome, {
      props: {
        pageIndex: 2,
        pageCount: 104,
        mode: "page",
        stitched: true,
      },
    })

    const stitchPrevious = wrapper.get("[data-reader-stitch-previous]")
    const stitchNext = wrapper.get("[data-reader-stitch-next]")
    const clearStitch = wrapper.get("[data-reader-clear-stitch]")

    expect(stitchPrevious.classes()).toContain("px-2.5")
    expect(stitchPrevious.text()).toContain("comics.readerStitchPreviousShort")
    expect(stitchPrevious.attributes("aria-label")).toBe("comics.readerStitchPrevious")
    expect(stitchPrevious.attributes("title")).toBe("comics.readerStitchPrevious")

    expect(stitchNext.classes()).toContain("px-2.5")
    expect(stitchNext.text()).toContain("comics.readerStitchNextShort")
    expect(stitchNext.attributes("aria-label")).toBe("comics.readerStitchNext")
    expect(stitchNext.attributes("title")).toBe("comics.readerStitchNext")

    expect(clearStitch.classes()).toContain("px-2.5")
    expect(clearStitch.text()).toContain("comics.readerClearStitchShort")
    expect(clearStitch.attributes("aria-label")).toBe("comics.readerClearStitch")
    expect(clearStitch.attributes("title")).toBe("comics.readerClearStitch")
  })

  it("renders the reader mode toggle inside the bottom toolbar", async () => {
    const wrapper = mount(ComicReaderChrome, {
      props: {
        pageIndex: 1,
        pageCount: 104,
        mode: "page",
      },
    })

    const toolbar = wrapper.get("[data-reader-chrome]")
    const modeToggle = toolbar.get("[data-reader-mode-toggle]")

    expect(modeToggle.attributes("aria-label")).toContain("settings.comicReaderModeScroll")

    await modeToggle.trigger("click")

    expect(wrapper.emitted("toggleMode")).toHaveLength(1)
  })
})
