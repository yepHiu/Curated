import type { Movie } from "@/domain/movie/types"

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
