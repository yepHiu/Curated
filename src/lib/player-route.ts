import type { LocationQuery } from "vue-router"
import type { LibraryMode } from "@/domain/library/types"
import { buildMovieRouteQuery } from "@/lib/library-query"
import { getResumeSecondsForOpenPlayer } from "@/lib/playback-progress-storage"

/** Open player from browse/detail with library query context and optional resume `t`. */
export function buildPlayerRouteFromBrowse(
  movieId: string,
  currentQuery: LocationQuery,
  sourceMode: LibraryMode,
) {
  const t = getResumeSecondsForOpenPlayer(movieId)
  const base = buildMovieRouteQuery(currentQuery, sourceMode, movieId)
  const query: LocationQuery = {
    ...base,
    autoplay: "1",
  }
  if (t !== undefined) {
    query.t = String(t)
  }
  return {
    name: "player" as const,
    params: { id: movieId },
    query,
  }
}

/** Open player from History page (minimal query, back stack uses `from`). */
export function buildPlayerRouteFromHistory(movieId: string, resumeSec: number) {
  const query: LocationQuery = {
    autoplay: "1",
    from: "history",
    t: String(Math.max(0, Math.floor(resumeSec))),
  }
  return {
    name: "player" as const,
    params: { id: movieId },
    query,
  }
}

/** 从萃取帧库跳转播放：保留时间点，返回栈可识别来源 */
export function buildPlayerRouteFromCuratedFrame(movieId: string, resumeSec: number) {
  const query: LocationQuery = {
    autoplay: "1",
    from: "curated-frames",
    t: String(Math.max(0, Math.floor(resumeSec))),
  }
  return {
    name: "player" as const,
    params: { id: movieId },
    query,
  }
}
