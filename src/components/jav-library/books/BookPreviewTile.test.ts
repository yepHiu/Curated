import { mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import BookPreviewTile from "./BookPreviewTile.vue"
let intersection: (entries: {isIntersecting: boolean}[]) => void
vi.mock("@vueuse/core", () => ({useIntersectionObserver: (_target: unknown, callback: typeof intersection) => {intersection = callback}}))
afterEach(() => vi.unstubAllGlobals())
describe('preview image scheduling', () => {
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
