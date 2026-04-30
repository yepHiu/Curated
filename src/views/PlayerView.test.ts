import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const routeState = vi.hoisted(() => ({
  params: {} as Record<string, unknown>,
  query: {} as Record<string, unknown>,
}))
const serviceState = vi.hoisted(() => ({
  movies: new Map<string, unknown>(),
}))
const serviceMocks = vi.hoisted(() => ({
  ensureMovieCached: vi.fn(),
}))
const recordMoviePlayedMock = vi.hoisted(() => vi.fn())

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    getMovieById: (id?: string) => (id ? serviceState.movies.get(id) : undefined),
    ensureMovieCached: serviceMocks.ensureMovieCached,
  }),
}))

vi.mock("@/lib/played-movies-storage", () => ({
  recordMoviePlayed: recordMoviePlayedMock,
}))

vi.mock("@/components/jav-library/PlayerPage.vue", () => ({
  default: {
    name: "PlayerPage",
    props: ["movie", "autoplay"],
    template:
      '<section data-player-page :data-movie-id="movie.id" :data-autoplay="String(autoplay)" />',
  },
}))

vi.mock("@/components/jav-library/NotFoundState.vue", () => ({
  default: {
    name: "NotFoundState",
    props: ["title", "description"],
    template: '<section data-not-found>{{ title }} {{ description }}</section>',
  },
}))

function movie(id: string) {
  return {
    id,
    title: `Title ${id}`,
    code: id.toUpperCase(),
  }
}

async function mountPlayerView() {
  const { default: PlayerView } = await import("./PlayerView.vue")
  return mount(PlayerView)
}

beforeEach(() => {
  vi.resetModules()
  vi.unstubAllEnvs()
  vi.stubEnv("VITE_USE_WEB_API", "false")
  routeState.params = {}
  routeState.query = {}
  serviceState.movies = new Map()
  serviceMocks.ensureMovieCached.mockReset()
  recordMoviePlayedMock.mockReset()
})

afterEach(() => {
  vi.unstubAllEnvs()
})

describe("PlayerView", () => {
  it("renders PlayerPage for a cached movie and passes autoplay from the route", async () => {
    routeState.params = { id: "movie-1" }
    routeState.query = { autoplay: "1" }
    serviceState.movies.set("movie-1", movie("movie-1"))

    const wrapper = await mountPlayerView()

    const player = wrapper.get("[data-player-page]")
    expect(player.attributes("data-movie-id")).toBe("movie-1")
    expect(player.attributes("data-autoplay")).toBe("true")
    expect(recordMoviePlayedMock).toHaveBeenCalledWith("movie-1")
  })

  it("renders NotFoundState when the target movie is unavailable", async () => {
    routeState.params = { id: "missing" }

    const wrapper = await mountPlayerView()

    expect(wrapper.get("[data-not-found]").text()).toContain("player.notFoundTitle")
    expect(wrapper.get("[data-not-found]").text()).toContain("player.notFoundDesc")
    expect(serviceMocks.ensureMovieCached).not.toHaveBeenCalled()
  })

  it("shows a loading state while Web API hydration is resolving", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    routeState.params = { id: "movie-1" }
    serviceMocks.ensureMovieCached.mockImplementationOnce(
      () =>
        new Promise<void>(() => {
          // Keep hydration pending so the loading branch remains visible.
        }),
    )

    const wrapper = await mountPlayerView()
    await flushPromises()

    expect(wrapper.text()).toContain("player.loadingTarget")
    expect(serviceMocks.ensureMovieCached).toHaveBeenCalledWith("movie-1")
  })
})
