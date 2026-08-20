import { resolveApiBaseUrl } from "./http-client"

type MediaUrlEnv = Parameters<typeof resolveApiBaseUrl>[0]

/** Rewrite same-origin `/api/...` asset URLs onto the active API base (dev :8080 / VITE_API_BASE_URL). */
export function resolveMediaUrl(
  src: string,
  env: MediaUrlEnv = import.meta.env,
  origin = window.location.origin,
): string {
  const trimmed = src.trim()
  if (!trimmed) return ""
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  if (!trimmed.startsWith("/api/")) return trimmed

  const base = resolveApiBaseUrl(env, origin)
  if (base.startsWith("http://") || base.startsWith("https://")) {
    return `${base.replace(/\/$/, "")}${trimmed.slice("/api".length)}`
  }
  return trimmed
}
