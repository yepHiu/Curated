export interface VideoQualitySample {
  totalVideoFrames: number
  droppedVideoFrames: number
  estimatedFps: number | null
}

export interface FrontendVideoSnapshot {
  available: boolean
  totalFrames: number | null
  droppedFrames: number | null
  droppedFrameRatePercent: number | null
  waitingCount30s: number
  estimatedFps: number | null
}

export interface FrontendMonitorSnapshot {
  fps: number | null
  longTaskCount30s: number
  memoryUsedMB: number | null
  routeName: string
  lastRouteChangeMs: number | null
  video: FrontendVideoSnapshot
}

export interface FrontendMonitorStore {
  setFPS(fps: number | null): void
  setMemoryUsedMB(memoryUsedMB: number | null): void
  setRouteSnapshot(routeName: string, lastRouteChangeMs: number | null): void
  recordLongTask(durationMs: number, recordedAt?: number): void
  recordVideoWaiting(recordedAt?: number): void
  updateVideoQuality(sample: VideoQualitySample | null): void
  clear(): void
  getSnapshot(): FrontendMonitorSnapshot
  subscribe(listener: (snapshot: FrontendMonitorSnapshot) => void): () => void
}

export interface FrontendMonitor extends FrontendMonitorStore {
  start(): void
  stop(): void
}

type FrontendMonitorStoreOptions = {
  now?: () => number
  windowMs?: number
}

type FrontendMonitorOptions = FrontendMonitorStoreOptions & {
  performance?: (Performance & { memory?: { usedJSHeapSize: number } }) | null
  window?: Window | null
  document?: Document | null
  queryVideo?: () => HTMLVideoElement | null
  sampleIntervalMs?: number
}

const DEFAULT_WINDOW_MS = 30_000
const DEFAULT_SAMPLE_INTERVAL_MS = 1_000
const MB = 1024 * 1024

function roundTo(value: number, digits: number): number {
  return Number(value.toFixed(digits))
}

function buildEmptyVideoSnapshot(waitingCount30s: number): FrontendVideoSnapshot {
  return {
    available: false,
    totalFrames: null,
    droppedFrames: null,
    droppedFrameRatePercent: null,
    waitingCount30s,
    estimatedFps: null,
  }
}

export function createFrontendMonitorStore(
  options: FrontendMonitorStoreOptions = {},
): FrontendMonitorStore {
  const now = options.now ?? (() => Date.now())
  const windowMs = options.windowMs ?? DEFAULT_WINDOW_MS

  let fps: number | null = null
  let memoryUsedMB: number | null = null
  let routeName = "unknown"
  let lastRouteChangeMs: number | null = null
  let videoQuality: VideoQualitySample | null = null
  const longTaskMarks: number[] = []
  const videoWaitingMarks: number[] = []
  const listeners = new Set<(snapshot: FrontendMonitorSnapshot) => void>()

  function pruneWindow(events: number[]): number[] {
    const threshold = now() - windowMs
    return events.filter((entryAt) => entryAt >= threshold)
  }

  function buildSnapshot(): FrontendMonitorSnapshot {
    const visibleLongTaskMarks = pruneWindow(longTaskMarks)
    const visibleVideoWaitingMarks = pruneWindow(videoWaitingMarks)
    longTaskMarks.length = 0
    longTaskMarks.push(...visibleLongTaskMarks)
    videoWaitingMarks.length = 0
    videoWaitingMarks.push(...visibleVideoWaitingMarks)

    if (!videoQuality) {
      return {
        fps,
        longTaskCount30s: visibleLongTaskMarks.length,
        memoryUsedMB,
        routeName,
        lastRouteChangeMs,
        video: buildEmptyVideoSnapshot(visibleVideoWaitingMarks.length),
      }
    }

    const droppedFrameRatePercent =
      videoQuality.totalVideoFrames > 0
        ? roundTo((videoQuality.droppedVideoFrames / videoQuality.totalVideoFrames) * 100, 2)
        : null

    return {
      fps,
      longTaskCount30s: visibleLongTaskMarks.length,
      memoryUsedMB,
      routeName,
      lastRouteChangeMs,
      video: {
        available: true,
        totalFrames: videoQuality.totalVideoFrames,
        droppedFrames: videoQuality.droppedVideoFrames,
        droppedFrameRatePercent,
        waitingCount30s: visibleVideoWaitingMarks.length,
        estimatedFps: videoQuality.estimatedFps,
      },
    }
  }

  function emitSnapshot() {
    const snapshot = buildSnapshot()
    for (const listener of listeners) {
      listener(snapshot)
    }
  }

  return {
    setFPS(nextFPS) {
      fps = nextFPS
      emitSnapshot()
    },

    setMemoryUsedMB(nextMemoryUsedMB) {
      memoryUsedMB = nextMemoryUsedMB
      emitSnapshot()
    },

    setRouteSnapshot(nextRouteName, nextRouteChangeMs) {
      routeName = nextRouteName
      lastRouteChangeMs = nextRouteChangeMs
      emitSnapshot()
    },

    recordLongTask(_durationMs, recordedAt = now()) {
      longTaskMarks.push(recordedAt)
      emitSnapshot()
    },

    recordVideoWaiting(recordedAt = now()) {
      videoWaitingMarks.push(recordedAt)
      emitSnapshot()
    },

    updateVideoQuality(sample) {
      videoQuality = sample
      emitSnapshot()
    },

    clear() {
      fps = null
      memoryUsedMB = null
      videoQuality = null
      longTaskMarks.length = 0
      videoWaitingMarks.length = 0
      emitSnapshot()
    },

    getSnapshot() {
      return buildSnapshot()
    },

    subscribe(listener) {
      listeners.add(listener)
      return () => {
        listeners.delete(listener)
      }
    },
  }
}

