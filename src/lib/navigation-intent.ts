import type { LibraryMode } from "@/domain/library/types"
import type { LocationQuery, RouteLocationNormalizedLoaded, RouteLocationRaw } from "vue-router"
import { buildMovieRouteQuery, getBrowseSourceMode, mergeLibraryQuery } from "@/lib/library-query"
import { getResumeSecondsForOpenPlayer } from "@/lib/playback-progress-storage"

const navigationBackTargets = ["home", "browse", "detail", "history", "curated-frames"] as const

export type NavigationBackTarget = (typeof navigationBackTargets)[number]

type RouteLike = Pick<RouteLocationNormalizedLoaded, "name" | "query">

function isNavigationBackTarget(value: unknown): value is NavigationBackTarget {
  return typeof value === "string" && navigationBackTargets.includes(value as NavigationBackTarget)
}

function formatResumeSecondsForRoute(resumeSec: number): string {
  const normalized = Math.max(0, resumeSec)
  return String(Number(normalized.toFixed(3)))
}

function buildPlayerQuery(
  movieId: string,
  currentQuery: LocationQuery,
  sourceMode: LibraryMode,
  back: Extract<NavigationBackTarget, "browse" | "detail">,
) {
  const query: LocationQuery = {
    ...buildMovieRouteQuery(currentQuery, sourceMode, movieId),
    autoplay: "1",
    back,
  }

  const resumeSec = getResumeSecondsForOpenPlayer(movieId)
  if (resumeSec !== undefined) {
    query.t = String(resumeSec)
  }

  return query
}

function buildBrowseBackLink(query: LocationQuery, movieId: string): RouteLocationRaw {
  return {
    name: getBrowseSourceMode(query),
    query: mergeLibraryQuery(query, {
      selected: movieId,
    }),
  }
}

export function getNavigationBackTarget(query: LocationQuery): NavigationBackTarget {
  if (isNavigationBackTarget(query.back)) {
    return query.back
  }
  if (query.from === "history" || query.from === "curated-frames") {
    return query.from
  }
  return "detail"
}

export function buildDetailRouteFromBrowse(
  movieId: string,
  currentQuery: LocationQuery,
  sourceMode: LibraryMode,
): RouteLocationRaw {
  return {
    name: "detail",
    params: { id: movieId },
    query: buildMovieRouteQuery(currentQuery, sourceMode, movieId),
  }
}

export function buildPlayerRouteFromBrowseIntent(
  movieId: string,
  currentQuery: LocationQuery,
  sourceMode: LibraryMode,
  back: Extract<NavigationBackTarget, "browse" | "detail">,
): RouteLocationRaw {
  return {
    name: "player",
    params: { id: movieId },
    query: buildPlayerQuery(movieId, currentQuery, sourceMode, back),
  }
}

export function buildPlayerRouteFromHistoryIntent(
  movieId: string,
  resumeSec: number,
): RouteLocationRaw {
  return {
    name: "player",
    params: { id: movieId },
    query: {
      autoplay: "1",
      back: "history",
      t: String(Math.max(0, Math.floor(resumeSec))),
    },
  }
}

export function buildPlayerRouteFromCuratedFrameIntent(
  movieId: string,
  resumeSec: number,
): RouteLocationRaw {
  return {
    name: "player",
    params: { id: movieId },
    query: {
      autoplay: "1",
      back: "curated-frames",
      t: formatResumeSecondsForRoute(resumeSec),
    },
  }
}

export function resolveNavigationBackLink(
  route: RouteLike,
  currentMovieId?: string,
): { to: RouteLocationRaw; labelKey: string } {
  if (route.name === "player" && currentMovieId) {
    const backTarget = getNavigationBackTarget(route.query)
    if (backTarget === "history") {
      return {
        to: { name: "history" },
        labelKey: "shell.backHistory",
      }
    }
    if (backTarget === "home") {
      return {
        to: { name: "home" },
        labelKey: "shell.backHome",
      }
    }
    if (backTarget === "curated-frames") {
      return {
        to: { name: "curated-frames" },
        labelKey: "shell.backCurated",
      }
    }
    if (backTarget === "browse") {
      return {
        to: buildBrowseBackLink(route.query, currentMovieId),
        labelKey: "shell.backLibrary",
      }
    }
    return {
      to: buildDetailRouteFromBrowse(
        currentMovieId,
        route.query,
        getBrowseSourceMode(route.query),
      ),
      labelKey: "shell.backDetail",
    }
  }

  if (route.name === "detail" && currentMovieId) {
    if (getNavigationBackTarget(route.query) === "home") {
      return {
        to: { name: "home" },
        labelKey: "shell.backHome",
      }
    }
    return {
      to: buildBrowseBackLink(route.query, currentMovieId),
      labelKey: "shell.backLibrary",
    }
  }

  return {
    to: { name: "library" },
    labelKey: "shell.backLibrary",
  }
}
