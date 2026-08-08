import { flushPromises, mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"

import AppSidebar from "./AppSidebar.vue"

const movies = ref([
  {
    id: "movie-1",
    actors: ["Actor A", "Actor B"],
    tags: ["Tag A"],
    userTags: [],
  },
])
const trashedMovies = ref([
  {
    id: "trash-1",
    actors: [],
    tags: [],
    userTags: [],
  },
])
const routeState = ref({
  name: "home" as unknown,
  params: {} as Record<string, unknown>,
  query: {} as Record<string, unknown>,
})
const activePlaybackSessionState = ref<unknown>(null)

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
  useRoute: () => routeState.value,
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: computed(() => movies.value),
    trashedMovies: computed(() => trashedMovies.value),
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

vi.mock("@/composables/use-active-playback-session", () => ({
  useActivePlaybackSession: () => ({
    activePlaybackSession: activePlaybackSessionState,
    dismissActivePlaybackSession: vi.fn(),
  }),
}))

vi.mock("@/components/ui/scroll-area", () => ({
  ScrollArea: { name: "ScrollArea", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button><slot /></button>" },
}))

vi.mock("@/components/ui/separator", () => ({
  Separator: { name: "Separator", template: "<hr />" },
}))

function setActivePlaybackSession() {
  activePlaybackSessionState.value = {
    movieId: "movie-1",
    title: "Movie title",
    positionSec: 42,
    durationSec: 120,
    progressPercent: 35,
    status: "paused",
    updatedAt: "2026-05-01T00:00:00.000Z",
    resumeRouteTarget: {
      name: "player",
      params: { id: "movie-1" },
      query: { autoplay: "1", t: "42", back: "browse" },
    },
  }
}

beforeEach(() => {
  routeState.value = {
    name: "home",
    params: {},
    query: {},
  }
  activePlaybackSessionState.value = null
})

describe("AppSidebar", () => {
  it("does not show numeric sidebar counts when movie data exists", async () => {
    const wrapper = mount(AppSidebar, { props: { compact: false } })
    await flushPromises()

    const libraryLink = wrapper
      .findAll("[data-sidebar-nav-link]")
      .find((link) => link.text().includes("nav.library"))

    expect(libraryLink?.text()).toBe("nav.library")
  })

  it("keeps the same core nav link count when toggling compact mode", async () => {
    const wrapper = mount(AppSidebar, { props: { compact: false } })
    await flushPromises()

    const expandedLinks = wrapper.findAll("[data-sidebar-nav-link]")

    await wrapper.setProps({ compact: true })
    await flushPromises()

    const compactLinks = wrapper.findAll("[data-sidebar-nav-link]")

    expect(expandedLinks.length).toBeGreaterThan(0)
    expect(expandedLinks).toHaveLength(compactLinks.length)
  })

  it("centers the brand icon without the collapsed label gap", async () => {
    const wrapper = mount(AppSidebar, { props: { compact: true } })
    await flushPromises()

    const brandLink = wrapper.get("[data-sidebar-brand-link]")
    expect(brandLink.classes()).toContain("justify-center")
    expect(brandLink.classes()).toContain("gap-0")
  })

  it("shows an expanded continue playback card above backend status", async () => {
    setActivePlaybackSession()

    const wrapper = mount(AppSidebar, { props: { compact: false } })
    await flushPromises()

    const card = wrapper.get("[data-active-playback-card]")
    expect(card.text()).toContain("nav.continuePlayback")
    expect(card.text()).toContain("Movie title")
    expect(card.attributes("data-to")).toContain('"t":"42"')
  })

  it("shows an icon-only continue playback entry in compact mode", async () => {
    setActivePlaybackSession()

    const wrapper = mount(AppSidebar, { props: { compact: true } })
    await flushPromises()

    expect(wrapper.find("[data-active-playback-card]").exists()).toBe(false)
    expect(wrapper.find("[data-active-playback-compact]").exists()).toBe(true)
  })

  it("hides the continue playback entry on the same player route", async () => {
    setActivePlaybackSession()
    routeState.value = {
      name: "player",
      params: { id: "movie-1" },
      query: {},
    }

    const wrapper = mount(AppSidebar, { props: { compact: false } })
    await flushPromises()

    expect(wrapper.find("[data-active-playback-card]").exists()).toBe(false)
    expect(wrapper.find("[data-active-playback-compact]").exists()).toBe(false)
  })

  it("keeps the actors nav item active on actor detail pages", async () => {
    routeState.value = {
      name: "actor-detail",
      params: { actorName: "Actor A" },
      query: {},
    }

    const wrapper = mount(AppSidebar, { props: { compact: false } })
    await flushPromises()

    const actorsLink = wrapper
      .findAll("[data-sidebar-nav-link]")
      .find((link) => link.text().includes("nav.actors"))

    expect(actorsLink?.classes()).toContain("bg-sidebar-accent")
  })

  it("exposes personal insights in the Yours group with a direct route target", async () => {
    const wrapper = mount(AppSidebar, { props: { compact: false } })
    await flushPromises()

    const insightsLink = wrapper
      .findAll("[data-sidebar-nav-link]")
      .find((link) => link.text().includes("nav.insights"))

    expect(insightsLink?.attributes("data-to")).toBe('{"name":"insights"}')
  })
})
