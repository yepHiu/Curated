import { describe, expect, it } from "vitest"
import { mockLibraryService } from "@/services/adapters/mock/mock-library-service"

describe("mockLibraryService", () => {
  it("returns undefined for an unknown movie id", () => {
    expect(mockLibraryService.getMovieById("missing-movie")).toBeUndefined()
  })

  it("toggles favorite state in the shared movie source", () => {
    const libraryService = mockLibraryService
    const movieId = libraryService.movies.value[0]?.id

    expect(movieId).toBeTruthy()

    if (!movieId) {
      return
    }

    const originalFavorite = libraryService.getMovieById(movieId)?.isFavorite ?? false

    libraryService.toggleFavorite(movieId, !originalFavorite)

    expect(libraryService.getMovieById(movieId)?.isFavorite).toBe(!originalFavorite)

    libraryService.toggleFavorite(movieId, originalFavorite)

    expect(libraryService.getMovieById(movieId)?.isFavorite).toBe(originalFavorite)
  })
})
