import { flushPromises, mount } from "@vue/test-utils"
import { defineComponent, h, nextTick, ref } from "vue"
import { createMemoryHistory, createRouter, RouterView, useRoute } from "vue-router"
import { providePlaybackHost, type PlaybackHost } from "@/composables/use-playback-host"
import { removeProgress } from "@/lib/playback-progress-storage"
import { clearActivePlaybackSession } from "@/composables/use-active-playback-session"
import ActivePlaybackHost from "./ActivePlaybackHost.vue"
import SidebarPlaybackEntry from "./SidebarPlaybackEntry.vue"
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
  prefetchMoviePlayback: vi.fn(),
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
    movies: { value: [movie(), movie({ id: "movie-2" })] },
    getMovieById: (id: string) => { /* 提供两部缓存影片给实际 PlayerView。 */ return ["movie-1", "movie-2"].includes(id) ? movie({ id }) : undefined },
    prefetchMoviePlayback: serviceMocks.prefetchMoviePlayback,
    trashedMovies: { value: [] },
    getMoviePlayback: serviceMocks.getMoviePlayback,
    createPlaybackSession: serviceMocks.createPlaybackSession,
    getPlaybackSession: serviceMocks.getPlaybackSession,
    deletePlaybackSession: serviceMocks.deletePlaybackSession,
  }),
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

beforeEach(() => {
  removeProgress("movie-1")
  removeProgress("movie-2")
  clearActivePlaybackSession()
  localStorage.clear()
  vi.stubEnv("VITE_USE_WEB_API", "false")
  vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(() => {})
  vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(() => {})
  routeState.query = {}
  routeState.hash = ""
  routerMocks.replace.mockReset()
  serviceMocks.getMoviePlayback.mockReset()
  serviceMocks.prefetchMoviePlayback.mockReset()
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


/** 创建真实路由、PlayerView、宿主及 PlayerPage，只模拟服务和原生媒体接口。 */
async function mountHost(mode: "direct" | "hls" = "direct") {
  const destroy = vi.fn()
  const { loadHlsLibrary } = await import("@/lib/hls-player")
  class FakeHls {
    /** 测试使用受支持的 HLS，不触发直放回退。 */
    static isSupported() { return true }
    /** 媒体 source 由假 HLS 持有，不需要实际下载分片。 */
    loadSource() {}
    /** 保留同一 video 元素即可验证跨路由的媒体所有权。 */
    attachMedia() {}
    destroy = destroy
  }
  vi.mocked(loadHlsLibrary).mockResolvedValue(FakeHls)
  serviceMocks.getMoviePlayback.mockResolvedValue({ movieId: "movie-1", mode, sessionId: mode === "hls" ? "session-1" : undefined, url: mode === "hls" ? "/video.m3u8" : "/video.mp4", fileName: "video.mp4", durationSec: 7200 })
  const { default: PlayerView } = await import("@/views/PlayerView.vue")
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: "/player/:id", name: "player", component: PlayerView },
    { path: "/library", name: "library", component: { template: '<input data-library-input /><p>Library</p>' } },
  ] })
  await router.push("/player/movie-1?back=browse&tag=original")
  await router.isReady()
  const compact = ref(false)
  let host!: PlaybackHost
  const wrapper = mount(defineComponent({
    /** 复制壳层拓扑：路由视图和持续播放宿主互为兄弟，导航只替换前者。 */
    setup() {
      host = providePlaybackHost(useRoute())
      return () => { /* 独立渲染宿主，视频节点不因路由导航重新创建。 */ return h("main", [h(SidebarPlaybackEntry, { compact: compact.value }), h(RouterView), host.target.value ? h(ActivePlaybackHost, { host }) : null]) }
    },
  }), { global: { plugins: [router], stubs: { Teleport: true, Transition: true } } })
  await flushPromises()
  await nextTick()
  const video = wrapper.get("video").element as HTMLVideoElement
  Object.defineProperty(video, "readyState", { configurable: true, value: 4 })
  Object.defineProperty(video, "videoWidth", { configurable: true, value: 1920 })
  Object.defineProperty(video, "duration", { configurable: true, value: 7200 })
  video.currentTime = 30
  await wrapper.get("video").trigger("loadedmetadata")
  video.currentTime = 30
  await wrapper.get("video").trigger("timeupdate")
  let paused = true
  Object.defineProperty(video, "paused", { configurable: true, get: () => { /* 读取模拟原生播放状态。 */ return paused } })
  Object.defineProperty(video, "buffered", { configurable: true, value: { length: 1, start: () => { /* 已缓存时间起点。 */ return 0 }, end: () => { /* 足够的起播缓冲。 */ return 7200 } } })
  video.play = vi.fn(async () => { /* 原生播放成功后派发媒体事件。 */ paused = false; video.dispatchEvent(new Event("play")) })
  video.pause = vi.fn(() => { /* 暂停同步事件供宿主和观看时长跟踪使用。 */ paused = true; video.dispatchEvent(new Event("pause")) })
  return { wrapper, video, host, router, destroy, compact }
}

