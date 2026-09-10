import type { PhotoBook } from "@/domain/photo/types"

export type PhotoLibrarySortValue = "addedAt" | "fileName" | "favorite"

const collator = new Intl.Collator("zh-CN", {
  numeric: true,
  sensitivity: "base",
})

function compareByAddedAtDesc(left: PhotoBook, right: PhotoBook): number {
  const addedAt = right.addedAt.localeCompare(left.addedAt)
  if (addedAt !== 0) return addedAt
  return compareByFileNameAsc(left, right)
}

function compareByFileNameAsc(left: PhotoBook, right: PhotoBook): number {
  const sourceFileName = collator.compare(left.sourceFileName, right.sourceFileName)
  if (sourceFileName !== 0) return sourceFileName
  return right.addedAt.localeCompare(left.addedAt)
}

function compareByFavoriteFirst(left: PhotoBook, right: PhotoBook): number {
  if (left.isFavorite !== right.isFavorite) {
    return left.isFavorite ? -1 : 1
  }
  return compareByAddedAtDesc(left, right)
}

export function sortPhotos(
  photos: readonly PhotoBook[],
  sort: PhotoLibrarySortValue,
): PhotoBook[] {
  const sorted = [...photos]
  switch (sort) {
    case "fileName":
      return sorted.sort(compareByFileNameAsc)
    case "favorite":
      return sorted.sort(compareByFavoriteFirst)
    case "addedAt":
    default:
      return sorted.sort(compareByAddedAtDesc)
  }
}
