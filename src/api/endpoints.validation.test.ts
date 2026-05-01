import { afterEach, describe, expect, it, vi } from "vitest"
import { api } from "./endpoints"

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("api endpoint response validation", () => {
  it("keeps valid health responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          name: "curated-dev",
          version: "20260430.120000",
          transport: "http",
          databasePath: "runtime/curated.db",
        }),
      ),
    )

    await expect(api.health()).resolves.toMatchObject({
      name: "curated-dev",
      transport: "http",
    })
  })

  it("rejects malformed health responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          name: "curated-dev",
          transport: "http",
        }),
      ),
    )

    await expect(api.health()).rejects.toThrow("Invalid API response for GET /health")
  })

  it("rejects malformed movie list pages", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          items: { id: "movie-1" },
          total: 1,
          limit: 500,
          offset: 0,
        }),
      ),
    )

    await expect(api.listMovies({ limit: 500, offset: 0 })).rejects.toThrow(
      "Invalid API response for GET /library/movies",
    )
  })

  it("rejects malformed movie details", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          title: "Missing id",
          code: "ABC-123",
        }),
      ),
    )

    await expect(api.getMovie("movie-1")).rejects.toThrow(
      "Invalid API response for GET /library/movies/:id",
    )
  })
})
