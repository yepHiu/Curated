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

  it("returns active movies for CSV export", async () => {
    const activeIds = mockLibraryService.movies.value.map((movie) => movie.id)

    const movies = await mockLibraryService.listMoviesForExport()

    expect(movies.map((movie) => movie.id)).toEqual(activeIds)
    expect(movies.every((movie) => !movie.trashedAt?.trim())).toBe(true)
  })

  it("persists ordered Saved Views and enforces normalized names", async () => {
    for (const item of [...mockLibraryService.savedViews.value]) {
      await mockLibraryService.deleteSavedView(item.id)
    }
    const first = await mockLibraryService.createSavedView("Unwatched", {
      schemaVersion: 1,
      mode: "library",
      playState: "unwatched",
      tab: "all",
    })
    const second = await mockLibraryService.createSavedView("4K", {
      schemaVersion: 1,
      mode: "library",
      resolution: "2160p",
      tab: "all",
      playState: "all",
    })
    expect(mockLibraryService.savedViews.value.map((item) => item.id)).toEqual([
      first.id,
      second.id,
    ])
    expect(mockLibraryService.savedViews.value[1]?.filters.resolution).toBe("4k")

    await expectHttpClientError(
      mockLibraryService.createSavedView("  uNwAtChEd ", {
        schemaVersion: 1,
      }),
      {
        status: 409,
        code: "SAVED_VIEW_NAME_CONFLICT",
        message: "saved view name already exists",
      },
    )

    await mockLibraryService.reorderSavedViews([second.id, first.id])
    await mockLibraryService.updateSavedView(first.id, { name: "Not watched" })
    await mockLibraryService.refreshSavedViews()
    expect(mockLibraryService.savedViews.value.map((item) => item.name)).toEqual([
      "4K",
      "Not watched",
    ])

    await mockLibraryService.deleteSavedView(second.id)
    await mockLibraryService.deleteSavedView(first.id)
    expect(mockLibraryService.savedViews.value).toEqual([])
  })

  it("persists explicit recommendation feedback and applies it to mock generation", async () => {
    const existing = await mockLibraryService.listHomepageRecommendationFeedback()
    for (const item of existing.items) {
      await mockLibraryService.deleteHomepageRecommendationFeedback(item.id)
    }

    const movie = mockLibraryService.movies.value.find((item) => item.actors.length > 0)
    expect(movie).toBeTruthy()
    const actor = movie!.actors[0]!
    const less = await mockLibraryService.createHomepageRecommendationFeedback({
      action: "less",
      targetType: "actor",
      targetValue: actor,
      sourceMovieId: movie!.id,
    })
    const duplicate = await mockLibraryService.createHomepageRecommendationFeedback({
      action: "less",
      targetType: "actor",
      targetValue: actor.toLocaleLowerCase(),
      sourceMovieId: movie!.id,
    })
    expect(duplicate.id).toBe(less.id)

    const blocked = await mockLibraryService.createHomepageRecommendationFeedback({
      action: "not_interested",
      targetType: "movie",
      targetValue: movie!.id,
      sourceMovieId: movie!.id,
    })
    const snapshot = await mockLibraryService.getHomepageDailyRecommendations()
    expect(snapshot.heroMovieIds).not.toContain(movie!.id)
    expect(snapshot.recommendationMovieIds).not.toContain(movie!.id)
    expect(snapshot.recommendations.every((item) => item.reasons.length > 0)).toBe(true)

    await mockLibraryService.deleteHomepageRecommendationFeedback(blocked.id)
    await mockLibraryService.deleteHomepageRecommendationFeedback(less.id)
    expect((await mockLibraryService.listHomepageRecommendationFeedback()).items).toEqual([])
  })

  it("returns visible connected client examples in mock mode", async () => {
    const dto = await mockLibraryService.listConnectedClients()

    expect(dto.total).toBeGreaterThanOrEqual(3)
    expect(dto.localCount).toBeGreaterThanOrEqual(1)
    expect(dto.remoteCount).toBeGreaterThanOrEqual(1)
    expect(dto.clients.some((client) => client.isLocalMachine)).toBe(true)
    expect(dto.clients.some((client) => client.accessKind === "remote")).toBe(true)
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

  it("defaults curated-frame export mode to raw and allows switching to watermarked", async () => {
    expect(mockLibraryService.curatedFrameExportMode.value).toBe("raw")

    await mockLibraryService.setCuratedFrameExportMode("watermarked")
    expect(mockLibraryService.curatedFrameExportMode.value).toBe("watermarked")

    await mockLibraryService.setCuratedFrameExportMode("raw")
    expect(mockLibraryService.curatedFrameExportMode.value).toBe("raw")
  })

  it("tracks default import library path and returns a mock import task", async () => {
    expect(mockLibraryService.defaultImportLibraryPathId.value).toBe("library-a")

    await mockLibraryService.setDefaultImportLibraryPathId("library-b")
    expect(mockLibraryService.defaultImportLibraryPathId.value).toBe("library-b")

    const task = await mockLibraryService.importMovies([
      new File(["movie"], "IMP-001.mp4", { type: "video/mp4" }),
    ])
    expect(task?.type).toBe("import.movies")
    expect(task?.status).toBe("completed")
    expect(task?.metadata).toMatchObject({
      targetLibraryPathId: "library-b",
      totalFiles: 1,
      completedFiles: 1,
      failedFiles: 0,
    })

    const first = mockLibraryService.movies.value[0]
    const check = await mockLibraryService.checkImportMovieCodes([
      `${first.code}-C.mp4`,
      "holiday.mp4",
    ])
    expect(check.matchedCount).toBe(1)
    expect(check.items[0]?.matches[0]?.movieId).toBe(first.id)
    expect(check.items[1]?.matches).toEqual([])

    await mockLibraryService.setDefaultImportLibraryPathId("library-a")
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
