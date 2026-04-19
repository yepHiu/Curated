import { flushPromises, mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { describe, expect, it, vi } from "vitest"

import AppSidebar from "./AppSidebar.vue"

const updateAvailable = ref(false)

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  RouterLink: {
    name: "RouterLink",
    props: ["to"],
    template: "<a :data-to=\"JSON.stringify(to)\"><slot /></a>",
  },
  useRoute: () => ({ name: "home", query: {} }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: computed(() => []),
    trashedMovies: computed(() => []),
  }),
}))

vi.mock("@/composables/use-backend-health", () => ({
  useBackendHealth: () => ({
    useWebApi: true,
    status: ref("online"),
    probing: ref(false),
    versionDisplay: ref("1.2.4"),
    checkNow: vi.fn(),
  }),
}))

vi.mock("@/composables/use-app-update", () => ({
  useAppUpdate: () => ({
    useWebApi: true,
    status: computed(() => (updateAvailable.value ? "update-available" : "up-to-date")),
    summary: computed(() =>
      updateAvailable.value
        ? {
            supported: true,
            status: "update-available",
            installedVersion: "1.2.7",
            latestVersion: "1.2.8",
            hasUpdate: true,
          }
        : {
            supported: true,
            status: "up-to-date",
            installedVersion: "1.2.8",
            latestVersion: "1.2.8",
            hasUpdate: false,
          },
    ),
    hasUpdateBadge: computed(() => updateAvailable.value),
    checkNow: vi.fn(),
    ensureLoaded: vi.fn(),
  }),
}))

vi.mock("@/lib/curated-frames/db", () => ({
  countCuratedFrames: vi.fn(async () => 0),
}))

vi.mock("@/lib/curated-frames/revision", () => ({
  curatedFramesRevision: ref(0),
}))

vi.mock("@/lib/playback-progress-storage", () => ({
  listSortedByUpdatedDesc: vi.fn(() => []),
  playbackProgressRevision: ref(0),
}))

vi.mock("@/components/ui/scroll-area", () => ({
  ScrollArea: { name: "ScrollArea", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button><slot /></button>" },
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: { name: "Badge", template: "<span><slot /></span>" },
}))

vi.mock("@/components/ui/separator", () => ({
  Separator: { name: "Separator", template: "<hr />" },
}))

describe("AppSidebar", () => {
  it("shows the brand update badge in expanded mode when a new version is available", async () => {
    updateAvailable.value = true

    const wrapper = mount(AppSidebar, { props: { compact: false } })
    await flushPromises()

    expect(wrapper.find("[data-update-badge]").exists()).toBe(true)
    expect(wrapper.find("[data-update-badge]").text()).toContain("New")
  })

  it("shows the compact brand update dot when a new version is available", async () => {
    updateAvailable.value = true

    const wrapper = mount(AppSidebar, { props: { compact: true } })
    await flushPromises()

    expect(wrapper.find("[data-update-dot]").exists()).toBe(true)
  })

  it("keeps the same core nav link count when toggling compact mode", async () => {
    updateAvailable.value = false

    const wrapper = mount(AppSidebar, { props: { compact: false } })
    await flushPromises()

    const expandedLinks = wrapper.findAll("[data-sidebar-nav-link]")

    await wrapper.setProps({ compact: true })
    await flushPromises()

    const compactLinks = wrapper.findAll("[data-sidebar-nav-link]")

    expect(expandedLinks.length).toBeGreaterThan(0)
    expect(expandedLinks).toHaveLength(compactLinks.length)
  })
})
