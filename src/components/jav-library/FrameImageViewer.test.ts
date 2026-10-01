import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import FrameImageViewer from './FrameImageViewer.vue'

// 使用翻译键保持查看器状态文案断言与语言无关。
vi.mock('vue-i18n', () => ({
  // 为组件提供最小翻译接口。
  useI18n: () => ({
    // 返回键名以识别重试文案。
    t: (key: string) => key,
  }),
}))

// 验证萃取帧大图展示及图片切换后的状态。
describe('FrameImageViewer', () => {
  // 加载完成后不遮挡图片，切换后仍可从加载失败中重试。
  it('fits the image without metadata controls and retains loading retry', async () => {
    const wrapper = mount(FrameImageViewer, { props: { src: 'blob:frame', alt: 'Frame' } })
    const image = wrapper.get('img')
    Object.defineProperties(image.element, { naturalWidth: { value: 1920 }, naturalHeight: { value: 1080 } })
    await image.trigger('load')
    expect(wrapper.text()).not.toContain('1920 × 1080')
    expect(wrapper.text()).not.toContain('100%')
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.find('.absolute').exists()).toBe(false)
    expect(image.classes()).toContain('object-contain')
    await wrapper.setProps({ src: 'blob:next' })
    expect(wrapper.find('.absolute').exists()).toBe(true)
    await wrapper.get('img').trigger('error')
    expect(wrapper.get('button').text()).toBe('curated.retryLoad')
    await wrapper.get('button').trigger('click')
    expect(wrapper.get('img').attributes('src')).toBe('blob:next')
    await wrapper.get('img').trigger('load')
    expect(wrapper.find('.absolute').exists()).toBe(false)
    wrapper.unmount()
  })
})