/** 模拟系统已进入小窗，保留原生标准所属元素与事件。 */
async function enterPip(video: HTMLVideoElement) {
  Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: video })
  video.dispatchEvent(new Event("enterpictureinpicture"))
  await nextTick()
}
const pipOriginals = ([
  [document, "pictureInPictureEnabled"], [document, "pictureInPictureElement"],
  [document, "exitPictureInPicture"], [HTMLVideoElement.prototype, "requestPictureInPicture"],
] as [object, string][]).map(([object, key]) => {
  // 保存原生接口，防止测试桩污染其它文件。
  return { object, key, descriptor: Object.getOwnPropertyDescriptor(object, key) }
})
beforeEach(() => {
  // 原生 PiP API 是测试唯一窗口层桩，不模拟 Vue 的生命周期。
  Object.defineProperty(document, "pictureInPictureEnabled", { configurable: true, value: true })
  Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: null })
  Object.defineProperty(document, "exitPictureInPicture", { configurable: true, value: vi.fn(async () => {
    const video = document.pictureInPictureElement
    Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: null })
    video?.dispatchEvent(new Event("leavepictureinpicture"))
  }) })
  Object.defineProperty(HTMLVideoElement.prototype, "requestPictureInPicture", { configurable: true, value: vi.fn().mockResolvedValue({}) })
})
afterEach(() => {
  // 恢复 PiP 属性并卸载各用例创建的实例。
  for (const original of pipOriginals) {
    if (original.descriptor) Object.defineProperty(original.object, original.key, original.descriptor)
    else Reflect.deleteProperty(original.object, original.key)
  }
})

