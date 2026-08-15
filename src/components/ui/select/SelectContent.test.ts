import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SelectContent from "./SelectContent.vue"

vi.mock("reka-ui", () => ({
  SelectContent: {
    name: "SelectContentRoot",
    template: '<div data-slot="select-content" v-bind="$attrs"><slot /></div>',
  },
  SelectPortal: { name: "SelectPortal", template: "<div><slot /></div>" },
  SelectViewport: {
    name: "SelectViewport",
    template: '<div data-reka-select-viewport data-slot="select-viewport" v-bind="$attrs"><slot /></div>',
  },
  useForwardPropsEmits: (props: unknown) => props,
}))

describe("SelectContent", () => {
  it("keeps overflow on the viewport and does not render scroll chevrons", () => {
    const wrapper = mount(SelectContent, {
      slots: {
        default: "<div>option</div>",
      },
    })

    expect(wrapper.get('[data-slot="select-content"]').classes()).toContain("overflow-hidden")
    expect(wrapper.get('[data-slot="select-content"]').classes()).not.toContain("overflow-y-auto")
    expect(wrapper.get("[data-reka-select-viewport]").classes()).toContain("[scrollbar-gutter:stable]")
    expect(wrapper.find("[data-slot='select-scroll-up-button']").exists()).toBe(false)
    expect(wrapper.find("[data-slot='select-scroll-down-button']").exists()).toBe(false)
  })
})
