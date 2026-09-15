import type { Movie } from "@/domain/movie/types"

/** Library grid sort keys shared by URL query, Saved Views, and filter UI. */
export type LibrarySortKey =
  | "added"
  | "release"
  | "rating"
  | "code"
  | "actor"
  | "studio"
  | "year"

export const librarySortKeys = [
  "added",
  "release",
  "rating",
  "code",
  "actor",
  "studio",
  "year",
] as const satisfies readonly LibrarySortKey[]

/** 比较入库时间，新到旧；可解析为时间时按时刻，否则按字符串。不使用番号或 id。 */
export function compareAddedAtDesc(left: string, right: string): number {
  const leftMs = Date.parse(left)
  const rightMs = Date.parse(right)
  if (!Number.isNaN(leftMs) && !Number.isNaN(rightMs) && leftMs !== rightMs) {
    return rightMs - leftMs
  }
  return right.localeCompare(left)
}

/** 按入库时间（addedAt）新到旧；相同时保持稳定次序，不用番号。 */
export function compareByAddedAtDesc(left: Movie, right: Movie): number {
  return compareAddedAtDesc(left.addedAt, right.addedAt)
}

/** 按有效评分（0–5）高到低；同分再以入库时间新到旧。 */
export function compareByRatingDesc(left: Movie, right: Movie): number {
  if (right.rating !== left.rating) {
    return right.rating - left.rating
  }
  return compareByAddedAtDesc(left, right)
}

/**
 * 按发行日（YYYY-MM-DD）新到旧；无发行日的条目排在后面，再以 addedAt 新到旧。
 */
export function compareByReleaseDateDesc(left: Movie, right: Movie): number {
  const a = (left.releaseDate ?? "").trim()
  const b = (right.releaseDate ?? "").trim()
  if (a && b) {
    return b.localeCompare(a) || compareByAddedAtDesc(left, right)
  }
  if (a && !b) {
    return -1
  }
  if (!a && b) {
    return 1
  }
  return compareByAddedAtDesc(left, right)
}

function compareTextAsc(left: string, right: string): number {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: "base" })
}

/** Empty values sort after non-empty, then stable by id. */
function compareOptionalTextAsc(left: string, right: string, tieBreak: () => number): number {
  const a = left.trim()
  const b = right.trim()
  if (a && b) {
    return compareTextAsc(a, b) || tieBreak()
  }
  if (a && !b) {
    return -1
  }
  if (!a && b) {
    return 1
  }
  return tieBreak()
}

/** 按番号（code）升序；空番号靠后。 */
export function compareByCodeAsc(left: Movie, right: Movie): number {
  return compareOptionalTextAsc(left.code, right.code, () => left.id.localeCompare(right.id))
}

/** 按首位演员名升序；无演员靠后。 */
export function compareByActorAsc(left: Movie, right: Movie): number {
  return compareOptionalTextAsc(
    left.actors[0] ?? "",
    right.actors[0] ?? "",
    () => compareByCodeAsc(left, right),
  )
}

/** 按厂商升序；空厂商靠后。 */
export function compareByStudioAsc(left: Movie, right: Movie): number {
  return compareOptionalTextAsc(left.studio, right.studio, () => compareByCodeAsc(left, right))
}

/** 按年份新到旧；无效年份靠后。 */
export function compareByYearDesc(left: Movie, right: Movie): number {
  const leftValid = Number.isInteger(left.year) && left.year >= 1800 && left.year <= 3000
  const rightValid = Number.isInteger(right.year) && right.year >= 1800 && right.year <= 3000
  if (leftValid && rightValid && left.year !== right.year) {
    return right.year - left.year
  }
  if (leftValid && !rightValid) {
    return -1
  }
  if (!leftValid && rightValid) {
    return 1
  }
  return compareByAddedAtDesc(left, right)
}

export function compareMoviesByLibrarySort(
  left: Movie,
  right: Movie,
  sort: LibrarySortKey,
): number {
  switch (sort) {
    case "release":
      return compareByReleaseDateDesc(left, right)
    case "rating":
      return compareByRatingDesc(left, right)
    case "code":
      return compareByCodeAsc(left, right)
    case "actor":
      return compareByActorAsc(left, right)
    case "studio":
      return compareByStudioAsc(left, right)
    case "year":
      return compareByYearDesc(left, right)
    case "added":
    default:
      return compareByAddedAtDesc(left, right)
  }
}

export function librarySortKeyFromTab(tab: "all" | "new" | "top-rated"): LibrarySortKey {
  switch (tab) {
    case "new":
      return "release"
    case "top-rated":
      return "rating"
    default:
      return "added"
  }
}

export function libraryTabFromSortKey(sort: LibrarySortKey): "all" | "new" | "top-rated" | "none" {
  switch (sort) {
    case "release":
      return "new"
    case "rating":
      return "top-rated"
    case "added":
      return "all"
    default:
      return "none"
  }
}
