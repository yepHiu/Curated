import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import { DropdownMenuContent } from "@/components/ui/dropdown-menu"
import BookReaderSettingsMenu from "./BookReaderSettingsMenu.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("BookReaderSettingsMenu", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("exposes comic stitch actions only in page mode", async () => {
    const wrapper = mount(BookReaderSettingsMenu, {
      attachTo: document.body,
      props: {
        kind: "comics",
        mode: "page",
        fit: "contain",
        direction: "ltr",
        showStitch: true,
        stitched: true,
      },
    })

    await wrapper.get("[data-reader-settings-trigger]").trigger("click")
    await flushPromises()

    const stitchPrevious = document.querySelector("[data-reader-stitch-previous]")
    const stitchNext = document.querySelector("[data-reader-stitch-next]")
    const clearStitch = document.querySelector("[data-reader-clear-stitch]")

    expect(stitchPrevious?.getAttribute("title")).toBe("comics.readerStitchPrevious")
    expect(stitchPrevious?.textContent).toContain("comics.readerStitchPreviousShort")
    expect(stitchNext?.getAttribute("title")).toBe("comics.readerStitchNext")
    expect(clearStitch?.getAttribute("title")).toBe("comics.readerClearStitch")

    stitchNext?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    await flushPromises()
    expect(wrapper.emitted("stitchNext")).toHaveLength(1)
    wrapper.unmount()
  })

  it("does not show stitch actions for photos", async () => {
    const wrapper = mount(BookReaderSettingsMenu, {
      attachTo: document.body,
      props: {
        kind: "photos",
        mode: "page",
        fit: "contain",
        direction: "ltr",
        showStitch: false,
      },
    })

    await wrapper.get("[data-reader-settings-trigger]").trigger("click")
    await flushPromises()

    expect(document.querySelector("[data-reader-stitch-previous]")).toBeNull()
    expect(document.querySelector("[data-reader-stitch-next]")).toBeNull()
    wrapper.unmount()
  })

  it("opens above the reader toolbar with a gap instead of covering it", async () => {
    const chrome = document.createElement("div")
    chrome.setAttribute("data-book-reader-chrome", "")
    document.body.appendChild(chrome)

    const wrapper = mount(BookReaderSettingsMenu, {
      attachTo: chrome,
      props: {
        kind: "photos",
        mode: "page",
        fit: "contain",
        direction: "ltr",
      },
    })

    await wrapper.get("[data-reader-settings-trigger]").trigger("click")
    await flushPromises()

    const menu = document.querySelector("[data-reader-settings-menu]")
    const content = wrapper.findComponent(DropdownMenuContent)

    expect(menu?.getAttribute("data-side")).toBe("top")
    expect(content.props("side")).toBe("top")
    expect(content.props("sideOffset")).toBe(8)
    expect(content.props("sideFlip")).toBe(false)
    expect(content.props("collisionPadding")).toBe(8)
    expect(content.props("reference")).toBe(chrome)
    wrapper.unmount()
  })
})
