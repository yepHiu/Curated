export interface RequestMonitorStartInput {
  method: string
  path: string
  startedAt?: number
}

export interface RequestMonitorFinishInput {
  status?: number | null
  failed?: boolean
  endedAt?: number
}

export interface RequestMonitorRecord {
  id: string
  method: string
  path: string
  startedAt: number
  endedAt: number
  durationMs: number
  status: number | null
  failed: boolean
}

export interface RequestMonitorSnapshot {
  activeRequestCount: number
  requestCount30s: number
  failedRequestCount30s: number
  avgLatencyMs30s: number | null
  recentRequests: RequestMonitorRecord[]
}

export interface RequestMonitor {
  startRequest(input: RequestMonitorStartInput): string
  finishRequest(requestId: string, input?: RequestMonitorFinishInput): void
  setPaused(paused: boolean): void
  clear(): void
  getSnapshot(): RequestMonitorSnapshot
  subscribe(listener: (snapshot: RequestMonitorSnapshot) => void): () => void
}

type ActiveRequest = {
  id: string
  method: string
  path: string
  startedAt: number
}

type RequestMonitorOptions = {
  now?: () => number
  windowMs?: number
  maxRecentRequests?: number
}

const DEFAULT_WINDOW_MS = 30_000
const DEFAULT_MAX_RECENT_REQUESTS = 50

export function createRequestMonitor(options: RequestMonitorOptions = {}): RequestMonitor {
  const now = options.now ?? (() => Date.now())
  const windowMs = options.windowMs ?? DEFAULT_WINDOW_MS
  const maxRecentRequests = options.maxRecentRequests ?? DEFAULT_MAX_RECENT_REQUESTS

  const activeRequests = new Map<string, ActiveRequest>()
  const recentRequests: RequestMonitorRecord[] = []
  const listeners = new Set<(snapshot: RequestMonitorSnapshot) => void>()
  let requestCounter = 0
  let paused = false

  function buildSnapshot(): RequestMonitorSnapshot {
    const threshold = now() - windowMs
    const windowedRequests = recentRequests.filter((request) => request.endedAt >= threshold)
    const totalDurationMs = windowedRequests.reduce((sum, request) => sum + request.durationMs, 0)

    return {
      activeRequestCount: activeRequests.size,
      requestCount30s: windowedRequests.length,
      failedRequestCount30s: windowedRequests.filter((request) => request.failed).length,
      avgLatencyMs30s:
        windowedRequests.length > 0 ? Math.round(totalDurationMs / windowedRequests.length) : null,
      recentRequests: [...recentRequests],
    }
  }

  function emitSnapshot() {
    const snapshot = buildSnapshot()
    for (const listener of listeners) {
      listener(snapshot)
    }
  }

  return {
    startRequest(input) {
      if (paused) {
        return ""
      }
      const requestId = `dev-request-${++requestCounter}`
      activeRequests.set(requestId, {
        id: requestId,
        method: input.method,
        path: input.path,
        startedAt: input.startedAt ?? now(),
      })
      emitSnapshot()
      return requestId
    },

    finishRequest(requestId, input = {}) {
      if (!requestId) {
        return
      }
      const activeRequest = activeRequests.get(requestId)
      if (!activeRequest) {
        return
      }

      activeRequests.delete(requestId)
      const endedAt = input.endedAt ?? now()
      const status = input.status ?? null
      recentRequests.unshift({
        id: activeRequest.id,
        method: activeRequest.method,
        path: activeRequest.path,
        startedAt: activeRequest.startedAt,
        endedAt,
        durationMs: Math.max(0, Math.round(endedAt - activeRequest.startedAt)),
        status,
        failed: input.failed ?? (status === null ? true : status >= 400),
      })
      if (recentRequests.length > maxRecentRequests) {
        recentRequests.length = maxRecentRequests
      }
      emitSnapshot()
    },

    setPaused(nextPaused) {
      paused = nextPaused
      if (paused) {
        activeRequests.clear()
      }
      emitSnapshot()
    },

    clear() {
      activeRequests.clear()
      recentRequests.length = 0
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

export const devRequestMonitor = createRequestMonitor()
