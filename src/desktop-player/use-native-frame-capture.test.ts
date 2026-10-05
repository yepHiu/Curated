import { flushPromises, mount } from "@vue/test-utils"
import { defineComponent, h, nextTick, ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import type { DesktopPlaybackCapture, DesktopPlaybackSnapshot, DesktopPlayerBridge } from "../../electron/playback-contract"
import { emptyNativePlayerState } from "../../electron/native-player-contract"
import { useNativeFrameCapture } from "./use-native-frame-capture"
import { playCuratedCaptureTriggerCue } from "@/lib/curated-frames/capture-feedback-sound"

vi.mock("@/lib/curated-frames/capture-feedback-sound", () => ({ playCuratedCaptureTriggerCue: vi.fn(), disposeCuratedCaptureFeedbackAudio: vi.fn() }))
const frame: DesktopPlaybackCapture = { id: "frame-one", movieId: "movie-one", fileId: "part-two", code: "TEST", positionSec: 12.125,
  capturedAt: "2026-10-06T01:02:03.000Z", preview: "data:image/png;base64,fixture", phase: "saved" }
function fixture() {
  const snapshot = ref<DesktopPlaybackSnapshot>({ sessionId: "one", revision: 1, engine: "native", windowOpen: true,
    fullscreen: false, queue: [], autoAdvance: false, locale: "zh-CN", movie: { id: "movie-one", code: "TEST", title: "Fixture", files: [] },
    state: { ...emptyNativePlayerState(), status: "paused", positionSec: 12, durationSec: 120 } })
  const bridge = { capture: vi.fn().mockResolvedValue(frame), capturePreferences: vi.fn().mockResolvedValue({ keyCode: "KeyV", feedbackSoundEnabled: false }),
    prepareCapture: vi.fn().mockResolvedValue({ ...frame, phase: "prepared" }), commitCapture: vi.fn().mockResolvedValue(frame), discardCapture: vi.fn().mockResolvedValue(undefined),
    cancelClip: vi.fn().mockResolvedValue(undefined), retryClip: vi.fn().mockResolvedValue(undefined) }
  let capture!: ReturnType<typeof useNativeFrameCapture>
  const wrapper = mount(defineComponent({ setup() {
    capture = useNativeFrameCapture(bridge as unknown as DesktopPlayerBridge, snapshot, code => "Translated " + code)
    return () => h("div")
  } }))
  return { snapshot, bridge, capture, wrapper }
}
describe("native frame receipt lifecycle", () => {
  it("commits a short press once after preparing the actual starting frame", async () => {
    const f = fixture()
    try {
      f.capture.startPress()
      f.capture.finishPress()
      await flushPromises()
      expect(f.bridge.prepareCapture).toHaveBeenCalledOnce()
      expect(f.bridge.commitCapture).toHaveBeenCalledExactlyOnceWith("one", frame.id, false)
      expect(f.bridge.capture).not.toHaveBeenCalled()
      expect(f.capture.receipt.value).toMatchObject({ positionSec: 12.125, committed: true })
    } finally { f.wrapper.unmount() }
  })
  it("uses a long press for GIF and does not add a second frame on release", async () => {
    vi.useFakeTimers()
    const f = fixture()
    try {
      f.capture.startPress()
      await flushPromises()
      f.snapshot.value.state.positionSec = 14
      await vi.advanceTimersByTimeAsync(500)
      expect(f.capture.recording.value).toBe(true)
      expect(f.capture.elapsedSec.value).toBe(1.875)
      f.capture.finishPress()
      await flushPromises()
      f.capture.finishPress()
      expect(f.bridge.commitCapture).toHaveBeenCalledExactlyOnceWith("one", frame.id, true)
      expect(f.bridge.capture).not.toHaveBeenCalled()
    } finally { f.wrapper.unmount(); vi.useRealTimers() }
  })
  it("automatically submits at six media seconds and treats a paused long press as one still frame", async () => {
    vi.useFakeTimers()
    const f = fixture()
    try {
      f.capture.startPress()
      await flushPromises()
      f.snapshot.value.state.positionSec = 18.125
      await vi.advanceTimersByTimeAsync(600)
      expect(f.bridge.commitCapture).toHaveBeenCalledExactlyOnceWith("one", frame.id, true)
      f.capture.finishPress()
      expect(f.bridge.commitCapture).toHaveBeenCalledOnce()
      f.snapshot.value.state.positionSec = 12.125
      f.capture.startPress()
      await flushPromises()
      await vi.advanceTimersByTimeAsync(1000)
      f.capture.finishPress()
      await flushPromises()
      expect(f.bridge.commitCapture.mock.calls[1]).toEqual(["one", frame.id, false])
    } finally { f.wrapper.unmount(); vi.useRealTimers() }
  })
  it("discards a late prepared frame after blur cancellation or replacing its session", async () => {
    const f = fixture()
    let finish!: (value: DesktopPlaybackCapture) => void
    f.bridge.prepareCapture.mockReturnValueOnce(new Promise(resolve => { finish = resolve }))
    try {
      f.capture.startPress()
      f.capture.cancelPress()
      finish({ ...frame, phase: "prepared" })
      await flushPromises()
      f.capture.finishPress()
      expect(f.bridge.discardCapture).toHaveBeenCalledExactlyOnceWith("one", frame.id)
      expect(f.bridge.commitCapture).not.toHaveBeenCalled()
      f.capture.startPress()
      f.snapshot.value = { ...f.snapshot.value, sessionId: "two" }
      await flushPromises()
      expect(f.capture.receipt.value).toBeUndefined()
      expect(f.bridge.commitCapture).not.toHaveBeenCalled()
    } finally { f.wrapper.unmount() }
  })
  it("allows a fresh press after preparation fails and follows the owned GIF result", async () => {
    const f = fixture()
    try {
      f.bridge.prepareCapture.mockRejectedValueOnce(new Error("CAPTURE_NOT_READY"))
      f.capture.startPress()
      await flushPromises()
      expect(f.capture.receipt.value?.error).toBe("Translated CAPTURE_NOT_READY")
      f.capture.startPress()
      await flushPromises()
      f.capture.finishPress()
      await flushPromises()
      expect(f.bridge.commitCapture).toHaveBeenCalledOnce()
      f.snapshot.value.clip = { frameId: frame.id, startSec: 12, endSec: 14, phase: "processing", progress: 25 }
      await nextTick()
      expect(f.capture.available.value).toBe(false)
      f.capture.startPress()
      expect(f.bridge.prepareCapture).toHaveBeenCalledTimes(2)
      f.snapshot.value.clip.phase = "error"
      await nextTick()
      expect(f.capture.phase.value).toBe("error")
      await f.capture.clipAction("retry")
      expect(f.bridge.retryClip).toHaveBeenCalledExactlyOnceWith("one")
    } finally { f.wrapper.unmount() }
  })
  it("uses actual capture time, retries the same failure ID and honors the current shortcut/sound preferences", async () => {
    const f = fixture()
    try {
      expect(f.capture.retryable.value).toBe(false)
      await f.capture.preferences()
      expect(f.capture.keyCode.value).toBe("KeyV")
      f.bridge.capture.mockResolvedValueOnce({ ...frame, phase: "error", error: "CAPTURE_SAVE_FAILED" })
      await f.capture.capture()
      expect(playCuratedCaptureTriggerCue).toHaveBeenLastCalledWith(false)
      expect(f.capture.receipt.value).toMatchObject({ positionSec: 12.125, phase: "error", error: "Translated CAPTURE_SAVE_FAILED" })
      expect(f.capture.retryable.value).toBe(true)
      f.snapshot.value.state.status = "ended"
      expect(f.capture.retryable.value).toBe(false)
      await f.capture.capture(true)
      expect(f.bridge.capture).toHaveBeenCalledOnce()
      f.snapshot.value.state.status = "paused"
      f.snapshot.value.state.positionSec = 38
      await f.capture.capture(true)
      expect(f.bridge.capture.mock.calls).toEqual([["one", undefined], ["one", "frame-one"]])
      expect(f.capture.receipt.value).toMatchObject({ committed: true, positionSec: 12.125 })
      expect(f.capture.retryable.value).toBe(false)
      f.capture.previewOpen.value = true
      await nextTick()
      f.capture.dismiss()
      expect(f.capture.receipt.value).toBeUndefined()
      expect(f.capture.previewOpen.value).toBe(false)
    } finally { f.wrapper.unmount() }
  })
  it("suppresses duplicate requests and ignores completion from a replaced session", async () => {
    const f = fixture()
    let finish!: (value: DesktopPlaybackCapture) => void
    f.bridge.capture.mockReturnValueOnce(new Promise(resolve => { finish = resolve }))
    try {
      const pending = f.capture.capture()
      expect(f.capture.available.value).toBe(false)
      await f.capture.capture()
      expect(f.bridge.capture).toHaveBeenCalledOnce()
      f.snapshot.value = { ...f.snapshot.value, sessionId: "two" }
      expect(f.capture.receipt.value).toBeUndefined()
      finish(frame)
      await pending
      expect(f.capture.receipt.value).toBeUndefined()
      expect(f.capture.busy.value).toBe(false)
      await f.capture.capture()
      expect(f.bridge.capture).toHaveBeenLastCalledWith("two", undefined)
    } finally { f.wrapper.unmount() }
  })
  it("shows localized pre-capture failure without claiming to have an image to retry", async () => {
    const f = fixture()
    try {
      f.bridge.capture.mockRejectedValueOnce(new Error("Error invoking remote method: CAPTURE_NOT_READY"))
      await f.capture.capture()
      expect(f.capture.receipt.value).toMatchObject({ phase: "error", error: "Translated CAPTURE_NOT_READY" })
      expect(f.capture.receipt.value?.preview).toBe("")
      expect(f.capture.retryable.value).toBe(false)
      f.bridge.capturePreferences.mockResolvedValueOnce({ keyCode: "ArrowLeft", feedbackSoundEnabled: true })
      await f.capture.preferences()
      expect(f.capture.keyCode.value).toBe("KeyC")
    } finally { f.wrapper.unmount() }
  })
})
