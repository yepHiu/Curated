import type { RouteLocationNormalizedLoaded } from "vue-router"
import type { AIChatActiveFiltersDTO, AIChatContextDTO } from "@/api/types"

function routeQueryString(route: RouteLocationNormalizedLoaded, key: string) {
  const value = route.query[key]
  return typeof value === "string" ? value.trim() : ""
}

/** A small allowlist, not a serialization of all URL state. */
export function agentActiveFilters(route: RouteLocationNormalizedLoaded): AIChatActiveFiltersDTO | undefined {
  const query = routeQueryString(route, "q")
  const tag = routeQueryString(route, "tag")
  const actor = routeQueryString(route, "actor")
  const rawPlayState = routeQueryString(route, "playState")
  const rawRuntime = routeQueryString(route, "runtime")
  const filters: AIChatActiveFiltersDTO = {}
  if (query) filters.query = query
  if (tag) filters.tag = tag
  if (actor) filters.actor = actor
  if (rawPlayState === "all" || rawPlayState === "unwatched" || rawPlayState === "in-progress" || rawPlayState === "completed") {
    filters.playState = rawPlayState
  }
  if (rawRuntime === "short" || rawRuntime === "standard" || rawRuntime === "long") {
    filters.runtime = rawRuntime
  }
  return Object.keys(filters).length > 0 ? filters : undefined
}

/** Build optional page context for the experimental agent from the current route. */
export function agentPageContext(route: RouteLocationNormalizedLoaded): AIChatContextDTO | undefined {
  const context: AIChatContextDTO = {}
  const name = typeof route.name === "string" ? route.name : ""
  if (name) {
    context.route = name
  }
  const movieId = typeof route.params.id === "string" ? route.params.id.trim() : ""
  if ((name === "detail" || name === "player") && movieId) {
    context.movieId = movieId
  }
  const actorParam = typeof route.params.actorName === "string" ? route.params.actorName.trim() : ""
  const actorQuery = typeof route.query.actor === "string" ? route.query.actor.trim() : ""
  if (actorParam || actorQuery) {
    context.actorName = actorParam || actorQuery
  }
  const q = routeQueryString(route, "q")
  if (q) {
    context.query = q
  }
  const filters = agentActiveFilters(route)
  if (filters) {
    context.contextVersion = 1
    context.activeFilters = filters
  }
  return Object.keys(context).length > 0 ? context : undefined
}
