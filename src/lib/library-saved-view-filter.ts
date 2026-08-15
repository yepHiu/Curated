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

function hasLocalUserRating(movie: Movie): boolean {
  return typeof movie.userRating === "number"
}

function matchesUserRating(movie: Movie, filters: SavedViewFiltersV1): boolean {
  if (filters.unrated) {
    return !hasLocalUserRating(movie)
  }
  if (filters.userRating === undefined) {
    return true
  }
  return hasLocalUserRating(movie) && movie.userRating! >= filters.userRating
}

function matchesYear(movie: Movie, year: string | undefined): boolean {
  const wanted = year?.trim().toLowerCase() ?? ""
  if (!wanted) {
    return true
  }
  if (wanted === "unknown") {
    return !Number.isInteger(movie.year) || movie.year < 1800 || movie.year > 3000
  }
  return movie.year === Number(wanted)
}

function matchesRuntime(movie: Movie, runtime: SavedViewFiltersV1["runtime"]): boolean {
  if (!runtime) {
    return true
  }
  const minutes = movie.runtimeMinutes
  if (!Number.isFinite(minutes) || minutes <= 0) {
    return false
  }
  if (runtime === "short") {
    return minutes < 90
  }
  if (runtime === "standard") {
    return minutes >= 90 && minutes <= 150
  }
  return minutes > 150
}

function matchesCatalog(movie: Movie, catalog: SavedViewFiltersV1["catalog"]): boolean {
  if (!catalog) {
    return true
  }
  if (catalog === "unscraped") {
    return movie.actors.length === 0 && movie.tags.length === 0
  }
  return !movie.coverUrl?.trim() && !movie.thumbUrl?.trim()
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
 * `userRating` is a minimum local score and never falls back to scraper/site rating.
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
    if (!matchesUserRating(movie, filters)) {
      return false
    }
    return (
      matchesResolution(movie, filters.resolution) &&
      matchesAddedWindow(movie, filters.addedWithinDays, now) &&
      matchesYear(movie, filters.year) &&
      matchesRuntime(movie, filters.runtime) &&
      matchesCatalog(movie, filters.catalog)
    )
  })
}