export function createFrontendMonitor(options: FrontendMonitorOptions = {}): FrontendMonitor {
  const now = options.now ?? (() => performance.now())
  const performanceApi = options.performance ?? performance
  const browserWindow = options.window ?? window
  const browserDocument = options.document ?? document
  const queryVideo = options.queryVideo ?? (() => browserDocument.querySelector("video"))
  const sampleIntervalMs = options.sampleIntervalMs ?? DEFAULT_SAMPLE_INTERVAL_MS
  const store = createFrontendMonitorStore(options)

  let running = false
  let fpsWindowStartedAt = 0
  let fpsFrameCount = 0
  let rafId: number | null = null
  let sampleTimer: ReturnType<typeof setInterval> | null = null
  let longTaskObserver: PerformanceObserver | null = null
  let attachedVideo: HTMLVideoElement | null = null
  let detachVideoWaitingListener: (() => void) | null = null
  let lastVideoSample:
    | {
      sampledAt: number
      totalVideoFrames: number
    }
    | null = null

  function syncVideoListener() {
    const nextVideo = queryVideo()
    if (nextVideo === attachedVideo) {
      return
    }

    detachVideoWaitingListener?.()
    detachVideoWaitingListener = null
    attachedVideo = nextVideo
    lastVideoSample = null

    if (!attachedVideo) {
      store.updateVideoQuality(null)
      return
    }

    const onVideoWaiting = () => {
      store.recordVideoWaiting()
    }

    attachedVideo.addEventListener("waiting", onVideoWaiting)
    detachVideoWaitingListener = () => {
      attachedVideo?.removeEventListener("waiting", onVideoWaiting)
    }
  }

  function sampleMemoryUsage() {
    const usedJSHeapSize = (
      performanceApi as (Performance & { memory?: { usedJSHeapSize: number } }) | null
    )?.memory?.usedJSHeapSize
    if (typeof usedJSHeapSize !== "number" || usedJSHeapSize <= 0) {
      store.setMemoryUsedMB(null)
      return
    }
    store.setMemoryUsedMB(roundTo(usedJSHeapSize / MB, 1))
  }

  function sampleVideoQuality() {
    syncVideoListener()
    if (!attachedVideo || typeof attachedVideo.getVideoPlaybackQuality !== "function") {
      store.updateVideoQuality(null)
      return
    }

    const quality = attachedVideo.getVideoPlaybackQuality()
    const sampledAt = now()
    let estimatedFps: number | null = null

    if (lastVideoSample && sampledAt > lastVideoSample.sampledAt) {
      const elapsedMs = sampledAt - lastVideoSample.sampledAt
      const totalFrameDelta = quality.totalVideoFrames - lastVideoSample.totalVideoFrames
      if (elapsedMs > 0 && totalFrameDelta >= 0) {
        estimatedFps = roundTo(totalFrameDelta / (elapsedMs / 1000), 2)
      }
    }

    lastVideoSample = {
      sampledAt,
      totalVideoFrames: quality.totalVideoFrames,
    }
    store.updateVideoQuality({
      totalVideoFrames: quality.totalVideoFrames,
      droppedVideoFrames: quality.droppedVideoFrames,
      estimatedFps,
    })
  }

  function onAnimationFrame(frameAt: number) {
    if (!running) {
      return
    }

    if (fpsWindowStartedAt === 0) {
      fpsWindowStartedAt = frameAt
    }
    fpsFrameCount += 1
    const elapsedMs = frameAt - fpsWindowStartedAt
    if (elapsedMs >= 1000) {
      store.setFPS(roundTo(fpsFrameCount / (elapsedMs / 1000), 1))
      fpsWindowStartedAt = frameAt
      fpsFrameCount = 0
    }

    rafId = browserWindow.requestAnimationFrame(onAnimationFrame)
  }

  function startLongTaskObserver() {
    if (typeof PerformanceObserver === "undefined") {
      return
    }
    try {
      longTaskObserver = new PerformanceObserver((entryList) => {
        for (const entry of entryList.getEntries()) {
          store.recordLongTask(entry.duration)
        }
      })
      longTaskObserver.observe({ entryTypes: ["longtask"] })
    } catch {
      longTaskObserver = null
    }
  }

  return {
    ...store,

    start() {
      if (running) {
        return
      }
      running = true
      fpsWindowStartedAt = 0
      fpsFrameCount = 0

      rafId = browserWindow.requestAnimationFrame(onAnimationFrame)
      startLongTaskObserver()
      sampleMemoryUsage()
      sampleVideoQuality()
      sampleTimer = setInterval(() => {
        sampleMemoryUsage()
        sampleVideoQuality()
      }, sampleIntervalMs)
    },

    stop() {
      running = false
      if (rafId !== null) {
        browserWindow.cancelAnimationFrame(rafId)
        rafId = null
      }
      if (sampleTimer !== null) {
        clearInterval(sampleTimer)
        sampleTimer = null
      }
      if (longTaskObserver) {
        longTaskObserver.disconnect()
        longTaskObserver = null
      }
      detachVideoWaitingListener?.()
      detachVideoWaitingListener = null
      attachedVideo = null
      lastVideoSample = null
    },
  }
}

export const devFrontendMonitor = createFrontendMonitor()
