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
    listTrustedAuthSessions: vi.fn(),
    revokeTrustedAuthSession: vi.fn(),
    revokeOtherTrustedAuthSessions: vi.fn(),
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

  it("lists and revokes trusted sessions by safe public id", async () => {
    const session = {
      publicId: "public-session-id",
      userAgent: "Test Browser",
      ip: "127.0.0.1",
      createdAt: "2026-07-19T10:00:00Z",
      lastSeenAt: "2026-07-19T11:00:00Z",
      trustedForever: true,
      current: false,
    }
    vi.mocked(api.listTrustedAuthSessions).mockResolvedValueOnce({ items: [session] })
    vi.mocked(api.revokeTrustedAuthSession).mockResolvedValueOnce({ items: [] })

    await expect(authLockService.listTrustedSessions()).resolves.toEqual([session])
    expect(authLockService.trustedSessions.value).toEqual([session])

    await expect(authLockService.revokeTrustedSession(session.publicId)).resolves.toEqual([])
    expect(api.revokeTrustedAuthSession).toHaveBeenCalledWith("public-session-id")
    expect(authLockService.trustedSessions.value).toEqual([])
  })

  /** 关闭 PIN 成功后清空本地受信任会话列表。 */
  it("clears trusted sessions after PIN is disabled", async () => {
    const session = {
      publicId: "public-session-id",
      userAgent: "Test Browser",
      ip: "127.0.0.1",
      createdAt: "2026-07-19T10:00:00Z",
      lastSeenAt: "2026-07-19T11:00:00Z",
      trustedForever: true,
      current: false,
    }
    vi.mocked(api.listTrustedAuthSessions).mockResolvedValueOnce({ items: [session] })
    await authLockService.listTrustedSessions()

    vi.mocked(api.patchAuthSettings).mockResolvedValueOnce({
      ...lockedStatus,
      pinEnabled: false,
      unlocked: true,
      setupRequired: true,
      pinLength: 0,
    })

    await expect(authLockService.patchSettings({ pinEnabled: false })).resolves.toMatchObject({
      pinEnabled: false,
    })
    expect(api.patchAuthSettings).toHaveBeenCalledWith({ pinEnabled: false })
    expect(authLockService.trustedSessions.value).toEqual([])
  })
})
