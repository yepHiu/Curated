import { beforeEach, describe, expect, it, vi } from "vitest"
import { createRouter, createMemoryHistory } from "vue-router"
vi.mock("@/services/library-service", () => ({ useLibraryService: () => ({ movies: { value: [] }, trashedMovies: { value: [] } }) }))
vi.mock("@/lib/playback-progress-storage", () => ({
  parseResumeSecondsFromQuery: (value: unknown) => typeof value === "string" ? Number(value) : undefined,
  getProgress: vi.fn(), hydratePlaybackProgress: vi.fn().mockResolvedValue(undefined),
}))
vi.mock("@/lib/played-movies-storage", () => ({ hasPlayedMovie: vi.fn(), hydratePlayedMovies: vi.fn().mockResolvedValue(undefined) }))
vi.mock("@/lib/player-playlist", () => ({ listPlayerPlaylistMovies: () => [{ id: "a" }, { id: "b" }],
  resolvePlayerPlaylistSource: () => "browse", readPlaylistAutoAdvance: () => true }))
vi.mock("@/lib/locale-storage", () => ({ resolveInitialLocale: () => "zh-CN" }))
vi.mock("@/composables/use-active-playback-session", () => ({ updateActivePlaybackSession: vi.fn() }))
const router = createRouter({ history: createMemoryHistory(), routes: [
  { path: "/library", name: "library", component: {} }, { path: "/player/:id", name: "player", component: {} },
] })
beforeEach(() => { vi.resetModules(); vi.stubEnv("VITE_USE_WEB_API", "true"); delete window.javLibrary })
describe("Desktop playback routing", () => {
  it("intercepts before PlayerView, passes frozen identities and leaves the source page available", async () => {
    const open = vi.fn().mockResolvedValue({ sessionId: "s", revision: 1, windowOpen: true })
    const capabilities = vi.fn().mockResolvedValue({ protocol: 1, available: true, preferences: { preferNative: true } })
    window.javLibrary = { playback: { open, capabilities } as unknown as NonNullable<Window["javLibrary"]>["playback"] }
    const { desktopPlaybackService: service } = await import("./desktop-playback-service")
    const stop = vi.fn(); service.bindWebStop(stop)
    const to = router.resolve("/player/a?fileId=p2&t=42&autoplay=1")
    expect(await service.intercept(to, router.resolve("/library?q=test"))).toBe("/library?q=test")
    expect(stop).toHaveBeenCalledOnce()
    expect(open).toHaveBeenCalledWith(expect.objectContaining({ movieId: "a", fileId: "p2", startSec: 42, autoplay: true, queue: ["a", "b"] }))
  })
  it("uses Web when the bridge is missing, disabled or explicitly bypassed", async () => {
    const { desktopPlaybackService: service } = await import("./desktop-playback-service")
    const to = router.resolve("/player/a")
    expect(await service.intercept(to, router.resolve("/library"))).toBe(true)
    const open = vi.fn()
    window.javLibrary = { playback: { open, capabilities: async () => ({ protocol: 1, available: true, preferences: { preferNative: false } }) } as unknown as NonNullable<Window["javLibrary"]>["playback"] }
    expect(await service.intercept(to, router.resolve("/library"))).toBe(true)
    expect(open).not.toHaveBeenCalled()
  })
  it("falls back once on ordinary engine failure and sends auth failure to unlock", async () => {
    const open = vi.fn().mockRejectedValue(new Error("MPV_START_FAILED"))
    window.javLibrary = { playback: { open, capabilities: async () => ({ protocol: 1, available: true, preferences: { preferNative: true } }) } as unknown as NonNullable<Window["javLibrary"]>["playback"] }
    const { desktopPlaybackService: service } = await import("./desktop-playback-service")
    const to = router.resolve("/player/a?autoplay=1")
    expect(await service.intercept(to, router.resolve("/library"))).toMatchObject({ name: "player", query: { engine: "web" } })
    open.mockRejectedValue(new Error("SERVER_LOCKED"))
    expect(await service.intercept(to, router.resolve("/library"))).toMatchObject({ name: "lock" })
  })
})
