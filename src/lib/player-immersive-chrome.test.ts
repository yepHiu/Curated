import { describe, expect, it, vi } from "vitest"
import { ref } from "vue"
import {
  formatImmersiveSeekFeedbackLabel,
  usePlayerImmersiveChrome,
} from "@/lib/player-immersive-chrome"

describe("player immersive chrome", () => {
  it("formats compact seek feedback labels", () => {
    expect(formatImmersiveSeekFeedbackLabel(10)).toBe("+10s")
    expect(formatImmersiveSeekFeedbackLabel(-7)).toBe("-7s")
    expect(formatImmersiveSeekFeedbackLabel(0)).toBe("+0s")
  })

  it("hides chrome after page idle while playback is active", () => {
    vi.useFakeTimers()

    const hasPlayback = ref(true)
    const isPlaying = ref(true)
    const immersiveChrome = usePlayerImmersiveChrome({
      hasPlayback,
      isPlaying,
      hideDelayMs: 5000,
      feedbackDurationMs: 650,
    })

    immersiveChrome.onPageMouseMove()
    expect(immersiveChrome.chromeVisible.value).toBe(true)

    vi.advanceTimersByTime(4999)
    expect(immersiveChrome.chromeVisible.value).toBe(true)

    vi.advanceTimersByTime(1)
    expect(immersiveChrome.chromeVisible.value).toBe(false)

    immersiveChrome.dispose()
    vi.useRealTimers()
  })

  it("keeps chrome hidden while showing transient seek feedback", () => {
    vi.useFakeTimers()

    const hasPlayback = ref(true)
    const isPlaying = ref(true)
    const immersiveChrome = usePlayerImmersiveChrome({
      hasPlayback,
      isPlaying,
      hideDelayMs: 5000,
      feedbackDurationMs: 650,
    })

    immersiveChrome.onPageMouseMove()
    vi.advanceTimersByTime(5000)
    expect(immersiveChrome.chromeVisible.value).toBe(false)

    immersiveChrome.showSeekFeedback(-10)
    expect(immersiveChrome.chromeVisible.value).toBe(false)
    expect(immersiveChrome.feedback.value?.kind).toBe("seek")
    expect(immersiveChrome.feedback.value?.label).toBe("-10s")
    expect(immersiveChrome.feedback.value?.direction).toBe("backward")

    vi.advanceTimersByTime(649)
    expect(immersiveChrome.feedback.value?.label).toBe("-10s")

    vi.advanceTimersByTime(1)
    expect(immersiveChrome.feedback.value).toBeNull()
    expect(immersiveChrome.chromeVisible.value).toBe(false)

    immersiveChrome.dispose()
    vi.useRealTimers()
  })

  it("shows curated feedback with its own visual kind while chrome stays hidden", () => {
    vi.useFakeTimers()

    const hasPlayback = ref(true)
    const isPlaying = ref(true)
    const immersiveChrome = usePlayerImmersiveChrome({
      hasPlayback,
      isPlaying,
      hideDelayMs: 5000,
      feedbackDurationMs: 650,
    })

    immersiveChrome.onPageMouseMove()
    vi.advanceTimersByTime(5000)
    expect(immersiveChrome.chromeVisible.value).toBe(false)

    immersiveChrome.showCuratedFeedback("Curated +1")
    expect(immersiveChrome.chromeVisible.value).toBe(false)
    expect(immersiveChrome.feedback.value?.kind).toBe("curated")
    expect(immersiveChrome.feedback.value?.label).toBe("Curated +1")

    vi.advanceTimersByTime(650)
    expect(immersiveChrome.feedback.value).toBeNull()

    immersiveChrome.dispose()
    vi.useRealTimers()
  })

  it("reveals chrome again only when the mouse moves", () => {
    vi.useFakeTimers()

    const hasPlayback = ref(true)
    const isPlaying = ref(true)
    const immersiveChrome = usePlayerImmersiveChrome({
      hasPlayback,
      isPlaying,
      hideDelayMs: 5000,
      feedbackDurationMs: 650,
    })

    immersiveChrome.onPageMouseMove()
    vi.advanceTimersByTime(5000)
    expect(immersiveChrome.chromeVisible.value).toBe(false)

    immersiveChrome.showSeekFeedback(10)
    expect(immersiveChrome.chromeVisible.value).toBe(false)

    immersiveChrome.onPageMouseMove()
    expect(immersiveChrome.chromeVisible.value).toBe(true)

    immersiveChrome.dispose()
    vi.useRealTimers()
  })

  it("does not re-show chrome just because playback pauses", () => {
    vi.useFakeTimers()

    const hasPlayback = ref(true)
    const isPlaying = ref(true)
    const immersiveChrome = usePlayerImmersiveChrome({
      hasPlayback,
      isPlaying,
      hideDelayMs: 5000,
      feedbackDurationMs: 650,
    })

    immersiveChrome.onPageMouseMove()
    vi.advanceTimersByTime(5000)
    expect(immersiveChrome.chromeVisible.value).toBe(false)

    isPlaying.value = false
    expect(immersiveChrome.chromeVisible.value).toBe(false)

    immersiveChrome.dispose()
    vi.useRealTimers()
  })
})
