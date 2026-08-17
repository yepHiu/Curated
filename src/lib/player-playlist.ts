import type { LocationQuery } from "vue-router"
import type { Movie } from "@/domain/movie/types"
import type { LibraryMode } from "@/domain/library/types"
import {
  buildSavedViewFiltersV1,
  getBrowseSourceMode,
  getLibraryActorExactFilters,
  getLibrarySearchQuery,
  getLibrarySortQuery,
} from "@/lib/library-query"
import { filterMoviesBySavedView } from "@/lib/library-saved-view-filter"
import { isMovieRecentlyAdded } from "@/lib/library-stats"
import { movieSearchHaystack } from "@/lib/movie-search"
import { compareMoviesByLibrarySort } from "@/lib/movie-sort"
import { getProgress, type PlaybackProgressEntry } from "@/lib/playback-progress-storage"

export const PLAYER_PLAYLIST_WINDOW_RADIUS = 10
export const PLAYER_PLAYLIST_PREFETCH = 4
export const PLAYER_PLAYLIST_WINDOW_MAX =
  PLAYER_PLAYLIST_WINDOW_RADIUS * 2 + 1 + PLAYER_PLAYLIST_PREFETCH
export const PLAYER_PLAYLIST_AUTO_ADVANCE_STORAGE_KEY = "curated-player-playlist-auto-advance-v1"

export type PlayerPlaylistSource = "browse" | "actor"

export type PlaylistWindow = {
  start: number
  end: number
}

function getDetailParent(query: LocationQuery): string {
  const raw = query.detailBack
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === "string" ? value.trim() : ""
}

/**
 * Queue follows the browsing origin, not the last hop.
 * Library / actor pages may open the player through the movie detail page.
 */
export function resolvePlayerPlaylistSource(query: LocationQuery): PlayerPlaylistSource | null {
  if (query.back === "browse") {
    return "browse"
  }
  if (query.back === "actor") {
    return "actor"
  }
  if (query.back !== "detail") {
    return null
  }

  const parent = getDetailParent(query)
  if (parent === "home") {
    return null
  }
  if (parent === "actor") {
    return "actor"
  }
  return "browse"
}

export function getPlayerPlaylistActorName(query: LocationQuery): string {
  const raw = query.actor
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === "string" ? value.trim() : ""
}

function resolveActorNameFromSearch(movies: readonly Movie[], query: LocationQuery): string {
  if (getLibraryActorExactFilters(query).length > 0) {
    return ""
  }
  const needle = getLibrarySearchQuery(query).trim().toLowerCase()
  if (!needle) {
    return ""
  }
  for (const movie of movies) {
    for (const raw of movie.actors) {
      const name = raw.trim()
      if (name && name.toLowerCase() === needle) {
        return name
      }
    }
  }
  return ""
}

export function listActorQueueMovies(movies: readonly Movie[], actorName: string): Movie[] {
  const actor = actorName.trim()
  if (!actor) {
    return []
  }
  return movies.filter((movie) => movie.actors.includes(actor))
}

export function listLibraryQueueMovies(input: {
  movies: readonly Movie[]
  trashedMovies: readonly Movie[]
  query: LocationQuery
  hasPlayedMovie: (movieId: string) => boolean
  getProgress: (movieId: string) => PlaybackProgressEntry | undefined
}): Movie[] {
  const mode: LibraryMode = getBrowseSourceMode(input.query)
  const raw = mode === "trash" ? input.trashedMovies : input.movies
  let list: Movie[]
  if (mode === "trash") {
    list = [...raw]
  } else if (mode === "favorites") {
    list = raw.filter((movie) => movie.isFavorite)
  } else if (mode === "recent") {
    list = raw
      .filter((movie) => isMovieRecentlyAdded(movie.addedAt))
      .slice()
      .sort((left, right) => right.addedAt.localeCompare(left.addedAt))
  } else {
    list = [...raw]
  }

  if (mode !== "trash") {
    const queryLower = getLibrarySearchQuery(input.query).trim().toLowerCase()
    const actorViaQ = resolveActorNameFromSearch(raw, input.query)
    if (queryLower && !actorViaQ) {
      list = list.filter((movie) => movieSearchHaystack(movie).includes(queryLower))
    }
    if (actorViaQ && getLibraryActorExactFilters(input.query).length === 0) {
      list = list.filter((movie) => movie.actors.includes(actorViaQ))
    }

    const filters = buildSavedViewFiltersV1(mode, input.query)
    list = filterMoviesBySavedView(list, filters, {
      hasPlayedMovie: input.hasPlayedMovie,
      getProgress: input.getProgress,
    })
  }

  const sort = getLibrarySortQuery(input.query)
  return list.slice().sort((left, right) => compareMoviesByLibrarySort(left, right, sort))
}

