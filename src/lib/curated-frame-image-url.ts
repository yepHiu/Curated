function apiBaseUrl(): string {
  const base = (import.meta.env.VITE_API_BASE_URL ?? "/api").replace(/\/$/, "")
  if (base.startsWith("http://") || base.startsWith("https://")) {
    return base
  }
  return `${window.location.origin}${base}`
}

export function curatedFrameImageUrl(frameId: string): string {
  return `${apiBaseUrl()}/curated-frames/${encodeURIComponent(frameId)}/image`
}

export function curatedFrameThumbnailUrl(frameId: string): string {
  return `${apiBaseUrl()}/curated-frames/${encodeURIComponent(frameId)}/thumbnail`
}
