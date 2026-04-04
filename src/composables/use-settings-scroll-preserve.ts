import { inject, nextTick, type Ref, ref } from "vue"

/** Provided by `SettingsView` — scroll container for the settings page. */
export const SETTINGS_SCROLL_EL_KEY = Symbol("settingsScrollEl")

/** Fallback when inject ref not yet bound — must match `SettingsView` root `id`. */
export const SETTINGS_SCROLL_ROOT_ID = "settings-scroll-root"

/** Updated by passive `scroll` on the settings root — survives bad sync reads of scrollTop. */
let lastScrollSnapshot = { top: 0, left: 0 }

/**
 * Call from `SettingsView` (scroll listener) so we always have the real position even if
 * a later sync read sees `scrollTop === 0` after layout thrash.
 */
export function noteSettingsScrollPosition(el: HTMLElement) {
  lastScrollSnapshot = { top: el.scrollTop, left: el.scrollLeft }
}

export function resetSettingsScrollSnapshot() {
  lastScrollSnapshot = { top: 0, left: 0 }
}

function getSettingsScrollEl(scrollElRef: Ref<HTMLElement | null>): HTMLElement | null {
  return scrollElRef.value ?? document.getElementById(SETTINGS_SCROLL_ROOT_ID)
}

function captureScroll(scrollElRef: Ref<HTMLElement | null>): { top: number; left: number } {
  const el = getSettingsScrollEl(scrollElRef)
  const snap = lastScrollSnapshot
  const rawTop = el?.scrollTop ?? 0
  const rawLeft = el?.scrollLeft ?? 0
  return {
    top: Math.max(rawTop, snap.top),
    left: Math.max(rawLeft, snap.left),
  }
}

function applyScroll(el: HTMLElement | null, top: number, left: number) {
  if (!el) return
  try {
    el.scrollTop = top
    el.scrollLeft = left
  } catch (err) {
    console.error("[settings-scroll-preserve] applyScroll failed", err)
  }
}

function runScrollRestoreBurst(
  scrollElRef: Ref<HTMLElement | null>,
  top: number,
  left: number,
  stopAt: number,
) {
  const tick = () => {
    applyScroll(getSettingsScrollEl(scrollElRef), top, left)
    if (performance.now() < stopAt) {
      requestAnimationFrame(tick)
    }
  }
  requestAnimationFrame(tick)
}

async function applyScrollRestoreSequence(
  scrollElRef: Ref<HTMLElement | null>,
  top: number,
  left: number,
) {
  await nextTick()
  await nextTick()
  await new Promise<void>((resolve) => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => resolve())
    })
  })

  const restore = () => {
    try {
      const t = getSettingsScrollEl(scrollElRef)
      applyScroll(t, top, left)
      if (t) noteSettingsScrollPosition(t)
    } catch (err) {
      console.error("[settings-scroll-preserve] restore failed", err)
    }
  }

  const el = getSettingsScrollEl(scrollElRef)
  const cleanupFns: Array<() => void> = []
  const cleanupObservers = () => {
    while (cleanupFns.length > 0) {
      cleanupFns.pop()?.()
    }
  }
  if (el) {
    let mutationRestoreRaf = 0
    const scheduleRestoreFromObserver = () => {
      if (mutationRestoreRaf !== 0) return
      mutationRestoreRaf = requestAnimationFrame(() => {
        mutationRestoreRaf = 0
        restore()
      })
    }

    const mutationObserver = new MutationObserver(() => {
      scheduleRestoreFromObserver()
    })
    mutationObserver.observe(el, { childList: true, subtree: true })
    cleanupFns.push(() => {
      if (mutationRestoreRaf !== 0) {
        cancelAnimationFrame(mutationRestoreRaf)
        mutationRestoreRaf = 0
      }
      mutationObserver.disconnect()
    })

    const onFocusIn = () => {
      restore()
      requestAnimationFrame(() => restore())
    }
    el.addEventListener("focusin", onFocusIn, true)
    cleanupFns.push(() => el.removeEventListener("focusin", onFocusIn, true))

    if (typeof ResizeObserver !== "undefined") {
      let resizeRestoreRaf = 0
      const resizeObserver = new ResizeObserver(() => {
        if (resizeRestoreRaf !== 0) return
        resizeRestoreRaf = requestAnimationFrame(() => {
          resizeRestoreRaf = 0
          restore()
        })
      })
      resizeObserver.observe(el)
      cleanupFns.push(() => {
        if (resizeRestoreRaf !== 0) {
          cancelAnimationFrame(resizeRestoreRaf)
          resizeRestoreRaf = 0
        }
        resizeObserver.disconnect()
      })
    }

    setTimeout(cleanupObservers, 900)
  }

  restore()
  queueMicrotask(restore)
  setTimeout(restore, 0)
  requestAnimationFrame(() => restore())
  setTimeout(restore, 50)
  setTimeout(restore, 150)
  setTimeout(restore, 300)
  setTimeout(restore, 500)
  setTimeout(restore, 750)
  runScrollRestoreBurst(scrollElRef, top, left, performance.now() + 900)
}

/**
 * Restores scroll position on the settings scroll root after async updates,
 * so PATCH/save flows do not jump back to the top.
 *
 * Important: call **before** setting any `*Saving` / busy flags that change layout,
 * or set those flags inside the `fn` callback so this helper reads `scrollTop` first.
 */
export function useSettingsScrollPreserve() {
  const scrollElRef = inject<Ref<HTMLElement | null>>(SETTINGS_SCROLL_EL_KEY, ref(null))

  async function withPreservedScroll<T>(fn: () => Promise<T>): Promise<T> {
    const { top, left } = captureScroll(scrollElRef)
    try {
      return await fn()
    } finally {
      await applyScrollRestoreSequence(scrollElRef, top, left)
    }
  }

  /**
   * Use after **synchronous** DOM mutations (e.g. delete row, reorder) so focus loss / reflow
   * does not scroll the container to the top before the async save runs.
   */
  function withSyncPreservedScroll(fn: () => void): void {
    const { top, left } = captureScroll(scrollElRef)
    fn()
    void applyScrollRestoreSequence(scrollElRef, top, left)
  }

  return { withPreservedScroll, withSyncPreservedScroll }
}
