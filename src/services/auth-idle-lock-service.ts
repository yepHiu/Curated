import type { AuthStatusDTO } from "@/api/types"
import { authLockService, isAuthLockEnabled } from "@/services/auth-lock-service"

export interface AuthIdleLockTarget {
  addEventListener: Window["addEventListener"]
  removeEventListener: Window["removeEventListener"]
}

interface AuthIdleLockService {
  status: { value: AuthStatusDTO }
  refreshStatus: () => Promise<AuthStatusDTO>
}

interface AuthIdleLockMonitorOptions {
  target: AuthIdleLockTarget
  service: AuthIdleLockService
  now?: () => number
  setTimer?: (callback: () => void, timeoutMs: number) => number
  clearTimer?: (handle: number) => void
  redirectToLock: (redirectPath: string) => void
  currentPath?: () => string
  minRefreshIntervalMs?: number
}

interface AuthIdleLockRouter {
  currentRoute: {
    value: {
      name?: unknown
      fullPath: string
    }
  }
  replace: (location: { name: string; query?: Record<string, string> }) => unknown
}

const activityEvents = [
  "pointerdown",
  "keydown",
  "wheel",
  "touchstart",
  "focus",
] as const

const defaultMinRefreshIntervalMs = 30_000

function shouldTrackIdle(status: AuthStatusDTO): boolean {
  return status.pinEnabled && status.unlocked && !status.trustedForever
}

function idleDeadlineMs(status: AuthStatusDTO): number | null {
  if (!status.sessionExpiresAt) return null
  const value = Date.parse(status.sessionExpiresAt)
  return Number.isFinite(value) ? value : null
}

export function createAuthIdleLockMonitor(options: AuthIdleLockMonitorOptions) {
  const now = options.now ?? (() => Date.now())
  const setTimer = options.setTimer ?? ((callback, timeoutMs) => window.setTimeout(callback, timeoutMs))
  const clearTimer = options.clearTimer ?? ((handle) => window.clearTimeout(handle))
  const currentPath = options.currentPath ?? (() => "/")
  const minRefreshIntervalMs = options.minRefreshIntervalMs ?? defaultMinRefreshIntervalMs
  let timerHandle: number | null = null
  let lastRefreshAt = 0
  let stopped = true

  function clearIdleTimer() {
    if (timerHandle !== null) {
      clearTimer(timerHandle)
      timerHandle = null
    }
  }

  function redirectIfLocked(status: AuthStatusDTO): boolean {
    if (status.pinEnabled && !status.unlocked) {
      options.redirectToLock(currentPath())
      return true
    }
    return false
  }

  function schedule(status: AuthStatusDTO = options.service.status.value) {
    clearIdleTimer()
    if (stopped || !shouldTrackIdle(status)) return
    const deadline = idleDeadlineMs(status)
    if (deadline === null) return
    const delay = Math.max(0, deadline - now())
    timerHandle = setTimer(() => {
      void refreshAndReschedule()
    }, delay)
  }

  async function refreshAndReschedule() {
    if (stopped) return
    const status = await options.service.refreshStatus()
    if (redirectIfLocked(status)) return
    schedule(status)
  }

  async function onActivity() {
    if (stopped) return
    const status = options.service.status.value
    if (!shouldTrackIdle(status)) {
      schedule(status)
      return
    }
    const nowMs = now()
    if (nowMs - lastRefreshAt < minRefreshIntervalMs) {
      schedule(status)
      return
    }
    lastRefreshAt = nowMs
    await refreshAndReschedule()
  }

  const listener = () => {
    void onActivity()
  }

  return {
    start() {
      if (!stopped) return this
      stopped = false
      for (const eventName of activityEvents) {
        options.target.addEventListener(eventName, listener, { passive: true })
      }
      schedule()
      return this
    },
    stop() {
      if (stopped) return
      stopped = true
      clearIdleTimer()
      for (const eventName of activityEvents) {
        options.target.removeEventListener(eventName, listener)
      }
    },
  }
}

export function startAuthIdleLockMonitor(router: AuthIdleLockRouter): () => void {
  if (!isAuthLockEnabled()) return () => {}
  const monitor = createAuthIdleLockMonitor({
    target: window,
    service: authLockService,
    redirectToLock: (redirectPath) => {
      if (router.currentRoute.value.name === "lock") return
      void router.replace({
        name: "lock",
        query: {
          redirect: redirectPath || "/",
        },
      })
    },
    currentPath: () => router.currentRoute.value.fullPath,
  })
  monitor.start()
  return () => monitor.stop()
}
