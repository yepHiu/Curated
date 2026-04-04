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

/**
 * 在异步保存后把设置页滚动条拉回捕获位置。
 * 刻意保持轻量：曾用 MutationObserver + 多段 setTimeout + 长时间 rAF burst，
 * 与 Switch 触发的 Vue 更新叠在一起会导致整页白屏。
 */
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

  restore()
  requestAnimationFrame(() => restore())
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
