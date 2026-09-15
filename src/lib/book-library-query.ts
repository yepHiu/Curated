import type { ComicReadStatus } from "@/domain/comic/types"
import type { LocationQuery } from "vue-router"

export type BookLibrarySortValue = "addedAt" | "fileName"

export type ComicLibraryBrowseState = {
  q: string
  tag: string
  favorite: boolean
  readStatus: ComicReadStatus | "all"
  sort: BookLibrarySortValue
}

export type PhotoLibraryBrowseState = {
  q: string
  tag: string
  sort: BookLibrarySortValue
}

export type BookLibraryQueryPatch = {
  q?: string | null
  tag?: string | null
  favorite?: boolean | null
  readStatus?: ComicReadStatus | "all" | null
  sort?: BookLibrarySortValue | null
}

/** 读取书库 URL 查询里的第一个非空字符串。 */
export function firstBookLibraryQueryString(query: LocationQuery, key: string): string {
  const raw = query[key]
  if (typeof raw === "string") return raw.trim()
  if (Array.isArray(raw)) {
    for (const item of raw) {
      if (typeof item === "string" && item.trim()) return item.trim()
    }
  }
  return ""
}

/** 只接受导入时间与文件名；旧书签 sort=favorite 不当作排序。 */
export function parseBookLibrarySort(raw: string): BookLibrarySortValue {
  return raw === "fileName" ? "fileName" : "addedAt"
}

/** 判断当前 URL 是否仍使用已废弃的收藏排序。 */
export function isLegacyFavoriteSort(query: LocationQuery): boolean {
  return firstBookLibraryQueryString(query, "sort") === "favorite"
}

/** 解析漫画墙的搜索、精确标签、收藏、阅读状态和排序。 */
export function parseComicLibraryBrowse(query: LocationQuery): ComicLibraryBrowseState {
  const favoriteRaw = firstBookLibraryQueryString(query, "favorite").toLowerCase()
  const readStatusRaw = firstBookLibraryQueryString(query, "readStatus")
  const readStatus: ComicReadStatus | "all" =
    readStatusRaw === "unread" || readStatusRaw === "reading" || readStatusRaw === "read"
      ? readStatusRaw
      : "all"
  return {
    q: firstBookLibraryQueryString(query, "q"),
    tag: firstBookLibraryQueryString(query, "tag"),
    favorite: favoriteRaw === "1" || favoriteRaw === "true" || isLegacyFavoriteSort(query),
    readStatus,
    sort: parseBookLibrarySort(firstBookLibraryQueryString(query, "sort")),
  }
}

/** 解析写真墙的搜索、精确标签和排序。 */
export function parsePhotoLibraryBrowse(query: LocationQuery): PhotoLibraryBrowseState {
  return {
    q: firstBookLibraryQueryString(query, "q"),
    tag: firstBookLibraryQueryString(query, "tag"),
    sort: parseBookLibrarySort(firstBookLibraryQueryString(query, "sort")),
  }
}

/** 写入或删除一个规范化后的查询字段。 */
function assignBookLibraryQueryValue(
  query: LocationQuery,
  key: string,
  value: string | undefined,
) {
  if (value) {
    query[key] = value
    return
  }
  delete query[key]
}

/**
 * 在当前书库 query 上打补丁，并去掉已废弃的 filter / sort=favorite。
 * 缺省排序、空搜索、空标签、未启用的收藏和“全部”阅读状态不会写进 URL。
 */
export function patchBookLibraryQuery(
  current: LocationQuery,
  patch: BookLibraryQueryPatch,
): LocationQuery {
  const next: LocationQuery = { ...current }
  delete next.filter
  if (next.sort === "favorite") {
    delete next.sort
  }

  if (Object.prototype.hasOwnProperty.call(patch, "q")) {
    assignBookLibraryQueryValue(next, "q", patch.q?.trim() || undefined)
  }
  if (Object.prototype.hasOwnProperty.call(patch, "tag")) {
    assignBookLibraryQueryValue(next, "tag", patch.tag?.trim() || undefined)
  }
  if (Object.prototype.hasOwnProperty.call(patch, "favorite")) {
    assignBookLibraryQueryValue(next, "favorite", patch.favorite ? "1" : undefined)
  }
  if (Object.prototype.hasOwnProperty.call(patch, "readStatus")) {
    const readStatus = patch.readStatus && patch.readStatus !== "all" ? patch.readStatus : undefined
    assignBookLibraryQueryValue(next, "readStatus", readStatus)
  }
  if (Object.prototype.hasOwnProperty.call(patch, "sort")) {
    const sort = patch.sort && patch.sort !== "addedAt" ? patch.sort : undefined
    assignBookLibraryQueryValue(next, "sort", sort)
  }
  return next
}

/** 漫画墙是否带有会缩小结果集的约束。 */
export function comicLibraryHasConstraints(state: ComicLibraryBrowseState): boolean {
  return Boolean(state.q || state.tag || state.favorite || state.readStatus !== "all")
}

/** 写真墙是否带有会缩小结果集的约束。 */
export function photoLibraryHasConstraints(state: PhotoLibraryBrowseState): boolean {
  return Boolean(state.q || state.tag)
}
