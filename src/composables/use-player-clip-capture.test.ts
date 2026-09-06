import { beforeEach, afterEach, describe, expect, it, vi } from "vitest"
import { ref } from "vue"
import { usePlayerClipCapture } from "./use-player-clip-capture"

describe("usePlayerClipCapture", () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it("keeps a short press available for single-frame capture", () => {
    const currentTime = ref(12)
    const onClipReady = vi.fn()
    const capture = usePlayerClipCapture({ currentTime, duration: ref(100), onClipReady })

    capture.startPress()
    expect(capture.phase.value).toBe("armed")
    const result = capture.finishPress()

    expect(result.wasLongPress).toBe(false)
    expect(capture.phase.value).toBe("idle")
    expect(onClipReady).not.toHaveBeenCalled()
  })

  it("emits a bounded clip after the long-press threshold", async () => {
    const currentTime = ref(12)
    const onClipReady = vi.fn().mockResolvedValue(undefined)
    const capture = usePlayerClipCapture({
      currentTime,
      duration: ref(100),
      longPressMs: 400,
      maxDurationSec: 6,
      onClipReady,
    })

    capture.startPress()
    vi.advanceTimersByTime(400)
    expect(capture.isRecording.value).toBe(true)
    currentTime.value = 13.2
    vi.advanceTimersByTime(60)
    const result = capture.finishPress()

    expect(result).toMatchObject({ wasLongPress: true, startSec: 12, endSec: 13.2 })
    expect(capture.phase.value).toBe("processing")
    await vi.waitFor(() => expect(onClipReady).toHaveBeenCalledWith({ startSec: 12, endSec: 13.2 }))
  })

  it("does not advance a paused media clock with wall time", () => {
    const currentTime = ref(12)
    const capture = usePlayerClipCapture({
      currentTime,
      duration: ref(100),
      onClipReady: vi.fn(),
    })

    capture.startPress()
    vi.advanceTimersByTime(400)
    vi.advanceTimersByTime(650)

    expect(capture.isRecording.value).toBe(true)
    expect(capture.elapsedSec.value).toBe(0)
  })

  it("falls back to a still when no media elapsed during the hold", () => {
    const currentTime = ref(12)
    const onClipReady = vi.fn().mockResolvedValue(undefined)
    const capture = usePlayerClipCapture({
      currentTime,
      duration: ref(100),
      onClipReady,
    })

    capture.startPress()
    vi.advanceTimersByTime(400)
    vi.advanceTimersByTime(2500)
    const result = capture.finishPress()

    expect(result.wasLongPress).toBe(false)
    expect(onClipReady).not.toHaveBeenCalled()
  })

  it("keeps each clip anchored to the time captured when the press began", () => {
    const currentTime = ref(100)
    const onClipReady = vi.fn().mockResolvedValue(undefined)
    const capture = usePlayerClipCapture({
      currentTime,
      duration: ref(1000),
      longPressMs: 400,
      onClipReady,
    })

    capture.startPress()
    currentTime.value = 100.5
    vi.advanceTimersByTime(400)
    currentTime.value = 105.5
    vi.advanceTimersByTime(100)
    const first = capture.finishPress()

    expect(first).toMatchObject({ wasLongPress: true, startSec: 100, endSec: 105.5 })
    capture.reset()

    currentTime.value = 200
    capture.startPress()
    currentTime.value = 200.5
    vi.advanceTimersByTime(400)
    currentTime.value = 205.5
    vi.advanceTimersByTime(100)
    const second = capture.finishPress()

    expect(second).toMatchObject({ wasLongPress: true, startSec: 200, endSec: 205.5 })
    expect(second.startSec).not.toBe(first.startSec)
  })

  it("automatically ends at the configured maximum duration", () => {
    const currentTime = ref(10)
    const onClipReady = vi.fn().mockResolvedValue(undefined)
    const capture = usePlayerClipCapture({
      currentTime,
      duration: ref(100),
      maxDurationSec: 6,
      onClipReady,
    })

    capture.startPress()
    vi.advanceTimersByTime(400)
    currentTime.value = 16
    vi.advanceTimersByTime(100)

    expect(onClipReady).toHaveBeenCalledWith({ startSec: 10, endSec: 16 })
  })

  it("suppresses the keyup click after automatic maximum-duration completion", () => {
    const currentTime = ref(10)
    const onClipReady = vi.fn().mockResolvedValue(undefined)
    const capture = usePlayerClipCapture({
      currentTime,
      duration: ref(100),
      maxDurationSec: 6,
      onClipReady,
    })

    capture.startPress()
    vi.advanceTimersByTime(400)
    currentTime.value = 16
    vi.advanceTimersByTime(100)

    expect(onClipReady).toHaveBeenCalledTimes(1)
    expect(capture.phase.value).toBe("processing")
    expect(capture.finishPress().wasLongPress).toBe(false)
    expect(capture.consumeClick()).toBe(true)
    expect(capture.consumeClick()).toBe(false)
  })
})
