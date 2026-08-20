import { describe, expect, it } from "vitest"
import type { RouteLocationNormalizedLoaded } from "vue-router"

import { agentPageContext } from "./agent-page-context"

function route(partial: Partial<RouteLocationNormalizedLoaded>): RouteLocationNormalizedLoaded {
  return {
    name: "library",
    params: {},
    query: {},
    ...partial,
  } as RouteLocationNormalizedLoaded
}

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
