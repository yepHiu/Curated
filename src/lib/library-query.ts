import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type {
  SavedViewCatalog,
  SavedViewFiltersV1,
  SavedViewPlayState,
  SavedViewRuntime,
  SavedViewSort,
} from "@/api/types"
import type { LocationQuery, RouteLocationNormalizedLoaded, RouteRecordName } from "vue-router"
import { librarySortKeyFromTab, librarySortKeys } from "@/lib/movie-sort"

const libraryModes = ["library", "favorites", "recent", "tags", "trash"] as const
const libraryTabs = ["all", "new", "top-rated"] as const
const libraryPlayStates = ["all", "unwatched", "in-progress", "completed"] as const
const libraryRuntimes = ["short", "standard", "long"] as const
const libraryCatalogs = ["unscraped", "no-cover"] as const
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

/** Retired `/tags` browse page: keep reading the mode, but never navigate there. */
export const canonicalLibraryRouteMode = (mode: LibraryMode): LibraryMode =>
  mode === "tags" ? "library" : mode

/**
 * 资料库浏览态（library / favorites / recent / tags / trash）路由解析。
 * `tags` 已退役为 `/library` 的兼容别名。优先用 path 末段：在 route.name 尚未就绪或与 path 短暂不一致时，
 * 避免仍把 `name === "library"` 当成主库，从而回收站误用主库列表 + URL `tab` 子筛选导致网格为空。
 */
export function resolveLibraryMode(
  route: Pick<RouteLocationNormalizedLoaded, "name" | "path">,
): LibraryMode {
  const parts = route.path.replace(/\/+$/, "").split("/").filter(Boolean)
  const seg = parts.length > 0 ? parts[parts.length - 1]! : ""
  if (isLibraryMode(seg)) {
    return canonicalLibraryRouteMode(seg)
  }
  if (isLibraryRouteName(route.name)) {
    return canonicalLibraryRouteMode(route.name)
  }
  return "library"
}

/** 是否处于资料库浏览（与 LibraryView 一致），用于壳层搜索等；比仅用 route.name 更耐短暂未就绪。 */
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
    return canonicalLibraryRouteMode(query.browse)
  }
  return isLibraryMode(query.from) ? canonicalLibraryRouteMode(query.from) : "library"
}

export type DetailBrowseTargetKind = "tag" | "actor" | "studio"

export const getDetailBrowseTargetMode = (
  sourceMode: LibraryMode,
  kind: DetailBrowseTargetKind,
): LibraryMode => {
  void kind
  return canonicalLibraryRouteMode(sourceMode)
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

const libraryTagDelimiter = ","

function uniqueTrimmedTags(values: readonly string[]): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const value of values) {
    const trimmed = value.trim()
    if (!trimmed) continue
    const key = trimmed.toLocaleLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    result.push(trimmed)
  }
  return result
}

function parseDelimitedExactValues(raw: LocationQuery["tag"]): string[] {
  if (typeof raw === "string") {
    return uniqueTrimmedTags(raw.split(libraryTagDelimiter))
  }
  if (Array.isArray(raw)) {
    return uniqueTrimmedTags(
      raw.flatMap((item) => (typeof item === "string" ? item.split(libraryTagDelimiter) : [])),
    )
  }
  return []
}

/** One or more exact library tag filters (metadata or user tags). AND semantics. */
export const getLibraryTagExactFilters = (query: LocationQuery): string[] =>
  parseDelimitedExactValues(query.tag)

export const serializeLibraryTagFilters = (tags: readonly string[]): string | undefined => {
  const normalized = uniqueTrimmedTags(tags)
  return normalized.length > 0 ? normalized.join(libraryTagDelimiter) : undefined
}

export const parseLibraryTagFilterText = (value: string | undefined): string[] =>
  uniqueTrimmedTags((value ?? "").split(libraryTagDelimiter))

/** First active tag; prefer getLibraryTagExactFilters for multi-select. */
export const getLibraryTagExactQuery = (query: LocationQuery): string =>
  getLibraryTagExactFilters(query)[0] ?? ""

