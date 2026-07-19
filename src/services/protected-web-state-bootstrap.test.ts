import { nextTick, ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import type { AuthStatusDTO } from "@/api/types"
import { createProtectedWebStateBootstrap } from "./protected-web-state-bootstrap"

function makeStatus(overrides: Partial<AuthStatusDTO> = {}): AuthStatusDTO {
  return {
    pinEnabled: true,
    unlocked: false,
    setupRequired: false,
    pinLength: 4,
    trustedForever: false,
    sessionTtlMinutes: 60,
    lanRequiresPin: true,
    lockOnRestart: true,
    ...overrides,
  }
}

function makeHarness(initialStatus = makeStatus()) {
  const status = ref(initialStatus)
  const scheduled: Array<() => void> = []
  const hydratePlaybackProgress = vi.fn().mockResolvedValue(undefined)
  const hydratePlayedMovies = vi.fn().mockResolvedValue(undefined)
  const refreshStatus = vi.fn().mockImplementation(async () => status.value)
  const bootstrap = createProtectedWebStateBootstrap({
    enabled: true,
    status,
    refreshStatus,
    hydratePlaybackProgress,
    hydratePlayedMovies,
    schedule: (callback) => scheduled.push(callback),
  })

  return {
    bootstrap,
    hydratePlaybackProgress,
    hydratePlayedMovies,
    refreshStatus,
    scheduled,
    status,
  }
}

describe("protected Web state bootstrap", () => {
  it("does not hydrate protected resources while startup authentication is locked", async () => {
    const harness = makeHarness()

    harness.bootstrap.start()
    await Promise.resolve()

    expect(harness.refreshStatus).toHaveBeenCalledTimes(1)
    expect(harness.scheduled).toHaveLength(0)
    expect(harness.hydratePlaybackProgress).not.toHaveBeenCalled()
    expect(harness.hydratePlayedMovies).not.toHaveBeenCalled()
  })

  it("hydrates both resources once after the session becomes unlocked", async () => {
    const harness = makeHarness()
    harness.bootstrap.start()
    await Promise.resolve()

    harness.status.value = makeStatus({ unlocked: true })
    await nextTick()

    expect(harness.scheduled).toHaveLength(1)
    harness.scheduled[0]?.()
    await Promise.resolve()

    expect(harness.hydratePlaybackProgress).toHaveBeenCalledTimes(1)
    expect(harness.hydratePlayedMovies).toHaveBeenCalledTimes(1)

    harness.status.value = makeStatus({ unlocked: true, trustedForever: true })
    await nextTick()

    expect(harness.scheduled).toHaveLength(1)
  })

  it("hydrates an already unlocked session only after refreshStatus confirms it", async () => {
    const harness = makeHarness(makeStatus({ unlocked: true }))

    harness.bootstrap.start()
    expect(harness.scheduled).toHaveLength(0)
    await Promise.resolve()

    expect(harness.scheduled).toHaveLength(1)
  })

  it("does nothing outside Web API mode", async () => {
    const harness = makeHarness(makeStatus({ unlocked: true }))
    const bootstrap = createProtectedWebStateBootstrap({
      enabled: false,
      status: harness.status,
      refreshStatus: harness.refreshStatus,
      hydratePlaybackProgress: harness.hydratePlaybackProgress,
      hydratePlayedMovies: harness.hydratePlayedMovies,
      schedule: (callback) => harness.scheduled.push(callback),
    })

    bootstrap.start()
    await Promise.resolve()

    expect(harness.refreshStatus).not.toHaveBeenCalled()
    expect(harness.scheduled).toHaveLength(0)
  })
})
