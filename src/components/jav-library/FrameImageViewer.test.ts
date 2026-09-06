import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import FrameImageViewer from './FrameImageViewer.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('FrameImageViewer', () => {
  it('offers original-pixel inspection by default after the image loads', async () => {
    const wrapper = mount(FrameImageViewer, { props: { src: 'blob:frame', alt: 'Frame' } })
    const image = wrapper.get('img')
    Object.defineProperties(image.element, { naturalWidth: { value: 1920 }, naturalHeight: { value: 1080 } })
    await image.trigger('load')
    expect(wrapper.text()).toContain('1920 × 1080')
    const zoom = wrapper.get('button')
    expect(zoom.text()).toBe('100%')
    await zoom.trigger('click')
    expect(zoom.attributes('aria-pressed')).toBe('true')
    expect(image.classes()).toContain('max-w-none')
    await wrapper.setProps({ src: 'blob:next' })
    expect(wrapper.find('button').exists()).toBe(false)
    wrapper.unmount()
  })
})
