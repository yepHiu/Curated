import { describe, expect, it } from "vitest"
import { buildDevPerformanceSummaryText } from "./monitor-summary"

describe("buildDevPerformanceSummaryText", () => {
  it("formats key frontend, backend, request, and decode metrics for copying", () => {
    const summary = buildDevPerformanceSummaryText({
      routeName: "player",
      fps: 58.3,
      longTaskCount30s: 2,
      memoryUsedMB: 256.4,
      requestCount30s: 18,
      failedRequestCount30s: 1,
      activeRequestCount: 2,
      avgLatencyMs30s: 84,
      backendHealthStatus: "online",
      backendHealthLatencyMs: 22,
      backendVersion: "20260409.010203 (dev)",
      systemCpuPercent: 18.4,
      backendCpuPercent: 6.1,
      videoAvailable: true,
      videoWaitingCount30s: 1,
      videoDroppedFrames: 12,
      videoTotalFrames: 500,
      videoDroppedFrameRatePercent: 2.4,
      videoEstimatedFps: 23.98,
    })

    expect(summary).toContain("Route: player")
    expect(summary).toContain("Frontend: fps=58.3")
    expect(summary).toContain("Requests: total30s=18, failed30s=1, active=2, avgMs=84")
    expect(summary).toContain("Backend: status=online, latencyMs=22, version=20260409.010203 (dev)")
    expect(summary).toContain("CPU: system=18.4%, backend=6.1%")
    expect(summary).toContain("Decode: fps=23.98, waiting30s=1, dropped=12/500 (2.4%)")
  })
})
