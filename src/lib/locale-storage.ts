export const LOCALE_STORAGE_KEY = "curated-locale"

export type SupportedLocale = "en" | "zh-CN" | "ja"

export const SUPPORTED_LOCALES: SupportedLocale[] = ["zh-CN", "en", "ja"]

export function normalizeNavigatorLanguage(): SupportedLocale {
  if (typeof navigator === "undefined") {
    return "zh-CN"
  }
  const raw = navigator.language || (navigator as Navigator & { userLanguage?: string }).userLanguage || ""
  const lower = raw.toLowerCase()
  if (lower.startsWith("ja")) return "ja"
  if (lower.startsWith("zh")) return "zh-CN"
  return "en"
}

export function resolveInitialLocale(): SupportedLocale {
  if (typeof localStorage === "undefined") {
    return normalizeNavigatorLanguage()
  }
  try {
    const v = localStorage.getItem(LOCALE_STORAGE_KEY)?.trim()
    if (v === "en" || v === "zh-CN" || v === "ja") {
      return v
    }
  } catch {
    // ignore
  }
  return normalizeNavigatorLanguage()
}

export function persistLocale(locale: SupportedLocale) {
  if (typeof localStorage === "undefined") return
  try {
    localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  } catch {
    // ignore
  }
}

/** `document.documentElement.lang` */
export function htmlLangFor(locale: SupportedLocale): string {
  return locale
}
