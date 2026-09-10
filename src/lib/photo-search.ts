import type { PhotoBook } from "@/domain/photo/types"

export interface PhotoSearchFilters {
  q?: string
}

export function filterPhotos(
  photos: readonly PhotoBook[],
  filters: PhotoSearchFilters = {},
): PhotoBook[] {
  const q = filters.q?.trim().toLocaleLowerCase()
  if (!q) return [...photos]
  return photos.filter((photo) => {
    const haystack = [
      photo.title,
      photo.sourceFileName,
      photo.location,
      ...photo.tags,
    ]
      .join(" ")
      .toLocaleLowerCase()
    return haystack.includes(q)
  })
}
