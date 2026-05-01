import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameEmptyState from "./CuratedFrameEmptyState.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("CuratedFrameEmptyState", () => {
  it("renders the library empty state", () => {
    const wrapper = mount(CuratedFrameEmptyState, {
      props: {
        variant: "library",
        showClearFilter: false,
      },
    })

    expect(wrapper.text()).toContain("curated.empty")
    expect(wrapper.find("button").exists()).toBe(false)
  })

  it("renders filtered empty state and forwards clear action", async () => {
    const wrapper = mount(CuratedFrameEmptyState, {
      props: {
        variant: "filtered",
        showClearFilter: true,
      },
    })

    expect(wrapper.text()).toContain("curated.tagFilterNoMatches")
    expect(wrapper.get("button").text()).toContain("curated.tagFilterAll")

    await wrapper.get("button").trigger("click")

    expect(wrapper.emitted("clearFilter")).toEqual([[]])
  })
})
