import { ref, watch, type Ref } from "vue"

export type ImmersiveChromePlaybackAction = "play" | "pause"

export type ImmersiveChromeFeedback =
  | {
      kind: "seek"
      label: string
      direction: "backward" | "forward"
    }
  | {
      kind: "curated"
      label: string
    }
  | {
      kind: "playback"
      label: string
      action: ImmersiveChromePlaybackAction
    }
  | {
      kind: "volume"
      label: string
      volumePercent: number
    }

export function formatImmersiveSeekFeedbackLabel(deltaSec: number): string {
  const rounded = Math.max(0, Math.round(Math.abs(deltaSec)))
  return `${deltaSec < 0 ? "-" : "+"}${rounded}s`
}

function clampImmersiveVolumePercent(percent: number): number {
  if (!Number.isFinite(percent)) return 0
  return Math.max(0, Math.min(100, Math.round(percent)))
}

export function formatImmersiveVolumeFeedbackLabel(percent: number): string {
  return `${clampImmersiveVolumePercent(percent)}%`
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

  function showPlaybackFeedback(action: ImmersiveChromePlaybackAction, label: string) {
    const normalized = label.trim()
    if (!normalized) return

    showFeedback({
      kind: "playback",
      action,
      label: normalized,
    })
  }

  function showVolumeFeedback(percent: number) {
    if (!Number.isFinite(percent)) return

    const volumePercent = clampImmersiveVolumePercent(percent)
    showFeedback({
      kind: "volume",
      label: formatImmersiveVolumeFeedbackLabel(volumePercent),
      volumePercent,
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
    showPlaybackFeedback,
    showVolumeFeedback,
    dispose,
  }
}
