import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import { DropdownMenuContent } from "@/components/ui/dropdown-menu"
import BookReaderChrome from "./BookReaderChrome.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("BookReaderChrome", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })
  it("shows page progress and previous/next actions with titles", () => {
    const wrapper = mount(BookReaderChrome, {
      props: {
        kind: "comics",
        pageIndex: 3,
        pageCount: 53,
        mode: "page",
        fit: "contain",
        direction: "ltr",
        showStitch: true,
        stitched: true,
      },
    })

    expect(wrapper.get("[data-reader-previous]").attributes("title")).toBe("comics.readerPrevious")
    expect(wrapper.get("[data-reader-next]").attributes("title")).toBe("comics.readerNext")
    expect(wrapper.get("[data-reader-settings-trigger]").attributes("title")).toBe(
      "bookBrowser.readerSettings",
    )
    expect(wrapper.get("[data-reader-progress]").attributes("aria-valuenow")).toBe("4")
    expect(wrapper.get("[data-book-reader-chrome]")).toBeTruthy()
    expect(wrapper.text()).toContain("4 / 53")
  })

  it("uses photo labels for the photo viewer chrome and omits stitch until settings open", () => {
    const wrapper = mount(BookReaderChrome, {
      props: {
        kind: "photos",
        pageIndex: 1,
        pageCount: 10,
        mode: "page",
        fit: "contain",
        direction: "ltr",
      },
    })

    expect(wrapper.find("[data-photo-viewer-chrome]").exists()).toBe(true)
    expect(wrapper.get("[data-photo-viewer-previous]").attributes("title")).toBe("photos.viewerPrevious")
    expect(wrapper.find("[data-reader-stitch-previous]").exists()).toBe(false)
  })

  it("opens the settings menu above the toolbar card with a gap", async () => {
    const wrapper = mount(BookReaderChrome, {
      attachTo: document.body,
      props: {
        kind: "comics",
        pageIndex: 0,
        pageCount: 12,
        mode: "page",
        fit: "contain",
        direction: "ltr",
      },
    })

    await wrapper.get("[data-reader-settings-trigger]").trigger("click")
    await flushPromises()

    const chrome = wrapper.get("[data-book-reader-chrome]").element
    const menu = document.querySelector("[data-reader-settings-menu]")
    const content = wrapper.findComponent(DropdownMenuContent)

    expect(menu?.getAttribute("data-side")).toBe("top")
    expect(content.props("sideOffset")).toBe(8)
    expect(content.props("sideFlip")).toBe(false)
    expect(content.props("reference")).toBe(chrome)
    wrapper.unmount()
  })
})
