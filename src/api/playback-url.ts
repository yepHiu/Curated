/**
 * Absolute URL for GET /api/library/movies/{id}/stream (same base rules as http-client).
 * Use as HTMLVideoElement.src when VITE_USE_WEB_API is enabled.
 */
export function moviePlaybackAbsoluteUrl(movieId: string): string {
  const base = (import.meta.env.VITE_API_BASE_URL ?? "/api").replace(/\/$/, "")
  const tail = `/library/movies/${encodeURIComponent(movieId)}/stream`
  if (base.startsWith("http://") || base.startsWith("https://")) {
    return `${base}${tail}`
  }
  return new URL(`${base}${tail}`, window.location.origin).href
}
