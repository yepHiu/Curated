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
  getPlaybackSession: vi.fn(),
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
    movies: { value: [] },
    trashedMovies: { value: [] },
    getMoviePlayback: serviceMocks.getMoviePlayback,
    createPlaybackSession: serviceMocks.createPlaybackSession,
    getPlaybackSession: serviceMocks.getPlaybackSession,
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
  vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(() => {})
  routeState.query = {}
  routeState.hash = ""
  routerMocks.replace.mockReset()
  serviceMocks.getMoviePlayback.mockReset()
  serviceMocks.createPlaybackSession.mockReset()
  serviceMocks.getPlaybackSession.mockReset()
  serviceMocks.getPlaybackSession.mockResolvedValue(null)
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

  it("restarts an HLS session at the beginning when the descriptor is already near the end", async () => {
    serviceMocks.getMoviePlayback.mockResolvedValueOnce({
      movieId: "movie-1",
      mode: "hls",
      url: "/api/playback/sessions/old/hls/index.m3u8",
      sessionId: "old-session",
      startPositionSec: 118,
      resumePositionSec: 119,
      durationSec: 120,
      canDirectPlay: false,
    })
    serviceMocks.createPlaybackSession.mockResolvedValueOnce({
      movieId: "movie-1",
      mode: "hls",
      url: "/api/playback/sessions/new/hls/index.m3u8",
      sessionId: "new-session",
      startPositionSec: 0,
      resumePositionSec: 0,
      durationSec: 120,
      canDirectPlay: false,
    })
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()
      expect(serviceMocks.createPlaybackSession).toHaveBeenCalledWith("movie-1", "hls", 0)
      expect(serviceMocks.deletePlaybackSession).toHaveBeenCalledWith("old-session")
    } finally {
      wrapper.unmount()
    }
  })

  it("loads video media with credentials for authenticated backend playback streams", async () => {
    serviceMocks.getMoviePlayback.mockResolvedValueOnce({
      movieId: "movie-1",
      mode: "direct",
      url: "http://127.0.0.1:8080/api/library/movies/movie-1/stream",
      durationSec: 120,
      canDirectPlay: true,
    })
    const wrapper = await mountPlayerPage()

    try {
      await flushPromises()
      await nextTick()

      expect(wrapper.get("video").attributes("crossorigin")).toBe("use-credentials")
    } finally {
      wrapper.unmount()
    }
  })

  it("steps to the previous and next frame with D and F outside form controls", async () => {
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

      const video = wrapper.get("video").element as HTMLVideoElement
      video.currentTime = 10
      await wrapper.get("video").trigger("timeupdate")
      Object.defineProperty(video, "paused", {
        configurable: true,
        value: false,
      })
      const pauseSpy = vi.mocked(HTMLMediaElement.prototype.pause)
      pauseSpy.mockClear()

      const previousFrame = new KeyboardEvent("keydown", {
        bubbles: true,
        cancelable: true,
        code: "KeyD",
        key: "d",
      })
      window.dispatchEvent(previousFrame)
      expect(previousFrame.defaultPrevented).toBe(true)
      expect(pauseSpy).toHaveBeenCalledTimes(1)
      expect(video.currentTime).toBeCloseTo(10 - 1 / 30)

      const nextFrame = new KeyboardEvent("keydown", {
        bubbles: true,
        cancelable: true,
        code: "KeyF",
        key: "f",
      })
      window.dispatchEvent(nextFrame)
      expect(nextFrame.defaultPrevented).toBe(true)
      expect(video.currentTime).toBeCloseTo(10)

      const input = document.createElement("input")
      document.body.appendChild(input)
      input.dispatchEvent(new KeyboardEvent("keydown", {
        bubbles: true,
        cancelable: true,
        code: "KeyD",
        key: "d",
      }))
      expect(video.currentTime).toBeCloseTo(10)
      input.remove()
    } finally {
      wrapper.unmount()
    }
  })

  it("tears down the video media pipeline when the player unmounts", async () => {
    serviceMocks.getMoviePlayback.mockResolvedValueOnce({
      movieId: "movie-1",
      mode: "direct",
      url: "/api/library/movies/movie-1/stream",
      durationSec: 120,
      canDirectPlay: true,
    })
    const pauseSpy = vi.mocked(HTMLMediaElement.prototype.pause)
    const removeAttributeSpy = vi.spyOn(HTMLMediaElement.prototype, "removeAttribute")
    const loadSpy = vi.mocked(HTMLMediaElement.prototype.load)
    const wrapper = await mountPlayerPage()

    await flushPromises()
    await nextTick()
    expect(wrapper.find("video").exists()).toBe(true)

    pauseSpy.mockClear()
    removeAttributeSpy.mockClear()
    loadSpy.mockClear()

    wrapper.unmount()

    expect(pauseSpy).toHaveBeenCalledTimes(1)
    expect(removeAttributeSpy).toHaveBeenCalledWith("src")
    expect(loadSpy).toHaveBeenCalledTimes(1)
  })
  it.each([[24, 0.5], [30, 2], [60, 1]])("steps a %ifps source at %sx and retains its cadence after pausing", async (fps, rate) => {
    serviceMocks.getMoviePlayback.mockResolvedValueOnce({ movieId: "movie-1", mode: "direct", url: "/api/library/movies/movie-1/stream", durationSec: 120, canDirectPlay: true })
    const wrapper = await mountPlayerPage()
    try {
      await flushPromises()
      const video = wrapper.get("video").element as HTMLVideoElement
      let callback!: VideoFrameRequestCallback
      video.requestVideoFrameCallback = vi.fn(cb => { callback = cb; return 1 })
      video.cancelVideoFrameCallback = vi.fn()
      Object.defineProperty(video, "paused", { configurable: true, value: false })
      await wrapper.get("video").trigger("loadedmetadata")
      video.playbackRate = rate
      const frame = (count: number, mediaTime: number, wallTime: number) => callback(wallTime, {
        mediaTime, presentedFrames: count, presentationTime: wallTime, expectedDisplayTime: wallTime,
        width: 1920, height: 1080, processingDuration: 0,
      })
      for (let index = 0; index <= 60; index++) frame(index + 1, index / fps, 1000 + index * 1000 / (fps * rate))
      video.currentTime = 10
      await wrapper.get("video").trigger("timeupdate")
      window.dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, cancelable: true, code: "KeyF", key: "f" }))
      expect(video.currentTime).toBeCloseTo(10 + 1 / fps, 5)
      Object.defineProperty(video, "paused", { configurable: true, value: true })
      await wrapper.get("video").trigger("pause")
      // Seek frames delivered after long pauses must not replace source cadence.
      for (let index = 0; index < 4; index++) {
        frame(62 + index, video.currentTime, 20000 + index * 1000)
        window.dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, cancelable: true, code: "KeyF", key: "f" }))
      }
      expect(video.currentTime).toBeCloseTo(10 + 5 / fps, 5)
      window.dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, cancelable: true, code: "KeyD", key: "d" }))
      expect(video.currentTime).toBeCloseTo(10 + 4 / fps, 5)
    } finally { wrapper.unmount() }
  })
})
