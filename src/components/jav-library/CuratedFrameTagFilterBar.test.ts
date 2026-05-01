import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameTagFilterBar from "./CuratedFrameTagFilterBar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

describe("CuratedFrameTagFilterBar", () => {
  it("renders visible tag facets with active state and forwards filter actions", async () => {
    const wrapper = mount(CuratedFrameTagFilterBar, {
      props: {
        facets: [
          { name: "favorite", count: 4 },
          { name: "close-up", count: 2 },
        ],
        visibleFacets: [
          { name: "favorite", count: 4 },
          { name: "close-up", count: 2 },
        ],
        activeTag: "close-up",
        hiddenCount: 3,
        expanded: false,
      },
    })
    const buttons = wrapper.findAll("button")
    const clearButton = buttons.find((button) => button.text().includes("curated.tagFilterAll"))!
    const favoriteButton = buttons.find((button) => button.text().includes("favorite"))!
    const activeButton = buttons.find((button) => button.text().includes("close-up"))!
    const expandButton = buttons.find((button) => button.text().includes("curated.tagFilterShowMore"))!

    expect(wrapper.attributes("aria-label")).toBe("curated.tagFilterTitle")
    expect(wrapper.text()).toContain("favorite")
    expect(wrapper.text()).toContain("close-up")
    expect(clearButton.attributes("aria-pressed")).toBe("false")
    expect(activeButton.attributes("aria-pressed")).toBe("true")

    await clearButton.trigger("click")
    await favoriteButton.trigger("click")
    await expandButton.trigger("click")

    expect(wrapper.emitted("clear")).toEqual([[]])
    expect(wrapper.emitted("toggleTag")).toEqual([["favorite"]])
    expect(wrapper.emitted("updateExpanded")).toEqual([[true]])
  })

  it("renders an empty hint when no facets are available", () => {
    const wrapper = mount(CuratedFrameTagFilterBar, {
      props: {
        facets: [],
        visibleFacets: [],
        activeTag: "",
        hiddenCount: 0,
        expanded: false,
      },
    })

    expect(wrapper.text()).toContain("curated.tagFilterEmpty")
    expect(wrapper.find("button").exists()).toBe(false)
  })
})
