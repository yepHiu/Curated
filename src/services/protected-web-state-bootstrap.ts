import { watch, type WatchStopHandle } from "vue"
import type { AuthStatusDTO } from "@/api/types"
import { hydratePlaybackProgress } from "@/lib/playback-progress-storage"
import { hydratePlayedMovies } from "@/lib/played-movies-storage"
import { authLockService, isAuthLockEnabled } from "@/services/auth-lock-service"

interface ProtectedWebStateBootstrapOptions {
  enabled: boolean
  status: { value: AuthStatusDTO }
  refreshStatus: () => Promise<AuthStatusDTO>
  hydratePlaybackProgress: () => Promise<void>
  hydratePlayedMovies: () => Promise<void>
  schedule: (callback: () => void) => void
  onRefreshError?: (error: unknown) => void
}

function scheduleWhenBrowserIdle(callback: () => void) {
  const idleCallback = (window as Window & {
    requestIdleCallback?: (idleCallback: () => void) => number
  }).requestIdleCallback

  if (typeof idleCallback === "function") {
    idleCallback(callback)
    return
  }

  window.setTimeout(callback, 0)
}

export function createProtectedWebStateBootstrap(options: ProtectedWebStateBootstrapOptions) {
  let started = false
  let hydrationQueued = false
  let stopWatching: WatchStopHandle | null = null

  function queueHydration(status: AuthStatusDTO) {
    if (!status.unlocked || hydrationQueued) return
    hydrationQueued = true
    options.schedule(() => {
      void Promise.allSettled([
        options.hydratePlaybackProgress(),
        options.hydratePlayedMovies(),
      ])
    })
  }

  return {
    start() {
      if (!options.enabled || started) return
      started = true
      stopWatching = watch(
        () => options.status.value,
        (status) => queueHydration(status),
      )
      void options.refreshStatus().then(queueHydration).catch((error) => {
        options.onRefreshError?.(error)
      })
    },
    stop() {
      stopWatching?.()
      stopWatching = null
    },
  }
}

export function startProtectedWebStateBootstrap(): () => void {
  const bootstrap = createProtectedWebStateBootstrap({
    enabled: isAuthLockEnabled(),
    status: authLockService.status,
    refreshStatus: () => authLockService.refreshStatus(),
    hydratePlaybackProgress,
    hydratePlayedMovies,
    schedule: scheduleWhenBrowserIdle,
    onRefreshError: (error) => {
      console.warn("[protected-state-bootstrap] auth status check failed", error)
    },
  })
  bootstrap.start()
  return () => bootstrap.stop()
}
