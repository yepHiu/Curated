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
})
