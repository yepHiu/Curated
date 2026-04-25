import { ref, watch, type Ref } from "vue"

export type ImmersiveChromeFeedback = {
  kind: "seek" | "curated"
  label: string
  direction?: "backward" | "forward"
}

export function formatImmersiveSeekFeedbackLabel(deltaSec: number): string {
  const rounded = Math.max(0, Math.round(Math.abs(deltaSec)))
  return `${deltaSec < 0 ? "-" : "+"}${rounded}s`
}

export function usePlayerImmersiveChrome(options: {
  hasPlayback: Ref<boolean>
  isPlaying: Ref<boolean>
  hideDelayMs?: number
  feedbackDurationMs?: number
}) {
  const hideDelayMs = options.hideDelayMs ?? 5000
  const feedbackDurationMs = options.feedbackDurationMs ?? 650

  const chromeVisible = ref(true)
  const feedback = ref<ImmersiveChromeFeedback | null>(null)

  let idleHideTimer: number | null = null
  let feedbackTimer: number | null = null

  function clearIdleHideTimer() {
    if (idleHideTimer !== null) {
      clearTimeout(idleHideTimer)
      idleHideTimer = null
    }
  }

  function clearFeedbackTimer() {
    if (feedbackTimer !== null) {
      clearTimeout(feedbackTimer)
      feedbackTimer = null
    }
  }

  function shouldAutoHide(): boolean {
    return options.hasPlayback.value && options.isPlaying.value
  }

  function scheduleChromeIdleHide() {
    clearIdleHideTimer()
    if (!shouldAutoHide()) {
      chromeVisible.value = true
      return
    }
    idleHideTimer = window.setTimeout(() => {
      idleHideTimer = null
      chromeVisible.value = false
    }, hideDelayMs)
  }

  function revealChrome() {
    chromeVisible.value = true
    scheduleChromeIdleHide()
  }

  function onPageMouseMove() {
    revealChrome()
  }

  function showFeedback(nextFeedback: ImmersiveChromeFeedback) {
    clearFeedbackTimer()
    feedback.value = nextFeedback

    feedbackTimer = window.setTimeout(() => {
      feedbackTimer = null
      feedback.value = null
    }, feedbackDurationMs)
  }

  function showSeekFeedback(deltaSec: number) {
    if (!Number.isFinite(deltaSec) || deltaSec === 0) return

    showFeedback({
      kind: "seek",
      direction: deltaSec < 0 ? "backward" : "forward",
      label: formatImmersiveSeekFeedbackLabel(deltaSec),
    })
  }

  function showCuratedFeedback(label: string) {
    const normalized = label.trim()
    if (!normalized) return

    showFeedback({
      kind: "curated",
      label: normalized,
    })
  }

  function dispose() {
    clearIdleHideTimer()
    clearFeedbackTimer()
  }

  watch(
    [options.hasPlayback, options.isPlaying],
    ([hasPlayback, isPlaying]) => {
      if (!hasPlayback) {
        clearIdleHideTimer()
        chromeVisible.value = true
        return
      }
      if (!isPlaying) {
        clearIdleHideTimer()
        return
      }
      scheduleChromeIdleHide()
    },
    { immediate: true },
  )

  return {
    chromeVisible,
    feedback,
    clearIdleHideTimer,
    clearFeedbackTimer,
    onPageMouseMove,
    revealChrome,
    scheduleChromeIdleHide,
    showFeedback,
    showSeekFeedback,
    showCuratedFeedback,
    dispose,
  }
}
