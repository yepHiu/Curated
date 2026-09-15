import type { ComicBook, ComicReadStatus } from "@/domain/comic/types"

export type ComicLibraryFilter = {
  q?: string
  tag?: string
  favorite?: boolean
  readStatus?: ComicReadStatus | "all" | string
}

/** 规范化自由文本搜索词，忽略首尾空白和大小写。 */
function normalizeSearchText(value: string): string {
  return value.trim().toLowerCase()
}

/** 拼出漫画墙自由文本可命中的标题、文件名、路径和标签。 */
export function comicSearchHaystack(comic: ComicBook): string {
  return [
    comic.title,
    comic.sourceFileName,
    comic.location,
    ...comic.tags,
  ]
    .filter(Boolean)
    .join("\n")
    .toLowerCase()
}

/** 按自由文本、精确标签、收藏和阅读状态缩小漫画墙。 */
export function filterComics(
  comics: readonly ComicBook[],
  filter: ComicLibraryFilter = {},
): ComicBook[] {
  const q = normalizeSearchText(filter.q ?? "")
  const tag = filter.tag?.trim() ?? ""
  const readStatus =
    typeof filter.readStatus === "string" ? filter.readStatus.trim() : filter.readStatus

  return comics.filter((comic) => {
    // 收藏、阅读状态、精确标签与自由文本同时收窄结果。
    // 收藏、阅读状态、精确标签与自由文本同时收窄结果。
    // 收藏、阅读状态、精确标签与自由文本同时收窄结果。
    if (filter.favorite === true && !comic.isFavorite) {
      return false
    }
    if (readStatus && readStatus !== "all" && comic.readStatus !== readStatus) {
      return false
    }
    if (tag && !comic.tags.includes(tag)) {
      return false
    }
    if (q && !comicSearchHaystack(comic).includes(q)) {
      return false
    }
    return true
  })
}
