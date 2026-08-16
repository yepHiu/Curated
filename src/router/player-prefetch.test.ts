import { afterEach, describe, expect, it, vi } from "vitest"

afterEach(() => {
  vi.resetModules()
  vi.clearAllMocks()
})

describe("router player playback prefetch", () => {
  it("prefetches the playback descriptor when navigating to the player", async () => {
    const prefetchMoviePlayback = vi.fn()
    vi.doMock("@/services/auth-lock-service", () => ({
      authLockService: { refreshStatus: vi.fn() },
      isAuthLockEnabled: () => false,
    }))
    vi.doMock("@/services/library-service", () => ({
      useLibraryService: () => ({ prefetchMoviePlayback }),
    }))

    const { default: router } = await import("@/router")
    await router.push("/player/movie-1?t=42&autoplay=1")
    await router.isReady()

    expect(prefetchMoviePlayback).toHaveBeenCalledWith("movie-1")
  })

  it("does not prefetch when the auth guard redirects to the lock screen", async () => {
    const prefetchMoviePlayback = vi.fn()
    vi.doMock("@/services/auth-lock-service", () => ({
      authLockService: {
        refreshStatus: vi.fn().mockResolvedValue({ pinEnabled: true, unlocked: false }),
      },
      isAuthLockEnabled: () => true,
    }))
    vi.doMock("@/services/library-service", () => ({
      useLibraryService: () => ({ prefetchMoviePlayback }),
    }))

    const { default: router } = await import("@/router")
    await router.push("/player/movie-1")
    await router.isReady()

    expect(router.currentRoute.value.name).toBe("lock")
    expect(prefetchMoviePlayback).not.toHaveBeenCalled()
  })

  it("does not prefetch outside the player route", async () => {
    const prefetchMoviePlayback = vi.fn()
    vi.doMock("@/services/auth-lock-service", () => ({
      authLockService: { refreshStatus: vi.fn() },
      isAuthLockEnabled: () => false,
    }))
    vi.doMock("@/services/library-service", () => ({
      useLibraryService: () => ({ prefetchMoviePlayback }),
    }))

    const { default: router } = await import("@/router")
    await router.push("/library")
    await router.isReady()

    expect(prefetchMoviePlayback).not.toHaveBeenCalled()
  })
})
