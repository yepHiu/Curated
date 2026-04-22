import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

import LibraryPage from "./LibraryPage.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: { value: "zh-CN" },
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => ({
    query: {},
  }),
}))

vi.mock("@/lib/library-stats", () => ({
  aggregateMetadataTagCounts: vi.fn(() => []),
  aggregateUserTagCounts: vi.fn(() => []),
}))

vi.mock("@/lib/library-query", () => ({
  getLibraryTagExactQuery: vi.fn(() => ""),
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: { template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    emits: ["click"],
    template: "<button @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { template: "<div><slot /></div>" },
  CardContent: { template: "<div><slot /></div>" },
  CardHeader: { template: "<div><slot /></div>" },
  CardTitle: { template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: { template: "<div data-tabs><slot /></div>" },
  TabsList: { template: "<div data-tabs-list><slot /></div>" },
  TabsTrigger: { template: "<button data-tabs-trigger><slot /></button>" },
}))

vi.mock("lucide-vue-next", () => ({
  CheckSquare: { template: "<span />" },
  ChevronDown: { template: "<span />" },
  ListChecks: { template: "<span />" },
  X: { template: "<span />" },
}))

vi.mock("@/components/jav-library/ActorProfileCard.vue", () => ({
  default: {
    props: ["actorName"],
    template: "<div data-actor-profile-card>{{ actorName }}</div>",
  },
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
        allMovies: [],
        visibleMovies: [],
        activeTab: "all",
        activeActorFilter: "Alpha Star",
        actorUserTagSuggestions: [],
      },
    })

    const html = wrapper.html()
    const tabsIndex = html.indexOf("data-tabs")
    const headerIndex = html.indexOf("data-virtual-masonry-header")
    const actorCardIndex = html.indexOf("data-actor-profile-card")
    const gridIndex = html.indexOf("data-virtual-masonry-grid")

    expect(tabsIndex).toBeGreaterThanOrEqual(0)
    expect(headerIndex).toBeGreaterThan(tabsIndex)
    expect(actorCardIndex).toBeGreaterThan(headerIndex)
    expect(gridIndex).toBeGreaterThan(actorCardIndex)
    expect(wrapper.find("[data-virtual-masonry-header] [data-actor-profile-card]").exists()).toBe(true)
  })

  it("does not render actor profile card inside tags mode layout", () => {
    const wrapper = mount(LibraryPage, {
      props: {
        mode: "tags",
        allMovies: [],
        visibleMovies: [],
        activeTab: "all",
        activeActorFilter: "Alpha Star",
        actorUserTagSuggestions: [],
      },
    })

    expect(wrapper.find("[data-actor-profile-card]").exists()).toBe(false)
  })
})
