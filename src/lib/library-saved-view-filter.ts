import type { SavedViewFiltersV1 } from "@/api/types"
import type { Movie } from "@/domain/movie/types"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"
import { normalizeLibraryResolutionFilter } from "@/lib/library-query"

export interface SavedViewFilterRuntime {
  now?: Date
  hasPlayedMovie(movieId: string): boolean
  getProgress(movieId: string): PlaybackProgressEntry | undefined
}

function matchesPlayState(
  movie: Movie,
  playState: SavedViewFiltersV1["playState"],
  runtime: SavedViewFilterRuntime,
): boolean {
  if (!playState || playState === "all") {
    return true
  }
  const progress = runtime.getProgress(movie.id)
  const hasMeaningfulProgress = Boolean(progress && progress.positionSec >= 5)
  if (playState === "unwatched") {
    return !runtime.hasPlayedMovie(movie.id) && !hasMeaningfulProgress
  }
  if (playState === "in-progress") {
    return Boolean(
      progress &&
        progress.positionSec >= 5 &&
        (progress.durationSec <= 0 || progress.positionSec < progress.durationSec * 0.95),
    )
  }
  return Boolean(
    progress &&
      progress.durationSec > 0 &&
      progress.positionSec >= progress.durationSec * 0.95,
  )
}

function matchesResolution(movie: Movie, resolution: string | undefined): boolean {
  const wanted = normalizeLibraryResolutionFilter(resolution ?? "")
  if (!wanted) {
    return true
  }
  return normalizeLibraryResolutionFilter(movie.resolution) === wanted
}

function matchesAddedWindow(movie: Movie, days: number | undefined, now: Date): boolean {
  if (days === undefined) {
    return true
  }
  const addedAt = Date.parse(movie.addedAt)
  if (!Number.isFinite(addedAt)) {
    return false
  }
  return addedAt >= now.getTime() - days * 24 * 60 * 60 * 1000
}

/**
 * Applies the Saved Views-only filters after the existing mode/q/entity filters.
 * Exact user rating deliberately ignores scraper/site rating fallbacks.
 */
export function filterMoviesBySavedView(
  movies: readonly Movie[],
  filters: SavedViewFiltersV1,
  runtime: SavedViewFilterRuntime,
): Movie[] {
  const now = runtime.now ?? new Date()
  return movies.filter((movie) => {
    if (!matchesPlayState(movie, filters.playState, runtime)) {
      return false
    }
    if (
      filters.userRating !== undefined &&
      (typeof movie.userRating !== "number" || movie.userRating !== filters.userRating)
    ) {
      return false
    }
    return (
      matchesResolution(movie, filters.resolution) &&
      matchesAddedWindow(movie, filters.addedWithinDays, now)
    )
  })
}
