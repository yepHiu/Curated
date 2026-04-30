import { flushPromises } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  getAppUpdateStatus: vi.fn(),
  checkAppUpdateNow: vi.fn(),
}))

vi.mock("@/api/endpoints", () => ({
  api: apiMocks,
}))

async function loadUseAppUpdate() {
  const { useAppUpdate } = await import("./use-app-update")
  return useAppUpdate()
}

beforeEach(() => {
  vi.resetModules()
  vi.unstubAllEnvs()
  vi.useRealTimers()
  apiMocks.getAppUpdateStatus.mockReset()
  apiMocks.checkAppUpdateNow.mockReset()
})

afterEach(() => {
  vi.unstubAllEnvs()
  vi.useRealTimers()
})

describe("useAppUpdate", () => {
  it("reports unsupported when Web API is disabled", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "false")

    const state = await loadUseAppUpdate()

    expect(state.useWebApi).toBe(false)
    expect(state.status.value).toBe("unsupported")
    expect(state.loaded.value).toBe(true)
    expect(state.summary.value?.supported).toBe(false)
    expect(apiMocks.getAppUpdateStatus).not.toHaveBeenCalled()
  })

  it("loads update status on demand", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.getAppUpdateStatus.mockResolvedValueOnce({
      supported: true,
      status: "update-available",
      hasUpdate: true,
      installedVersion: "1.0.0",
      latestVersion: "1.1.0",
    })

    const state = await loadUseAppUpdate()
    state.ensureLoaded()

    expect(state.status.value).toBe("checking")
    expect(state.loading.value).toBe(true)
    await flushPromises()

    expect(apiMocks.getAppUpdateStatus).toHaveBeenCalledTimes(1)
    expect(state.loaded.value).toBe(true)
    expect(state.status.value).toBe("update-available")
    expect(state.hasUpdateBadge.value).toBe(true)
    expect(state.summary.value?.latestVersion).toBe("1.1.0")
  })

  it("runs a manual update check and stores errors", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.checkAppUpdateNow.mockRejectedValueOnce(new Error("release lookup failed"))

    const state = await loadUseAppUpdate()
    const result = await state.checkNow()

    expect(apiMocks.checkAppUpdateNow).toHaveBeenCalledTimes(1)
    expect(result?.status).toBe("error")
    expect(state.status.value).toBe("error")
    expect(state.loaded.value).toBe(true)
    expect(state.errorMessage.value).toBe("release lookup failed")
  })

  it("schedules only one silent auto check for multiple consumers", async () => {
    vi.useFakeTimers()
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.getAppUpdateStatus.mockResolvedValueOnce({
      supported: true,
      status: "up-to-date",
      hasUpdate: false,
      installedVersion: "1.0.0",
      latestVersion: "1.0.0",
    })

    const first = await loadUseAppUpdate()
    const second = await loadUseAppUpdate()
    expect(first.loaded.value).toBe(false)
    expect(second.loaded.value).toBe(false)

    await vi.advanceTimersByTimeAsync(12_000)
    await flushPromises()

    expect(apiMocks.getAppUpdateStatus).toHaveBeenCalledTimes(1)
    expect(first.status.value).toBe("up-to-date")
    expect(second.status.value).toBe("up-to-date")
    expect(first.loading.value).toBe(false)
  })
})
