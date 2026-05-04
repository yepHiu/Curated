const MAX_KNOWN_LOADED_IMAGES = 2000

const loadedImageUrls = new Set<string>()

function normalizeImageUrl(url: string | undefined): string {
  return url?.trim() ?? ""
}

export function isImageUrlLoaded(url: string | undefined): boolean {
  const normalized = normalizeImageUrl(url)
  return normalized ? loadedImageUrls.has(normalized) : false
}

export function markImageUrlLoaded(url: string | undefined): void {
  const normalized = normalizeImageUrl(url)
  if (!normalized) return

  loadedImageUrls.delete(normalized)
  loadedImageUrls.add(normalized)

  while (loadedImageUrls.size > MAX_KNOWN_LOADED_IMAGES) {
    const oldest = loadedImageUrls.values().next().value
    if (typeof oldest !== "string") break
    loadedImageUrls.delete(oldest)
  }
}
