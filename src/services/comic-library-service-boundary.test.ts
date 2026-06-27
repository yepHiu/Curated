import { afterEach, describe, expect, it, vi } from "vitest"
import serviceSelectorSource from "./comic-library-service.ts?raw"
import contractSource from "./contracts/comic-library-service.ts?raw"
import mockAdapterSource from "./adapters/mock/mock-comic-library-service.ts?raw"
import webAdapterSource from "./adapters/web/web-comic-library-service.ts?raw"

afterEach(() => {
  vi.unstubAllEnvs()
  vi.resetModules()
})

describe("comic library service boundary", () => {
  it("uses the web comic adapter when VITE_USE_WEB_API is true", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    vi.resetModules()

    const [{ useComicLibraryService }, { webComicLibraryService }] = await Promise.all([
      import("./comic-library-service"),
      import("./adapters/web/web-comic-library-service"),
    ])

    expect(useComicLibraryService()).toBe(webComicLibraryService)
  })

  it("keeps comic service modules independent from movie services and movie endpoints", () => {
    const sources = [
      serviceSelectorSource,
      contractSource,
      mockAdapterSource,
      webAdapterSource,
    ].join("\n")

    expect(sources).not.toContain("@/services/library-service")
    expect(sources).not.toContain("@/services/adapters/mock/mock-library-service")
    expect(sources).not.toContain("@/services/adapters/web/web-library-service")
    expect(sources).not.toContain("@/api/endpoints")
    expect(sources).not.toContain("@/domain/movie")
  })
})
