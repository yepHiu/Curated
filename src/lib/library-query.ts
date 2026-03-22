import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type { LocationQuery, RouteRecordName } from "vue-router"

const libraryModes = ["library", "favorites", "recent", "tags"] as const
const libraryTabs = ["all", "new", "top-rated"] as const

const hasOwnKey = <T extends object>(value: T, key: PropertyKey) =>
  Object.prototype.hasOwnProperty.call(value, key)

export const isLibraryMode = (value: unknown): value is LibraryMode =>
  typeof value === "string" && libraryModes.includes(value as LibraryMode)

export const isLibraryRouteName = (
  value: RouteRecordName | null | undefined,
): value is LibraryMode => isLibraryMode(value)

export const getBrowseSourceMode = (query: LocationQuery): LibraryMode =>
  isLibraryMode(query.from) ? query.from : "library"

export const getLibrarySearchQuery = (query: LocationQuery): string => {
  const raw = query.q
  if (typeof raw === "string") {
    return raw
  }
  if (Array.isArray(raw)) {
    const first = raw.find((x): x is string => typeof x === "string" && x.trim() !== "")
    return first ?? ""
  }
  return ""
}

/** 精确标签筛选（元数据或用户标签字段完全匹配）；与 `q` 可同时生效（交集） */
export const getLibraryTagExactQuery = (query: LocationQuery) =>
  typeof query.tag === "string" ? query.tag : ""

/** 精确演员筛选（`actors` 数组元素完全匹配）；与 `q`、`tag` 可同时生效（交集） */
export const getLibraryActorExactQuery = (query: LocationQuery): string => {
  const raw = query.actor
  if (typeof raw === "string") {
    return raw
  }
  if (Array.isArray(raw)) {
    const first = raw.find((x): x is string => typeof x === "string" && x.trim() !== "")
    return first ?? ""
  }
  return ""
}

export const getLibraryTabQuery = (query: LocationQuery): LibraryTab => {
  const value = typeof query.tab === "string" ? query.tab : "all"
  return libraryTabs.includes(value as LibraryTab) ? (value as LibraryTab) : "all"
}

export const getSelectedMovieQuery = (query: LocationQuery) =>
  typeof query.selected === "string" ? query.selected : undefined

export const getBrowseContextQuery = (query: LocationQuery) => ({
  from: getBrowseSourceMode(query),
  q: getLibrarySearchQuery(query) || undefined,
  tag: getLibraryTagExactQuery(query).trim() || undefined,
  actor: getLibraryActorExactQuery(query).trim() || undefined,
  tab: getLibraryTabQuery(query) === "all" ? undefined : getLibraryTabQuery(query),
  selected: getSelectedMovieQuery(query),
})

export const mergeLibraryQuery = (
  sourceQuery: LocationQuery,
  patch: Partial<
    Record<"q" | "tab" | "selected" | "from" | "tag" | "actor", string | undefined>
  >,
) => {
  const nextQuery: LocationQuery = {
    ...sourceQuery,
  }

  const applyValue = (
    key: "q" | "tab" | "selected" | "from" | "tag" | "actor",
    value: string | undefined,
  ) => {
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

  if (hasOwnKey(patch, "tag")) {
    applyValue("tag", patch.tag?.trim() || undefined)
  }

  if (hasOwnKey(patch, "actor")) {
    applyValue("actor", patch.actor?.trim() || undefined)
  }

  return nextQuery
}

export const buildBrowseRouteTarget = (page: LibraryMode, currentQuery: LocationQuery) => ({
  name: page,
  query: mergeLibraryQuery(currentQuery, {
    q: getLibrarySearchQuery(currentQuery) || undefined,
    tag: getLibraryTagExactQuery(currentQuery).trim() || undefined,
    actor: getLibraryActorExactQuery(currentQuery).trim() || undefined,
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
    tag: getLibraryTagExactQuery(currentQuery).trim() || undefined,
    actor: getLibraryActorExactQuery(currentQuery).trim() || undefined,
    tab: getLibraryTabQuery(currentQuery),
    selected: selectedMovieId,
  })

/** 萃取帧库专用搜索（与影片库 `q` 隔离） */
export const getCuratedFrameSearchQuery = (query: LocationQuery) =>
  typeof query.cfq === "string" ? query.cfq : ""

export const mergeCuratedFramesQuery = (
  sourceQuery: LocationQuery,
  patch: Partial<{ cfq: string | undefined }>,
) => {
  const nextQuery: LocationQuery = { ...sourceQuery }
  if (hasOwnKey(patch, "cfq")) {
    const t = patch.cfq?.trim()
    if (t) {
      nextQuery.cfq = t
    } else {
      delete nextQuery.cfq
    }
  }
  return nextQuery
}
