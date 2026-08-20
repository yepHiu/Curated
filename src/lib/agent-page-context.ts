import type { RouteLocationNormalizedLoaded } from "vue-router"
import type { AIChatContextDTO } from "@/api/types"

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
  const q = typeof route.query.q === "string" ? route.query.q.trim() : ""
  if (q) {
    context.query = q
  }
  return Object.keys(context).length > 0 ? context : undefined
}
