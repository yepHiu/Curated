import type { ComicBook, ComicReadStatus } from "@/domain/comic/types"

export type ComicLibraryFilter = {
  q?: string
  favorite?: boolean
  readStatus?: ComicReadStatus | "all" | string
}

function normalizeSearchText(value: string): string {
  return value.trim().toLowerCase()
}

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

export function filterComics(
  comics: readonly ComicBook[],
  filter: ComicLibraryFilter = {},
): ComicBook[] {
  const q = normalizeSearchText(filter.q ?? "")
  const readStatus =
    typeof filter.readStatus === "string" ? filter.readStatus.trim() : filter.readStatus

  return comics.filter((comic) => {
    if (filter.favorite === true && !comic.isFavorite) {
      return false
    }
    if (readStatus && readStatus !== "all" && comic.readStatus !== readStatus) {
      return false
    }
    if (q && !comicSearchHaystack(comic).includes(q)) {
      return false
    }
    return true
  })
}
