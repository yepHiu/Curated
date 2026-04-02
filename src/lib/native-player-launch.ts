import type { NativePlayerPreset } from "@/api/types"

export const NATIVE_PLAYER_BROWSER_TEMPLATE_STORAGE_KEY =
  "curated-native-player-browser-template-v1"

export type NativePlayerLaunchContext = {
  url: string
  path?: string
  movieId?: string
  code?: string
  startSec?: number
  startMs?: number
}

export function normalizeNativePlayerPresetForBrowserLaunch(
  preset: NativePlayerPreset | undefined,
  command?: string,
): NativePlayerPreset {
  switch (preset) {
    case "mpv":
    case "potplayer":
    case "custom":
      return preset
  }
  const cmd = (command ?? "").trim().toLowerCase()
  if (cmd.includes("potplayer")) return "potplayer"
  if (!cmd || cmd.includes("mpv")) return "mpv"
  return "custom"
}

export function defaultNativePlayerBackendCommand(
  preset: NativePlayerPreset | undefined,
): string {
  return normalizeNativePlayerPresetForBrowserLaunch(preset) === "potplayer"
    ? "PotPlayerMini64.exe"
    : "mpv"
}

export function defaultNativePlayerBrowserTemplate(
  preset: NativePlayerPreset | undefined,
): string {
  switch (normalizeNativePlayerPresetForBrowserLaunch(preset)) {
    case "potplayer":
      return "potplayer:{url}"
    case "custom":
      return "your-player:{url}"
    case "mpv":
    default:
      return ""
  }
}

export function getStoredNativePlayerBrowserTemplate(): string | null {
  if (typeof localStorage === "undefined") return null
  try {
    const value = localStorage.getItem(NATIVE_PLAYER_BROWSER_TEMPLATE_STORAGE_KEY)?.trim()
    return value || null
  } catch {
    return null
  }
}

export function resolveNativePlayerBrowserTemplate(
  preset: NativePlayerPreset | undefined,
  storedTemplate?: string | null,
): string {
  const stored = (storedTemplate ?? getStoredNativePlayerBrowserTemplate() ?? "").trim()
  if (stored) return stored
  return defaultNativePlayerBrowserTemplate(preset)
}

export function persistNativePlayerBrowserTemplate(
  preset: NativePlayerPreset | undefined,
  template: string,
): string {
  const trimmed = template.trim()
  const fallback = defaultNativePlayerBrowserTemplate(preset)
  const next = trimmed || fallback
  if (typeof localStorage === "undefined") return next
  try {
    if (!trimmed || trimmed === fallback) {
      localStorage.removeItem(NATIVE_PLAYER_BROWSER_TEMPLATE_STORAGE_KEY)
    } else {
      localStorage.setItem(NATIVE_PLAYER_BROWSER_TEMPLATE_STORAGE_KEY, trimmed)
    }
  } catch {
    // ignore quota / private mode
  }
  return next
}

export function buildNativePlayerLaunchUrl(
  template: string,
  context: NativePlayerLaunchContext,
): string {
  const startSec = Math.max(0, Number.isFinite(context.startSec) ? context.startSec ?? 0 : 0)
  const startMs = Math.max(
    0,
    Number.isFinite(context.startMs) ? context.startMs ?? 0 : Math.round(startSec * 1000),
  )
  const replacements = new Map<string, string>([
    ["{url}", context.url ?? ""],
    ["{urlEncoded}", encodeURIComponent(context.url ?? "")],
    ["{path}", context.path ?? ""],
    ["{pathEncoded}", encodeURIComponent(context.path ?? "")],
    ["{movieId}", context.movieId ?? ""],
    ["{movieIdEncoded}", encodeURIComponent(context.movieId ?? "")],
    ["{code}", context.code ?? ""],
    ["{codeEncoded}", encodeURIComponent(context.code ?? "")],
    ["{startSec}", String(Math.floor(startSec))],
    ["{startSecEncoded}", encodeURIComponent(String(Math.floor(startSec)))],
    ["{startMs}", String(Math.floor(startMs))],
    ["{startMsEncoded}", encodeURIComponent(String(Math.floor(startMs)))],
  ])

  let next = template.trim()
  for (const [token, value] of replacements) {
    next = next.replaceAll(token, value)
  }
  return next
}

export function looksLikeBrowserProtocolLaunchTarget(value: string): boolean {
  const trimmed = value.trim()
  if (!/^[a-z][a-z0-9+.-]*:/i.test(trimmed)) return false
  return !/^https?:/i.test(trimmed)
}
