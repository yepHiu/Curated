import type { ComicBook } from "@/domain/comic/types"

export type ComicLibrarySortValue = "addedAt" | "fileName" | "favorite"

const collator = new Intl.Collator("zh-CN", {
  numeric: true,
  sensitivity: "base",
})

function compareByAddedAtDesc(left: ComicBook, right: ComicBook): number {
  const addedAt = right.addedAt.localeCompare(left.addedAt)
  if (addedAt !== 0) return addedAt
  return compareByFileNameAsc(left, right)
}

function compareByFileNameAsc(left: ComicBook, right: ComicBook): number {
  const sourceFileName = collator.compare(left.sourceFileName, right.sourceFileName)
  if (sourceFileName !== 0) return sourceFileName
  return right.addedAt.localeCompare(left.addedAt)
}

function compareByFavoriteFirst(left: ComicBook, right: ComicBook): number {
  if (left.isFavorite !== right.isFavorite) {
    return left.isFavorite ? -1 : 1
  }
  return compareByAddedAtDesc(left, right)
}

export function sortComics(
  comics: readonly ComicBook[],
  sort: ComicLibrarySortValue,
): ComicBook[] {
  const sorted = [...comics]
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
