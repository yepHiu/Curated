import { flushPromises, mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import BookMediaInfoDialog from "./BookMediaInfoDialog.vue"

vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe("BookMediaInfoDialog", () => {
  it("shows source details read-only and supports closing", async () => {
    const wrapper = mount(BookMediaInfoDialog, {
      props: { open: true, book: { title: "A book", sourceFileName: "archive.cbz", location: "D:/Books/archive.cbz", pageCount: 24, addedAt: "2026-09-01T00:00:00Z", updatedAt: "2026-09-02T00:00:00Z" } },
      global: { stubs: { DialogContent: { template: "<section><slot /></section>" } } },
    })
    await flushPromises()
    const dialog = wrapper.get('[data-book-media-info]')
    expect(dialog.text()).toContain("A book")
    expect(dialog.text()).toContain("D:/Books/archive.cbz")
    expect(dialog.findAll('dd').map(row => row.text())).toEqual([
      "archive.cbz", "D:/Books/archive.cbz", "CBZ", "24",
      new Date("2026-09-01T00:00:00Z").toLocaleString(), new Date("2026-09-02T00:00:00Z").toLocaleString(),
    ])
    expect(dialog.find('input').exists()).toBe(false)
    const close = dialog.findAll('button').find(button => button.text() === "common.close")!
    await close.trigger('click')
    expect(wrapper.emitted('update:open')).toEqual([[false]])
    wrapper.unmount()
  })

  it("shows placeholders for absent source fields and invalid dates", async () => {
    const wrapper = mount(BookMediaInfoDialog, {
      props: { open: true, book: { title: "No source", sourceFileName: "", location: "", pageCount: 0, addedAt: "", updatedAt: "invalid" } },
      global: { stubs: { DialogContent: { template: "<section><slot /></section>" } } },
    })
    await flushPromises()
    expect(wrapper.findAll('dd').map(row => row.text())).toEqual(["—", "—", "—", "0", "—", "—"])
    wrapper.unmount()
  })
})
