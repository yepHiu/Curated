/** 与 index.html 内联脚本共用，避免首屏闪烁 */
export const THEME_STORAGE_KEY = "curated-ui-theme"

export type ThemePreference = "light" | "dark" | "system"

export function parseThemePreference(raw: string | null): ThemePreference {
  if (raw === "light" || raw === "dark" || raw === "system") return raw
  return "dark"
}

export function getStoredThemePreference(): ThemePreference {
  if (typeof localStorage === "undefined") return "dark"
  try {
    return parseThemePreference(localStorage.getItem(THEME_STORAGE_KEY))
  } catch {
    return "dark"
  }
}

export function setStoredThemePreference(pref: ThemePreference): void {
  if (typeof localStorage === "undefined") return
  try {
    localStorage.setItem(THEME_STORAGE_KEY, pref)
  } catch {
    /* ignore quota / private mode */
  }
}

export function isDarkForPreference(pref: ThemePreference): boolean {
  if (pref === "dark") return true
  if (pref === "light") return false
  if (typeof window === "undefined") return true
  return window.matchMedia("(prefers-color-scheme: dark)").matches
}

export type ApplyThemeToDocumentOptions = {
  /** 使用 View Transitions API（需浏览器支持；首屏与减少动画偏好下应关闭） */
  viewTransition?: boolean
}

function prefersReducedMotion(): boolean {
  if (typeof window === "undefined") return true
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches
}

export function applyThemeToDocument(
  pref: ThemePreference,
  options?: ApplyThemeToDocumentOptions,
): void {
  if (typeof document === "undefined") return
  const root = document.documentElement
  const apply = () => {
    if (isDarkForPreference(pref)) root.classList.add("dark")
    else root.classList.remove("dark")
  }

  const useVt =
    options?.viewTransition === true &&
    typeof document.startViewTransition === "function" &&
    !prefersReducedMotion()

  if (useVt) {
    document.startViewTransition(apply)
  } else {
    apply()
  }
}
