import { computed, onBeforeUnmount, ref, shallowRef, watch, type Ref } from "vue"
import type { DesktopPlaybackCapture, DesktopPlaybackSnapshot, DesktopPlayerBridge } from "../../electron/playback-contract"
import type { CaptureJob } from "@/composables/use-curated-capture-queue"
import { normalizeCuratedCaptureKeyCode } from "@/lib/player-shortcuts"
import { disposeCuratedCaptureFeedbackAudio, playCuratedCaptureTriggerCue } from "@/lib/curated-frames/capture-feedback-sound"
import { usePlayerClipCapture } from "@/composables/use-player-clip-capture"

type Receipt = Pick<CaptureJob, "phase" | "positionSec" | "preview" | "error" | "committed"> & { movie: { code: string } }

/** 会话切换后丢弃旧回执；保存失败的重试使用主进程保留的同一帧。 */
export function useNativeFrameCapture(bridge: DesktopPlayerBridge | undefined, snapshot: Ref<DesktopPlaybackSnapshot | undefined>,
  message: (code: string) => string) {
  const busy = ref(false), receipt = ref<Receipt>(), previewOpen = ref(false)
  const keyCode = ref("KeyC"), feedbackSoundEnabled = ref(true)
  const result = shallowRef<DesktopPlaybackCapture>()
  let timer: ReturnType<typeof setTimeout> | undefined
  let disposed = false
  let generation = 0
  let prepared: Promise<DesktopPlaybackCapture | undefined> | undefined
  let pressSession = ""
  let pressed = false
  const available = computed(() => Boolean(bridge && !busy.value && snapshot.value && snapshot.value.clip?.phase !== "processing" && ["playing", "paused"].includes(snapshot.value.state.status)))
  const retryable = computed(() => available.value && result.value?.phase === "error" && receipt.value?.phase === "error")
  const clip = usePlayerClipCapture({ currentTime: computed(() => snapshot.value?.state.positionSec ?? 0),
    duration: computed(() => snapshot.value?.state.durationSec ?? 0), onClipReady: () => commit(true) })
  const recording = clip.isRecording
  const clipStatus = computed(() => snapshot.value?.clip)

  function dismiss() { clearTimeout(timer); receipt.value = undefined; result.value = undefined; previewOpen.value = false }
  function expire() {
    clearTimeout(timer)
    if (receipt.value?.committed && !previewOpen.value) timer = setTimeout(dismiss, 8000)
  }
  watch(() => snapshot.value?.sessionId, () => { invalidatePress(); clip.reset(); dismiss() }, { flush: "sync" })
  watch(previewOpen, expire)
  watch(() => snapshot.value?.clip?.phase, phase => {
    if (phase && phase !== "processing") {
      clip.phase.value = phase === "saved" ? "success" : phase === "error" ? "error" : "idle"
      clearTimeout(timer)
      if (phase !== "error") timer = setTimeout(() => { clip.reset(); dismiss() }, 8000)
    }
  })

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
    if (!bridge || !current || !available.value || pressed || prepared) return
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
      receipt.value = { movie: { code: next.code }, phase: next.phase === "saved" ? "saved" : "error", committed: next.phase === "saved",
        positionSec: next.positionSec, preview: next.preview, error: next.error ? message(next.error) : "" }
      expire()
    } catch (failure) {
      if (disposed || snapshot.value?.sessionId !== current.sessionId) return
      const code = failure instanceof Error ? failure.message.match(/\b[A-Z][A-Z_0-9]{3,}\b/g)?.at(-1) : undefined
      receipt.value = { ...receipt.value!, phase: "error", error: message(code ?? "CAPTURE_FAILED") }
    } finally { busy.value = false }
  }
  function startPress() {
    if (!bridge || !available.value || pressed || prepared || previewOpen.value) return
    clip.reset(); dismiss()
    const current = snapshot.value!
    const epoch = ++generation
    pressSession = current.sessionId; pressed = true
    void playCuratedCaptureTriggerCue(feedbackSoundEnabled.value)
    clip.startPress()
    prepared = bridge.prepareCapture(pressSession).then(async frame => {
      if (disposed || epoch !== generation || snapshot.value?.sessionId !== current.sessionId) {
        await bridge.discardCapture(current.sessionId, frame.id).catch(() => {})
        return undefined
      }
      // 以引擎冻结帧的真实时间修正起点，避免 IPC/编码耗时混入媒体范围。
      clip.startSec.value = frame.positionSec
      return frame
    }).catch(failure => {
      if (epoch === generation && !disposed) {
        clip.cancelPress(); pressed = false; prepared = undefined
        receipt.value = { movie: { code: current.movie?.code ?? "" }, positionSec: current.state.positionSec, preview: "", phase: "error", committed: false,
          error: message(failure instanceof Error ? failure.message.match(/\b[A-Z][A-Z_0-9]{3,}\b/g)?.at(-1) ?? "CAPTURE_FAILED" : "CAPTURE_FAILED") }
      }
      return undefined
    })
  }
  async function commit(gif: boolean) {
    const pending = prepared, epoch = generation, sessionId = pressSession
    if (!pending || !bridge) return
    pressed = false
    busy.value = true
    try {
      const frame = await pending
      if (!frame || epoch !== generation || snapshot.value?.sessionId !== sessionId || disposed) return
      receipt.value = { movie: { code: frame.code }, positionSec: frame.positionSec, preview: frame.preview, phase: "saving", committed: false, error: "" }
      const next = await bridge.commitCapture(sessionId, frame.id, gif)
      if (epoch !== generation || snapshot.value?.sessionId !== sessionId || disposed) return
      result.value = next
      receipt.value = { movie: { code: next.code }, positionSec: next.positionSec, preview: next.preview, phase: next.phase === "saved" ? "saved" : "error",
        committed: next.phase === "saved", error: next.error ? message(next.error) : "" }
      if (!snapshot.value.clip) { clip.reset(); expire() }
    } catch (failure) {
      if (epoch === generation && !disposed) {
        clip.reset()
        receipt.value = { ...receipt.value!, phase: "error", error: message(failure instanceof Error ? failure.message.match(/\b[A-Z][A-Z_0-9]{3,}\b/g)?.at(-1) ?? "CAPTURE_FAILED" : "CAPTURE_FAILED") }
      }
    } finally { if (prepared === pending) prepared = undefined; busy.value = false }
  }
  function finishPress() {
    if (!pressed) return
    const finished = clip.finishPress()
    if (!finished.wasLongPress && !clip.consumeClick()) void commit(false)
  }
  function invalidatePress() {
    const pending = prepared, sessionId = pressSession
    generation++; pressed = false; prepared = undefined
    clip.cancelPress()
    if (pending && bridge) void pending.then(frame => frame && bridge.discardCapture(sessionId, frame.id)).catch(() => {})
  }
  function cancelPress() {
    if (pressed) invalidatePress()
  }
  async function clipAction(action: "cancel" | "retry") {
    const current = snapshot.value
    if (!current || !bridge || busy.value) return
    busy.value = true
    try { await (action === "cancel" ? bridge.cancelClip(current.sessionId) : bridge.retryClip(current.sessionId)) }
    catch { /* 主进程快照保留当前任务结果，切会话的迟到操作不显示。 */ }
    finally { busy.value = false }
  }
  onBeforeUnmount(() => { disposed = true; invalidatePress(); dismiss(); disposeCuratedCaptureFeedbackAudio() })
  return { busy, receipt, previewOpen, keyCode, available, retryable, capture, dismiss, preferences, startPress, finishPress, cancelPress,
    recording, phase: clip.phase, elapsedSec: clip.elapsedSec, progress: clip.progress, clipStatus, clipAction }
}
