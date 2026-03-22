import type { Movie } from "@/domain/movie/types"

/** 番号宽松比较：小写并去掉空格、连字符、下划线，便于 `mkb100` 命中 `MKB-100` */
export function normalizeLooseCode(s: string): string {
  return s.toLowerCase().replace(/[\s\-_]/g, "")
}

/** 供库页搜索：一次性拼成可 match 的小写串，避免多处重复 join 逻辑 */
export function movieSearchHaystack(movie: Movie): string {
  const loose = movie.code?.trim() ? normalizeLooseCode(movie.code) : ""
  const base = `${movie.title} ${movie.code} ${movie.studio} ${movie.actors.join(" ")} ${movie.tags.join(" ")} ${movie.userTags.join(" ")}`
  const withLoose = loose ? `${base} ${loose}` : base
  return withLoose.trim().toLowerCase()
}
