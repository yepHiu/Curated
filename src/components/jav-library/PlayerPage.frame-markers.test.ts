import { flushPromises, mount } from "@vue/test-utils"
import { nextTick } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"

const routeState = vi.hoisted(() => ({
  query: {} as Record<string, unknown>,
  hash: "",
}))

const routerMocks = vi.hoisted(() => ({
  replace: vi.fn(),
}))

const serviceMocks = vi.hoisted(() => ({
  getMoviePlayback: vi.fn(),
  createPlaybackSession: vi.fn(),
  deletePlaybackSession: vi.fn(),
}))

const framePageMocks = vi.hoisted(() => ({
  listCuratedFramesPage: vi.fn(),
}))

const activePlaybackMocks = vi.hoisted(() => ({
  updateActivePlaybackSession: vi.fn(),
  clearActivePlaybackSession: vi.fn(),
}))

const serviceState = vi.hoisted(() => ({
  playerSettings: {
    value: {
      seekBackwardStepSec: 10,
      seekForwardStepSec: 10,
      nativePlayerPreset: "custom",
      nativePlayerCommand: "",
      nativePlayerEnabled: false,
    },
  },
}))

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
  useRouter: () => routerMocks,
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: { value: "zh-CN" },
    t: (key: string) => key,
  }),
}))

vi.mock("@/i18n", () => ({
  i18n: {
    global: {
      t: (key: string) => key,
    },
  },
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    playerSettings: serviceState.playerSettings,
    movies: { value: [] },
    trashedMovies: { value: [] },
    getMoviePlayback: serviceMocks.getMoviePlayback,
    createPlaybackSession: serviceMocks.createPlaybackSession,
    deletePlaybackSession: serviceMocks.deletePlaybackSession,
  }),
}))

vi.mock("@/composables/use-active-playback-session", () => ({
  updateActivePlaybackSession: activePlaybackMocks.updateActivePlaybackSession,
  clearActivePlaybackSession: activePlaybackMocks.clearActivePlaybackSession,
}))

vi.mock("@/lib/hls-player", () => ({
  buildHlsPlaybackConfig: vi.fn(() => ({})),
  canPlayHlsNatively: vi.fn(() => false),
  loadHlsLibrary: vi.fn(),
  preloadHlsLibrary: vi.fn(),
  prewarmHlsResources: vi.fn(),
  startHlsLoadingAtSessionOrigin: vi.fn(),
}))

vi.mock("@/lib/curated-frames/db", () => ({
  listCuratedFramesPage: framePageMocks.listCuratedFramesPage,
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: {
    name: "Badge",
    template: "<span><slot /></span>",
  },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled"],
    template: '<button :disabled="disabled"><slot /></button>',
  },
}))

vi.mock("@/components/ui/slider", () => ({
  Slider: {
    name: "Slider",
    props: ["disabled", "modelValue"],
    template: '<div data-slider :data-disabled="String(disabled)" />',
  },
}))

vi.mock("@/components/jav-library/PlayerPlaybackSettingsMenu.vue", () => ({
  default: {
    name: "PlayerPlaybackSettingsMenu",
    template: "<div data-playback-settings-menu />",
  },
}))

function movie(overrides: Partial<Movie> = {}): Movie {
  return {
    id: "movie-1",
    title: "Movie title",
    code: "ABC-123",
    studio: "Studio",
    actors: ["Mina"],
    tags: [],
    userTags: [],
    runtimeMinutes: 120,
    rating: 4,
    summary: "Summary",
    isFavorite: false,
    addedAt: "2026-04-30T00:00:00.000Z",
    location: "D:/media/movie-1.mp4",
    resolution: "1080p",
    year: 2026,
    tone: "neutral",
    coverClass: "bg-muted",
    ...overrides,
  }
}

async function mountPlayerPage(props: { movie?: Movie; autoplay?: boolean } = {}) {
  const { default: PlayerPage } = await import("./PlayerPage.vue")
  const wrapper = mount(PlayerPage, {
    props: {
      movie: props.movie ?? movie(),
      autoplay: props.autoplay ?? false,
    },
    global: {
      stubs: {
        Teleport: true,
        Transition: true,
      },
    },
  })
  await nextTick()
  return wrapper
}

function frameRow(id: string, positionSec: number, movieId = "movie-1") {
  return {
    id,
    movieId,
    title: "Movie title",
    code: "ABC-123",
    actors: ["Mina"],
    positionSec,
    capturedAt: "2026-08-16T00:00:00.000Z",
    tags: [],
  }
}

function directDescriptor(durationSec = 120) {
  return {
    movieId: "movie-1",
    mode: "direct",
    url: "/api/library/movies/movie-1/stream",
    durationSec,
    canDirectPlay: true,
  }
}

