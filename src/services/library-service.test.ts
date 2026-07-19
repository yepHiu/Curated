import { beforeEach, describe, expect, it, vi } from "vitest"

const adapterMocks = vi.hoisted(() => ({
  mockLibraryService: { adapter: "mock" },
  startWebLibraryService: vi.fn(),
  webLibraryService: { adapter: "web" },
}))

vi.mock("@/services/adapters/mock/mock-library-service", () => ({
  mockLibraryService: adapterMocks.mockLibraryService,
}))

vi.mock("@/services/adapters/web/web-library-service", () => ({
  startWebLibraryService: adapterMocks.startWebLibraryService,
  webLibraryService: adapterMocks.webLibraryService,
}))

beforeEach(() => {
  vi.resetModules()
  vi.unstubAllEnvs()
  adapterMocks.startWebLibraryService.mockReset()
})

describe("library service adapter bootstrap", () => {
  it("selects Mock without starting the imported Web adapter", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "false")

    const { useLibraryService } = await import("./library-service")

    expect(useLibraryService()).toBe(adapterMocks.mockLibraryService)
    expect(adapterMocks.startWebLibraryService).not.toHaveBeenCalled()
  })

  it("starts and selects the Web adapter only when Web API mode is enabled", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")

    const { useLibraryService } = await import("./library-service")

    expect(adapterMocks.startWebLibraryService).toHaveBeenCalledTimes(1)
    expect(useLibraryService()).toBe(adapterMocks.webLibraryService)
  })
})
