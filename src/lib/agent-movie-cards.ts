import type { AIAgentMovieCardDTO } from "@/api/types"

export function parsePresentMoviesContent(content: string): AIAgentMovieCardDTO[] {
  const trimmed = content.trim()
  if (!trimmed.startsWith("{")) return []
  try {
    const parsed = JSON.parse(trimmed) as { movies?: unknown }
    if (!Array.isArray(parsed.movies)) return []
    return parsed.movies.flatMap((item) => {
      if (!item || typeof item !== "object") return []
      const row = item as AIAgentMovieCardDTO
      const movieId = typeof row.movieId === "string" ? row.movieId.trim() : ""
      if (!movieId) return []
      return [{
        movieId,
        title: typeof row.title === "string" ? row.title : undefined,
        code: typeof row.code === "string" ? row.code : undefined,
        actors: Array.isArray(row.actors) ? row.actors.filter((name): name is string => typeof name === "string") : undefined,
        coverUrl: typeof row.coverUrl === "string" ? row.coverUrl : undefined,
        thumbUrl: typeof row.thumbUrl === "string" ? row.thumbUrl : undefined,
        reason: typeof row.reason === "string" ? row.reason : undefined,
      }]
    })
  } catch {
    return []
  }
}
