import type { SavedViewFiltersV1 } from "@/api/types"
import type { Movie } from "@/domain/movie/types"
import { buildSavedViewRouteTarget } from "@/lib/library-query"
import { listLibraryQueueMovies } from "@/lib/player-playlist"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"

/** Visible rows before the confirm card folds the rest. */
export const SAVED_VIEW_CONFIRM_PREVIEW = 4
/** Maximum rows after expanding the folded list. */
export const SAVED_VIEW_CONFIRM_EXPAND_MAX = 20

export function listMoviesMatchingSavedView(input: {
  movies: readonly Movie[]
  trashedMovies: readonly Movie[]
  filters: SavedViewFiltersV1
  hasPlayedMovie: (movieId: string) => boolean
  getProgress: (movieId: string) => PlaybackProgressEntry | undefined
}): Movie[] {
  const target = buildSavedViewRouteTarget(input.filters)
  return listLibraryQueueMovies({
    movies: input.movies,
    trashedMovies: input.trashedMovies,
    query: {
      ...target.query,
      browse: String(target.name),
    },
    hasPlayedMovie: input.hasPlayedMovie,
    getProgress: input.getProgress,
  })
}

export function splitSavedViewConfirmMovies(movies: readonly Movie[], expanded: boolean) {
  const total = movies.length
  const limit = expanded
    ? Math.min(total, SAVED_VIEW_CONFIRM_EXPAND_MAX)
    : Math.min(total, SAVED_VIEW_CONFIRM_PREVIEW)
  const visible = movies.slice(0, limit)
  return {
    visible,
    total,
    foldedCount: Math.max(0, total - SAVED_VIEW_CONFIRM_PREVIEW),
    remainderAfterExpand: Math.max(0, total - visible.length),
    canExpand: !expanded && total > SAVED_VIEW_CONFIRM_PREVIEW,
    canCollapse: expanded && total > SAVED_VIEW_CONFIRM_PREVIEW,
  }
}
