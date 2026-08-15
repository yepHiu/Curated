import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

import LibraryPage from "./LibraryPage.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: { value: "zh-CN" },
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    emits: ["click"],
    template: "<button @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("lucide-vue-next", () => ({
  CheckSquare: { template: "<span />" },
  ListChecks: { template: "<span />" },
  X: { template: "<span />" },
}))

vi.mock("@/components/jav-library/ActorProfileCard.vue", () => ({
  default: {
    props: ["actorName"],
    template: "<div data-actor-profile-card>{{ actorName }}</div>",
  },
}))

vi.mock("@/components/jav-library/LibrarySavedViewsControls.vue", () => ({
  default: { template: '<div data-saved-view-controls><slot /></div>' },
}))

vi.mock("@/components/jav-library/VirtualMovieMasonry.vue", () => ({
  default: {
    template: `
      <div data-virtual-masonry>
        <div data-virtual-masonry-header><slot name="header" /></div>
        <div data-virtual-masonry-grid />
      </div>
    `,
  },
}))

describe("LibraryPage", () => {
  it("passes actor profile card into masonry header in normal library mode", () => {
    const wrapper = mount(LibraryPage, {
      props: {
        mode: "library",
        visibleMovies: [],
        activeActorFilter: "Alpha Star",
        actorUserTagSuggestions: [],
      },
    })

    const html = wrapper.html()
    const toolbarIndex = html.indexOf("data-saved-view-controls")
    const headerIndex = html.indexOf("data-virtual-masonry-header")
    const actorCardIndex = html.indexOf("data-actor-profile-card")
    const gridIndex = html.indexOf("data-virtual-masonry-grid")

    expect(toolbarIndex).toBeGreaterThanOrEqual(0)
    expect(headerIndex).toBeGreaterThan(toolbarIndex)
    expect(actorCardIndex).toBeGreaterThan(headerIndex)
    expect(gridIndex).toBeGreaterThan(actorCardIndex)
    expect(wrapper.find("[data-virtual-masonry-header] [data-actor-profile-card]").exists()).toBe(true)
  })

  it("keeps a compact toolbar without the old sort tabs", () => {
    const wrapper = mount(LibraryPage, {
      props: {
        mode: "library",
        visibleMovies: [],
      },
    })

    expect(wrapper.get("h1").classes()).toContain("sr-only")
    expect(wrapper.find("[data-library-filter-tabs]").exists()).toBe(false)
    expect(wrapper.find("[data-library-tab-trigger]").exists()).toBe(false)
    expect(wrapper.get("[data-library-batch-toggle]").classes()).toContain("min-h-11")
    expect(wrapper.get("[data-library-batch-toggle]").classes()).not.toContain("min-h-12")
    expect(wrapper.get("[data-saved-view-controls]").element.parentElement?.className).toContain(
      "justify-end",
    )
  })

  it("places batch manage in the same saved-view action cluster as filters", () => {
    const wrapper = mount(LibraryPage, {
      props: {
        mode: "library",
        visibleMovies: [],
      },
    })

    expect(
      wrapper.get("[data-saved-view-controls]").find("[data-library-batch-toggle]").exists(),
    ).toBe(true)
  })
})
