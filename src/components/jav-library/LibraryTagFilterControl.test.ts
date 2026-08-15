import { nextTick } from "vue"
import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import LibraryTagFilterControl from "./LibraryTagFilterControl.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

function mountControl(props: {
  metadataTags: { tag: string; count: number }[]
  userTags: { tag: string; count: number }[]
  selectedTags: string[]
}) {
  return mount(LibraryTagFilterControl, { props })
}

describe("LibraryTagFilterControl", () => {
  it("renders a grouped picker with counts, selected chips, and multi-select", async () => {
    const wrapper = mountControl({
      metadataTags: [{ tag: "Featured", count: 4 }],
      userTags: [{ tag: "mine", count: 2 }],
      selectedTags: ["mine"],
    })

    expect(wrapper.get('[data-library-tag-filter-chip="mine"]').text()).toContain("mine")
    expect(wrapper.get('[data-library-tag-filter-option="mine"]').attributes("aria-pressed")).toBe("true")
    expect(wrapper.get('[data-library-tag-filter-option="Featured"]').attributes("aria-pressed")).toBe("false")
    expect(wrapper.get('[data-library-tag-filter-option="mine"]').attributes("data-library-tag-filter-kind")).toBe(
      "user",
    )
    expect(wrapper.get('[data-library-tag-filter-option="Featured"]').attributes("data-library-tag-filter-kind")).toBe(
      "metadata",
    )
    expect(wrapper.get('[data-library-tag-filter-option="mine"]').text()).toContain("2")
    expect(wrapper.get('[data-library-tag-filter-option="Featured"]').text()).toContain("4")
    expect(wrapper.findAll("[data-library-tag-filter-grid]")).toHaveLength(2)
    expect(wrapper.get("[data-library-tag-filter-grid]").classes().join(" ")).toContain(
      "minmax(11rem,1fr)",
    )

    await wrapper.get("[data-library-tag-filter-clear]").trigger("click")
    await wrapper.get('[data-library-tag-filter-option="Featured"]').trigger("click")
    await wrapper.get('[data-library-tag-filter-chip="mine"]').trigger("click")

    expect(wrapper.emitted("clear")).toEqual([[]])
    expect(wrapper.emitted("toggleTag")).toEqual([["Featured"], ["mine"]])
  })

  it("filters grouped options by search without hiding selected chips", async () => {
    const wrapper = mountControl({
      metadataTags: [{ tag: "Featured", count: 4 }],
      userTags: [{ tag: "mine", count: 2 }],
      selectedTags: ["mine"],
    })

    const search = wrapper.get("[data-library-tag-filter-search]")
    await search.setValue("feat")
    await nextTick()

    expect(wrapper.find('[data-library-tag-filter-option="Featured"]').exists()).toBe(true)
    expect(wrapper.find('[data-library-tag-filter-option="mine"]').exists()).toBe(false)
    expect(wrapper.get('[data-library-tag-filter-chip="mine"]').exists()).toBe(true)
  })

  it("shows the empty library copy when there are no tag facets", () => {
    const wrapper = mountControl({
      metadataTags: [],
      userTags: [],
      selectedTags: [],
    })

    expect(wrapper.text()).toContain("library.savedViewTagEmpty")
    expect(wrapper.find("[data-library-tag-filter-search]").exists()).toBe(false)
  })
})
