import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"

import Slider from "./Slider.vue"

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

// reka-ui slider measures layout on mount.
globalThis.ResizeObserver = ResizeObserverStub as typeof ResizeObserver

describe("Slider", () => {
  it("keeps a slim visual track while exposing a larger click target", () => {
    const wrapper = mount(Slider, {
      props: {
        modelValue: [25],
        max: 100,
      },
    })

    const root = wrapper.get('[data-slot="slider"]')
    const track = wrapper.get('[data-slot="slider-track"]')

    expect(root.classes()).toContain("data-[orientation=horizontal]:h-6")
    expect(track.classes()).toContain("absolute")
    expect(track.classes()).toContain("top-1/2")
    expect(track.classes()).toContain("-translate-y-1/2")
    expect(track.classes()).toContain("data-[orientation=horizontal]:h-1.5")
  })
})
