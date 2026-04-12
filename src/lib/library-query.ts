import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type { LocationQuery, RouteLocationNormalizedLoaded, RouteRecordName } from "vue-router"

const libraryModes = ["library", "favorites", "recent", "tags", "trash"] as const
const libraryTabs = ["all", "new", "top-rated"] as const
const libraryNavigationTransientKeys = ["from", "browse", "back", "autoplay", "t"] as const

const hasOwnKey = <T extends object>(value: T, key: PropertyKey) =>
  Object.prototype.hasOwnProperty.call(value, key)

function omitLibraryNavigationTransientKeys(sourceQuery: LocationQuery) {
  const nextQuery: LocationQuery = { ...sourceQuery }
  for (const key of libraryNavigationTransientKeys) {
    delete nextQuery[key]
  }
  return nextQuery
}

export const isLibraryMode = (value: unknown): value is LibraryMode =>
  typeof value === "string" && libraryModes.includes(value as LibraryMode)

export const isLibraryRouteName = (
  value: RouteRecordName | null | undefined,
): value is LibraryMode => isLibraryMode(value)

/**
 * 资料库五态（library / favorites / recent / tags / trash）路由解析。
 * 优先用 path 末段：在 route.name 尚未就绪或与 path 短暂不一致时，避免仍把 `name === "library"` 当成主库，
 * 从而回收站误用主库列表 + URL `tab` 子筛选导致网格为空（侧栏回收站计数仍来自 trashed 列表）。
 */
export function resolveLibraryMode(
  route: Pick<RouteLocationNormalizedLoaded, "name" | "path">,
): LibraryMode {
  const parts = route.path.replace(/\/+$/, "").split("/").filter(Boolean)
  const seg = parts.length > 0 ? parts[parts.length - 1]! : ""
  if (isLibraryMode(seg)) {
    return seg
  }
  if (isLibraryRouteName(route.name)) {
    return route.name
  }
  return "library"
}

/** 是否处于资料库五态浏览（与 LibraryView 一致），用于壳层搜索等；比仅用 route.name 更耐短暂未就绪。 */
export function isLibraryBrowseRoute(
  route: Pick<RouteLocationNormalizedLoaded, "name" | "path">,
): boolean {
  if (isLibraryRouteName(route.name)) {
    return true
  }
  const parts = route.path.replace(/\/+$/, "").split("/").filter(Boolean)
  if (parts.length === 0) {
    return false
  }
  return isLibraryMode(parts[parts.length - 1]!)
}

export const getBrowseSourceMode = (query: LocationQuery): LibraryMode => {
  if (isLibraryMode(query.browse)) {
    return query.browse
  }
  return isLibraryMode(query.from) ? query.from : "library"
}

export type DetailBrowseTargetKind = "tag" | "actor" | "studio"

export const getDetailBrowseTargetMode = (
  sourceMode: LibraryMode,
  kind: DetailBrowseTargetKind,
): LibraryMode => {
  if (sourceMode === "tags" && (kind === "actor" || kind === "studio")) {
    return "library"
  }
  return sourceMode
}

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
export const getLibraryTagExactQuery = (query: LocationQuery): string => {
  const raw = query.tag
  if (typeof raw === "string") {
    return raw
  }
  if (Array.isArray(raw)) {
    const first = raw.find((x): x is string => typeof x === "string" && x.trim() !== "")
    return first ?? ""
  }
  return ""
}

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