/** One or more exact actor filters. AND semantics (must co-star). */
export const getLibraryActorExactFilters = (query: LocationQuery): string[] =>
  parseDelimitedExactValues(query.actor)

/** First active actor; used by the profile card when exactly one actor is selected. */
export const getLibraryActorExactQuery = (query: LocationQuery): string =>
  getLibraryActorExactFilters(query)[0] ?? ""

/** One or more exact studio filters. OR semantics. */
export const getLibraryStudioExactFilters = (query: LocationQuery): string[] =>
  parseDelimitedExactValues(query.studio)

/** First active studio; prefer getLibraryStudioExactFilters for multi-select. */
export const getLibraryStudioExactQuery = (query: LocationQuery): string =>
  getLibraryStudioExactFilters(query)[0] ?? ""

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

export const getLibraryUnratedQuery = (query: LocationQuery): boolean => {
  const raw = typeof query.unrated === "string" ? query.unrated.trim().toLowerCase() : ""
  return raw === "1" || raw === "true"
}

export const normalizeLibraryYearFilter = (value: string): string => {
  const normalized = value.trim().toLowerCase()
  if (!normalized) {
    return ""
  }
  if (normalized === "unknown") {
    return "unknown"
  }
  if (!/^\d{4}$/.test(normalized)) {
    return ""
  }
  const year = Number(normalized)
  return year >= 1800 && year <= 3000 ? normalized : ""
}

export const getLibraryYearQuery = (query: LocationQuery): string =>
  normalizeLibraryYearFilter(typeof query.year === "string" ? query.year : "")

export const normalizeLibraryRuntimeFilter = (value: string): SavedViewRuntime | "" => {
  const normalized = value.trim().toLowerCase()
  return libraryRuntimes.includes(normalized as SavedViewRuntime)
    ? (normalized as SavedViewRuntime)
    : ""
}

export const getLibraryRuntimeQuery = (query: LocationQuery): SavedViewRuntime | "" =>
  normalizeLibraryRuntimeFilter(typeof query.runtime === "string" ? query.runtime : "")

export const normalizeLibraryCatalogFilter = (value: string): SavedViewCatalog | "" => {
  const normalized = value.trim().toLowerCase()
  return libraryCatalogs.includes(normalized as SavedViewCatalog)
    ? (normalized as SavedViewCatalog)
    : ""
}

export const getLibraryCatalogQuery = (query: LocationQuery): SavedViewCatalog | "" =>
  normalizeLibraryCatalogFilter(typeof query.catalog === "string" ? query.catalog : "")

export const normalizeLibrarySortFilter = (value: string): SavedViewSort | "" => {
  const normalized = value.trim().toLowerCase()
  return librarySortKeys.includes(normalized as SavedViewSort) ? (normalized as SavedViewSort) : ""
}

/** Explicit `sort=` wins; otherwise derive from the legacy tab pills. */
export const getLibrarySortQuery = (query: LocationQuery): SavedViewSort => {
  const explicit = normalizeLibrarySortFilter(typeof query.sort === "string" ? query.sort : "")
  if (explicit) {
    return explicit
  }
  return librarySortKeyFromTab(getLibraryTabQuery(query))
}

/**
 * URL / browse helpers only keep `sort` when it is not already expressed by `tab`
 * (added / release / rating). Custom sorts (code / actor / studio / year) stay explicit.
 */
export const getLibrarySortQueryForRoute = (query: LocationQuery): SavedViewSort | undefined => {
  const explicit = normalizeLibrarySortFilter(typeof query.sort === "string" ? query.sort : "")
  if (!explicit || explicit === "added") {
    return undefined
  }
  if (explicit === librarySortKeyFromTab(getLibraryTabQuery(query))) {
    return undefined
  }
  return explicit
}

