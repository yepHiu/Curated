/**
 * 后端持久化萃取帧时，缩略图由 GET /api/curated-frames/{id}/image 提供。
 */
export function curatedFrameImageUrl(frameId: string): string {
  const base = (import.meta.env.VITE_API_BASE_URL ?? "/api").replace(/\/$/, "")
  return `${window.location.origin}${base}/curated-frames/${encodeURIComponent(frameId)}/image`
}
