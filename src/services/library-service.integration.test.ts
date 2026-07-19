import { afterEach, describe, expect, it, vi } from "vitest"

afterEach(() => {
  vi.unstubAllEnvs()
  vi.unstubAllGlobals()
  vi.resetModules()
})

describe("library service Mock runtime", () => {
  it("runs the selected Mock adapter without issuing backend fetches", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "false")
    const fetchMock = vi.fn<typeof fetch>()
    vi.stubGlobal("fetch", fetchMock)

    const { useLibraryService } = await import("./library-service")
    const service = useLibraryService()

    await expect(service.listMoviesForExport()).resolves.toEqual(service.movies.value)
    expect(service.moviesLoaded.value).toBe(true)
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
