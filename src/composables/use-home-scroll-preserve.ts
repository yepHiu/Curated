import { nextTick, onBeforeUnmount, onMounted, type Ref } from "vue"

const HOME_SCROLL_TOP_KEY = "curated:home-scroll-top"
const HOME_DETAIL_RETURN_ARMED_KEY = "curated:home-detail-return-armed"

function readNumber(key: string): number | null {
  if (typeof window === "undefined") return null

  const raw = window.sessionStorage.getItem(key)
  if (!raw) return null

  const value = Number(raw)
  return Number.isFinite(value) ? value : null
}

function writeNumber(key: string, value: number) {
  if (typeof window === "undefined") return

  window.sessionStorage.setItem(key, String(Math.max(0, Math.round(value))))
}

export function saveHomeScrollSnapshot(scrollTop: number) {
  writeNumber(HOME_SCROLL_TOP_KEY, scrollTop)
}

export function readHomeScrollSnapshot(): number | null {
  return readNumber(HOME_SCROLL_TOP_KEY)
}

export function armHomeDetailReturnRestore() {
  if (typeof window === "undefined") return

  window.sessionStorage.setItem(HOME_DETAIL_RETURN_ARMED_KEY, "1")
}

export function consumeHomeDetailReturnRestore(): number | null {
  if (typeof window === "undefined") return null

  const armed = window.sessionStorage.getItem(HOME_DETAIL_RETURN_ARMED_KEY) === "1"
  window.sessionStorage.removeItem(HOME_DETAIL_RETURN_ARMED_KEY)
  if (!armed) return null

  return readHomeScrollSnapshot()
}

export function resetHomeScrollRestoreState() {
  if (typeof window === "undefined") return

  window.sessionStorage.removeItem(HOME_SCROLL_TOP_KEY)
  window.sessionStorage.removeItem(HOME_DETAIL_RETURN_ARMED_KEY)
}

export function useHomeScrollPreserve(options: {
  scrollElRef: Ref<HTMLElement | null>
}) {
  const { scrollElRef } = options

  function persist() {
    const el = scrollElRef.value
    if (!el) return

    saveHomeScrollSnapshot(el.scrollTop)
  }

  onMounted(async () => {
    const restoreTop = consumeHomeDetailReturnRestore()
    if (restoreTop === null) return

    await nextTick()
    requestAnimationFrame(() => {
      const el = scrollElRef.value
      if (!el) return

      el.scrollTop = restoreTop
      setTimeout(() => {
        if (scrollElRef.value) {
          scrollElRef.value.scrollTop = restoreTop
        }
      }, 60)
    })
  })

  onBeforeUnmount(() => {
    persist()
  })

  return { persist }
}
