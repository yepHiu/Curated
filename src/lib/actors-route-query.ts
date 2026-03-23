import type { LocationQuery } from "vue-router"

/** 演员库页专用搜索 query，避免与资料库 `q` 混用 */
export const ACTORS_SEARCH_QUERY_KEY = "actorsQ"

const hasOwnKey = <T extends object>(value: T, key: PropertyKey) =>
  Object.prototype.hasOwnProperty.call(value, key)

export function getActorsSearchQuery(query: LocationQuery): string {
  const raw = query[ACTORS_SEARCH_QUERY_KEY]
  if (typeof raw === "string") {
    return raw
  }
  if (Array.isArray(raw)) {
    const first = raw.find((x): x is string => typeof x === "string" && x.trim() !== "")
    return first ?? ""
  }
  return ""
}

/** 演员用户标签精确筛选（勿与影片 `tag=` 混用） */
export function getActorsTagQuery(query: LocationQuery): string {
  const raw = query.actorTag
  if (typeof raw === "string") {
    return raw
  }
  if (Array.isArray(raw)) {
    const first = raw.find((x): x is string => typeof x === "string" && x.trim() !== "")
    return first ?? ""
  }
  return ""
}

export function mergeActorsQuery(
  sourceQuery: LocationQuery,
  patch: Partial<Record<typeof ACTORS_SEARCH_QUERY_KEY | "actorTag", string | undefined>>,
) {
  const nextQuery: LocationQuery = { ...sourceQuery }

  const apply = (key: typeof ACTORS_SEARCH_QUERY_KEY | "actorTag", value: string | undefined) => {
    if (value && value.trim() !== "") {
      nextQuery[key] = value.trim()
    } else {
      delete nextQuery[key]
    }
  }

  if (hasOwnKey(patch, ACTORS_SEARCH_QUERY_KEY)) {
    apply(ACTORS_SEARCH_QUERY_KEY, patch[ACTORS_SEARCH_QUERY_KEY])
  }
  if (hasOwnKey(patch, "actorTag")) {
    apply("actorTag", patch.actorTag)
  }

  return nextQuery
}
