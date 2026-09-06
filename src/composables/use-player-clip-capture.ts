import { computed, onBeforeUnmount, ref, type Ref } from "vue"

export type ClipCapturePhase = "idle" | "armed" | "recording" | "processing" | "success" | "error"

export interface PlayerClipCaptureOptions {
  currentTime: Ref<number>
  duration: Ref<number>
  minDurationSec?: number
  maxDurationSec?: number
  longPressMs?: number
  mediaTime?: () => number
  onClipReady: (input: { startSec: number; endSec: number }) => void | Promise<void>
}

/**
 * Shared short-press/long-press state machine for the Curated player button.
 * A short press is intentionally left to the caller (single-frame capture); a
 * long press emits one bounded clip range when the pointer/key is released.
 */
export function usePlayerClipCapture(options: PlayerClipCaptureOptions) {
  const phase = ref<ClipCapturePhase>("idle")
  const startSec = ref<number | null>(null)
  const endSec = ref<number | null>(null)
  const elapsedSec = ref(0)
  const error = ref("")
  const taskId = ref<string | null>(null)
  const suppressNextClick = ref(false)

  const minDurationSec = options.minDurationSec ?? 0.4
  const maxDurationSec = options.maxDurationSec ?? 6
  const longPressMs = options.longPressMs ?? 400
  let thresholdTimer: number | null = null
  let ticker: number | null = null
  let pointerActive = false
  const recordingTickMs = 100
  const mediaTime = () => options.mediaTime?.() ?? options.currentTime.value

  const isRecording = computed(() => phase.value === "recording")
  const progress = computed(() => Math.min(1, elapsedSec.value / maxDurationSec))

  function clearTimers() {
    if (thresholdTimer !== null) window.clearTimeout(thresholdTimer)
    if (ticker !== null) window.clearInterval(ticker)
    thresholdTimer = null
    ticker = null
  }

  function reset() {
    clearTimers()
    phase.value = "idle"
    startSec.value = null
    endSec.value = null
    elapsedSec.value = 0
    error.value = ""
    taskId.value = null
    pointerActive = false
  }

  function startPress() {
    if (phase.value !== "idle") return
    suppressNextClick.value = false
    const now = Number.isFinite(mediaTime()) ? Math.max(0, mediaTime()) : 0
    startSec.value = now
    endSec.value = null
    elapsedSec.value = 0
    error.value = ""
    pointerActive = true
    phase.value = "armed"
    thresholdTimer = window.setTimeout(() => {
      thresholdTimer = null
      if (!pointerActive || phase.value !== "armed") return
      // Keep the timestamp captured on press. Re-reading the media clock here
      // can pick up a stale seek/timeupdate value and make consecutive clips
      // start from the previous recording's position.
      phase.value = "recording"
      ticker = window.setInterval(() => {
        const mediaNow = Number.isFinite(mediaTime()) ? Math.max(0, mediaTime()) : now
        const mediaElapsed = Math.max(0, mediaNow - (startSec.value ?? now))
        elapsedSec.value = mediaElapsed
        if (elapsedSec.value >= maxDurationSec) finishPress()
      }, recordingTickMs)
    }, longPressMs)
  }

  function finishPress(): { wasLongPress: boolean; startSec?: number; endSec?: number } {
    if (phase.value !== "armed" && phase.value !== "recording") {
      return { wasLongPress: false }
    }
    const wasLongPress = phase.value === "recording"
    pointerActive = false
    clearTimers()
    if (!wasLongPress) {
      reset()
      return { wasLongPress: false }
    }
    const start = startSec.value ?? options.currentTime.value
    // Read the live media clock at release: pauses/buffering must not create
    // unseen footage and playback rate must not be confused with wall time.
    const end = Math.min(Math.max(start, mediaTime()), start + maxDurationSec, options.duration.value || Infinity)
    if (end - start < minDurationSec) {
      reset()
      return { wasLongPress: false }
    }
    startSec.value = start
    endSec.value = end
    elapsedSec.value = Math.max(0, end - start)
    suppressNextClick.value = true
    phase.value = "processing"
    void Promise.resolve(options.onClipReady({ startSec: start, endSec: end })).catch((err: unknown) => {
      error.value = err instanceof Error ? err.message : String(err)
      phase.value = "error"
    })
    return { wasLongPress: true, startSec: start, endSec: end }
  }

  function cancelPress() {
    if (phase.value === "recording" || phase.value === "armed") reset()
  }

  function consumeClick(): boolean {
    if (!suppressNextClick.value) return false
    suppressNextClick.value = false
    return true
  }

  onBeforeUnmount(() => {
    clearTimers()
  })

  return {
    phase,
    startSec,
    endSec,
    elapsedSec,
    error,
    taskId,
    isRecording,
    progress,
    startPress,
    finishPress,
    cancelPress,
    consumeClick,
    reset,
  }
}
