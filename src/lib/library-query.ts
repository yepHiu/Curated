import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type { SavedViewFiltersV1, SavedViewPlayState } from "@/api/types"
import type { LocationQuery, RouteLocationNormalizedLoaded, RouteRecordName } from "vue-router"

const libraryModes = ["library", "favorites", "recent", "tags", "trash"] as const
const libraryTabs = ["all", "new", "top-rated"] as const
const libraryPlayStates = ["all", "unwatched", "in-progress", "completed"] as const
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

export const getLibraryPlayStateQuery = (query: LocationQuery): SavedViewPlayState => {
  const value = typeof query.playState === "string" ? query.playState : "all"
  return libraryPlayStates.includes(value as SavedViewPlayState)
    ? (value as SavedViewPlayState)
    : "all"
}

export const getLibraryUserRatingQuery = (query: LocationQuery): number | undefined => {
  if (typeof query.userRating !== "string" || query.userRating.trim() === "") {
    return undefined
  }
  const value = Number(query.userRating)
  return Number.isFinite(value) && value >= 0 && value <= 5 ? value : undefined
}

export const normalizeLibraryResolutionFilter = (value: string): string => {
  switch (value.trim().toLowerCase()) {
    case "":
      return ""
    case "4k":
    case "2160p":
    case "uhd":
    case "3840x2160":
      return "4k"
    case "1080p":
    case "full hd":
    case "fhd":
      return "1080p"
    case "720p":
    case "hd":
      return "720p"
    case "480p":
    case "sd":
      return "480p"
    default:
      return value.trim().toLowerCase()
  }
}

export const getLibraryResolutionQuery = (query: LocationQuery): string =>
  normalizeLibraryResolutionFilter(typeof query.resolution === "string" ? query.resolution : "")

export const getLibraryAddedWithinDaysQuery = (query: LocationQuery): number | undefined => {
  if (typeof query.addedWithinDays !== "string") {
    return undefined
  }
  const value = Number(query.addedWithinDays)
  return Number.isInteger(value) && value >= 1 && value <= 3650 ? value : undefined
}

export const buildSavedViewFiltersV1 = (
  mode: LibraryMode,
  query: LocationQuery,
): SavedViewFiltersV1 => ({
  schemaVersion: 1,
  mode,
  q: getLibrarySearchQuery(query).trim() || undefined,
  tag: getLibraryTagExactQuery(query).trim() || undefined,
  actor: getLibraryActorExactQuery(query).trim() || undefined,
  studio: getLibraryStudioExactQuery(query).trim() || undefined,
  tab: getLibraryTabQuery(query),
  playState: getLibraryPlayStateQuery(query),
  userRating: getLibraryUserRatingQuery(query),
  resolution: getLibraryResolutionQuery(query) || undefined,
  addedWithinDays: getLibraryAddedWithinDaysQuery(query),
})

export const buildSavedViewRouteTarget = (filters: SavedViewFiltersV1) => ({
  name: filters.mode ?? "library",
  query: mergeLibraryQuery({}, {
    q: filters.q,
    tag: filters.tag,
    actor: filters.actor,
    studio: filters.studio,
    tab: filters.tab,
    playState: filters.playState,
    userRating: filters.userRating === undefined ? undefined : String(filters.userRating),
    resolution: normalizeLibraryResolutionFilter(filters.resolution ?? "") || undefined,
    addedWithinDays:
      filters.addedWithinDays === undefined ? undefined : String(filters.addedWithinDays),
  }),
})

export const getSelectedMovieQuery = (query: LocationQuery) => {
  const raw = query.selected
  if (typeof raw === "string") return raw
  if (Array.isArray(raw)) {
    return raw.find((x): x is string => typeof x === "string" && x.trim() !== "")
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
  playState:
    getLibraryPlayStateQuery(query) === "all" ? undefined : getLibraryPlayStateQuery(query),
  userRating:
    getLibraryUserRatingQuery(query) === undefined
      ? undefined
      : String(getLibraryUserRatingQuery(query)),
  resolution: getLibraryResolutionQuery(query) || undefined,
  addedWithinDays:
    getLibraryAddedWithinDaysQuery(query) === undefined
      ? undefined
      : String(getLibraryAddedWithinDaysQuery(query)),
  selected: getSelectedMovieQuery(query),
})

type LibraryQueryPatchKey =
  | "q"
  | "tab"
  | "selected"
  | "from"
  | "tag"
  | "actor"
  | "studio"
  | "playState"
  | "userRating"
  | "resolution"
  | "addedWithinDays"

export const mergeLibraryQuery = (
  sourceQuery: LocationQuery,
  patch: Partial<
    Record<LibraryQueryPatchKey, string | undefined>
  >,
) => {
  const nextQuery: LocationQuery = omitLibraryNavigationTransientKeys(sourceQuery)

  const applyValue = (
    key: LibraryQueryPatchKey,
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

  if (hasOwnKey(patch, "playState")) {
    applyValue("playState", patch.playState && patch.playState !== "all" ? patch.playState : undefined)
  }

  if (hasOwnKey(patch, "userRating")) {
    applyValue("userRating", patch.userRating)
  }

  if (hasOwnKey(patch, "resolution")) {
    applyValue("resolution", normalizeLibraryResolutionFilter(patch.resolution ?? "") || undefined)
  }

  if (hasOwnKey(patch, "addedWithinDays")) {
    applyValue("addedWithinDays", patch.addedWithinDays)
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
    playState: getLibraryPlayStateQuery(currentQuery),
    userRating:
      getLibraryUserRatingQuery(currentQuery) === undefined
        ? undefined
        : String(getLibraryUserRatingQuery(currentQuery)),
    resolution: getLibraryResolutionQuery(currentQuery) || undefined,
    addedWithinDays:
      getLibraryAddedWithinDaysQuery(currentQuery) === undefined
        ? undefined
        : String(getLibraryAddedWithinDaysQuery(currentQuery)),
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
