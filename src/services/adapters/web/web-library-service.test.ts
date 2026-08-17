import { flushPromises } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { HttpClientError } from "@/api/http-client"
import type { MovieDetailDTO, MovieListItemDTO, SettingsDTO } from "@/api/types"

const apiMocks = vi.hoisted(() => ({
  listMovies: vi.fn(),
  listConnectedClients: vi.fn(),
  getMovie: vi.fn(),
  getMoviePlayback: vi.fn(),
  patchMovie: vi.fn(),
  deleteMovie: vi.fn(),
  restoreMovie: vi.fn(),
  getSettings: vi.fn(),
  patchSettings: vi.fn(),
  importMovies: vi.fn(),
  getMovieImportUpload: vi.fn(),
  deleteMovieImportUpload: vi.fn(),
  listSavedViews: vi.fn(),
  createSavedView: vi.fn(),
  patchSavedView: vi.fn(),
  deleteSavedView: vi.fn(),
  reorderSavedViews: vi.fn(),
  listHomepageRecommendationFeedback: vi.fn(),
  createHomepageRecommendationFeedback: vi.fn(),
  deleteHomepageRecommendationFeedback: vi.fn(),
  previewActorMerge: vi.fn(),
  applyActorMerge: vi.fn(),
  listActorMergeAudits: vi.fn(),
  getPersonalInsightsOverview: vi.fn(),
  getPersonalInsightsBreakdown: vi.fn(),
}))

function movieListDto(id: string, overrides: Partial<MovieListItemDTO> = {}): MovieListItemDTO {
  return {
    id,
    title: `Title ${id}`,
    code: id.toUpperCase(),
    studio: "Studio",
    actors: ["Actor"],
    tags: ["tag"],
    runtimeMinutes: 120,
    rating: 4,
    isFavorite: false,
    addedAt: "2026-01-01T00:00:00.000Z",
    location: `D:/media/${id}.mp4`,
    resolution: "1080p",
    year: 2026,
    ...overrides,
  }
}

function movieDetailDto(id: string, overrides: Partial<MovieDetailDTO> = {}): MovieDetailDTO {
  return {
    ...movieListDto(id),
    summary: `Summary ${id}`,
    previewImages: [],
    metadataRating: 4,
    ...overrides,
  }
}

function settingsDto(overrides: Partial<SettingsDTO> = {}): SettingsDTO {
  return {
    libraryPaths: [],
    backupDirectory: "",
    player: {
      hardwareDecode: true,
      hardwareEncoder: "auto",
      nativePlayerPreset: "custom",
      nativePlayerEnabled: false,
      nativePlayerCommand: "",
      streamPushEnabled: true,
      forceStreamPush: false,
      ffmpegCommand: "ffmpeg",
      preferNativePlayer: false,
      seekForwardStepSec: 10,
      seekBackwardStepSec: 10,
    },
    organizeLibrary: true,
    autoLibraryWatch: true,
    autoActorProfileScrape: false,
    autoDownloadUpdates: false,
    launchAtLogin: false,
    launchAtLoginSupported: false,
    curatedFrameExportFormat: "jpg",
    curatedFrameExportMode: "raw",
    metadataMovieProvider: "",
    metadataMovieProviders: [],
    metadataMovieProviderChain: [],
    metadataMovieScrapeMode: "auto",
    proxy: { enabled: false },
    backendLog: {
      logDir: "",
      logLevel: "info",
    },
    ...overrides,
  }
}

vi.mock("@/api/endpoints", () => ({
  api: apiMocks,
}))

vi.mock("@/i18n", () => ({
  i18n: {
    global: {
      locale: ref("zh-CN"),
      t: (key: string) => key,
    },
  },
}))

async function loadStartedWebLibraryService() {
  const serviceModule = await import("./web-library-service")
  void serviceModule.startWebLibraryService()
  return serviceModule
}

beforeEach(() => {
  vi.resetModules()
  window.location.hash = ""
  window.localStorage.clear()
  apiMocks.listMovies.mockReset()
  apiMocks.listConnectedClients.mockReset()
  apiMocks.getMovie.mockReset()
  apiMocks.getMoviePlayback.mockReset()
  apiMocks.patchMovie.mockReset()
  apiMocks.deleteMovie.mockReset()
  apiMocks.restoreMovie.mockReset()
  apiMocks.getSettings.mockReset()
  apiMocks.patchSettings.mockReset()
  apiMocks.importMovies.mockReset()
  apiMocks.getMovieImportUpload.mockReset()
  apiMocks.deleteMovieImportUpload.mockReset()
  apiMocks.listSavedViews.mockReset()
  apiMocks.createSavedView.mockReset()
  apiMocks.patchSavedView.mockReset()
  apiMocks.deleteSavedView.mockReset()
  apiMocks.reorderSavedViews.mockReset()
  apiMocks.listHomepageRecommendationFeedback.mockReset()
  apiMocks.createHomepageRecommendationFeedback.mockReset()
  apiMocks.deleteHomepageRecommendationFeedback.mockReset()
  apiMocks.previewActorMerge.mockReset()
  apiMocks.applyActorMerge.mockReset()
  apiMocks.listActorMergeAudits.mockReset()
  apiMocks.getPersonalInsightsOverview.mockReset()
  apiMocks.getPersonalInsightsBreakdown.mockReset()
  vi.useRealTimers()
})