export const buildSavedViewFiltersV1 = (
  mode: LibraryMode,
  query: LocationQuery,
): SavedViewFiltersV1 => {
  const sort = getLibrarySortQuery(query)
  return {
    schemaVersion: 1,
    mode: canonicalLibraryRouteMode(mode),
    q: getLibrarySearchQuery(query).trim() || undefined,
    tag: serializeLibraryTagFilters(getLibraryTagExactFilters(query)),
    actor: serializeLibraryTagFilters(getLibraryActorExactFilters(query)),
    studio: serializeLibraryTagFilters(getLibraryStudioExactFilters(query)),
    tab: getLibraryTabQuery(query),
    sort: sort === "added" ? undefined : sort,
    playState: getLibraryPlayStateQuery(query),
    userRating: getLibraryUnratedQuery(query) ? undefined : getLibraryUserRatingQuery(query),
    unrated: getLibraryUnratedQuery(query) || undefined,
    resolution: getLibraryResolutionQuery(query) || undefined,
    addedWithinDays: getLibraryAddedWithinDaysQuery(query),
    year: getLibraryYearQuery(query) || undefined,
    runtime: getLibraryRuntimeQuery(query) || undefined,
    catalog: getLibraryCatalogQuery(query) || undefined,
  }
}

export const buildSavedViewRouteTarget = (filters: SavedViewFiltersV1) => {
  const tab =
    filters.tab === "new" || filters.tab === "top-rated" ? filters.tab : ("all" as const)
  const sort = filters.sort && filters.sort !== "added" ? filters.sort : undefined
  const sortForRoute = sort && sort !== librarySortKeyFromTab(tab) ? sort : undefined
  return {
    name: canonicalLibraryRouteMode(filters.mode ?? "library"),
    query: mergeLibraryQuery({}, {
      q: filters.q,
      tag: filters.tag,
      actor: filters.actor,
      studio: filters.studio,
      tab: filters.tab,
      sort: sortForRoute,
      playState: filters.playState,
      userRating:
        filters.unrated || filters.userRating === undefined ? undefined : String(filters.userRating),
      unrated: filters.unrated ? "1" : undefined,
      resolution: normalizeLibraryResolutionFilter(filters.resolution ?? "") || undefined,
      addedWithinDays:
        filters.addedWithinDays === undefined ? undefined : String(filters.addedWithinDays),
      year: normalizeLibraryYearFilter(filters.year ?? "") || undefined,
      runtime: normalizeLibraryRuntimeFilter(filters.runtime ?? "") || undefined,
      catalog: normalizeLibraryCatalogFilter(filters.catalog ?? "") || undefined,
    }),
  }
}

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
  tag: serializeLibraryTagFilters(getLibraryTagExactFilters(query)),
  actor: serializeLibraryTagFilters(getLibraryActorExactFilters(query)),
  studio: serializeLibraryTagFilters(getLibraryStudioExactFilters(query)),
  tab: getLibraryTabQuery(query) === "all" ? undefined : getLibraryTabQuery(query),
  sort: getLibrarySortQueryForRoute(query),
  playState:
    getLibraryPlayStateQuery(query) === "all" ? undefined : getLibraryPlayStateQuery(query),
  userRating:
    getLibraryUnratedQuery(query) || getLibraryUserRatingQuery(query) === undefined
      ? undefined
      : String(getLibraryUserRatingQuery(query)),
  unrated: getLibraryUnratedQuery(query) ? "1" : undefined,
  resolution: getLibraryResolutionQuery(query) || undefined,
  addedWithinDays:
    getLibraryAddedWithinDaysQuery(query) === undefined
      ? undefined
      : String(getLibraryAddedWithinDaysQuery(query)),
  year: getLibraryYearQuery(query) || undefined,
  runtime: getLibraryRuntimeQuery(query) || undefined,
  catalog: getLibraryCatalogQuery(query) || undefined,
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
  | "sort"
  | "playState"
  | "userRating"
  | "unrated"
  | "resolution"
  | "addedWithinDays"
  | "year"
  | "runtime"
  | "catalog"

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

  if (hasOwnKey(patch, "sort")) {
    const sort = normalizeLibrarySortFilter(patch.sort ?? "")
    applyValue("sort", sort && sort !== "added" ? sort : undefined)
  }

  if (hasOwnKey(patch, "selected")) {
    applyValue("selected", patch.selected)
  }

  if (hasOwnKey(patch, "from")) {
    applyValue("from", patch.from)
  }

  if (hasOwnKey(patch, "tag")) {
    applyValue("tag", serializeLibraryTagFilters(parseLibraryTagFilterText(patch.tag)))
  }

  if (hasOwnKey(patch, "actor")) {
    applyValue("actor", serializeLibraryTagFilters(parseLibraryTagFilterText(patch.actor)))
  }

  if (hasOwnKey(patch, "studio")) {
    applyValue("studio", serializeLibraryTagFilters(parseLibraryTagFilterText(patch.studio)))
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

  if (hasOwnKey(patch, "unrated")) {
    applyValue("unrated", patch.unrated === "1" || patch.unrated === "true" ? "1" : undefined)
  }

  if (hasOwnKey(patch, "year")) {
    applyValue("year", normalizeLibraryYearFilter(patch.year ?? "") || undefined)
  }

  if (hasOwnKey(patch, "runtime")) {
    applyValue("runtime", normalizeLibraryRuntimeFilter(patch.runtime ?? "") || undefined)
  }

  if (hasOwnKey(patch, "catalog")) {
    applyValue("catalog", normalizeLibraryCatalogFilter(patch.catalog ?? "") || undefined)
  }

  return nextQuery
}

