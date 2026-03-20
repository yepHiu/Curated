import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type { LocationQuery, RouteRecordName } from "vue-router"

const libraryModes = ["library", "favorites", "recent", "tags"] as const
const libraryTabs = ["all", "new", "favorites", "top-rated"] as const

const hasOwnKey = <T extends object>(value: T, key: PropertyKey) =>
  Object.prototype.hasOwnProperty.call(value, key)

export const isLibraryMode = (value: unknown): value is LibraryMode =>
  typeof value === "string" && libraryModes.includes(value as LibraryMode)

export const isLibraryRouteName = (
  value: RouteRecordName | null | undefined,
): value is LibraryMode => isLibraryMode(value)

export const getBrowseSourceMode = (query: LocationQuery): LibraryMode =>
  isLibraryMode(query.from) ? query.from : "library"

export const getLibrarySearchQuery = (query: LocationQuery) =>
  typeof query.q === "string" ? query.q : ""

export const getLibraryTabQuery = (query: LocationQuery): LibraryTab => {
  const value = typeof query.tab === "string" ? query.tab : "all"
  return libraryTabs.includes(value as LibraryTab) ? (value as LibraryTab) : "all"
}

export const getSelectedMovieQuery = (query: LocationQuery) =>
  typeof query.selected === "string" ? query.selected : undefined

export const getBrowseContextQuery = (query: LocationQuery) => ({
  from: getBrowseSourceMode(query),
  q: getLibrarySearchQuery(query) || undefined,
  tab: getLibraryTabQuery(query) === "all" ? undefined : getLibraryTabQuery(query),
  selected: getSelectedMovieQuery(query),
})

export const mergeLibraryQuery = (
  sourceQuery: LocationQuery,
  patch: Partial<Record<"q" | "tab" | "selected" | "from", string | undefined>>,
) => {
  const nextQuery: LocationQuery = {
    ...sourceQuery,
  }

  const applyValue = (key: "q" | "tab" | "selected" | "from", value: string | undefined) => {
    if (value) {
      nextQuery[key] = value
      return
    }

    delete nextQuery[key]
  }

  if (hasOwnKey(patch, "q")) {
    applyValue("q", patch.q)
  }

  if (hasOwnKey(patch, "tab")) {
    applyValue("tab", patch.tab && patch.tab !== "all" ? patch.tab : undefined)
  }

  if (hasOwnKey(patch, "selected")) {
    applyValue("selected", patch.selected)
  }

  if (hasOwnKey(patch, "from")) {
    applyValue("from", patch.from)
  }

  return nextQuery
}

export const buildBrowseRouteTarget = (page: LibraryMode, currentQuery: LocationQuery) => ({
  name: page,
  query: mergeLibraryQuery(currentQuery, {
    q: getLibrarySearchQuery(currentQuery) || undefined,
    tab: getLibraryTabQuery(currentQuery),
    selected: getSelectedMovieQuery(currentQuery),
  }),
})

export const buildMovieRouteQuery = (
  currentQuery: LocationQuery,
  sourceMode: LibraryMode,
  selectedMovieId: string,
) =>
  mergeLibraryQuery(currentQuery, {
    from: sourceMode,
    q: getLibrarySearchQuery(currentQuery) || undefined,
    tab: getLibraryTabQuery(currentQuery),
    selected: selectedMovieId,
  })
