import type { AIAgentBookCardDTO } from "@/api/types"

/** 从历史工具正文还原本轮书卡，忽略缺少 kind/id 的残缺行。 */
export function parsePresentBooksContent(content: string): AIAgentBookCardDTO[] {
  const trimmed = content.trim()
  if (!trimmed.startsWith("{")) return []
  try {
    const parsed = JSON.parse(trimmed) as { books?: unknown }
    if (!Array.isArray(parsed.books)) return []
    return parsed.books.flatMap((item) => {
      if (!item || typeof item !== "object") return []
      const row = item as AIAgentBookCardDTO
      const kind = row.kind === "comic" || row.kind === "photo" ? row.kind : ""
      const comicId = typeof row.comicId === "string" ? row.comicId.trim() : ""
      const photoId = typeof row.photoId === "string" ? row.photoId.trim() : ""
      if (!kind || (kind === "comic" && !comicId) || (kind === "photo" && !photoId)) return []
      return [{
        kind,
        comicId: comicId || undefined,
        photoId: photoId || undefined,
        title: typeof row.title === "string" ? row.title : undefined,
        coverUrl: typeof row.coverUrl === "string" ? row.coverUrl : undefined,
        tags: Array.isArray(row.tags) ? row.tags.filter((tag): tag is string => typeof tag === "string") : undefined,
      }]
    })
  } catch {
    return []
  }
}
