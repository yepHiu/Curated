import { mount } from "@vue/test-utils"
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
  const bridge = { capture: vi.fn().mockResolvedValue(frame), capturePreferences: vi.fn().mockResolvedValue({ keyCode: "KeyV", feedbackSoundEnabled: false }) }
  let capture!: ReturnType<typeof useNativeFrameCapture>
  const wrapper = mount(defineComponent({ setup() {
    capture = useNativeFrameCapture(bridge as unknown as DesktopPlayerBridge, snapshot, code => "Translated " + code)
    return () => h("div")
  } }))
  return { snapshot, bridge, capture, wrapper }
}
describe("native frame receipt lifecycle", () => {
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
