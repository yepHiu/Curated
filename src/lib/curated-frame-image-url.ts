import { resolveApiBaseUrl } from "@/api/http-client"

function apiBaseUrl(): string {
  return new URL(resolveApiBaseUrl(import.meta.env), window.location.origin).href.replace(/\/$/, "")
}

export function curatedFrameImageUrl(frameId: string): string {
  return `${apiBaseUrl()}/curated-frames/${encodeURIComponent(frameId)}/image`
}

export function curatedFrameThumbnailUrl(frameId: string): string {
  return `${apiBaseUrl()}/curated-frames/${encodeURIComponent(frameId)}/thumbnail`
}
