import type { LocationQuery } from "vue-router"
import type { LibraryMode } from "@/domain/library/types"
import {
  buildPlayerRouteFromBrowseIntent,
  buildPlayerRouteFromCuratedFrameIntent,
  buildPlayerRouteFromHistoryIntent,
} from "@/lib/navigation-intent"

/** Open player from browse/detail with library query context and optional resume `t`. */
export function buildPlayerRouteFromBrowse(
  movieId: string,
  currentQuery: LocationQuery,
  sourceMode: LibraryMode,
  options?: { back?: "browse" | "detail"; fileId?: string },
) {
  return buildPlayerRouteFromBrowseIntent(
    movieId,
    currentQuery,
    sourceMode,
    options?.back ?? "detail",
    options?.fileId,
  )
}

/** Open player from History page with an explicit history return target. */
export function buildPlayerRouteFromHistory(movieId: string, resumeSec: number, fileId?: string, sourceMode: LibraryMode = "library") {
  return buildPlayerRouteFromHistoryIntent(movieId, resumeSec, fileId, sourceMode)
}

/** Open player from curated-frame capture time with an explicit curated return target. */
export function buildPlayerRouteFromCuratedFrame(movieId: string, resumeSec: number, fileId?: string) {
  return buildPlayerRouteFromCuratedFrameIntent(movieId, resumeSec, fileId)
}
