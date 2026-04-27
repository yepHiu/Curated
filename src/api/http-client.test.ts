import { describe, expect, it } from "vitest"
import { resolveApiBaseUrl } from "./http-client"

describe("resolveApiBaseUrl", () => {
  it("defaults to the same-origin API path", () => {
    expect(resolveApiBaseUrl({})).toBe("/api")
  })

  it("keeps an explicit API base URL override", () => {
    expect(resolveApiBaseUrl({ VITE_API_BASE_URL: "http://192.168.1.10:8081/api" })).toBe(
      "http://192.168.1.10:8081/api",
    )
  })
})
