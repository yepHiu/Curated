import { describe, expect, it } from "vitest"
import { createRequestMonitor } from "./request-monitor"

describe("createRequestMonitor", () => {
  it("summarizes recent completed requests within the sliding window", () => {
    let now = 100_000
    const monitor = createRequestMonitor({
      now: () => now,
      windowMs: 30_000,
    })

    const first = monitor.startRequest({ method: "GET", path: "/api/health" })
    now += 120
    monitor.finishRequest(first, { status: 200 })

    now += 5_000
    const second = monitor.startRequest({ method: "POST", path: "/api/library/movies" })
    now += 80
    monitor.finishRequest(second, { status: 503 })

    now += 25_000
    const snapshot = monitor.getSnapshot()

    expect(snapshot.requestCount30s).toBe(1)
    expect(snapshot.failedRequestCount30s).toBe(1)
    expect(snapshot.avgLatencyMs30s).toBe(80)
    expect(snapshot.recentRequests).toHaveLength(2)
    expect(snapshot.recentRequests[0]).toMatchObject({
      method: "POST",
      path: "/api/library/movies",
      status: 503,
      failed: true,
      durationMs: 80,
    })
  })

  it("tracks active requests separately from completed request stats", () => {
    let now = 1_000
    const monitor = createRequestMonitor({
      now: () => now,
      windowMs: 30_000,
    })

    const inFlight = monitor.startRequest({ method: "GET", path: "/api/settings" })

    expect(monitor.getSnapshot().activeRequestCount).toBe(1)
    expect(monitor.getSnapshot().requestCount30s).toBe(0)

    now += 45
    monitor.finishRequest(inFlight, { status: 200 })

    const snapshot = monitor.getSnapshot()
    expect(snapshot.activeRequestCount).toBe(0)
    expect(snapshot.requestCount30s).toBe(1)
    expect(snapshot.avgLatencyMs30s).toBe(45)
  })

  it("limits retained recent requests and clears both history and active state", () => {
    let now = 10_000
    const monitor = createRequestMonitor({
      now: () => now,
      windowMs: 30_000,
      maxRecentRequests: 3,
    })

    for (let index = 0; index < 4; index += 1) {
      const requestId = monitor.startRequest({
        method: "GET",
        path: `/api/items/${index}`,
      })
      now += 25
      monitor.finishRequest(requestId, { status: 200 })
      now += 5
    }

    expect(monitor.getSnapshot().recentRequests.map((item) => item.path)).toEqual([
      "/api/items/3",
      "/api/items/2",
      "/api/items/1",
    ])

    const active = monitor.startRequest({ method: "PATCH", path: "/api/items/3" })
    expect(monitor.getSnapshot().activeRequestCount).toBe(1)

    monitor.clear()

    expect(monitor.getSnapshot()).toMatchObject({
      activeRequestCount: 0,
      requestCount30s: 0,
      failedRequestCount30s: 0,
      avgLatencyMs30s: null,
    })
    expect(monitor.getSnapshot().recentRequests).toEqual([])

    // Clear should also forget dangling request ids rather than throwing later.
    monitor.finishRequest(active, { status: 200 })
    expect(monitor.getSnapshot().activeRequestCount).toBe(0)
  })

  it("ignores new request samples while paused and resumes cleanly", () => {
    const monitor = createRequestMonitor()

    monitor.setPaused(true)
    const ignored = monitor.startRequest({ method: "GET", path: "/api/ignored" })
    monitor.finishRequest(ignored, { status: 200 })

    expect(monitor.getSnapshot()).toMatchObject({
      activeRequestCount: 0,
      requestCount30s: 0,
    })

    monitor.setPaused(false)
    const tracked = monitor.startRequest({ method: "GET", path: "/api/tracked" })
    monitor.finishRequest(tracked, { status: 200 })

    expect(monitor.getSnapshot().requestCount30s).toBe(1)
  })
})