export const buildBrowseRouteTarget = (page: LibraryMode, currentQuery: LocationQuery) => ({
  name: canonicalLibraryRouteMode(page),
  query: mergeLibraryQuery(currentQuery, {
    q: getLibrarySearchQuery(currentQuery) || undefined,
    tag: serializeLibraryTagFilters(getLibraryTagExactFilters(currentQuery)),
    actor: serializeLibraryTagFilters(getLibraryActorExactFilters(currentQuery)),
    studio: serializeLibraryTagFilters(getLibraryStudioExactFilters(currentQuery)),
    tab: getLibraryTabQuery(currentQuery),
    sort: getLibrarySortQueryForRoute(currentQuery),
    playState: getLibraryPlayStateQuery(currentQuery),
    userRating:
      getLibraryUnratedQuery(currentQuery) || getLibraryUserRatingQuery(currentQuery) === undefined
        ? undefined
        : String(getLibraryUserRatingQuery(currentQuery)),
    unrated: getLibraryUnratedQuery(currentQuery) ? "1" : undefined,
    resolution: getLibraryResolutionQuery(currentQuery) || undefined,
    addedWithinDays:
      getLibraryAddedWithinDaysQuery(currentQuery) === undefined
        ? undefined
        : String(getLibraryAddedWithinDaysQuery(currentQuery)),
    year: getLibraryYearQuery(currentQuery) || undefined,
    runtime: getLibraryRuntimeQuery(currentQuery) || undefined,
    catalog: getLibraryCatalogQuery(currentQuery) || undefined,
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
      tag: serializeLibraryTagFilters(getLibraryTagExactFilters(currentQuery)),
      actor: serializeLibraryTagFilters(getLibraryActorExactFilters(currentQuery)),
      studio: serializeLibraryTagFilters(getLibraryStudioExactFilters(currentQuery)),
      tab: getLibraryTabQuery(currentQuery),
      sort: getLibrarySortQueryForRoute(currentQuery),
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

/** Parse one or more curated-frame tag filters from `cft` (comma-separated or repeated). */
export const getCuratedFrameTagFilters = (query: LocationQuery): string[] =>
  parseDelimitedExactValues(query.cft)

/** @deprecated Prefer getCuratedFrameTagFilters; returns the first active tag for legacy callers. */
export const getCuratedFrameTagQuery = (query: LocationQuery) =>
  getCuratedFrameTagFilters(query)[0] ?? ""

export const serializeCuratedFrameTagFilters = (tags: readonly string[]): string | undefined => {
  const normalized = uniqueTrimmedTags(tags)
  return normalized.length > 0 ? normalized.join(libraryTagDelimiter) : undefined
}

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
