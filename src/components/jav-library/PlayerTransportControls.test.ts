import { defineComponent, nextTick, ref } from "vue"
import { mount } from "@vue/test-utils"
import { afterEach, describe, expect, it } from "vitest"
import PlayerTransportControls from "./PlayerTransportControls.vue"

class ResizeObserverStub {
  /** 滑块挂载时会注册尺寸观察，测试里不需要真实布局回调。 */
  observe() {}
  /** 尺寸观察的注销在测试里是空操作。 */
  unobserve() {}
  /** 组件卸载时断开观察。 */
  disconnect() {}
}

const labels = {
  play: "播放", pause: "暂停", seekBack: "后退", seekForward: "前进",
  volume: "音量", mute: "静音", unmute: "取消静音",
}

/** 返回 jsdom 缺少的矩形序列化结果。 */
function emptyRectJson() {
  return {}
}

/** 给滑块一个稳定的横向轨道，避免 jsdom 里矩形全是 0。 */
function stubSliderRect(slider: HTMLElement) {
  /** 把轨道固定成 100 像素宽，便于用 clientX 直接对应百分比。 */
  function sliderRect() {
    return { x: 0, y: 0, left: 0, top: 0, right: 100, bottom: 24, width: 100, height: 24, toJSON: emptyRectJson }
  }
  /** 拇指宽度置 0，换算时不必再扣半宽。 */
  function thumbRect() {
    return { x: 0, y: 0, left: 0, top: 0, right: 0, bottom: 0, width: 0, height: 0, toJSON: emptyRectJson }
  }
  slider.getBoundingClientRect = sliderRect
  slider.querySelectorAll<HTMLElement>("[data-slot=slider-thumb]").forEach(markThumbRect)
  /** 逐个拇指套上零宽度矩形。 */
  function markThumbRect(thumb: HTMLElement) {
    thumb.getBoundingClientRect = thumbRect
  }
}

/** 在音量条上按下并移动，模拟捕获没有把后续移动送回按下节点。 */
function dragVolume(slider: HTMLElement, from: number, to: number) {
  const track = slider.querySelector("[data-slot=slider-track]") ?? slider
  track.dispatchEvent(new PointerEvent("pointerdown", {
    bubbles: true, cancelable: true, button: 0, pointerId: 7, clientX: from,
  }))
  window.dispatchEvent(new PointerEvent("pointermove", {
    bubbles: true, pointerId: 7, clientX: to, buttons: 1,
  }))
}

/** 把音量写回滑块，确认拖动中的数值会反映到拇指。 */
const VolumeHarness = defineComponent({
  components: { PlayerTransportControls },
  setup() {
    const volume = ref(20)
    /** 父组件按播放器的方式立即采用拖动中的音量。 */
    function onVolume(values?: number[]) {
      const next = values?.[0]
      if (next !== undefined) volume.value = next
    }
    return { volume, onVolume, labels }
  },
  template: `<PlayerTransportControls follow-volume-pointer :playing="true" :volume-values="[volume]" :volume-percent="volume" :muted="volume === 0" :labels="labels" @volume="onVolume" />`,
})

describe("PlayerTransportControls volume drag", () => {
  const originalCapture = HTMLElement.prototype.setPointerCapture

  afterEach(() => {
    HTMLElement.prototype.setPointerCapture = originalCapture
    document.body.innerHTML = ""
  })

  it("follows pointer moves when the native window does not retarget pointer capture", async () => {
    globalThis.ResizeObserver = ResizeObserverStub as typeof ResizeObserver
    /** 测试里不让指针捕获抛错，移动仍从窗口事件送入。 */
    function ignorePointerCapture() {}
    HTMLElement.prototype.setPointerCapture = ignorePointerCapture
    const wrapper = mount(VolumeHarness, { attachTo: document.body })
    const slider = wrapper.get("[data-slot=slider]").element as HTMLElement
    stubSliderRect(slider)
    const seen: number[] = []
    /** 记录滑块自己是否还收到按下，接管成功时不应再进入组件库。 */
    function noteSliderPointerDown() { seen.push(-1) }
    slider.addEventListener("pointerdown", noteSliderPointerDown)
    dragVolume(slider, 25, 80)
    await nextTick()
    expect(seen).toEqual([])
    expect(slider.querySelector("[data-slot=slider-thumb]")?.getAttribute("aria-valuenow")).toBe("80")
    wrapper.unmount()
  })

  it("leaves the web player slider in charge when pointer following is off", () => {
    globalThis.ResizeObserver = ResizeObserverStub as typeof ResizeObserver
    /** 关闭跟随时仍可能走到组件库的捕获调用，这里只避免 jsdom 抛错。 */
    function ignorePointerCapture() {}
    HTMLElement.prototype.setPointerCapture = ignorePointerCapture
    const wrapper = mount(PlayerTransportControls, {
      props: { playing: true, volumeValues: [20], volumePercent: 20, muted: false, labels },
      attachTo: document.body,
    })
    const slider = wrapper.get("[data-slot=slider]").element as HTMLElement
    stubSliderRect(slider)
    dragVolume(slider, 25, 80)
    const volumes = wrapper.emitted("volume") ?? []
    expect(volumes).not.toContainEqual([[80]])
    wrapper.unmount()
  })
})
