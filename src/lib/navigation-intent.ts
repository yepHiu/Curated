import type { LibraryMode } from "@/domain/library/types"
import type { LocationQuery, RouteLocationNormalizedLoaded, RouteLocationRaw } from "vue-router"
import {
  buildMovieRouteQuery,
  getBrowseSourceMode,
  getDetailBrowseTargetMode,
  mergeLibraryQuery,
  type DetailBrowseTargetKind,
} from "@/lib/library-query"
import { getResumeSecondsForOpenPlayer } from "@/lib/playback-progress-storage"

const navigationBackTargets = ["home", "browse", "detail", "actor", "history", "curated-frames"] as const

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

function getActorNameQuery(query: LocationQuery): string {
  const raw = query.actor
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === "string" ? value.trim() : ""
}

export function buildActorDetailRoute(
  actorName: string,
  selectedMovieId?: string,
): RouteLocationRaw {
  const selected = selectedMovieId?.trim()
  return {
    name: "actor-detail",
    params: { actorName },
    query: selected ? { selected } : {},
  }
}

export function buildActorDetailRouteFromDetail(
  actorName: string,
  movieId: string,
  currentQuery: LocationQuery,
  sourceMode: LibraryMode,
): RouteLocationRaw {
  return {
    name: "actor-detail",
    params: { actorName },
    query: {
      ...buildMovieRouteQuery(currentQuery, sourceMode, movieId),
      back: "detail",
    },
  }
}

function buildActorBackLink(query: LocationQuery, movieId: string): RouteLocationRaw {
  const actorName = getActorNameQuery(query)
  if (!actorName) {
    return buildBrowseBackLink(query, movieId)
  }
  return buildActorDetailRoute(actorName, movieId)
}

function hasExplicitBackTarget(query: LocationQuery, target: NavigationBackTarget): boolean {
  return query.back === target
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

export interface FilteredBrowseRouteFromDetailInput {
  movieId: string
  currentQuery: LocationQuery
  sourceMode: LibraryMode
  kind: DetailBrowseTargetKind
  value: string
}

export function buildFilteredBrowseRouteFromDetail({
  movieId,
  currentQuery,
  sourceMode,
  kind,
  value,
}: FilteredBrowseRouteFromDetailInput): RouteLocationRaw {
  const trimmed = value.trim()
  const filterPatch: Partial<
    Record<"q" | "tab" | "selected" | "from" | "tag" | "actor" | "studio", string | undefined>
  > = {
    q: undefined,
    tag: undefined,
    actor: undefined,
    studio: undefined,
    tab: "all",
    selected: movieId,
  }

  filterPatch[kind] = trimmed || undefined

  return {
    name: getDetailBrowseTargetMode(sourceMode, kind),
    query: {
      ...mergeLibraryQuery(currentQuery, filterPatch),
      back: "detail",
      browse: sourceMode,
      selected: movieId,
    },
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

export function buildDetailRouteFromActor(movieId: string, actorName: string): RouteLocationRaw {
  return {
    name: "detail",
    params: { id: movieId },
    query: {
      actor: actorName,
      back: "actor",
      selected: movieId,
    },
  }
}

export function buildPlayerRouteFromActorIntent(
  movieId: string,
  actorName: string,
): RouteLocationRaw {
  const query: LocationQuery = {
    actor: actorName,
    autoplay: "1",
    back: "actor",
    selected: movieId,
  }

  const resumeSec = getResumeSecondsForOpenPlayer(movieId)
  if (resumeSec !== undefined) {
    query.t = String(resumeSec)
  }

  return {
    name: "player",
    params: { id: movieId },
    query,
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
  if (route.name === "actor-detail") {
    if (currentMovieId && hasExplicitBackTarget(route.query, "detail")) {
      return {
        to: buildDetailRouteFromBrowse(
          currentMovieId,
          route.query,
          getBrowseSourceMode(route.query),
        ),
        labelKey: "shell.backDetail",
      }
    }

    return {
      to: { name: "actors" },
      labelKey: "shell.backActors",
    }
  }

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
    if (backTarget === "actor") {
      return {
        to: buildActorBackLink(route.query, currentMovieId),
        labelKey: "shell.backActor",
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

  if (
    route.name !== "detail" &&
    currentMovieId &&
    hasExplicitBackTarget(route.query, "detail")
  ) {
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
    const backTarget = getNavigationBackTarget(route.query)
    if (backTarget === "home") {
      return {
        to: { name: "home" },
        labelKey: "shell.backHome",
      }
    }
    if (backTarget === "actor") {
      return {
        to: buildActorBackLink(route.query, currentMovieId),
        labelKey: "shell.backActor",
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
