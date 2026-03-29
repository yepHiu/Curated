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

export function applyThemeToDocument(pref: ThemePreference): void {
  if (typeof document === "undefined") return
  const root = document.documentElement
  if (isDarkForPreference(pref)) root.classList.add("dark")
  else root.classList.remove("dark")
}
