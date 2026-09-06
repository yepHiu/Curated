import { describe, expect, it } from "vitest"
import type { RouteLocationNormalizedLoaded } from "vue-router"

import { agentActiveFilters, agentPageContext } from "./agent-page-context"

function route(partial: Partial<RouteLocationNormalizedLoaded>): RouteLocationNormalizedLoaded {
  return {
    name: "library",
    params: {},
    query: {},
    ...partial,
  } as RouteLocationNormalizedLoaded
}

it("projects only the allowlisted library filters into context v1", () => {
  const current = route({
    name: "library",
    query: { q: "hello", tag: "fav", playState: "unwatched", runtime: "short", offset: "50", secret: "no" },
  })
  expect(agentActiveFilters(current)).toEqual({ query: "hello", tag: "fav", playState: "unwatched", runtime: "short" })
  expect(agentPageContext(current)).toEqual({
    contextVersion: 1,
    route: "library",
    query: "hello",
    activeFilters: { query: "hello", tag: "fav", playState: "unwatched", runtime: "short" },
  })
})

describe("agentPageContext", () => {
  it("captures movie and actor from dedicated routes", () => {
    expect(agentPageContext(route({ name: "detail", params: { id: "m1" } }))).toEqual({
      route: "detail",
      movieId: "m1",
    })
    expect(agentPageContext(route({ name: "actor-detail", params: { actorName: "A" } }))).toEqual({
      route: "actor-detail",
      actorName: "A",
    })
  })
})
