import { describe, expect, it } from "vitest"
import { HttpClientError } from "@/api/http-client"
import { mockLibraryService } from "@/services/adapters/mock/mock-library-service"

async function expectHttpClientError(
  promise: Promise<unknown>,
  expected: { status: number; code: string; message: string },
) {
  try {
    await promise
    throw new Error("expected promise to reject")
  } catch (err) {
    expect(err).toBeInstanceOf(HttpClientError)
    expect((err as HttpClientError).status).toBe(expected.status)
    expect((err as HttpClientError).apiError).toMatchObject({
      code: expected.code,
      message: expected.message,
      retryable: false,
    })
  }
}

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
  it("reports movies as already loaded in mock mode", () => {
    expect(mockLibraryService.moviesLoaded.value).toBe(true)
  })

  it("tracks launch-at-login in local mock state while remaining unsupported", async () => {
    expect(mockLibraryService.launchAtLogin.value).toBe(false)
    expect(mockLibraryService.launchAtLoginSupported.value).toBe(false)

    await mockLibraryService.setLaunchAtLogin(true)

    expect(mockLibraryService.launchAtLogin.value).toBe(true)
    expect(mockLibraryService.launchAtLoginSupported.value).toBe(false)
  })

  it("defaults curated-frame export format to jpg and allows switching formats", async () => {
    expect(mockLibraryService.curatedFrameExportFormat.value).toBe("jpg")

    await mockLibraryService.setCuratedFrameExportFormat("png")
    expect(mockLibraryService.curatedFrameExportFormat.value).toBe("png")

    await mockLibraryService.setCuratedFrameExportFormat("jpg")
    expect(mockLibraryService.curatedFrameExportFormat.value).toBe("jpg")
  })

  it("ensureMovieCached resolves (mock is fully in-memory)", async () => {
    await expect(mockLibraryService.ensureMovieCached("any-id")).resolves.toBeUndefined()
  })

  it("rejects opening a library path in file manager in mock mode", async () => {
    await expectHttpClientError(mockLibraryService.revealLibraryPathInFileManager("library-a"), {
      status: 501,
      code: "MOCK_REVEAL_NOT_SUPPORTED",
      message: "MOCK_REVEAL_NOT_SUPPORTED",
    })
  })

  it("uses HTTP-shaped errors for unsupported mock-only operations", async () => {
    await expectHttpClientError(mockLibraryService.revealMovieInFileManager("mkb-100"), {
      status: 501,
      code: "MOCK_REVEAL_NOT_SUPPORTED",
      message: "MOCK_REVEAL_NOT_SUPPORTED",
    })
    await expectHttpClientError(
      mockLibraryService.exportCuratedFrames({
        ids: ["frame-1"],
        format: "png",
      }),
      {
        status: 501,
        code: "MOCK_CURATED_EXPORT_NOT_SUPPORTED",
        message: "MOCK_CURATED_EXPORT_NOT_SUPPORTED",
      },
    )
  })

  it("uses HTTP-shaped errors for mock validation and not-found failures", async () => {
    await expectHttpClientError(mockLibraryService.addLibraryPath("relative/path"), {
      status: 400,
      code: "COMMON_BAD_REQUEST",
      message: "library path must be an absolute path",
    })
    await expectHttpClientError(mockLibraryService.getActorProfile("Missing Actor"), {
      status: 404,
      code: "COMMON_NOT_FOUND",
      message: "actor not found",
    })
  })

  it("returns undefined for an unknown movie id", () => {
    expect(mockLibraryService.getMovieById("missing-movie")).toBeUndefined()
  })

  it("finds trashed movies by trimmed id like the web adapter", async () => {
    const movieId = mockLibraryService.movies.value[0]?.id
    expect(movieId).toBeTruthy()

    if (!movieId) {
      return
    }

    await mockLibraryService.deleteMovie(movieId)

    try {
      expect(mockLibraryService.movies.value.some((movie) => movie.id === movieId)).toBe(false)
      expect(mockLibraryService.trashedMovies.value.some((movie) => movie.id === movieId)).toBe(
        true,
      )
      expect(mockLibraryService.getMovieById(` ${movieId} `)?.id).toBe(movieId)
      await expect(mockLibraryService.loadMovieDetail(` ${movieId} `)).resolves.toMatchObject({
        id: movieId,
      })
    } finally {
      await mockLibraryService.restoreMovie(movieId)
    }
  })

  it("resolves loadMovieDetail asynchronously", async () => {
    const movieId = mockLibraryService.movies.value[0]?.id
    expect(movieId).toBeTruthy()

    if (!movieId) {
      return
    }

    let settled = false
    const detailPromise = mockLibraryService.loadMovieDetail(movieId).then((movie) => {
      settled = true
      return movie
    })

    expect(settled).toBe(false)
    await expect(detailPromise).resolves.toMatchObject({ id: movieId })
    expect(settled).toBe(true)
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
