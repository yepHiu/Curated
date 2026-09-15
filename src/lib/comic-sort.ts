import type { ComicBook } from "@/domain/comic/types"

export type ComicLibrarySortValue = "addedAt" | "fileName"

const collator = new Intl.Collator("zh-CN", {
  numeric: true,
  sensitivity: "base",
})

/** 新导入的漫画排在前面；导入时间相同则按文件名。 */
function compareByAddedAtDesc(left: ComicBook, right: ComicBook): number {
  const addedAt = right.addedAt.localeCompare(left.addedAt)
  if (addedAt !== 0) return addedAt
  return compareByFileNameAsc(left, right)
}

/** 按源文件名自然排序；文件名相同则新导入的在前。 */
function compareByFileNameAsc(left: ComicBook, right: ComicBook): number {
  const sourceFileName = collator.compare(left.sourceFileName, right.sourceFileName)
  if (sourceFileName !== 0) return sourceFileName
  return right.addedAt.localeCompare(left.addedAt)
}

/** 按导入时间或文件名排序漫画墙；收藏不再作为排序档。 */
export function sortComics(
  comics: readonly ComicBook[],
  sort: ComicLibrarySortValue,
): ComicBook[] {
  const sorted = [...comics]
  switch (sort) {
    case "fileName":
      return sorted.sort(compareByFileNameAsc)
    case "addedAt":
    default:
      return sorted.sort(compareByAddedAtDesc)
  }
}