/** 精确厂商筛选（展示用 `studio`，与用户覆盖一致）；与 `q`、`tag`、`actor` 可同时生效（交集） */
export const getLibraryStudioExactQuery = (query: LocationQuery): string => {
  const raw = query.studio
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

export const getSelectedMovieQuery = (query: LocationQuery) => {
  const raw = query.selected
  if (typeof raw === "string") {
    return raw
  }
  if (Array.isArray(raw)) {
    const first = raw.find((x): x is string => typeof x === "string" && x.trim() !== "")
    return first
  }
  return undefined
}

export const getBrowseContextQuery = (query: LocationQuery) => ({
  browse: getBrowseSourceMode(query),
  q: getLibrarySearchQuery(query) || undefined,
  tag: getLibraryTagExactQuery(query).trim() || undefined,
  actor: getLibraryActorExactQuery(query).trim() || undefined,
  studio: getLibraryStudioExactQuery(query).trim() || undefined,
  tab: getLibraryTabQuery(query) === "all" ? undefined : getLibraryTabQuery(query),
  selected: getSelectedMovieQuery(query),
})

export const mergeLibraryQuery = (
  sourceQuery: LocationQuery,
  patch: Partial<
    Record<"q" | "tab" | "selected" | "from" | "tag" | "actor" | "studio", string | undefined>
  >,
) => {
  const nextQuery: LocationQuery = omitLibraryNavigationTransientKeys(sourceQuery)

  const applyValue = (
    key: "q" | "tab" | "selected" | "from" | "tag" | "actor" | "studio",
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

  if (hasOwnKey(patch, "studio")) {
    applyValue("studio", patch.studio?.trim() || undefined)
  }

  return nextQuery
}

export const buildBrowseRouteTarget = (page: LibraryMode, currentQuery: LocationQuery) => ({
  name: page,
  query: mergeLibraryQuery(currentQuery, {
    q: getLibrarySearchQuery(currentQuery) || undefined,
    tag: getLibraryTagExactQuery(currentQuery).trim() || undefined,
    actor: getLibraryActorExactQuery(currentQuery).trim() || undefined,
    studio: getLibraryStudioExactQuery(currentQuery).trim() || undefined,
    tab: getLibraryTabQuery(currentQuery),
    selected: getSelectedMovieQuery(currentQuery),
  }),
})

export const buildMovieRouteQuery = (
  currentQuery: LocationQuery,
  sourceMode: LibraryMode,
  selectedMovieId: string,
) =>
  ({
    browse: sourceMode,
    ...mergeLibraryQuery(currentQuery, {
      q: getLibrarySearchQuery(currentQuery) || undefined,
      tag: getLibraryTagExactQuery(currentQuery).trim() || undefined,
      actor: getLibraryActorExactQuery(currentQuery).trim() || undefined,
      studio: getLibraryStudioExactQuery(currentQuery).trim() || undefined,
      tab: getLibraryTabQuery(currentQuery),
      selected: selectedMovieId,
    }),
  })

export const buildClearLibraryActorFilterQuery = (
  currentQuery: LocationQuery,
  activeActorName: string,
) => {
  const actorName = activeActorName.trim().toLowerCase()
  const searchQuery = getLibrarySearchQuery(currentQuery).trim()
  const shouldClearSearch = actorName !== "" && searchQuery.toLowerCase() === actorName

  return mergeLibraryQuery(currentQuery, {
    actor: undefined,
    q: shouldClearSearch ? undefined : searchQuery || undefined,
    selected: undefined,
  })
}

/** 萃取帧库专用搜索（与影片库 `q` 隔离） */
export const getCuratedFrameSearchQuery = (query: LocationQuery) =>
  typeof query.cfq === "string" ? query.cfq : ""

/** 萃取帧库专用标签筛选（与 `cfq` 独立） */
export const getCuratedFrameTagQuery = (query: LocationQuery) =>
  typeof query.cft === "string" ? query.cft : ""

export const mergeCuratedFramesQuery = (
  sourceQuery: LocationQuery,
  patch: Partial<{ cfq: string | undefined; cft: string | undefined }>,
) => {
  const nextQuery: LocationQuery = { ...sourceQuery }
  const applyValue = (key: "cfq" | "cft", value: string | undefined) => {
    const t = value?.trim()
    if (t) {
      nextQuery[key] = t
    } else {
      delete nextQuery[key]
    }
  }
  if (hasOwnKey(patch, "cfq")) {
    applyValue("cfq", patch.cfq)
  }
  if (hasOwnKey(patch, "cft")) {
    applyValue("cft", patch.cft)
  }
  return nextQuery
}
