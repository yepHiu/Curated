import { nextTick, onBeforeUnmount, readonly, watch, type Ref, ref } from "vue"

type ScrollSnapshot = {
  top: number
  left: number
}

const libraryScrollSnapshots = new Map<string, ScrollSnapshot>()

function captureScroll(el: HTMLElement | null): ScrollSnapshot {
  return {
    top: el?.scrollTop ?? 0,
    left: el?.scrollLeft ?? 0,
  }
}

function applyScroll(el: HTMLElement | null, snapshot: ScrollSnapshot) {
  if (!el) return
  el.scrollTop = snapshot.top
  el.scrollLeft = snapshot.left
}

async function restoreScrollSequence(
  scrollElRef: Ref<HTMLElement | null>,
  snapshot: ScrollSnapshot,
) {
  await nextTick()
  await new Promise<void>((resolve) => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => resolve())
    })
  })

  const restore = () => {
    applyScroll(scrollElRef.value, snapshot)
  }

  restore()
  queueMicrotask(restore)
  requestAnimationFrame(restore)
  setTimeout(restore, 40)
  setTimeout(restore, 120)
  setTimeout(restore, 260)
  setTimeout(restore, 480)
}

export function useLibraryScrollPreserve(options: {
  scrollElRef: Ref<HTMLElement | null>
  preserveKey: Ref<string>
}) {
  const { scrollElRef, preserveKey } = options
  const scrollTop = ref(0)
  let detachScrollListener: (() => void) | undefined

  function storeSnapshot(key = preserveKey.value) {
    const normalizedKey = key.trim()
    if (!normalizedKey) return
    libraryScrollSnapshots.set(normalizedKey, captureScroll(scrollElRef.value))
  }

  async function restoreSnapshot(key = preserveKey.value) {
    const normalizedKey = key.trim()
    if (!normalizedKey) return
    const snapshot = libraryScrollSnapshots.get(normalizedKey)
    if (!snapshot) return
    scrollTop.value = snapshot.top
    await restoreScrollSequence(scrollElRef, snapshot)
  }

  function scrollToTop() {
    const el = scrollElRef.value
    if (!el) return
    el.scrollTo({ top: 0, behavior: "smooth" })
  }

  watch(
    scrollElRef,
    (el) => {
      detachScrollListener?.()
      detachScrollListener = undefined

      if (!el) {
        scrollTop.value = 0
        return
      }

      const onScroll = () => {
        scrollTop.value = el.scrollTop
        storeSnapshot()
      }

      scrollTop.value = el.scrollTop
      el.addEventListener("scroll", onScroll, { passive: true })
      detachScrollListener = () => el.removeEventListener("scroll", onScroll)
      if (libraryScrollSnapshots.has(preserveKey.value.trim())) {
        void restoreSnapshot()
      }
    },
    { immediate: true },
  )

  watch(
    preserveKey,
    (nextKey, prevKey) => {
      if (prevKey.trim()) {
        storeSnapshot(prevKey)
      }

      if (!nextKey.trim()) {
        scrollTop.value = scrollElRef.value?.scrollTop ?? 0
        return
      }

      if (libraryScrollSnapshots.has(nextKey.trim())) {
        void restoreSnapshot(nextKey)
        return
      }

      scrollTop.value = scrollElRef.value?.scrollTop ?? 0
    },
    { flush: "post" },
  )

  onBeforeUnmount(() => {
    storeSnapshot()
    detachScrollListener?.()
  })

  return {
    scrollTop: readonly(scrollTop),
    scrollToTop,
  }
}
