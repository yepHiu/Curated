import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import BookPagePreviewGrid from "./BookPagePreviewGrid.vue"
const width = ref(792)
vi.mock("@vueuse/core", async original => ({ ...await original<typeof import("@vueuse/core")>(), useElementSize: () => ({width, height: ref(0)}) }))
vi.mock("vue-i18n", () => ({useI18n: () => ({t: (key: string) => key})}))
const pages = Array.from({length: 1000}, (_, index) => ({index, thumbUrl: `/thumb/${index}`}))
beforeEach(() => { width.value = 792 })
describe("adaptive book previews", () => {
  it("sizes two rows to container width and bounds DOM even for a thousand pages", async () => {
    const wrapper = mount(BookPagePreviewGrid, {props: {bookId: "a", pages, total: 1000, kind: "photos"}})
    expect(wrapper.findAll('[data-book-preview-tile]')).toHaveLength(12)
    width.value = 300; await flushPromises()
    expect(wrapper.findAll('[data-book-preview-tile]')).toHaveLength(4)
    width.value = 3000; await flushPromises()
    expect(wrapper.findAll('[data-book-preview-tile]')).toHaveLength(20)
    wrapper.unmount()
  })
  it("replaces batches; locates the last page; opens its actual index and resets for another book", async () => {
    const wrapper = mount(BookPagePreviewGrid, {props: {bookId: "a", pages, total: 1000, kind: "comics"}})
    await wrapper.get('[data-preview-next]').trigger('click')
    expect(wrapper.findAll('[data-book-preview-tile]')[0]!.text()).toBe('13')
    await wrapper.get('[data-preview-previous]').trigger('click')
    expect(wrapper.findAll('[data-book-preview-tile]')[0]!.text()).toBe('1')
    await wrapper.get('[data-preview-jump]').setValue('1000')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.findAll('[data-book-preview-tile]')).toHaveLength(4)
    expect(wrapper.get<HTMLButtonElement>('[data-preview-next]').element.disabled).toBe(true)
    await wrapper.findAll('[data-book-preview-tile]').at(-1)!.trigger('click')
    expect(wrapper.emitted('open')).toEqual([[999]])
    await wrapper.setProps({bookId: 'b', pages: pages.slice(0,2), total: 2})
    expect(wrapper.findAll('[data-book-preview-tile]')[0]!.text()).toBe('1')
    expect(wrapper.get<HTMLInputElement>('[data-preview-jump]').element.value).toBe('1')
    wrapper.unmount()
  })
  it("rejects out-of-range and fractional page jumps", async () => {
    const wrapper = mount(BookPagePreviewGrid, {props: {bookId: "a", pages: pages.slice(0,10), total: 10, kind: "photos"}})
    for (const value of ['0','11','1.5']) {
      await wrapper.get('[data-preview-jump]').setValue(value)
      expect(wrapper.get<HTMLButtonElement>('button[type="submit"]').element.disabled).toBe(true)
      await wrapper.get('form').trigger('submit')
      expect(wrapper.findAll('[data-book-preview-tile]')[0]!.text()).toBe('1')
    }
    wrapper.unmount()
  })
})
