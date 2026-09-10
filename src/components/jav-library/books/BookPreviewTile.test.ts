import { mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import BookPreviewTile from "./BookPreviewTile.vue"
let intersection: (entries: {isIntersecting: boolean}[]) => void
vi.mock("@vueuse/core", () => ({useIntersectionObserver: (_target: unknown, callback: typeof intersection) => {intersection = callback}}))
afterEach(() => vi.unstubAllGlobals())
describe('preview image scheduling', () => {
  it('adapts the card to landscape and portrait images and resets when the source changes', async () => {
    const wrapper = mount(BookPreviewTile, {props: {src:'/landscape', label:'Page 1', pageNumber:1, retryLabel:'Retry'}})
    intersection([{isIntersecting:true}]); await wrapper.vm.$nextTick()
    const image = wrapper.get('img')
    Object.defineProperties(image.element, {
      naturalWidth: {value: 1200, configurable: true},
      naturalHeight: {value: 800, configurable: true},
    })
    await image.trigger('load')
    expect(wrapper.attributes('data-aspect-ratio')).toBe('1.5')
    await wrapper.setProps({src:'/portrait'})
    expect(Number(wrapper.attributes('data-aspect-ratio'))).toBeCloseTo(2 / 3)
    Object.defineProperties(image.element, {
      naturalWidth: {value: 600, configurable: true},
      naturalHeight: {value: 1200, configurable: true},
    })
    await image.trigger('load')
    expect(wrapper.attributes('data-aspect-ratio')).toBe('0.5')
    wrapper.unmount()
  })
  it('waits for nearby viewport; errors expose retry without opening the reader', async () => {
    vi.stubGlobal('IntersectionObserver', class {})
    const wrapper = mount(BookPreviewTile, {props: {src:'/thumb', label:'Page 8', pageNumber:8, retryLabel:'Retry'}})
    expect(wrapper.find('img').exists()).toBe(false)
    intersection([{isIntersecting:true}]); await wrapper.vm.$nextTick()
    expect(wrapper.get('img').attributes('decoding')).toBe('async')
    await wrapper.get('img').trigger('error')
    expect(wrapper.attributes('aria-label')).toBe('Retry')
    await wrapper.trigger('click')
    expect(wrapper.find('img').exists()).toBe(true)
    expect(wrapper.emitted('open')).toBeUndefined()
    await wrapper.get('img').trigger('load')
    await wrapper.trigger('click')
    expect(wrapper.emitted('open')).toHaveLength(1)
    wrapper.unmount()
  })
})
