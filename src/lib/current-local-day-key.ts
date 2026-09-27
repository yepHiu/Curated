import { onBeforeUnmount, onMounted, readonly, ref } from "vue"

export function getCurrentLocalDayKey(now: Date = new Date()): string {
  const month = String(now.getMonth() + 1).padStart(2, "0")
  const day = String(now.getDate()).padStart(2, "0")
  return `${now.getFullYear()}-${month}-${day}`
}

const dayKey = ref(getCurrentLocalDayKey())
let subscribers = 0
let timer: ReturnType<typeof setInterval> | undefined

function refresh() {
  dayKey.value = getCurrentLocalDayKey()
}

/** Share one clock across all visible release badges. */
export function useCurrentLocalDayKey() {
  refresh()
  onMounted(() => {
    refresh()
    if (subscribers++ === 0) {
      timer = setInterval(refresh, 60_000)
      window.addEventListener("focus", refresh)
      document.addEventListener("visibilitychange", refresh)
    }
  })
  onBeforeUnmount(() => {
    if (--subscribers === 0) {
      clearInterval(timer)
      timer = undefined
      window.removeEventListener("focus", refresh)
      document.removeEventListener("visibilitychange", refresh)
    }
  })
  return readonly(dayKey)
}