export function listPlayerPlaylistMovies(input: {
  source: PlayerPlaylistSource | null
  movies: readonly Movie[]
  trashedMovies: readonly Movie[]
  query: LocationQuery
  hasPlayedMovie: (movieId: string) => boolean
  getProgress?: (movieId: string) => PlaybackProgressEntry | undefined
}): Movie[] {
  if (input.source === "actor") {
    return listActorQueueMovies(input.movies, getPlayerPlaylistActorName(input.query))
  }
  if (input.source === "browse") {
    return listLibraryQueueMovies({
      movies: input.movies,
      trashedMovies: input.trashedMovies,
      query: input.query,
      hasPlayedMovie: input.hasPlayedMovie,
      getProgress: input.getProgress ?? getProgress,
    })
  }
  return []
}

export function findPlaylistIndex(movies: readonly Movie[], movieId: string): number {
  const id = movieId.trim()
  if (!id) {
    return -1
  }
  return movies.findIndex((movie) => movie.id === id)
}

export function recenterPlaylistWindow(currentIndex: number, total: number): PlaylistWindow {
  if (total <= 0 || currentIndex < 0) {
    return { start: 0, end: -1 }
  }
  return {
    start: Math.max(0, currentIndex - PLAYER_PLAYLIST_WINDOW_RADIUS),
    end: Math.min(total - 1, currentIndex + PLAYER_PLAYLIST_WINDOW_RADIUS),
  }
}

export function slidePlaylistWindow(
  window: PlaylistWindow,
  direction: "up" | "down",
  _currentIndex: number,
  total: number,
): PlaylistWindow {
  if (total <= 0) {
    return { start: 0, end: -1 }
  }

  let nextStart = Math.max(0, window.start)
  let nextEnd = Math.min(total - 1, window.end)
  if (nextEnd < nextStart) {
    return { start: 0, end: Math.min(total - 1, PLAYER_PLAYLIST_WINDOW_MAX - 1) }
  }

  if (direction === "up") {
    nextStart = Math.max(0, nextStart - PLAYER_PLAYLIST_PREFETCH)
    if (nextEnd - nextStart + 1 > PLAYER_PLAYLIST_WINDOW_MAX) {
      nextEnd = Math.min(total - 1, nextStart + PLAYER_PLAYLIST_WINDOW_MAX - 1)
    }
  } else {
    nextEnd = Math.min(total - 1, nextEnd + PLAYER_PLAYLIST_PREFETCH)
    if (nextEnd - nextStart + 1 > PLAYER_PLAYLIST_WINDOW_MAX) {
      nextStart = Math.max(0, nextEnd - PLAYER_PLAYLIST_WINDOW_MAX + 1)
    }
  }

  return { start: nextStart, end: nextEnd }
}

export function readPlaylistAutoAdvance(): boolean {
  if (typeof localStorage === "undefined") {
    return true
  }
  try {
    const raw = localStorage.getItem(PLAYER_PLAYLIST_AUTO_ADVANCE_STORAGE_KEY)
    if (raw === "0" || raw === "false") {
      return false
    }
    return true
  } catch {
    return true
  }
}

export function writePlaylistAutoAdvance(enabled: boolean): void {
  if (typeof localStorage === "undefined") {
    return
  }
  try {
    localStorage.setItem(PLAYER_PLAYLIST_AUTO_ADVANCE_STORAGE_KEY, enabled ? "1" : "0")
  } catch {
    // Ignore quota / private-mode failures; in-memory state still applies.
  }
}