describe("webLibraryService bootstrap", () => {
  it("does not call the API merely because the adapter module is imported", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })

    await import("./web-library-service")
    await flushPromises()

    expect(apiMocks.listMovies).not.toHaveBeenCalled()
  })

  it("loads movies only after the selected adapter is explicitly started", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()

    expect(apiMocks.listMovies).toHaveBeenCalledTimes(1)
    expect(webLibraryService.moviesLoaded.value).toBe(true)
  })
})

describe("webLibraryService loadError", () => {
  it("stores a visible load error when the initial movie list request fails", async () => {
    apiMocks.listMovies.mockRejectedValueOnce(new Error("list failed"))

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()

    expect(webLibraryService.loadError.value).toBe("list failed")
  })

  it("stores a visible load error when movie detail loading fails", async () => {
    apiMocks.listMovies.mockResolvedValue({ items: [], total: 0 })
    apiMocks.getMovie.mockRejectedValueOnce(new Error("detail failed"))

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await expect(webLibraryService.loadMovieDetail("movie-1")).resolves.toBeUndefined()

    expect(webLibraryService.loadError.value).toBe("detail failed")
  })

  it("uses the API error message when movie detail loading fails with an HTTP error", async () => {
    apiMocks.listMovies.mockResolvedValue({ items: [], total: 0 })
    apiMocks.getMovie.mockRejectedValueOnce(
      new HttpClientError(404, {
        code: "MOVIE_NOT_FOUND",
        message: "Movie is gone",
        retryable: false,
      }),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await expect(webLibraryService.loadMovieDetail("movie-1")).resolves.toBeUndefined()

    expect(webLibraryService.loadError.value).toBe("Movie is gone")
  })
})

describe("webLibraryService mutations", () => {
  it("delegates recommendation feedback lifecycle to the Web API", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    const item = {
      id: "feedback-1",
      action: "less" as const,
      targetType: "actor" as const,
      targetValue: "Actor A",
      sourceMovieId: "movie-1",
      createdAt: "2026-07-21T00:00:00Z",
      updatedAt: "2026-07-21T00:00:00Z",
    }
    apiMocks.listHomepageRecommendationFeedback.mockResolvedValueOnce({ items: [item] })
    apiMocks.createHomepageRecommendationFeedback.mockResolvedValueOnce(item)
    apiMocks.deleteHomepageRecommendationFeedback.mockResolvedValueOnce(undefined)

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await expect(webLibraryService.listHomepageRecommendationFeedback()).resolves.toEqual({ items: [item] })
    await expect(webLibraryService.createHomepageRecommendationFeedback({
      action: "less",
      targetType: "actor",
      targetValue: "Actor A",
      sourceMovieId: "movie-1",
    })).resolves.toEqual(item)
    await expect(webLibraryService.deleteHomepageRecommendationFeedback(item.id)).resolves.toBeUndefined()
    expect(apiMocks.deleteHomepageRecommendationFeedback).toHaveBeenCalledWith(item.id)
  })

  it("keeps the ordered Saved Views cache in sync with API mutations", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    const first = {
      id: "view-1",
      name: "Unwatched",
      filters: { schemaVersion: 1 as const, mode: "library" as const, playState: "unwatched" as const },
      sortOrder: 0,
      createdAt: "2026-07-20T00:00:00Z",
      updatedAt: "2026-07-20T00:00:00Z",
    }
    const second = {
      ...first,
      id: "view-2",
      name: "4K",
      filters: { schemaVersion: 1 as const, mode: "library" as const, resolution: "4k" },
      sortOrder: 1,
    }
    apiMocks.listSavedViews.mockResolvedValueOnce({ items: [first] })
    apiMocks.createSavedView.mockResolvedValueOnce(second)
    apiMocks.patchSavedView.mockResolvedValueOnce({ ...first, name: "Not watched" })
    apiMocks.reorderSavedViews.mockResolvedValueOnce({ items: [second, { ...first, name: "Not watched", sortOrder: 1 }] })
    apiMocks.deleteSavedView.mockResolvedValueOnce(undefined)

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await webLibraryService.refreshSavedViews()
    await webLibraryService.createSavedView("4K", second.filters)
    await webLibraryService.updateSavedView(first.id, { name: "Not watched" })
    await webLibraryService.reorderSavedViews([second.id, first.id])
    expect(webLibraryService.savedViews.value.map((item) => item.id)).toEqual([second.id, first.id])
    await webLibraryService.deleteSavedView(second.id)
    expect(webLibraryService.savedViews.value).toHaveLength(1)
  })

  it("forwards connected clients requests to the API", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.listConnectedClients.mockResolvedValueOnce({
      clients: [],
      total: 0,
      localCount: 0,
      remoteCount: 0,
      sampledAt: "2026-05-15T10:00:00Z",
    })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()

    await expect(webLibraryService.listConnectedClients()).resolves.toMatchObject({
      total: 0,
      sampledAt: "2026-05-15T10:00:00Z",
    })
    expect(apiMocks.listConnectedClients).toHaveBeenCalledTimes(1)
  })

  it("recovers organize library state from settings when saving fails", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.patchSettings.mockRejectedValueOnce(new Error("save failed"))
    apiMocks.getSettings.mockResolvedValueOnce(
      settingsDto({
        organizeLibrary: true,
        libraryPaths: [
          {
            id: "library-1",
            path: "D:/media",
            title: "Media",
            firstLibraryScanPending: true,
          },
        ],
      }),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await expect(webLibraryService.setOrganizeLibrary(false)).rejects.toThrow("save failed")

    expect(apiMocks.patchSettings).toHaveBeenCalledWith({ organizeLibrary: false })
    expect(apiMocks.getSettings).toHaveBeenCalledTimes(1)
    expect(webLibraryService.organizeLibrary.value).toBe(true)
    expect(webLibraryService.libraryPaths.value).toEqual([
      {
        id: "library-1",
        path: "D:/media",
        title: "Media",
        firstLibraryScanPending: true,
      },
    ])
  })

  it("does not let a stale failed settings save override a newer successful save", async () => {
    let rejectFirstPatch: (reason?: unknown) => void
    const firstPatch = new Promise<SettingsDTO>((_, reject) => {
      rejectFirstPatch = reject
    })
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.patchSettings
      .mockReturnValueOnce(firstPatch)
      .mockResolvedValueOnce(settingsDto({ organizeLibrary: true }))

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()

    const firstSave = webLibraryService.setOrganizeLibrary(false)
    await Promise.resolve()
    expect(webLibraryService.organizeLibrary.value).toBe(false)

    await webLibraryService.setOrganizeLibrary(true)
    rejectFirstPatch!(new Error("old save failed"))
    await expect(firstSave).rejects.toThrow("old save failed")

    expect(apiMocks.getSettings).not.toHaveBeenCalled()
    expect(webLibraryService.organizeLibrary.value).toBe(true)
  })

  it("persists auto-download updates through settings", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.patchSettings.mockResolvedValueOnce(
      settingsDto({ autoDownloadUpdates: true }),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await webLibraryService.setAutoDownloadUpdates(true)

    expect(apiMocks.patchSettings).toHaveBeenCalledWith({ autoDownloadUpdates: true })
    expect(webLibraryService.autoDownloadUpdates.value).toBe(true)
  })

  it("recovers proxy settings from settings when saving fails", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.patchSettings.mockRejectedValueOnce(new Error("proxy failed"))
    apiMocks.getSettings.mockResolvedValueOnce(
      settingsDto({
        proxy: { enabled: false },
      }),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await expect(
      webLibraryService.setProxy({
        enabled: true,
        url: "http://127.0.0.1:7890",
      }),
    ).rejects.toThrow("proxy failed")

    expect(apiMocks.patchSettings).toHaveBeenCalledWith({
      proxy: {
        enabled: true,
        url: "http://127.0.0.1:7890",
      },
    })
    expect(apiMocks.getSettings).toHaveBeenCalledTimes(1)
    expect(webLibraryService.proxy.value).toEqual({ enabled: false })
  })

  it("persists the default import library path through settings", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.patchSettings.mockResolvedValueOnce(
      settingsDto({ defaultImportLibraryPathId: "library-b" }),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await webLibraryService.setDefaultImportLibraryPathId(" library-b ")

    expect(apiMocks.patchSettings).toHaveBeenCalledWith({
      defaultImportLibraryPathId: "library-b",
    })
    expect(webLibraryService.defaultImportLibraryPathId.value).toBe("library-b")
  })

  it("forwards movie import files and upload progress options to the API", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.importMovies.mockResolvedValueOnce({
      taskId: "import-1",
      type: "import.movies",
      status: "completed",
      createdAt: "2026-05-01T00:00:00.000Z",
      progress: 100,
    })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    const file = new File(["movie"], "IMP-001.mp4", { type: "video/mp4" })
    const onUploadProgress = vi.fn()
    const task = await webLibraryService.importMovies([file], { onUploadProgress })

    expect(apiMocks.importMovies).toHaveBeenCalledWith(
      [file],
      expect.objectContaining({ onUploadProgress, onUploadSessionCreated: expect.any(Function) }),
    )
    expect(task?.taskId).toBe("import-1")
  })

  it("resumes a resumable session by uploadId and clears the local ledger entry on success", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.importMovies.mockResolvedValueOnce({
      taskId: "import-resume-1",
      type: "import.movies",
      status: "completed",
      createdAt: "2026-05-01T00:00:00.000Z",
      progress: 100,
    })
    window.localStorage.setItem(
      "curated-movie-import-uploads-v1",
      JSON.stringify([
        {
          uploadId: "upload_resume00000001",
          targetLibraryPathId: "library-a",
          chunkSize: 32,
          files: [{ relativePath: "IMP-BIG.mp4", size: 8, lastModified: 1234 }],
          createdAt: "2026-08-17T00:00:00.000Z",
          lastActiveAt: new Date().toISOString(),
        },
      ]),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    const file = new File(["fake-mp4"], "IMP-BIG.mp4", { type: "video/mp4" })
    const task = await webLibraryService.importMovies([file], {
      resumeUploadId: "upload_resume00000001",
    })

    expect(apiMocks.importMovies).toHaveBeenCalledWith(
      [file],
      expect.objectContaining({ resumeUploadId: "upload_resume00000001" }),
    )
    expect(task?.taskId).toBe("import-resume-1")
    expect(window.localStorage.getItem("curated-movie-import-uploads-v1")).toBe("[]")
  })

  it("keeps the ledger entry when a resumable import attempt fails", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.importMovies.mockRejectedValueOnce(new Error("network down"))
    window.localStorage.setItem(
      "curated-movie-import-uploads-v1",
      JSON.stringify([
        {
          uploadId: "upload_resume00000002",
          targetLibraryPathId: "library-a",
          chunkSize: 32,
          files: [{ relativePath: "IMP-BIG2.mp4", size: 8, lastModified: 1234 }],
          createdAt: "2026-08-17T00:00:00.000Z",
          lastActiveAt: new Date().toISOString(),
        },
      ]),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    const file = new File(["fake-mp4"], "IMP-BIG2.mp4", { type: "video/mp4" })
    await expect(
      webLibraryService.importMovies([file], { resumeUploadId: "upload_resume00000002" }),
    ).rejects.toThrow("network down")

    const stored = JSON.parse(
      window.localStorage.getItem("curated-movie-import-uploads-v1") ?? "[]",
    )
    expect(stored).toHaveLength(1)
    expect(stored[0].uploadId).toBe("upload_resume00000002")
  })

  it("lists resumable sessions and prunes terminal or missing ones from the ledger", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    // resetModules 后适配器持有新模块图的 HttpClientError；动态导入保证 instanceof 同一身份
    const { HttpClientError: FreshHttpClientError } = await import("@/api/http-client")
    const now = new Date().toISOString()
    const seed = [
      {
        uploadId: "upload_alive00000001",
        targetLibraryPathId: "library-a",
        chunkSize: 32,
        files: [{ relativePath: "IMP-A.mp4", size: 10, lastModified: 1 }],
        createdAt: now,
        lastActiveAt: now,
      },
      {
        uploadId: "upload_dead0000000002",
        targetLibraryPathId: "library-a",
        chunkSize: 32,
        files: [{ relativePath: "IMP-B.mp4", size: 10, lastModified: 2 }],
        createdAt: now,
        lastActiveAt: now,
      },
      {
        uploadId: "upload_gone0000000003",
        targetLibraryPathId: "library-a",
        chunkSize: 32,
        files: [{ relativePath: "IMP-C.mp4", size: 10, lastModified: 3 }],
        createdAt: now,
        lastActiveAt: now,
      },
    ]
    window.localStorage.setItem("curated-movie-import-uploads-v1", JSON.stringify(seed))
    apiMocks.getMovieImportUpload.mockImplementation(async (uploadId: string) => {
      if (uploadId === "upload_alive00000001") {
        return {
          uploadId,
          targetPath: "D:/Library",
          chunkSize: 32,
          bytesReceived: 4,
          totalBytes: 10,
          state: "uploading",
          expiresAt: new Date(Date.now() + 12 * 3_600_000).toISOString(),
          files: [],
          task: {} as never,
        }
      }
      if (uploadId === "upload_dead0000000002") {
        return {
          uploadId,
          targetPath: "D:/Library",
          chunkSize: 32,
          bytesReceived: 10,
          totalBytes: 10,
          state: "committed",
          files: [],
          task: {} as never,
        }
      }
      throw new FreshHttpClientError(404)
    })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    const sessions = await webLibraryService.listResumableMovieImports()

    expect(sessions).toHaveLength(1)
    expect(sessions[0]).toMatchObject({
      uploadId: "upload_alive00000001",
      totalBytes: 10,
      bytesReceived: 4,
    })
    expect(sessions[0].files).toEqual([{ relativePath: "IMP-A.mp4", size: 10, lastModified: 1 }])
    const stored = JSON.parse(
      window.localStorage.getItem("curated-movie-import-uploads-v1") ?? "[]",
    )
    expect(stored.map((entry: { uploadId: string }) => entry.uploadId)).toEqual([
      "upload_alive00000001",
    ])
  })

  it("abandons a resumable upload by deleting the server session and ledger entry", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    const now = new Date().toISOString()
    window.localStorage.setItem(
      "curated-movie-import-uploads-v1",
      JSON.stringify([
        {
          uploadId: "upload_abandon000001",
          targetLibraryPathId: "library-a",
          chunkSize: 32,
          files: [{ relativePath: "IMP-D.mp4", size: 10, lastModified: 4 }],
          createdAt: now,
          lastActiveAt: now,
        },
      ]),
    )
    apiMocks.deleteMovieImportUpload.mockResolvedValueOnce(undefined)

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await webLibraryService.abandonMovieImportUpload("upload_abandon000001")

    expect(apiMocks.deleteMovieImportUpload).toHaveBeenCalledWith("upload_abandon000001")
    expect(window.localStorage.getItem("curated-movie-import-uploads-v1")).toBe("[]")
  })

  it("short-circuits blank movie ids without waiting for library loading", async () => {
    let resolveList: (value: {
      items: MovieListItemDTO[]
      total: number
      limit: number
      offset: number
    }) => void
    const pendingList = new Promise<{
      items: MovieListItemDTO[]
      total: number
      limit: number
      offset: number
    }>((resolve) => {
      resolveList = resolve
    })
    apiMocks.listMovies.mockReturnValueOnce(pendingList)

    const { webLibraryService } = await loadStartedWebLibraryService()
    let resolved = false
    const ensurePromise = webLibraryService.ensureMovieCached("   ").then(() => {
      resolved = true
    })

    try {
      await Promise.resolve()
      expect(resolved).toBe(true)
      expect(apiMocks.getMovie).not.toHaveBeenCalled()
    } finally {
      resolveList!({ items: [], total: 0, limit: 500, offset: 0 })
      await ensurePromise
    }
  })

  it("does not load detail when the movie is already cached in the active list", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({
      items: [movieListDto("movie-1")],
      total: 1,
      limit: 500,
      offset: 0,
    })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    apiMocks.listMovies.mockClear()
    await webLibraryService.ensureMovieCached(" movie-1 ")

    expect(apiMocks.listMovies).not.toHaveBeenCalled()
    expect(apiMocks.getMovie).not.toHaveBeenCalled()
  })

  it("does not load detail when the movie is already cached in trash", async () => {
    window.location.hash = "#/trash"
    apiMocks.listMovies
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        limit: 500,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [
          movieListDto("movie-1", {
            trashedAt: "2026-01-02T00:00:00.000Z",
          }),
        ],
        total: 1,
        limit: 500,
        offset: 0,
      })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    apiMocks.listMovies.mockClear()
    await webLibraryService.ensureMovieCached(" movie-1 ")

    expect(apiMocks.listMovies).not.toHaveBeenCalled()
    expect(apiMocks.getMovie).not.toHaveBeenCalled()
  })

  it("moves a deleted movie out of active cache and refreshes trash cache", async () => {
    apiMocks.listMovies
      .mockResolvedValueOnce({
        items: [movieListDto("movie-1"), movieListDto("movie-2")],
        total: 2,
        limit: 500,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [
          movieListDto("movie-1", {
            trashedAt: "2026-01-02T00:00:00.000Z",
          }),
        ],
        total: 1,
        limit: 500,
        offset: 0,
      })
    apiMocks.deleteMovie.mockResolvedValueOnce(undefined)

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await webLibraryService.deleteMovie(" movie-1 ")

    expect(apiMocks.deleteMovie).toHaveBeenCalledWith("movie-1")
    expect(apiMocks.listMovies).toHaveBeenLastCalledWith(
      expect.objectContaining({ mode: "trash" }),
    )
    expect(webLibraryService.movies.value.map((movie) => movie.id)).toEqual(["movie-2"])
    expect(webLibraryService.trashedMovies.value.map((movie) => movie.id)).toEqual([
      "movie-1",
    ])
  })

  it("reloads active and trash caches after restoring a movie", async () => {
    window.location.hash = "#/trash"
    apiMocks.listMovies
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        limit: 500,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [
          movieListDto("movie-1", {
            trashedAt: "2026-01-02T00:00:00.000Z",
          }),
        ],
        total: 1,
        limit: 500,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [movieListDto("movie-1")],
        total: 1,
        limit: 500,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        limit: 500,
        offset: 0,
      })
    apiMocks.restoreMovie.mockResolvedValueOnce(undefined)

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await webLibraryService.restoreMovie(" movie-1 ")

    expect(apiMocks.restoreMovie).toHaveBeenCalledWith("movie-1")
    expect(webLibraryService.movies.value.map((movie) => movie.id)).toEqual(["movie-1"])
    expect(webLibraryService.trashedMovies.value).toEqual([])
  })

  it("removes a permanently deleted movie from trash cache", async () => {
    window.location.hash = "#/trash"
    apiMocks.listMovies
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        limit: 500,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [
          movieListDto("movie-1", {
            trashedAt: "2026-01-02T00:00:00.000Z",
          }),
          movieListDto("movie-2", {
            trashedAt: "2026-01-03T00:00:00.000Z",
          }),
        ],
        total: 2,
        limit: 500,
        offset: 0,
      })
    apiMocks.deleteMovie.mockResolvedValueOnce(undefined)

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    apiMocks.listMovies.mockClear()
    await webLibraryService.deleteMoviePermanently(" movie-1 ")

    expect(apiMocks.deleteMovie).toHaveBeenCalledWith("movie-1", { permanent: true })
    expect(apiMocks.listMovies).not.toHaveBeenCalled()
    expect(webLibraryService.trashedMovies.value.map((movie) => movie.id)).toEqual([
      "movie-2",
    ])
  })

  it("patches a cached movie and merges the returned detail into cache", async () => {
    apiMocks.listMovies.mockResolvedValue({
      items: [movieListDto("movie-1", { title: "Original title" })],
      total: 1,
      limit: 500,
      offset: 0,
    })
    apiMocks.patchMovie.mockResolvedValueOnce(
      movieDetailDto("movie-1", { title: "Updated title", summary: "Updated summary" }),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    const updated = await webLibraryService.patchMovie("movie-1", {
      userTitle: "Updated title",
    })

    expect(apiMocks.patchMovie).toHaveBeenCalledWith("movie-1", {
      userTitle: "Updated title",
    })
    expect(updated?.title).toBe("Updated title")
    expect(updated?.summary).toBe("Updated summary")
    expect(webLibraryService.getMovieById("movie-1")?.title).toBe("Updated title")
  })

  it("loads a missing movie into cache before patching it", async () => {
    apiMocks.listMovies.mockResolvedValue({
      items: [],
      total: 0,
      limit: 500,
      offset: 0,
    })
    apiMocks.getMovie.mockResolvedValueOnce(
      movieDetailDto("movie-1", { title: "Loaded before patch" }),
    )
    apiMocks.patchMovie.mockResolvedValueOnce(
      movieDetailDto("movie-1", { title: "Patched after load" }),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    const updated = await webLibraryService.patchMovie("movie-1", {
      userTitle: "Patched after load",
    })

    expect(apiMocks.getMovie).toHaveBeenCalledWith("movie-1")
    expect(apiMocks.patchMovie).toHaveBeenCalledWith("movie-1", {
      userTitle: "Patched after load",
    })
    expect(updated?.title).toBe("Patched after load")
    expect(webLibraryService.getMovieById("movie-1")?.title).toBe("Patched after load")
  })

  it("patches the requested favorite state and updates the movie cache", async () => {
    apiMocks.listMovies.mockResolvedValue({
      items: [movieListDto("movie-1", { isFavorite: false })],
      total: 1,
      limit: 500,
      offset: 0,
    })
    apiMocks.patchMovie.mockResolvedValueOnce(
      movieDetailDto("movie-1", { isFavorite: true }),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    const updated = await webLibraryService.toggleFavorite("movie-1", true)

    expect(apiMocks.patchMovie).toHaveBeenCalledWith("movie-1", { isFavorite: true })
    expect(updated?.isFavorite).toBe(true)
    expect(webLibraryService.getMovieById("movie-1")?.isFavorite).toBe(true)
  })

  it("keeps the cached movie unchanged when favorite patching fails", async () => {
    apiMocks.listMovies.mockResolvedValue({
      items: [movieListDto("movie-1", { isFavorite: false })],
      total: 1,
      limit: 500,
      offset: 0,
    })
    apiMocks.patchMovie.mockRejectedValueOnce(new Error("patch failed"))

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await expect(webLibraryService.toggleFavorite("movie-1", true)).rejects.toThrow(
      "patch failed",
    )

    expect(webLibraryService.getMovieById("movie-1")?.isFavorite).toBe(false)
  })
})

describe("webLibraryService reloadMoviesFromApi", () => {
  it("debounces repeated reload requests into one API refresh", async () => {
    vi.useFakeTimers()
    apiMocks.listMovies.mockResolvedValue({
      items: [movieListDto("movie-1")],
      total: 1,
      limit: 500,
      offset: 0,
    })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    apiMocks.listMovies.mockClear()
    apiMocks.listMovies.mockResolvedValue({
      items: [movieListDto("movie-2")],
      total: 1,
      limit: 500,
      offset: 0,
    })

    await webLibraryService.reloadMoviesFromApi()
    await webLibraryService.reloadMoviesFromApi()
    await vi.advanceTimersByTimeAsync(449)
    expect(apiMocks.listMovies).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1)
    await flushPromises()

    expect(apiMocks.listMovies).toHaveBeenCalledTimes(1)
    expect(webLibraryService.movies.value.map((movie) => movie.id)).toEqual(["movie-2"])
  })
})

