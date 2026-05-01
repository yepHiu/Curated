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

beforeEach(() => {
  vi.resetModules()
  vi.stubEnv("VITE_USE_WEB_API", "false")
  vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(() => {})
  routeState.query = {}
  routeState.hash = ""
  routerMocks.replace.mockReset()
  serviceMocks.getMoviePlayback.mockReset()
  serviceMocks.createPlaybackSession.mockReset()
  serviceMocks.deletePlaybackSession.mockReset()
  activePlaybackMocks.updateActivePlaybackSession.mockReset()
  activePlaybackMocks.clearActivePlaybackSession.mockReset()
  serviceState.playerSettings.value = {
    seekBackwardStepSec: 10,
    seekForwardStepSec: 10,
    nativePlayerPreset: "custom",
    nativePlayerCommand: "",
    nativePlayerEnabled: false,
  }
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllEnvs()
})

describe("PlayerPage loading states", () => {
  it("shows the preparing overlay while the playback descriptor is loading", async () => {
    serviceMocks.getMoviePlayback.mockReturnValueOnce(new Promise(() => {}))
    const wrapper = await mountPlayerPage()

    try {
      expect(serviceMocks.getMoviePlayback).toHaveBeenCalledWith("movie-1")
      expect(wrapper.text()).toContain("common.loading")
      expect(wrapper.text()).toContain("player.preparingPlayback")
    } finally {
      wrapper.unmount()
    }
  })

  it("shows the mock no-stream hint when no playback descriptor is available", async () => {
    serviceMocks.getMoviePlayback.mockResolvedValueOnce(undefined)
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()
      expect(wrapper.text()).toContain("player.noOnlineSrc")
      expect(wrapper.text()).toContain("player.mockNoPlay")
      expect(wrapper.text()).not.toContain("player.preparingPlayback")
    } finally {
      wrapper.unmount()
    }
  })

  it("shows a visible playback error when descriptor loading fails", async () => {
    serviceMocks.getMoviePlayback.mockRejectedValueOnce(new Error("offline"))
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()
      expect(wrapper.text()).toContain("player.errGeneric")
      expect(wrapper.text()).toContain("player.noOnlineSrc")
    } finally {
      wrapper.unmount()
    }
  })

  it("uses the Web API no-stream hint in Web mode", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    serviceMocks.getMoviePlayback.mockResolvedValueOnce(undefined)
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()
      expect(wrapper.text()).toContain("player.noOnlineSrc")
      expect(wrapper.text()).toContain("player.errNoSrc")
    } finally {
      wrapper.unmount()
    }
  })

  it("publishes active playback state and clears it after playback ends", async () => {
    serviceMocks.getMoviePlayback.mockResolvedValueOnce({
      movieId: "movie-1",
      mode: "direct",
      url: "/api/library/movies/movie-1/stream",
      durationSec: 120,
      canDirectPlay: true,
    })
    routeState.query = { back: "browse", browse: "library", autoplay: "1" }
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()

      expect(activePlaybackMocks.updateActivePlaybackSession).toHaveBeenCalledWith(
        expect.objectContaining({
          movieId: "movie-1",
          title: "Movie title",
          positionSec: 0,
          durationSec: 120,
          status: "paused",
          routeQuery: routeState.query,
        }),
      )

      const video = wrapper.get("video").element as HTMLVideoElement
      Object.defineProperty(video, "duration", {
        configurable: true,
        value: 120,
      })
      video.currentTime = 42

      await wrapper.get("video").trigger("loadedmetadata")
      await wrapper.get("video").trigger("timeupdate")

      expect(activePlaybackMocks.updateActivePlaybackSession).toHaveBeenCalledWith(
        expect.objectContaining({
          movieId: "movie-1",
          title: "Movie title",
          positionSec: 42,
          durationSec: 120,
          status: "paused",
          routeQuery: routeState.query,
        }),
      )

      await wrapper.get("video").trigger("ended")

      expect(activePlaybackMocks.clearActivePlaybackSession).toHaveBeenCalledWith("movie-1")
    } finally {
      wrapper.unmount()
    }
  })
})
