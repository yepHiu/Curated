import { describe, expect, it } from "vitest"
import { mockLibraryService } from "@/services/adapters/mock/mock-library-service"

describe("mockLibraryService.patchPlayerSettings", () => {
  it("does not enable forceStreamPush when only enabling stream push", async () => {
    await mockLibraryService.patchPlayerSettings({
      streamPushEnabled: false,
      forceStreamPush: false,
    })
    expect(mockLibraryService.playerSettings.value.streamPushEnabled).toBe(false)
    expect(mockLibraryService.playerSettings.value.forceStreamPush).toBe(false)

    await mockLibraryService.patchPlayerSettings({ streamPushEnabled: true })
    expect(mockLibraryService.playerSettings.value.streamPushEnabled).toBe(true)
    expect(mockLibraryService.playerSettings.value.forceStreamPush).toBe(false)
  })
})

describe("mockLibraryService", () => {
  it("tracks launch-at-login in local mock state while remaining unsupported", async () => {
    expect(mockLibraryService.launchAtLogin.value).toBe(false)
    expect(mockLibraryService.launchAtLoginSupported.value).toBe(false)

    await mockLibraryService.setLaunchAtLogin(true)

    expect(mockLibraryService.launchAtLogin.value).toBe(true)
    expect(mockLibraryService.launchAtLoginSupported.value).toBe(false)
  })

  it("ensureMovieCached resolves (mock is fully in-memory)", async () => {
    await expect(mockLibraryService.ensureMovieCached("any-id")).resolves.toBeUndefined()
  })

  it("returns undefined for an unknown movie id", () => {
    expect(mockLibraryService.getMovieById("missing-movie")).toBeUndefined()
  })

  it("toggles favorite state in the shared movie source", async () => {
    const libraryService = mockLibraryService
    const movieId = libraryService.movies.value[0]?.id

    expect(movieId).toBeTruthy()

    if (!movieId) {
      return
    }

    const originalFavorite = libraryService.getMovieById(movieId)?.isFavorite ?? false

    await libraryService.toggleFavorite(movieId, !originalFavorite)

    expect(libraryService.getMovieById(movieId)?.isFavorite).toBe(!originalFavorite)

    await libraryService.toggleFavorite(movieId, originalFavorite)

    expect(libraryService.getMovieById(movieId)?.isFavorite).toBe(originalFavorite)
  })
})
