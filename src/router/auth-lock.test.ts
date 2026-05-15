import { afterEach, describe, expect, it, vi } from "vitest"

const lockedStatus = {
  pinEnabled: true,
  unlocked: false,
  setupRequired: false,
  trustedForever: false,
  sessionTtlMinutes: 60,
  lanRequiresPin: true,
  lockOnRestart: true,
}

afterEach(() => {
  vi.resetModules()
  vi.clearAllMocks()
})

describe("router auth lock guard", () => {
  it("redirects locked Web API pages to lock route", async () => {
    const refreshStatus = vi.fn().mockResolvedValue(lockedStatus)
    vi.doMock("@/services/auth-lock-service", () => ({
      authLockService: { refreshStatus },
      isAuthLockEnabled: () => true,
    }))

    const { default: router } = await import("@/router")
    await router.push("/library?actor=Mina")
    await router.isReady()

    expect(refreshStatus).toHaveBeenCalled()
    expect(router.currentRoute.value.name).toBe("lock")
    expect(router.currentRoute.value.query.redirect).toBe("/library?actor=Mina")
  })

  it("allows regular pages when auth status is already unlocked", async () => {
    const refreshStatus = vi.fn().mockResolvedValue({
      ...lockedStatus,
      unlocked: true,
    })
    vi.doMock("@/services/auth-lock-service", () => ({
      authLockService: { refreshStatus },
      isAuthLockEnabled: () => true,
    }))

    const { default: router } = await import("@/router")
    await router.push("/library")
    await router.isReady()

    expect(router.currentRoute.value.name).toBe("library")
  })
})