beforeEach(() => {
  vi.resetModules()
  vi.stubEnv("VITE_USE_WEB_API", "false")
  vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(() => {})
  vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(() => {})
  routeState.query = {}
  routeState.hash = ""
  routerMocks.replace.mockReset()
  serviceMocks.getMoviePlayback.mockReset()
  serviceMocks.createPlaybackSession.mockReset()
  serviceMocks.deletePlaybackSession.mockReset()
  activePlaybackMocks.updateActivePlaybackSession.mockReset()
  activePlaybackMocks.clearActivePlaybackSession.mockReset()
  framePageMocks.listCuratedFramesPage.mockReset()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllEnvs()
})

describe("PlayerPage progress frame markers", () => {
  it("loads this movie's curated frames and renders them once duration is known", async () => {
    framePageMocks.listCuratedFramesPage.mockResolvedValue({
      items: [frameRow("f1", 30), frameRow("f2", 60), frameRow("f3", 90)],
      total: 3,
      limit: 200,
      offset: 0,
    })
    serviceMocks.getMoviePlayback.mockResolvedValueOnce(directDescriptor())
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()

      expect(framePageMocks.listCuratedFramesPage).toHaveBeenCalledWith({
        movieId: "movie-1",
        limit: 200,
        offset: 0,
      })
      const ticks = wrapper.findAll("[data-frame-marker]")
      expect(ticks).toHaveLength(3)
      // jsdom 未实现 ResizeObserver → 宽度未知前每帧独立展示，位置按时间换算
      expect(ticks[0].attributes("style")).toContain("left: 25%")
      expect(ticks[1].attributes("style")).toContain("left: 50%")
      expect(ticks[2].attributes("style")).toContain("left: 75%")
    } finally {
      wrapper.unmount()
    }
  })

  it("fetches follow-up pages until the reported total is covered", async () => {
    // 位置都压在 120 秒时长内，避免触发越界脏数据过滤
    const firstPage = Array.from({ length: 200 }, (_, index) => frameRow(`a${index}`, index / 10))
    framePageMocks.listCuratedFramesPage
      .mockResolvedValueOnce({ items: firstPage, total: 201, limit: 200, offset: 0 })
      .mockResolvedValueOnce({
        items: [frameRow("b0", 20.5)],
        total: 201,
        limit: 200,
        offset: 200,
      })
    serviceMocks.getMoviePlayback.mockResolvedValueOnce(directDescriptor())
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()

      expect(framePageMocks.listCuratedFramesPage).toHaveBeenCalledTimes(2)
      expect(framePageMocks.listCuratedFramesPage).toHaveBeenLastCalledWith({
        movieId: "movie-1",
        limit: 200,
        offset: 200,
      })
      expect(wrapper.findAll("[data-frame-marker]").length).toBe(201)
    } finally {
      wrapper.unmount()
    }
  })

  it("degrades silently to no markers when loading fails", async () => {
    framePageMocks.listCuratedFramesPage.mockRejectedValue(new Error("idb unavailable"))
    serviceMocks.getMoviePlayback.mockResolvedValueOnce(directDescriptor())
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()

      expect(wrapper.findAll("[data-frame-marker]")).toHaveLength(0)
      expect(wrapper.text()).not.toContain("player.errGeneric")
    } finally {
      wrapper.unmount()
    }
  })

  it("does not render markers while duration is still unknown", async () => {
    framePageMocks.listCuratedFramesPage.mockResolvedValue({
      items: [frameRow("f1", 30)],
      total: 1,
      limit: 200,
      offset: 0,
    })
    serviceMocks.getMoviePlayback.mockReturnValueOnce(new Promise(() => {}))
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()

      expect(wrapper.findAll("[data-frame-marker]")).toHaveLength(0)
    } finally {
      wrapper.unmount()
    }
  })

  it("reloads markers when the player advances to another movie", async () => {
    framePageMocks.listCuratedFramesPage
      .mockResolvedValueOnce({
        items: [frameRow("f1", 30)],
        total: 1,
        limit: 200,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [frameRow("f2", 90, "movie-2")],
        total: 1,
        limit: 200,
        offset: 0,
      })
    serviceMocks.getMoviePlayback.mockResolvedValue(directDescriptor())
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()
      expect(wrapper.findAll("[data-frame-marker]")).toHaveLength(1)
      expect(wrapper.get("[data-frame-marker]").attributes("style")).toContain("left: 25%")

      await wrapper.setProps({
        movie: movie({ id: "movie-2", title: "Next title", code: "DEF-456" }),
      })
      await flushPromises()
      await nextTick()

      expect(framePageMocks.listCuratedFramesPage).toHaveBeenLastCalledWith({
        movieId: "movie-2",
        limit: 200,
        offset: 0,
      })
      expect(wrapper.findAll("[data-frame-marker]")).toHaveLength(1)
      expect(wrapper.get("[data-frame-marker]").attributes("style")).toContain("left: 75%")
    } finally {
      wrapper.unmount()
    }
  })
})
