import type { PhotoBook } from "@/domain/photo/types"

export type PhotoLibrarySortValue = "addedAt" | "fileName"

const collator = new Intl.Collator("zh-CN", {
  numeric: true,
  sensitivity: "base",
})

/** 新导入的写真集排在前面；导入时间相同则按文件名。 */
function compareByAddedAtDesc(left: PhotoBook, right: PhotoBook): number {
  const addedAt = right.addedAt.localeCompare(left.addedAt)
  if (addedAt !== 0) return addedAt
  return compareByFileNameAsc(left, right)
}

/** 按源文件名自然排序；文件名相同则新导入的在前。 */
function compareByFileNameAsc(left: PhotoBook, right: PhotoBook): number {
  const sourceFileName = collator.compare(left.sourceFileName, right.sourceFileName)
  if (sourceFileName !== 0) return sourceFileName
  return right.addedAt.localeCompare(left.addedAt)
}

/** 按导入时间或文件名排序写真墙；收藏在能改收藏之前不作为排序档。 */
export function sortPhotos(
  photos: readonly PhotoBook[],
  sort: PhotoLibrarySortValue,
): PhotoBook[] {
  const sorted = [...photos]
  switch (sort) {
    case "fileName":
      return sorted.sort(compareByFileNameAsc)
    case "addedAt":
    default:
      return sorted.sort(compareByAddedAtDesc)
  }
}
