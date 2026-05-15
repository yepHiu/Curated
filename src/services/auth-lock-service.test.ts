import { afterEach, describe, expect, it, vi } from "vitest"
import { api } from "@/api/endpoints"
import { authLockService } from "./auth-lock-service"

vi.mock("@/api/endpoints", () => ({
  api: {
    authStatus: vi.fn(),
    setupPin: vi.fn(),
    unlockPin: vi.fn(),
    changePin: vi.fn(),
    lockApp: vi.fn(),
    patchAuthSettings: vi.fn(),
  },
}))

const lockedStatus = {
  pinEnabled: true,
  unlocked: false,
  setupRequired: false,
  pinLength: 4,
  trustedForever: false,
  sessionTtlMinutes: 60,
  lanRequiresPin: true,
  lockOnRestart: true,
}

afterEach(() => {
  vi.clearAllMocks()
})

describe("authLockService", () => {
  it("refreshes and stores auth status", async () => {
    vi.mocked(api.authStatus).mockResolvedValueOnce(lockedStatus)

    await expect(authLockService.refreshStatus()).resolves.toEqual(lockedStatus)

    expect(authLockService.status.value).toEqual(lockedStatus)
  })

  it("passes trustedForever to unlock", async () => {
    const trustedStatus = {
      ...lockedStatus,
      unlocked: true,
      trustedForever: true,
    }
    vi.mocked(api.unlockPin).mockResolvedValueOnce(trustedStatus)

    await expect(authLockService.unlock({ pin: "123456", trustedForever: true })).resolves.toEqual(
      trustedStatus,
    )

    expect(api.unlockPin).toHaveBeenCalledWith({ pin: "123456", trustedForever: true })
    expect(authLockService.status.value.trustedForever).toBe(true)
  })

  it("passes current and new PIN values when changing PIN", async () => {
    const changedStatus = {
      ...lockedStatus,
      unlocked: true,
      pinLength: 5,
    }
    vi.mocked(api.changePin).mockResolvedValueOnce(changedStatus)

    await expect(authLockService.changePin({
      currentPin: "1234",
      newPin: "98765",
      confirmPin: "98765",
    })).resolves.toEqual(changedStatus)

    expect(api.changePin).toHaveBeenCalledWith({
      currentPin: "1234",
      newPin: "98765",
      confirmPin: "98765",
    })
    expect(authLockService.status.value.pinLength).toBe(5)
  })
})