describe("application-owned playback", () => {
  // direct 和 HLS 都必须保留同一 video、会话和当前位置，返回时不预热第二次。
  it.each(["direct", "hls"] as const)("keeps %s playback across browsing and reuses it on return", async (mode) => {
    const test = await mountHost(mode)
    await enterPip(test.video)
    test.video.currentTime = 42
    await test.wrapper.get("video").trigger("timeupdate")
    await test.router.push("/library?tag=changed")
    await flushPromises()
    expect(test.wrapper.get("video").element).toBe(test.video)
    expect(test.video.currentTime).toBe(42)
    expect(test.wrapper.get("[data-active-player-host]").attributes("inert")).toBeDefined()
    expect(test.wrapper.find("[data-active-playback-controls]").exists()).toBe(true)
    expect(test.wrapper.find("[data-background-playback]").exists()).toBe(false)
    expect(test.host.playerRoute.query.tag).toBe("original")
    expect(test.host.pipActive.value).toBe(true)
    expect(test.destroy).not.toHaveBeenCalled()
    expect(serviceMocks.deletePlaybackSession).not.toHaveBeenCalled()
    await test.wrapper.get("[data-active-playback-card]").trigger("click")
    await flushPromises()
    expect(test.wrapper.get("video").element).toBe(test.video)
    expect(document.pictureInPictureElement).toBeNull()
    expect(serviceMocks.getMoviePlayback).toHaveBeenCalledTimes(1)
    expect(serviceMocks.prefetchMoviePlayback).toHaveBeenCalledTimes(1)
    expect(test.wrapper.find("[data-active-playback-controls]").exists()).toBe(false)
    expect(test.wrapper.get("[data-active-player-host]").attributes("inert")).toBeUndefined()
    test.wrapper.unmount()
    await flushPromises()
    if (mode === "hls") expect(serviceMocks.deletePlaybackSession).toHaveBeenCalledWith("session-1")
  })

  // 返回图标先完成路由导航再关闭小窗，保持同一媒体、进度及真实暂停/播放状态。
  it.each([
    ["direct", false], ["direct", true], ["hls", false], ["hls", true],
  ] as const)("returns %s PiP to normal playback while playing=%s", async (mode, playing) => {
    const test = await mountHost(mode)
    if (playing) await test.video.play()
    await enterPip(test.video)
    await test.router.push("/library")
    await flushPromises()
    const controls = test.wrapper.get("[data-active-playback-controls]")
    expect(controls.element.lastElementChild?.hasAttribute("data-active-playback-return")).toBe(true)
    expect(controls.findAll("button")).toHaveLength(3)
    await test.wrapper.get("[data-active-playback-return]").trigger("click")
    await flushPromises()
    expect(test.router.currentRoute.value.name).toBe("player")
    expect(test.wrapper.get("video").element).toBe(test.video)
    expect(document.pictureInPictureElement).toBeNull()
    expect(test.host.pipActive.value).toBe(false)
    expect(test.video.currentTime).toBe(30)
    expect(test.video.paused).toBe(!playing)
    expect(test.destroy).not.toHaveBeenCalled()
    expect(serviceMocks.getMoviePlayback).toHaveBeenCalledTimes(1)
    expect(test.router.currentRoute.value.query.t).toBeUndefined()
    test.wrapper.unmount()
  })

  // 路由守卫拒绝返回时保持后台小窗，不能先退出从而触发停止。
  it("keeps PiP alive when returning to the player is cancelled", async () => {
    const test = await mountHost("hls")
    await enterPip(test.video)
    await test.router.push("/library")
    await flushPromises()
    const removeGuard = test.router.beforeEach(() => false)
    await test.wrapper.get("[data-active-playback-return]").trigger("click")
    await flushPromises()
    expect(test.router.currentRoute.value.name).toBe("library")
    expect(document.pictureInPictureElement).toBe(test.video)
    expect(document.exitPictureInPicture).not.toHaveBeenCalled()
    expect(test.wrapper.get("video").element).toBe(test.video)
    removeGuard()
    test.wrapper.unmount()
  })

  // 未开启小窗不能留下无可见控制的后台声音。
  it("releases normal playback on navigation without PiP", async () => {
    const test = await mountHost("hls")
    await test.router.push("/library")
    await flushPromises()
    expect(test.wrapper.find("video").exists()).toBe(false)
    expect(test.destroy).toHaveBeenCalledTimes(1)
    expect(serviceMocks.deletePlaybackSession).toHaveBeenCalledWith("session-1")
    test.wrapper.unmount()
  })

  // 小窗在后台被系统关闭时停止，已保存的进度仍供下次恢复。
  it("stops when the system closes PiP on another page", async () => {
    const test = await mountHost("hls")
    await enterPip(test.video)
    await test.router.push("/library")
    Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: null })
    test.video.dispatchEvent(new Event("leavepictureinpicture"))
    await flushPromises()
    expect(test.host.target.value).toBeNull()
    expect(test.wrapper.find("video").exists()).toBe(false)
    expect(serviceMocks.deletePlaybackSession).toHaveBeenCalledWith("session-1")
    test.wrapper.unmount()
  })

  // 全局快捷键在其它页面必须保留默认行为，停止按钮必须清理续播卡片。
  it("does not consume background hotkeys and explicitly stops playback", async () => {
    const test = await mountHost("hls")
    await enterPip(test.video)
    await test.router.push("/library")
    await flushPromises()
    const key = new KeyboardEvent("keydown", { code: "Space", bubbles: true, cancelable: true })
    window.dispatchEvent(key)
    expect(key.defaultPrevented).toBe(false)
    const { activePlaybackSession } = await import("@/composables/use-active-playback-session")
    expect(activePlaybackSession.value?.movieId).toBe("movie-1")
    await test.wrapper.get('button[aria-label="player.stopBackgroundPlayback"]').trigger("click")
    await flushPromises()
    expect(test.wrapper.find("video").exists()).toBe(false)
    expect(activePlaybackSession.value).toBeNull()
    expect(serviceMocks.deletePlaybackSession).toHaveBeenCalledWith("session-1")
    test.wrapper.unmount()
  })

  // 新影片接管旧小窗，旧实例迟到的关闭事件不能结束后继实例。
  it("replaces the active movie with exactly one player", async () => {
    const test = await mountHost("hls")
    await enterPip(test.video)
    await test.router.push("/player/movie-2")
    await flushPromises()
    expect(test.wrapper.findAll("video")).toHaveLength(1)
    expect(test.wrapper.get("video").element).not.toBe(test.video)
    expect(test.host.target.value?.movie.id).toBe("movie-2")
    expect(serviceMocks.getMoviePlayback).toHaveBeenCalledTimes(2)
    test.host.setPipActive("movie-1", false)
    expect(test.host.target.value?.movie.id).toBe("movie-2")
    expect(serviceMocks.deletePlaybackSession).toHaveBeenCalledWith("session-1")
    test.wrapper.unmount()
  })
  // 后台控制采用同一个 video，暂停及恢复不创建新会话。
  it("pauses and resumes from the unified sidebar entry", async () => {
    const test = await mountHost("hls")
    await enterPip(test.video)
    await test.router.push("/library")
    await flushPromises()
    await test.wrapper.get('button[aria-label="player.ariaPlay"]').trigger("click")
    await flushPromises()
    expect(test.video.play).toHaveBeenCalledTimes(1)
    expect(test.host.playing.value).toBe(true)
    await test.wrapper.get('button[aria-label="player.ariaPause"]').trigger("click")
    expect(test.video.pause).toHaveBeenCalled()
    expect(test.host.playing.value).toBe(false)
    expect(serviceMocks.createPlaybackSession).not.toHaveBeenCalled()
    test.wrapper.unmount()
  })

  // 收起侧栏仍能控制同一实例；起播与近结尾不能沿用续播隐藏规则丢失控制。
  it("keeps compact sidebar controls before resume eligibility and near the end", async () => {
    const test = await mountHost("hls")
    test.video.currentTime = 1
    await test.wrapper.get("video").trigger("timeupdate")
    await enterPip(test.video)
    await test.router.push("/library")
    test.compact.value = true
    await flushPromises()
    expect(test.wrapper.find("[data-active-playback-compact]").exists()).toBe(true)
    expect(test.wrapper.find("[data-active-playback-dismiss]").exists()).toBe(false)
    await test.wrapper.get('button[aria-label="player.ariaPlay"]').trigger("click")
    await flushPromises()
    expect(test.video.play).toHaveBeenCalledTimes(1)
    expect(test.router.currentRoute.value.name).toBe("library")
    test.video.currentTime = 7100
    await test.wrapper.get("video").trigger("timeupdate")
    expect(test.wrapper.find("[data-active-playback-controls]").exists()).toBe(true)
    expect(test.wrapper.get("[data-active-playback-compact]").attributes("title")).toContain("1:58:20")
    await test.wrapper.get('button[aria-label="player.stopBackgroundPlayback"]').trigger("click")
    await flushPromises()
    expect(test.wrapper.find("video").exists()).toBe(false)
    expect(test.wrapper.find("[data-active-playback-controls]").exists()).toBe(false)
    test.wrapper.unmount()
  })

  // 认证失效先停止媒体，随后路由是否及时跳锁定页不影响安全边界。
  it("stops immediately when the authenticated session locks", async () => {
    const test = await mountHost("hls")
    await enterPip(test.video)
    await test.router.push("/library")
    vi.stubEnv("VITE_USE_WEB_API", "true")
    const { api } = await import("@/api/endpoints")
    const { authLockService } = await import("@/services/auth-lock-service")
    vi.spyOn(api, "lockApp").mockResolvedValue({ ...authLockService.status.value, pinEnabled: true, unlocked: false })
    await authLockService.lock()
    expect(test.video.pause).toHaveBeenCalled()
    expect(test.host.target.value).toBeNull()
    await flushPromises()
    expect(test.wrapper.find("video").exists()).toBe(false)
    expect(serviceMocks.deletePlaybackSession).toHaveBeenCalledWith("session-1")
    test.wrapper.unmount()
    vi.spyOn(api, "authStatus").mockResolvedValue({ ...authLockService.status.value, pinEnabled: false, unlocked: true })
    await authLockService.refreshStatus()
  })

  // 同片的显式帧时间链接需要寻址；普通返回入口不重新应用快照进度。
  it("seeks an existing movie for an explicit frame link", async () => {
    const test = await mountHost("direct")
    await enterPip(test.video)
    await test.router.push("/library")
    await test.router.push("/player/movie-1?t=120&autoplay=1&back=curated-frames")
    await flushPromises()
    expect(test.wrapper.get("video").element).toBe(test.video)
    expect(test.video.currentTime).toBe(120)
    expect(serviceMocks.getMoviePlayback).toHaveBeenCalledTimes(1)
    expect(test.router.currentRoute.value.query.t).toBeUndefined()
    test.wrapper.unmount()
  })

  // 后台播放列表结束不能把用户从资料库带走；保留停止或返回入口。
  it("keeps browsing when a background movie ends", async () => {
    const test = await mountHost("direct")
    await enterPip(test.video)
    await test.router.push("/library")
    test.video.currentTime = 7200
    await test.wrapper.get("video").trigger("ended")
    await flushPromises()
    expect(test.router.currentRoute.value.name).toBe("library")
    expect(test.host.target.value?.movie.id).toBe("movie-1")
    test.wrapper.unmount()
  })

})
