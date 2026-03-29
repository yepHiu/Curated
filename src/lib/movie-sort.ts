import type { Movie } from "@/domain/movie/types"

/** 按入库时间（addedAt）新到旧。 */
export function compareByAddedAtDesc(left: Movie, right: Movie): number {
  return right.addedAt.localeCompare(left.addedAt)
}

/** 按有效评分（0–5）高到低；同分再以入库时间新到旧。 */
export function compareByRatingDesc(left: Movie, right: Movie): number {
  if (right.rating !== left.rating) {
    return right.rating - left.rating
  }
  return right.addedAt.localeCompare(left.addedAt)
}

/**
 * 按发行日（YYYY-MM-DD）新到旧；无发行日的条目排在后面，再以 addedAt 新到旧。
 */
export function compareByReleaseDateDesc(left: Movie, right: Movie): number {
  const a = (left.releaseDate ?? "").trim()
  const b = (right.releaseDate ?? "").trim()
  if (a && b) {
    return b.localeCompare(a)
  }
  if (a && !b) {
    return -1
  }
  if (!a && b) {
    return 1
  }
  return right.addedAt.localeCompare(left.addedAt)
}
