import type { Movie } from "@/domain/movie/types"

/** 供库页搜索：一次性拼成可 match 的小写串，避免多处重复 join 逻辑 */
export function movieSearchHaystack(movie: Movie): string {
  return `${movie.title} ${movie.code} ${movie.studio} ${movie.actors.join(" ")} ${movie.tags.join(" ")}`.toLowerCase()
}
