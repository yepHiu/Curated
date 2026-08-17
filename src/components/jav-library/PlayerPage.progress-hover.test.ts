import { flushPromises, mount } from "@vue/test-utils"
import type { VueWrapper } from "@vue/test-utils"
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

async function mountPlayerPage() {
  const { default: PlayerPage } = await import("./PlayerPage.vue")
  const wrapper = mount(PlayerPage, {
    props: { movie: movie(), autoplay: false },
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

function progressRoot(wrapper: VueWrapper): HTMLElement {
  return (wrapper.get("[data-slider]").element as HTMLElement).parentElement as HTMLElement
}

async function dispatchMouse(root: HTMLElement, type: string, clientX?: number) {
  root.dispatchEvent(new MouseEvent(type, { bubbles: true, clientX }))
  await nextTick()
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
  framePageMocks.listCuratedFramesPage.mockResolvedValue({
    items: [],
    total: 0,
    limit: 200,
    offset: 0,
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllEnvs()
})

describe("PlayerPage progress hover indicator", () => {
  it("shows paired primary triangles following the pointer and hides on leave", async () => {
    serviceMocks.getMoviePlayback.mockResolvedValueOnce({
      movieId: "movie-1",
      mode: "direct",
      url: "/api/library/movies/movie-1/stream",
      durationSec: 120,
      canDirectPlay: true,
    })
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()

      const root = progressRoot(wrapper)
      vi.spyOn(root, "getBoundingClientRect").mockReturnValue({
        left: 0,
        width: 200,
        right: 200,
        top: 0,
        bottom: 24,
        height: 24,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      } as DOMRect)

      expect(wrapper.find("[data-slot='player-progress-hover-indicator']").exists()).toBe(false)

      await dispatchMouse(root, "mousemove", 50)
      const indicator = wrapper.get("[data-slot='player-progress-hover-indicator']")
      expect(indicator.attributes("style")).toContain("left: 25%")
      expect(indicator.findAll("span")).toHaveLength(2)
      expect(indicator.classes()).toContain("pointer-events-none")

      await dispatchMouse(root, "mousemove", 170)
      expect(
        wrapper.get("[data-slot='player-progress-hover-indicator']").attributes("style"),
      ).toContain("left: 85%")

      await dispatchMouse(root, "mouseleave")
      expect(wrapper.find("[data-slot='player-progress-hover-indicator']").exists()).toBe(false)
    } finally {
      wrapper.unmount()
    }
  })

  it("keeps the indicator hidden while duration is unknown", async () => {
    serviceMocks.getMoviePlayback.mockReturnValueOnce(new Promise(() => {}))
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()

      const root = progressRoot(wrapper)
      vi.spyOn(root, "getBoundingClientRect").mockReturnValue({
        left: 0,
        width: 200,
      } as DOMRect)
      await dispatchMouse(root, "mousemove", 100)

      expect(wrapper.find("[data-slot='player-progress-hover-indicator']").exists()).toBe(false)
    } finally {
      wrapper.unmount()
    }
  })
})
