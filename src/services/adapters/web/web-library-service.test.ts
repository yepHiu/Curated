import { flushPromises } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { HttpClientError } from "@/api/http-client"
import type { MovieDetailDTO, MovieListItemDTO, SettingsDTO } from "@/api/types"

const apiMocks = vi.hoisted(() => ({
  listMovies: vi.fn(),
  getMovie: vi.fn(),
  patchMovie: vi.fn(),
  deleteMovie: vi.fn(),
  restoreMovie: vi.fn(),
  getSettings: vi.fn(),
  patchSettings: vi.fn(),
  importMovies: vi.fn(),
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

beforeEach(() => {
  vi.resetModules()
  window.location.hash = ""
  apiMocks.listMovies.mockReset()
  apiMocks.getMovie.mockReset()
  apiMocks.patchMovie.mockReset()
  apiMocks.deleteMovie.mockReset()
  apiMocks.restoreMovie.mockReset()
  apiMocks.getSettings.mockReset()
  apiMocks.patchSettings.mockReset()
  apiMocks.importMovies.mockReset()
  vi.useRealTimers()
})

describe("webLibraryService loadError", () => {
  it("stores a visible load error when the initial movie list request fails", async () => {
    apiMocks.listMovies.mockRejectedValueOnce(new Error("list failed"))

    const { webLibraryService } = await import("./web-library-service")
    await flushPromises()

    expect(webLibraryService.loadError.value).toBe("list failed")
  })

  it("stores a visible load error when movie detail loading fails", async () => {
    apiMocks.listMovies.mockResolvedValue({ items: [], total: 0 })
    apiMocks.getMovie.mockRejectedValueOnce(new Error("detail failed"))

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
    await flushPromises()
    await expect(webLibraryService.loadMovieDetail("movie-1")).resolves.toBeUndefined()

    expect(webLibraryService.loadError.value).toBe("Movie is gone")
  })
})

describe("webLibraryService mutations", () => {
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
    await flushPromises()
    const file = new File(["movie"], "IMP-001.mp4", { type: "video/mp4" })
    const onUploadProgress = vi.fn()
    const task = await webLibraryService.importMovies([file], { onUploadProgress })

    expect(apiMocks.importMovies).toHaveBeenCalledWith([file], { onUploadProgress })
    expect(task?.taskId).toBe("import-1")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

    const { webLibraryService } = await import("./web-library-service")
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

  it("coalesces concurrent movie detail loads and merges the detail into cache", async () => {
    apiMocks.listMovies.mockResolvedValue({ items: [], total: 0, limit: 500, offset: 0 })
    apiMocks.getMovie.mockResolvedValueOnce(movieDetailDto("movie-1"))

    const { webLibraryService } = await import("./web-library-service")
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
})
