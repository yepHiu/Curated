import type { LocationQueryValue, RouteLocationNormalizedLoaded } from "vue-router"

function normalizeQueryValues(value: LocationQueryValue | LocationQueryValue[]) {
  if (Array.isArray(value)) {
    return value
      .filter((item): item is string => typeof item === "string" && item.trim().length > 0)
      .slice()
      .sort()
  }

  if (typeof value === "string" && value.trim().length > 0) {
    return [value]
  }

  return []
}

export function buildLibraryBrowseScrollKey(
  route: Pick<RouteLocationNormalizedLoaded, "name" | "query">,
) {
  const routeName = typeof route.name === "string" ? route.name : String(route.name ?? "library")
  const params = new URLSearchParams()

  for (const key of Object.keys(route.query).sort()) {
    if (key === "selected") {
      continue
    }

    for (const value of normalizeQueryValues(route.query[key])) {
      params.append(key, value)
    }
  }

  const query = params.toString()
  return query ? `${routeName}?${query}` : routeName
}
