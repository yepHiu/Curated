import { nextTick, onBeforeUnmount, readonly, watch, type Ref, ref } from "vue"

type ScrollSnapshot = {
  top: number
  left: number
}

const libraryScrollSnapshots = new Map<string, ScrollSnapshot>()

export function clearLibraryScrollSnapshot(key: string) {
  libraryScrollSnapshots.delete(key.trim())
}

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
  isCurrent: () => boolean,
) {
  await nextTick()
  await new Promise<void>((resolve) => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => resolve())
    })
  })

  const restore = () => {
    if (!isCurrent()) return
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
  let scrollAnimationFrame: number | undefined
  let restoreRevision = 0

  function cancelScrollAnimation() {
    if (scrollAnimationFrame === undefined) return
    cancelAnimationFrame(scrollAnimationFrame)
    scrollAnimationFrame = undefined
  }

  function onScrollInput() {
    restoreRevision++
    cancelScrollAnimation()
  }

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
    const revision = ++restoreRevision
    const el = scrollElRef.value
    scrollTop.value = snapshot.top
    await restoreScrollSequence(scrollElRef, snapshot, () =>
      revision === restoreRevision && el === scrollElRef.value && normalizedKey === preserveKey.value.trim(),
    )
  }

  function scrollToTop() {
    const el = scrollElRef.value
    if (!el) return
    onScrollInput()
    if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) {
      el.scrollTo({ top: 0, behavior: "instant" })
      return
    }

    // DynamicScroller adjusts scrollTop when measured item heights change,
    // which cancels native smooth scrolling. Drive the animation ourselves.
    const startTop = el.scrollTop
    let startTime: number | undefined
    const animate = (time: number) => {
      startTime ??= time
      const progress = Math.min(1, (time - startTime) / 500)
      el.scrollTo({ top: startTop * (1 - progress) ** 3, behavior: "instant" })
      scrollAnimationFrame = progress < 1 ? requestAnimationFrame(animate) : undefined
    }
    scrollAnimationFrame = requestAnimationFrame(animate)
  }

  watch(
    scrollElRef,
    (el) => {
      onScrollInput()
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
      el.addEventListener("wheel", onScrollInput, { passive: true })
      el.addEventListener("touchstart", onScrollInput, { passive: true })
      el.addEventListener("pointerdown", onScrollInput, { passive: true })
      el.addEventListener("keydown", onScrollInput)
      detachScrollListener = () => {
        el.removeEventListener("scroll", onScroll)
        el.removeEventListener("wheel", onScrollInput)
        el.removeEventListener("touchstart", onScrollInput)
        el.removeEventListener("pointerdown", onScrollInput)
        el.removeEventListener("keydown", onScrollInput)
      }
      if (libraryScrollSnapshots.has(preserveKey.value.trim())) {
        void restoreSnapshot()
      }
    },
    { immediate: true },
  )

  watch(
    preserveKey,
    (nextKey, prevKey) => {
      onScrollInput()
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
    onScrollInput()
    storeSnapshot()
    detachScrollListener?.()
  })

  return {
    scrollTop: readonly(scrollTop),
    scrollToTop,
  }
}
