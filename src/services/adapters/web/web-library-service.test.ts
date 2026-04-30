import { flushPromises } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  listMovies: vi.fn(),
  getMovie: vi.fn(),
}))

function movieListDto(id: string) {
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
  }
}

function movieDetailDto(id: string) {
  return {
    ...movieListDto(id),
    summary: `Summary ${id}`,
    previewImages: [],
    metadataRating: 4,
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
