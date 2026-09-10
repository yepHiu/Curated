import { afterEach, describe, expect, it, vi } from "vitest"

const comicServiceState = vi.hoisted(() => ({
  enabled: false,
  refreshSettings: vi.fn(),
}))

afterEach(() => {
  vi.resetModules()
  vi.clearAllMocks()
  comicServiceState.enabled = false
})

function mockRouteDependencies() {
  vi.doMock("@/services/auth-lock-service", () => ({
    authLockService: { refreshStatus: vi.fn() },
    isAuthLockEnabled: () => false,
  }))
  vi.doMock("@/services/comic-library-service", () => ({
    useComicLibraryService: () => ({
      comicLibraryEnabled: {
        get value() {
          return comicServiceState.enabled
        },
      },
      refreshSettings: comicServiceState.refreshSettings,
    }),
  }))
  vi.doMock("@/layouts/AppShell.vue", () => ({
    default: { name: "MockAppShell", template: "<div />" },
  }))
  vi.doMock("@/views/ComicsView.vue", () => ({
    default: { name: "MockComicsView", template: "<div />" },
  }))
  vi.doMock("@/views/SettingsView.vue", () => ({
    default: { name: "MockSettingsView", template: "<div />" },
  }))
}

describe("comic library routes", () => {
  it("redirects the comic wall to comic settings when comics are disabled", async () => {
    comicServiceState.enabled = false
    mockRouteDependencies()

    const { default: router } = await import("@/router")
    await router.push("/comics")
    await router.isReady()

    expect(comicServiceState.refreshSettings).toHaveBeenCalled()
    expect(router.currentRoute.value.name).toBe("settings")
    expect(router.currentRoute.value.query.section).toBe("comics")
  })

  it("allows the comic wall route when comics are enabled", async () => {
    comicServiceState.enabled = true
    mockRouteDependencies()

    const { default: router } = await import("@/router")
    await router.push("/comics")
    await router.isReady()

    expect(router.currentRoute.value.name).toBe("comics")
  })
})
