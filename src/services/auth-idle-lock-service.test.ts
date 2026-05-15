import { describe, expect, it, vi } from "vitest"
import { createAuthIdleLockMonitor, type AuthIdleLockTarget } from "./auth-idle-lock-service"
import type { AuthStatusDTO } from "@/api/types"

function makeStatus(overrides: Partial<AuthStatusDTO> = {}): AuthStatusDTO {
  return {
    pinEnabled: true,
    unlocked: true,
    setupRequired: false,
    pinLength: 4,
    trustedForever: false,
    sessionTtlMinutes: 1,
    lanRequiresPin: true,
    lockOnRestart: true,
    sessionExpiresAt: "2026-05-15T10:01:00Z",
    ...overrides,
  }
}

function makeTarget(): AuthIdleLockTarget & { dispatch: (type: string) => void } {
  const listeners = new Map<string, Set<EventListener>>()
  return {
    addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
      const next = listeners.get(type) ?? new Set<EventListener>()
      next.add(listener as EventListener)
      listeners.set(type, next)
    },
    removeEventListener(type: string, listener: EventListenerOrEventListenerObject) {
      listeners.get(type)?.delete(listener as EventListener)
    },
    dispatch(type) {
      for (const listener of listeners.get(type) ?? []) {
        listener(new Event(type))
      }
    },
  }
}

describe("auth idle lock monitor", () => {
  it("refreshes the auth session on user activity before idle expiry", async () => {
    let now = Date.parse("2026-05-15T10:00:00Z")
    const target = makeTarget()
    const timers: Array<{ timeoutMs: number; callback: () => void }> = []
    const service = {
      status: { value: makeStatus() },
      refreshStatus: vi.fn().mockImplementation(async () => {
        service.status.value = makeStatus({ sessionExpiresAt: "2026-05-15T10:01:30Z" })
        return service.status.value
      }),
    }

    createAuthIdleLockMonitor({
      target,
      service,
      now: () => now,
      setTimer: (callback, timeoutMs) => {
        timers.push({ timeoutMs, callback })
        return timers.length
      },
      clearTimer: vi.fn(),
      redirectToLock: vi.fn(),
      minRefreshIntervalMs: 0,
    }).start()

    now = Date.parse("2026-05-15T10:00:30Z")
    target.dispatch("keydown")
    await Promise.resolve()

    expect(service.refreshStatus).toHaveBeenCalledTimes(1)
    expect(timers.at(-1)?.timeoutMs).toBe(60_000)
  })

  it("redirects to lock when the idle deadline has expired", async () => {
    const target = makeTarget()
    const redirectToLock = vi.fn()
    let idleCallback: (() => void) | undefined
    const service = {
      status: { value: makeStatus() },
      refreshStatus: vi.fn().mockResolvedValue(makeStatus({
        unlocked: false,
        sessionExpiresAt: undefined,
      })),
    }

    createAuthIdleLockMonitor({
      target,
      service,
      now: () => Date.parse("2026-05-15T10:00:00Z"),
      setTimer: (callback) => {
        idleCallback = callback
        return 1
      },
      clearTimer: vi.fn(),
      redirectToLock,
      currentPath: () => "/library?actor=Mina",
      minRefreshIntervalMs: 0,
    }).start()

    idleCallback?.()
    await Promise.resolve()

    expect(service.refreshStatus).toHaveBeenCalledTimes(1)
    expect(redirectToLock).toHaveBeenCalledWith("/library?actor=Mina")
  })
})
