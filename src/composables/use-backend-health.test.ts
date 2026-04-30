import { flushPromises, mount } from "@vue/test-utils"
import { defineComponent, nextTick } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import type { useBackendHealth } from "./use-backend-health"

const apiMocks = vi.hoisted(() => ({
  health: vi.fn(),
}))

vi.mock("@/api/endpoints", () => ({
  api: apiMocks,
}))

type BackendHealthResult = ReturnType<typeof useBackendHealth>

async function mountBackendHealth() {
  let state!: BackendHealthResult
  const { useBackendHealth } = await import("./use-backend-health")
  const wrapper = mount(
    defineComponent({
      setup() {
        state = useBackendHealth()
        return () => null
      },
    }),
  )
  return { wrapper, state }
}

beforeEach(() => {
  vi.resetModules()
  vi.unstubAllEnvs()
  vi.useRealTimers()
  apiMocks.health.mockReset()
})

afterEach(() => {
  vi.unstubAllEnvs()
  vi.useRealTimers()
})

describe("useBackendHealth", () => {
  it("stays in mock status when Web API is disabled", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "false")

    const { state } = await mountBackendHealth()

    expect(state.useWebApi).toBe(false)
    expect(state.status.value).toBe("mock")
    expect(apiMocks.health).not.toHaveBeenCalled()
  })

  it("marks the backend online and exposes version display after a successful probe", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.health.mockResolvedValueOnce({
      ok: true,
      version: "1.2.3",
      channel: "dev",
    })

    const { state } = await mountBackendHealth()
    await flushPromises()

    expect(apiMocks.health).toHaveBeenCalledTimes(1)
    expect(state.status.value).toBe("online")
    expect(state.versionDisplay.value).toBe("1.2.3 (dev)")
  })

  it("marks the backend offline after a failed probe", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.health.mockRejectedValueOnce(new Error("offline"))

    const { state } = await mountBackendHealth()
    await flushPromises()

    expect(state.status.value).toBe("offline")
    expect(state.health.value).toBeNull()
  })

  it("polls while mounted and stops polling after unmount", async () => {
    vi.useFakeTimers()
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.health.mockResolvedValue({ ok: true, version: "1.2.3" })

    const { wrapper } = await mountBackendHealth()
    await flushPromises()
    expect(apiMocks.health).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(30_000)
    await flushPromises()
    expect(apiMocks.health).toHaveBeenCalledTimes(2)

    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(30_000)
    await flushPromises()
    expect(apiMocks.health).toHaveBeenCalledTimes(2)
  })

  it("keeps manual recheck spinner visible for the minimum duration", async () => {
    vi.useFakeTimers()
    vi.setSystemTime(0)
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.health.mockResolvedValue({ ok: true, version: "1.2.3" })

    const { state } = await mountBackendHealth()
    await flushPromises()
    apiMocks.health.mockClear()

    state.checkNow()
    await nextTick()
    expect(state.probing.value).toBe(true)
    await flushPromises()
    expect(state.probing.value).toBe(true)

    await vi.advanceTimersByTimeAsync(500)
    await flushPromises()
    expect(state.probing.value).toBe(false)
    expect(apiMocks.health).toHaveBeenCalledTimes(1)
  })
})
