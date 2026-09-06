import { describe, expect, it } from "vitest"
import { parsePresentMoviesContent } from "./agent-movie-cards"

describe("parsePresentMoviesContent", () => {
  it("reads persisted present_movies payloads", () => {
    expect(parsePresentMoviesContent(JSON.stringify({
      movies: [{ movieId: "m1", title: "Hello", reason: "轻松" }],
    }))).toEqual([{ movieId: "m1", title: "Hello", reason: "轻松" }])
  })

  it("ignores tool summaries that are not JSON slates", () => {
    expect(parsePresentMoviesContent("present_movies: ok")).toEqual([])
  })
})
