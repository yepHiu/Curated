import { computed, onBeforeUnmount, ref, shallowRef, watch, type Ref } from "vue"
import type { DesktopPlaybackCapture, DesktopPlaybackSnapshot, DesktopPlayerBridge } from "../../electron/playback-contract"
import type { CaptureJob } from "@/composables/use-curated-capture-queue"
import { normalizeCuratedCaptureKeyCode } from "@/lib/player-shortcuts"
import { disposeCuratedCaptureFeedbackAudio, playCuratedCaptureTriggerCue } from "@/lib/curated-frames/capture-feedback-sound"

type Receipt = Pick<CaptureJob, "phase" | "positionSec" | "preview" | "error" | "committed"> & { movie: { code: string } }

/** 会话切换后丢弃旧回执；保存失败的重试使用主进程保留的同一帧。 */
export function useNativeFrameCapture(bridge: DesktopPlayerBridge | undefined, snapshot: Ref<DesktopPlaybackSnapshot | undefined>,
  message: (code: string) => string) {
  const busy = ref(false), receipt = ref<Receipt>(), previewOpen = ref(false)
  const keyCode = ref("KeyC"), feedbackSoundEnabled = ref(true)
  const result = shallowRef<DesktopPlaybackCapture>()
  let timer: ReturnType<typeof setTimeout> | undefined
  let disposed = false
  const available = computed(() => Boolean(bridge && !busy.value && snapshot.value && ["playing", "paused"].includes(snapshot.value.state.status)))
  const retryable = computed(() => available.value && result.value?.phase === "error" && receipt.value?.phase === "error")

  function dismiss() { clearTimeout(timer); receipt.value = undefined; result.value = undefined; previewOpen.value = false }
  function expire() {
    clearTimeout(timer)
    if (receipt.value?.committed && !previewOpen.value) timer = setTimeout(dismiss, 8000)
  }
  watch(() => snapshot.value?.sessionId, dismiss, { flush: "sync" })
  watch(previewOpen, expire)

  async function preferences() {
    try {
      const value = await bridge?.capturePreferences()
      if (disposed || !value) return
      keyCode.value = normalizeCuratedCaptureKeyCode(value.keyCode)
      feedbackSoundEnabled.value = value.feedbackSoundEnabled
    } catch { /* 当前主页面暂不可用时保留已知偏好。 */ }
  }
  async function capture(retry = false) {
    const current = snapshot.value
    if (!bridge || !current || !available.value) return
    const retryId = retry ? result.value?.id : undefined
    if (retry && !retryable.value) return
    clearTimeout(timer)
    busy.value = true
    receipt.value = { movie: { code: current.movie?.code ?? "" }, phase: "saving", committed: false,
      positionSec: retry ? result.value!.positionSec : current.state.positionSec, preview: retry ? receipt.value?.preview ?? "" : "", error: "" }
    if (!retry) { result.value = undefined; void playCuratedCaptureTriggerCue(feedbackSoundEnabled.value) }
    try {
      const next = await bridge.capture(current.sessionId, retryId)
      if (disposed || snapshot.value?.sessionId !== current.sessionId) return
      result.value = next
      receipt.value = { movie: { code: next.code }, phase: next.phase, committed: next.phase === "saved",
        positionSec: next.positionSec, preview: next.preview, error: next.error ? message(next.error) : "" }
      expire()
    } catch (failure) {
      if (disposed || snapshot.value?.sessionId !== current.sessionId) return
      const code = failure instanceof Error ? failure.message.match(/\b[A-Z][A-Z_0-9]{3,}\b/g)?.at(-1) : undefined
      receipt.value = { ...receipt.value!, phase: "error", error: message(code ?? "CAPTURE_FAILED") }
    } finally { busy.value = false }
  }
  onBeforeUnmount(() => { disposed = true; dismiss(); disposeCuratedCaptureFeedbackAudio() })
  return { busy, receipt, previewOpen, keyCode, available, retryable, capture, dismiss, preferences }
}