describe("webLibraryService loading", () => {
  it("loads all active movie pages for CSV export", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({
      items: [],
      total: 0,
      limit: 500,
      offset: 0,
    })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()

    apiMocks.listMovies.mockClear()
    apiMocks.listMovies
      .mockResolvedValueOnce({
        items: [movieListDto("movie-1")],
        total: 2,
        limit: 500,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [movieListDto("movie-2")],
        total: 2,
        limit: 500,
        offset: 1,
      })

    const movies = await webLibraryService.listMoviesForExport()

    expect(apiMocks.listMovies).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ limit: 500, offset: 0 }),
    )
    expect(apiMocks.listMovies).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ limit: 500, offset: 1 }),
    )
    expect(apiMocks.listMovies.mock.calls.every(([params]) => params.mode !== "trash")).toBe(
      true,
    )
    expect(movies.map((movie) => movie.id)).toEqual(["movie-1", "movie-2"])
    expect(webLibraryService.movies.value.map((movie) => movie.id)).toEqual([
      "movie-1",
      "movie-2",
    ])
    expect(webLibraryService.moviesLoaded.value).toBe(true)
    expect(webLibraryService.loadError.value).toBeNull()
  })

  it("loads and persists the remembered backup directory through settings", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.getSettings.mockResolvedValueOnce(
      settingsDto({ backupDirectory: "D:\\Remembered" }),
    )
    apiMocks.patchSettings.mockResolvedValueOnce(
      settingsDto({ backupDirectory: "D:\\Backups" }),
    )

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await webLibraryService.refreshSettings()
    expect(webLibraryService.backupDirectory.value).toBe("D:\\Remembered")

    await webLibraryService.setBackupDirectory("  D:\\Backups  ")

    expect(apiMocks.patchSettings).toHaveBeenCalledWith({ backupDirectory: "D:\\Backups" })
    expect(webLibraryService.backupDirectory.value).toBe("D:\\Backups")
  })

  it("marks the movie list loaded after the first page while remaining pages continue in the background", async () => {
    let resolveSecondPage: (value: {
      items: MovieListItemDTO[]
      total: number
      limit: number
      offset: number
    }) => void
    const secondPage = new Promise<{
      items: MovieListItemDTO[]
      total: number
      limit: number
      offset: number
    }>((resolve) => {
      resolveSecondPage = resolve
    })

    apiMocks.listMovies
      .mockResolvedValueOnce({
        items: [movieListDto("movie-1")],
        total: 2,
        limit: 500,
        offset: 0,
      })
      .mockReturnValueOnce(secondPage)

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()

    expect(apiMocks.listMovies).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ limit: 500, offset: 0 }),
    )
    expect(apiMocks.listMovies).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ limit: 500, offset: 1 }),
    )
    expect(webLibraryService.moviesLoaded.value).toBe(true)
    expect(webLibraryService.movies.value.map((movie) => movie.id)).toEqual(["movie-1"])

    resolveSecondPage!({
      items: [movieListDto("movie-2")],
      total: 2,
      limit: 500,
      offset: 1,
    })
    await flushPromises()

    expect(webLibraryService.movies.value.map((movie) => movie.id)).toEqual([
      "movie-1",
      "movie-2",
    ])
    expect(webLibraryService.loadError.value).toBeNull()
  })

  it("loads all movie list pages on first initialization", async () => {
    apiMocks.listMovies
      .mockResolvedValueOnce({
        items: [movieListDto("movie-1")],
        total: 2,
        limit: 500,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [movieListDto("movie-2")],
        total: 2,
        limit: 500,
        offset: 1,
      })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()

    expect(apiMocks.listMovies).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ limit: 500, offset: 0 }),
    )
    expect(apiMocks.listMovies).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ limit: 500, offset: 1 }),
    )
    expect(webLibraryService.movies.value.map((movie) => movie.id)).toEqual([
      "movie-1",
      "movie-2",
    ])
    expect(webLibraryService.moviesLoaded.value).toBe(true)
    expect(webLibraryService.loadError.value).toBeNull()
  })

  it("continues loading movie list pages beyond ten thousand items", async () => {
    const total = 10_001
    const batchSize = 500
    for (let offset = 0; offset < total; offset += batchSize) {
      const count = Math.min(batchSize, total - offset)
      apiMocks.listMovies.mockResolvedValueOnce({
        items: Array.from({ length: count }, (_, index) =>
          movieListDto(`movie-${offset + index}`),
        ),
        total,
        limit: batchSize,
        offset,
      })
    }

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()

    expect(apiMocks.listMovies).toHaveBeenLastCalledWith(
      expect.objectContaining({ limit: batchSize, offset: 10_000 }),
    )
    expect(webLibraryService.movies.value).toHaveLength(total)
    expect(webLibraryService.moviesLoaded.value).toBe(true)
    expect(webLibraryService.loadError.value).toBeNull()
  })

  it("coalesces concurrent movie detail loads and merges the detail into cache", async () => {
    apiMocks.listMovies.mockResolvedValue({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.getMovie.mockResolvedValueOnce(movieDetailDto("movie-1"))

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    const [first, second] = await Promise.all([
      webLibraryService.loadMovieDetail("movie-1"),
      webLibraryService.loadMovieDetail("movie-1"),
    ])

    expect(apiMocks.getMovie).toHaveBeenCalledTimes(1)
    expect(first?.summary).toBe("Summary movie-1")
    expect(second?.summary).toBe("Summary movie-1")
    expect(webLibraryService.getMovieById("movie-1")?.summary).toBe("Summary movie-1")
    expect(webLibraryService.loadError.value).toBeNull()
  })

  it("passes actor merge operations through and refreshes canonical movie actors after apply", async () => {
    apiMocks.listMovies.mockResolvedValue({
      items: [movieListDto("movie-1", { actors: ["Target"] })],
      total: 1,
      limit: 500,
      offset: 0,
    })
    apiMocks.previewActorMerge.mockResolvedValue({ previewToken: "token" })
    apiMocks.applyActorMerge.mockResolvedValue({
      id: "amrg_1",
      sourceName: "Source",
      targetName: "Target",
    })
    apiMocks.listActorMergeAudits.mockResolvedValue({
      items: [],
      total: 0,
      limit: 50,
      offset: 0,
    })

    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()
    await expect(
      webLibraryService.previewActorMerge({ sourceName: "Source", targetName: "Target" }),
    ).resolves.toEqual({ previewToken: "token" })
    await expect(
      webLibraryService.applyActorMerge({
        sourceName: "Source",
        targetName: "Target",
        previewToken: "token",
        confirm: true,
      }),
    ).resolves.toMatchObject({ id: "amrg_1" })
    await expect(webLibraryService.listActorMergeAudits()).resolves.toMatchObject({ total: 0 })

    expect(apiMocks.applyActorMerge).toHaveBeenCalledWith(
      expect.objectContaining({ confirm: true, previewToken: "token" }),
    )
    expect(apiMocks.listMovies).toHaveBeenCalledTimes(2)
    expect(webLibraryService.movies.value[0]?.actors).toEqual(["Target"])
  })

  it("delegates bounded personal insights queries to the Web API", async () => {
    apiMocks.listMovies.mockResolvedValue({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.getPersonalInsightsOverview.mockResolvedValue({
      range: "30d",
      watchedSeconds: 120,
    })
    apiMocks.getPersonalInsightsBreakdown.mockResolvedValue({
      range: "30d",
      dimension: "tag",
      items: [],
    })
    const { webLibraryService } = await loadStartedWebLibraryService()
    await flushPromises()

    await expect(webLibraryService.getPersonalInsightsOverview({
      range: "30d",
      timezone: "Asia/Shanghai",
    })).resolves.toMatchObject({ watchedSeconds: 120 })
    await expect(webLibraryService.getPersonalInsightsBreakdown({
      range: "30d",
      timezone: "Asia/Shanghai",
      dimension: "tag",
      limit: 10,
    })).resolves.toMatchObject({ dimension: "tag" })

    expect(apiMocks.getPersonalInsightsOverview).toHaveBeenCalledWith({
      range: "30d",
      timezone: "Asia/Shanghai",
    })
    expect(apiMocks.getPersonalInsightsBreakdown).toHaveBeenCalledWith({
      range: "30d",
      timezone: "Asia/Shanghai",
      dimension: "tag",
      limit: 10,
    })
  })
})

describe("webLibraryService playback prefetch", () => {
  function playbackDto(id: string, url = "") {
    return {
      movieId: id,
      mode: "direct" as const,
      url,
      canDirectPlay: true,
    }
  }

  it("resolves an uncached movie via the detail endpoint without the full library load", async () => {
    let resolveList: ((value: { items: MovieListItemDTO[]; total: number; limit: number; offset: number }) => void) | undefined
    apiMocks.listMovies.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveList = resolve
        }),
    )
    apiMocks.getMovie.mockResolvedValueOnce(movieDetailDto("movie-deep"))

    // Cold service: never started, so the list request above stays pending.
    const serviceModule = await import("./web-library-service")
    await serviceModule.webLibraryService.ensureMovieCached(" movie-deep ")

    expect(apiMocks.getMovie).toHaveBeenCalledWith("movie-deep")
    expect(apiMocks.listMovies).not.toHaveBeenCalled()
    expect(
      serviceModule.webLibraryService.movies.value.some((movie) => movie.id === "movie-deep"),
    ).toBe(true)
    resolveList?.({ items: [], total: 0, limit: 500, offset: 0 })
  })

  it("falls back to the full library load when the detail request finds nothing", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.getMovie.mockResolvedValueOnce(undefined)

    const serviceModule = await import("./web-library-service")
    await serviceModule.webLibraryService.ensureMovieCached(" movie-gone ")

    expect(apiMocks.getMovie).toHaveBeenCalledWith("movie-gone")
    expect(apiMocks.listMovies).toHaveBeenCalledTimes(1)
  })

  it("consumes a fresh prefetch exactly once and fills the stream URL", async () => {
    apiMocks.getMoviePlayback.mockResolvedValueOnce(playbackDto("movie-1"))
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })

    const { webLibraryService } = await loadStartedWebLibraryService()
    webLibraryService.prefetchMoviePlayback(" movie-1 ")
    await flushPromises()

    await expect(webLibraryService.getMoviePlayback("movie-1")).resolves.toMatchObject({
      movieId: "movie-1",
      mode: "direct",
    })
    // Consume-once: the descriptor reflects server state at request time, so a
    // later request must hit the API again instead of reusing the snapshot.
    apiMocks.getMoviePlayback.mockResolvedValueOnce(playbackDto("movie-1", "/api/library/movies/movie-1/stream"))
    await webLibraryService.getMoviePlayback("movie-1")

    expect(apiMocks.getMoviePlayback).toHaveBeenCalledTimes(2)
  })

  it("reuses an in-flight prefetch and never serves a failed one", async () => {
    apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.getMoviePlayback.mockRejectedValueOnce(new Error("descriptor failed"))

    const { webLibraryService } = await loadStartedWebLibraryService()
    webLibraryService.prefetchMoviePlayback("movie-1")
    webLibraryService.prefetchMoviePlayback(" movie-1 ")
    expect(apiMocks.getMoviePlayback).toHaveBeenCalledTimes(1)

    await expect(webLibraryService.getMoviePlayback("movie-1")).rejects.toBeTruthy()

    apiMocks.getMoviePlayback.mockResolvedValueOnce(playbackDto("movie-1"))
    await expect(webLibraryService.getMoviePlayback("movie-1")).resolves.toMatchObject({
      movieId: "movie-1",
    })
    expect(apiMocks.getMoviePlayback).toHaveBeenCalledTimes(2)
  })

  it("drops a stale prefetch instead of serving an outdated descriptor", async () => {
    vi.useFakeTimers()
    try {
      apiMocks.listMovies.mockResolvedValueOnce({ items: [], total: 0, limit: 500, offset: 0 })
      const { webLibraryService } = await loadStartedWebLibraryService()

      apiMocks.getMoviePlayback.mockResolvedValueOnce(playbackDto("movie-1"))
      webLibraryService.prefetchMoviePlayback("movie-1")
      vi.advanceTimersByTime(15_000)

      apiMocks.getMoviePlayback.mockResolvedValueOnce(playbackDto("movie-1", "/fresh"))
      await expect(webLibraryService.getMoviePlayback("movie-1")).resolves.toMatchObject({
        url: "/fresh",
      })
      expect(apiMocks.getMoviePlayback).toHaveBeenCalledTimes(2)
    } finally {
      vi.useRealTimers()
    }
  })
})
