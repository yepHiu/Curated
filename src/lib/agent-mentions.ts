import type { Movie } from "@/domain/movie/types"

export type AgentMentionKind = "movie" | "actor" | "tag" | "comic" | "photo"

export interface AgentMention {
  kind: AgentMentionKind
  id: string
  label: string
}

const MENTION_AT = /(?:^|\s)@([^\s@]*)$/

export function mentionQueryAtCursor(text: string, cursor: number): { start: number; query: string } | null {
  const prefix = text.slice(0, Math.max(0, cursor))
  const match = prefix.match(MENTION_AT)
  if (!match || match.index === undefined) return null
  const at = prefix.lastIndexOf("@")
  return { start: at, query: match[1] ?? "" }
}

export function applyMention(text: string, cursor: number, mention: AgentMention): { text: string; cursor: number } {
  const token = `@${mention.label} `
  const found = mentionQueryAtCursor(text, cursor)
  if (!found) {
    const next = `${text}${token}`
    return { text: next, cursor: next.length }
  }
  const next = `${text.slice(0, found.start)}${token}${text.slice(cursor)}`
  return { text: next, cursor: found.start + token.length }
}

export function mentionsStillInText(mentions: AgentMention[], text: string): AgentMention[] {
  return mentions.filter((item) => text.includes(`@${item.label}`))
}

/** Bare `@` stays compact; typing a query can surface a few more hits. */
export function mentionPickerLimit(query: string): number {
  return query.trim() ? 4 : 2
}

export function searchMovieMentions(movies: readonly Movie[], query: string, limit = 6): AgentMention[] {
  const q = query.trim().toLowerCase()
  const hits = movies.filter((movie) => {
    if (movie.trashedAt) return false
    if (!q) return true
    const haystack = [movie.title, movie.code, ...movie.actors].join(" ").toLowerCase()
    return haystack.includes(q)
  })
  return hits.slice(0, limit).map((movie) => ({
    kind: "movie",
    id: movie.id,
    label: movie.title || movie.code || movie.id,
  }))
}

export function searchActorMentions(
  actors: readonly { name: string }[],
  query: string,
  limit = 6,
): AgentMention[] {
  const q = query.trim().toLowerCase()
  const hits = actors.filter((actor) => {
    const name = actor.name.trim()
    if (!name) return false
    if (!q) return true
    return name.toLowerCase().includes(q)
  })
  return hits.slice(0, limit).map((actor) => ({
    kind: "actor",
    id: actor.name,
    label: actor.name,
  }))
}

export function searchTagMentions(movies: readonly Movie[], query: string, limit = 6): AgentMention[] {
  const q = query.trim().toLowerCase()
  const seen = new Set<string>()
  const tags: string[] = []
  for (const movie of movies) {
    for (const tag of [...(movie.tags ?? []), ...(movie.userTags ?? [])]) {
      const value = tag.trim()
      if (!value) continue
      const key = value.toLowerCase()
      if (seen.has(key)) continue
      if (q && !key.includes(q)) continue
      seen.add(key)
      tags.push(value)
      if (tags.length >= limit) {
        return tags.map((label) => ({ kind: "tag", id: label, label }))
      }
    }
  }
  return tags.map((label) => ({ kind: "tag", id: label, label }))
}

/** 从已加载漫画列表生成 @ 提及候选项。 */
export function searchComicMentions(
  comics: readonly { id: string; title?: string; tags?: string[] }[],
  query: string,
  limit = 6,
): AgentMention[] {
  const q = query.trim().toLowerCase()
  const hits = comics.filter((comic) => {
    if (!comic.id.trim()) return false
    if (!q) return true
    const haystack = [comic.title ?? "", ...(comic.tags ?? [])].join(" ").toLowerCase()
    return haystack.includes(q)
  })
  return hits.slice(0, limit).map((comic) => ({
    kind: "comic",
    id: comic.id,
    label: comic.title?.trim() || comic.id,
  }))
}

/** 从已加载写真列表生成 @ 提及候选项。 */
export function searchPhotoMentions(
  photos: readonly { id: string; title?: string; tags?: string[] }[],
  query: string,
  limit = 6,
): AgentMention[] {
  const q = query.trim().toLowerCase()
  const hits = photos.filter((photo) => {
    if (!photo.id.trim()) return false
    if (!q) return true
    const haystack = [photo.title ?? "", ...(photo.tags ?? [])].join(" ").toLowerCase()
    return haystack.includes(q)
  })
  return hits.slice(0, limit).map((photo) => ({
    kind: "photo",
    id: photo.id,
    label: photo.title?.trim() || photo.id,
  }))
}
