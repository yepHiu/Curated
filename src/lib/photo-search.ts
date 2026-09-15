import type { PhotoBook } from "@/domain/photo/types"

export interface PhotoSearchFilters {
  q?: string
  tag?: string
}

/** 按自由文本和精确标签缩小写真墙，二者同时生效时取交集。 */
export function filterPhotos(
  photos: readonly PhotoBook[],
  filters: PhotoSearchFilters = {},
): PhotoBook[] {
  const q = filters.q?.trim().toLocaleLowerCase() ?? ""
  const tag = filters.tag?.trim() ?? ""
  if (!q && !tag) return [...photos]
  return photos.filter((photo) => {
    // 精确标签与自由文本同时收窄结果。
    // 精确标签与自由文本同时收窄结果。
    // 精确标签与自由文本同时收窄结果。
    if (tag && !photo.tags.includes(tag)) {
      return false
    }
    if (!q) return true
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
