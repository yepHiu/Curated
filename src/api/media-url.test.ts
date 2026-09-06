import { describe, expect, it } from "vitest"
import { resolveMediaUrl } from "./media-url"

describe("resolveMediaUrl", () => {
  it("rewrites relative /api assets onto an explicit backend base", () => {
    expect(
      resolveMediaUrl(
        "/api/library/movies/m1/asset/cover",
        { VITE_API_BASE_URL: "http://127.0.0.1:8090/api" },
        "http://127.0.0.1:5173",
      ),
    ).toBe("http://127.0.0.1:8090/api/library/movies/m1/asset/cover")
  })

  it("keeps remote URLs unchanged", () => {
    expect(resolveMediaUrl("https://cdn.example/cover.jpg")).toBe("https://cdn.example/cover.jpg")
  })

  it("keeps same-origin /api paths when the API base is relative", () => {
    expect(resolveMediaUrl("/api/library/movies/m1/asset/cover", {}, "http://127.0.0.1:8081")).toBe(
      "/api/library/movies/m1/asset/cover",
    )
  })
})
