/**
 * 设置页滚动容器与「保存时尽量保持滚动位置」相关符号。
 *
 * 历史实现会在每次异步保存的 `finally` 里多次写回 `scrollTop`（曾含 MutationObserver、
 * 多段 setTimeout、长时间 rAF burst）。与 Vue 3 在同一滚动根上的 patch 叠在一起时，
 * 会在多类开关 / 多页保存路径上触发整页白屏。
 *
 * 现改为 **透传**：`withPreservedScroll` / `withSyncPreservedScroll` 仅执行回调，不再做
 * 滚动回写。若保存后偶发滚到顶部，可接受；稳定优先。
 */
export const SETTINGS_SCROLL_EL_KEY = Symbol("settingsScrollEl")

/** Fallback when inject ref not yet bound — must match `SettingsView` root `id`. */
export const SETTINGS_SCROLL_ROOT_ID = "settings-scroll-root"

/**
 * Call from `SettingsView` (scroll listener).
 * 滚动恢复逻辑已移除；保留空实现以免改 `SettingsView` 调用点。
 */
export function noteSettingsScrollPosition(el: HTMLElement) {
  void el
  // no-op
}

export function resetSettingsScrollSnapshot() {
  // no-op
}

export function useSettingsScrollPreserve() {
  async function withPreservedScroll<T>(fn: () => Promise<T>): Promise<T> {
    return await fn()
  }

  function withSyncPreservedScroll(fn: () => void): void {
    fn()
  }

  return { withPreservedScroll, withSyncPreservedScroll }
}
