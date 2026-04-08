import { computed, onMounted, onUnmounted, ref } from "vue"
import { useRoute, useRouter, type RouteRecordName } from "vue-router"
import { api } from "@/api/endpoints"
import type { DevPerformanceSummaryDTO } from "@/api/types"
import { pushAppToast } from "@/composables/use-app-toast"
import { devFrontendMonitor } from "@/lib/dev-performance/frontend-monitor"
import { buildDevPerformanceSummaryText } from "@/lib/dev-performance/monitor-summary"
import { devRequestMonitor } from "@/lib/dev-performance/request-monitor"

const USE_WEB_API = import.meta.env.VITE_USE_WEB_API === "true"
const BACKEND_POLL_MS = 10_000

export type DevPerformanceBackendStatus = "mock" | "checking" | "online" | "offline"

function resolveRouteName(name: RouteRecordName | null | undefined): string {
  if (typeof name === "string" && name.trim()) {
    return name
  }
  return "unknown"
}

function formatBackendVersion(version?: string, channel?: string): string | null {
  if (!version?.trim()) {
    return null
  }
  return channel?.trim() ? `${version} (${channel})` : version
}

export function useDevPerformanceMonitor() {
  const route = useRoute()
  const router = useRouter()

  const expanded = ref(false)
  const paused = ref(false)
  const frontendSnapshot = ref(devFrontendMonitor.getSnapshot())
  const requestSnapshot = ref(devRequestMonitor.getSnapshot())
  const backendHealthStatus = ref<DevPerformanceBackendStatus>(USE_WEB_API ? "checking" : "mock")
  const backendHealthLatencyMs = ref<number | null>(null)
  const backendVersion = ref<string | null>(null)
  const backendPerformance = ref<DevPerformanceSummaryDTO | null>(null)

  let backendPollTimer: ReturnType<typeof setInterval> | null = null
  let routeNavigationStartedAt: number | null = null
  let detachFrontendSubscription: (() => void) | null = null
  let detachRequestSubscription: (() => void) | null = null
  let detachBeforeEachGuard: (() => void) | null = null
  let detachAfterEachGuard: (() => void) | null = null

  const summaryText = computed(() =>
    buildDevPerformanceSummaryText({
      routeName: frontendSnapshot.value.routeName,
      fps: frontendSnapshot.value.fps,
      longTaskCount30s: frontendSnapshot.value.longTaskCount30s,
      memoryUsedMB: frontendSnapshot.value.memoryUsedMB,
      requestCount30s: requestSnapshot.value.requestCount30s,
      failedRequestCount30s: requestSnapshot.value.failedRequestCount30s,
      activeRequestCount: requestSnapshot.value.activeRequestCount,
      avgLatencyMs30s: requestSnapshot.value.avgLatencyMs30s,
      backendHealthStatus: backendHealthStatus.value,
      backendHealthLatencyMs: backendHealthLatencyMs.value,
      backendVersion: backendVersion.value,
      systemCpuPercent:
        backendPerformance.value?.supported === true
          ? backendPerformance.value.systemCpuPercent ?? null
          : null,
      backendCpuPercent:
        backendPerformance.value?.supported === true
          ? backendPerformance.value.backendCpuPercent ?? null
          : null,
      videoAvailable: frontendSnapshot.value.video.available,
      videoWaitingCount30s: frontendSnapshot.value.video.waitingCount30s,
      videoDroppedFrames: frontendSnapshot.value.video.droppedFrames,
      videoTotalFrames: frontendSnapshot.value.video.totalFrames,
      videoDroppedFrameRatePercent: frontendSnapshot.value.video.droppedFrameRatePercent,
      videoEstimatedFps: frontendSnapshot.value.video.estimatedFps,
    }),
  )

  function syncFrontendRoute(routeName: string, durationMs: number | null) {
    devFrontendMonitor.setRouteSnapshot(routeName, durationMs)
  }

  function stopBackendPolling() {
    if (backendPollTimer !== null) {
      clearInterval(backendPollTimer)
      backendPollTimer = null
    }
  }

  function startBackendPolling() {
    if (!USE_WEB_API || backendPollTimer !== null) {
      return
    }
    backendPollTimer = setInterval(() => {
      void pollBackendState()
    }, BACKEND_POLL_MS)
  }

  async function pollBackendState() {
    if (!USE_WEB_API || paused.value) {
      return
    }

    const healthStartedAt = performance.now()
    try {
      const health = await api.health()
      backendHealthStatus.value = "online"
      backendHealthLatencyMs.value = Math.round(performance.now() - healthStartedAt)
      backendVersion.value = formatBackendVersion(health.version, health.channel)
    } catch {
      backendHealthStatus.value = "offline"
      backendHealthLatencyMs.value = null
      backendVersion.value = null
      backendPerformance.value = { supported: false }
      return
    }

    try {
      backendPerformance.value = await api.getDevPerformanceSummary()
    } catch {
      backendPerformance.value = { supported: false }
    }
  }

  function applyPausedState(nextPaused: boolean) {
    paused.value = nextPaused
    devRequestMonitor.setPaused(nextPaused)

    if (nextPaused) {
      devFrontendMonitor.stop()
      stopBackendPolling()
      return
    }

    devFrontendMonitor.start()
    syncFrontendRoute(resolveRouteName(route.name), null)
    if (USE_WEB_API) {
      startBackendPolling()
      void pollBackendState()
    }
  }

  function toggleExpanded() {
    expanded.value = !expanded.value
  }

  function togglePaused() {
    applyPausedState(!paused.value)
  }

  function clearStats() {
    devFrontendMonitor.clear()
    devRequestMonitor.clear()
    backendPerformance.value = null
    backendHealthLatencyMs.value = null
    syncFrontendRoute(resolveRouteName(route.name), frontendSnapshot.value.lastRouteChangeMs)

    if (USE_WEB_API && !paused.value) {
      void pollBackendState()
    }
  }

  async function copySummary() {
    try {
      await navigator.clipboard.writeText(summaryText.value)
      pushAppToast("Dev performance summary copied.", {
        variant: "success",
        durationMs: 2200,
      })
    } catch {
      pushAppToast("Failed to copy dev performance summary.", {
        variant: "destructive",
      })
    }
  }

  onMounted(() => {
    detachFrontendSubscription = devFrontendMonitor.subscribe((snapshot) => {
      frontendSnapshot.value = snapshot
    })
    detachRequestSubscription = devRequestMonitor.subscribe((snapshot) => {
      requestSnapshot.value = snapshot
    })

    frontendSnapshot.value = devFrontendMonitor.getSnapshot()
    requestSnapshot.value = devRequestMonitor.getSnapshot()
    syncFrontendRoute(resolveRouteName(route.name), null)

    detachBeforeEachGuard = router.beforeEach(() => {
      routeNavigationStartedAt = performance.now()
    })
    detachAfterEachGuard = router.afterEach((to) => {
      const durationMs =
        routeNavigationStartedAt === null
          ? null
          : Math.round(performance.now() - routeNavigationStartedAt)
      routeNavigationStartedAt = null
      syncFrontendRoute(resolveRouteName(to.name), durationMs)
    })

    applyPausedState(false)
  })

  onUnmounted(() => {
    stopBackendPolling()
    devFrontendMonitor.stop()
    devRequestMonitor.setPaused(false)
    detachFrontendSubscription?.()
    detachRequestSubscription?.()
    detachBeforeEachGuard?.()
    detachAfterEachGuard?.()
    detachFrontendSubscription = null
    detachRequestSubscription = null
    detachBeforeEachGuard = null
    detachAfterEachGuard = null
  })

  return {
    useWebApi: USE_WEB_API,
    expanded,
    paused,
    frontendSnapshot,
    requestSnapshot,
    backendHealthStatus,
    backendHealthLatencyMs,
    backendVersion,
    backendPerformance,
    summaryText,
    toggleExpanded,
    togglePaused,
    clearStats,
    copySummary,
  }
}
