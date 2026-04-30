import { flushPromises } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { MovieDetailDTO, MovieListItemDTO } from "@/api/types"

const apiMocks = vi.hoisted(() => ({
  listMovies: vi.fn(),
  getMovie: vi.fn(),
  patchMovie: vi.fn(),
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
  apiMocks.listMovies.mockReset()
  apiMocks.getMovie.mockReset()
  apiMocks.patchMovie.mockReset()
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
})

describe("webLibraryService mutations", () => {
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
