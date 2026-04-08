import { describe, expect, it } from "vitest"
import { createFrontendMonitorStore } from "./frontend-monitor"

describe("createFrontendMonitorStore", () => {
  it("keeps long tasks and video waiting counts inside the sliding window", () => {
    let now = 10_000
    const monitor = createFrontendMonitorStore({
      now: () => now,
      windowMs: 30_000,
    })

    monitor.recordLongTask(120)
    monitor.recordVideoWaiting()

    now += 12_000
    monitor.recordLongTask(42)
    monitor.recordVideoWaiting()

    now += 20_000
    const snapshot = monitor.getSnapshot()

    expect(snapshot.longTaskCount30s).toBe(1)
    expect(snapshot.video.waitingCount30s).toBe(1)
  })

  it("summarizes route, memory, and video playback quality", () => {
    const monitor = createFrontendMonitorStore()

    monitor.setRouteSnapshot("player", 148)
    monitor.setFPS(58.3)
    monitor.setMemoryUsedMB(256.4)
    monitor.updateVideoQuality({
      totalVideoFrames: 500,
      droppedVideoFrames: 12,
      estimatedFps: 23.98,
    })

    expect(monitor.getSnapshot()).toMatchObject({
      fps: 58.3,
      memoryUsedMB: 256.4,
      routeName: "player",
      lastRouteChangeMs: 148,
      video: {
        available: true,
        totalFrames: 500,
        droppedFrames: 12,
        droppedFrameRatePercent: 2.4,
        estimatedFps: 23.98,
      },
    })
  })

  it("clears transient counts while preserving the most recent route context", () => {
    const monitor = createFrontendMonitorStore()

    monitor.setRouteSnapshot("settings", 96)
    monitor.setFPS(60)
    monitor.setMemoryUsedMB(128)
    monitor.recordLongTask(80)
    monitor.recordVideoWaiting()
    monitor.updateVideoQuality({
      totalVideoFrames: 320,
      droppedVideoFrames: 4,
      estimatedFps: 29.97,
    })

    monitor.clear()

    expect(monitor.getSnapshot()).toMatchObject({
      fps: null,
      memoryUsedMB: null,
      routeName: "settings",
      lastRouteChangeMs: 96,
      longTaskCount30s: 0,
      video: {
        available: false,
        waitingCount30s: 0,
        totalFrames: null,
        droppedFrames: null,
        droppedFrameRatePercent: null,
        estimatedFps: null,
      },
    })
  })
})
