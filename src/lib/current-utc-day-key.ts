import { onBeforeUnmount, onMounted, readonly, ref } from "vue"

export interface UseCurrentUtcDayKeyOptions {
  pollIntervalMs?: number
  now?: () => Date
}

export function getCurrentUtcDayKey(now: Date = new Date()): string {
  return now.toISOString().slice(0, 10)
}

export function useCurrentUtcDayKey(options: UseCurrentUtcDayKeyOptions = {}) {
  const pollIntervalMs = Math.max(1_000, options.pollIntervalMs ?? 60_000)
  const now = options.now ?? (() => new Date())
  const dayKey = ref(getCurrentUtcDayKey(now()))
  let timer: number | null = null

  const refresh = () => {
    const nextDayKey = getCurrentUtcDayKey(now())
    if (nextDayKey !== dayKey.value) {
      dayKey.value = nextDayKey
    }
  }

  onMounted(() => {
    refresh()
    timer = window.setInterval(refresh, pollIntervalMs)
    window.addEventListener("focus", refresh)
    document.addEventListener("visibilitychange", refresh)
  })

  onBeforeUnmount(() => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
    window.removeEventListener("focus", refresh)
    document.removeEventListener("visibilitychange", refresh)
  })

  return {
    dayKey: readonly(dayKey),
    refresh,
  }
}
