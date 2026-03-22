import type { Movie } from "@/domain/movie/types"
import { normalizeLooseCode } from "@/lib/movie-search"

export type ActorSearchSuggestion = { kind: "actor"; canonical: string }
export type TagSearchSuggestion = { kind: "tag"; canonical: string }
export type CodeSearchSuggestion = { kind: "code"; code: string; movieId: string }

export type LibrarySearchSuggestion =
  | ActorSearchSuggestion
  | TagSearchSuggestion
  | CodeSearchSuggestion

export interface LibrarySearchSuggestionGroups {
  actors: ActorSearchSuggestion[]
  tags: TagSearchSuggestion[]
  codes: CodeSearchSuggestion[]
}

export const DEFAULT_SUGGESTION_LIMITS = {
  actor: 10,
  tag: 10,
  code: 10,
} as const

function scoreTextMatch(label: string, needleLower: string): number {
  const l = label.toLowerCase()
  if (l.startsWith(needleLower)) {
    return 0
  }
  if (l.includes(needleLower)) {
    return 1
  }
  return 2
}

function sortStringsByRelevance(candidates: string[], needleLower: string): string[] {
  return [...candidates].sort((a, b) => {
    const da = scoreTextMatch(a, needleLower)
    const db = scoreTextMatch(b, needleLower)
    if (da !== db) {
      return da - db
    }
    return a.localeCompare(b)
  })
}

function rankCodeMatch(code: string, needleLower: string, needleLoose: string): number {
  const low = code.toLowerCase()
  const nl = normalizeLooseCode(code)
  if (low.startsWith(needleLower)) {
    return 0
  }
  if (needleLoose && nl.startsWith(needleLoose)) {
    return 1
  }
  if (low.includes(needleLower)) {
    return 2
  }
  if (needleLoose && nl.includes(needleLoose)) {
    return 3
  }
  return 4
}

function sortCodesByRelevance(
  entries: [string, string][],
  needleLower: string,
  needleLoose: string,
): [string, string][] {
  return [...entries].sort(([a], [b]) => {
    const ra = rankCodeMatch(a, needleLower, needleLoose)
    const rb = rankCodeMatch(b, needleLower, needleLoose)
    if (ra !== rb) {
      return ra - rb
    }
    return a.localeCompare(b)
  })
}

/** 从已缓存影片聚合演员 / 标签 / 番号联想（trim 为空则返回空分组） */
export function buildLibrarySearchSuggestions(
  needle: string,
  movies: readonly Movie[],
  limits: Partial<Record<keyof typeof DEFAULT_SUGGESTION_LIMITS, number>> = {},
): LibrarySearchSuggestionGroups {
  const n = needle.trim()
  if (!n) {
    return { actors: [], tags: [], codes: [] }
  }

  const needleLower = n.toLowerCase()
  const needleLoose = normalizeLooseCode(n)

  const capActor = limits.actor ?? DEFAULT_SUGGESTION_LIMITS.actor
  const capTag = limits.tag ?? DEFAULT_SUGGESTION_LIMITS.tag
  const capCode = limits.code ?? DEFAULT_SUGGESTION_LIMITS.code

  const actorByLower = new Map<string, string>()
  const tagSet = new Set<string>()
  const codeToMovieId = new Map<string, string>()

  for (const m of movies) {
    for (const raw of m.actors) {
      const name = raw.trim()
      if (!name) {
        continue
      }
      const k = name.toLowerCase()
      if (!actorByLower.has(k)) {
        actorByLower.set(k, name)
      }
    }
    for (const t of m.tags) {
      const x = t.trim()
      if (x) {
        tagSet.add(x)
      }
    }
    for (const t of m.userTags) {
      const x = t.trim()
      if (x) {
        tagSet.add(x)
      }
    }
    const code = m.code?.trim()
    if (code && !codeToMovieId.has(code)) {
      codeToMovieId.set(code, m.id)
    }
  }

  const actorMatches = [...actorByLower.values()].filter((name) =>
    name.toLowerCase().includes(needleLower),
  )
  const actors = sortStringsByRelevance(actorMatches, needleLower)
    .slice(0, capActor)
    .map((canonical) => ({ kind: "actor" as const, canonical }))

  const tagMatches = [...tagSet].filter((tag) => tag.toLowerCase().includes(needleLower))
  const tags = sortStringsByRelevance(tagMatches, needleLower)
    .slice(0, capTag)
    .map((canonical) => ({ kind: "tag" as const, canonical }))

  const codeEntries = [...codeToMovieId.entries()].filter(([code]) => {
    const low = code.toLowerCase()
    if (low.includes(needleLower)) {
      return true
    }
    if (!needleLoose) {
      return false
    }
    return normalizeLooseCode(code).includes(needleLoose)
  })

  const codes = sortCodesByRelevance(codeEntries, needleLower, needleLoose)
    .slice(0, capCode)
    .map(([code, movieId]) => ({ kind: "code" as const, code, movieId }))

  return { actors, tags, codes }
}

export function librarySearchSuggestionsHasAny(groups: LibrarySearchSuggestionGroups): boolean {
  return groups.actors.length > 0 || groups.tags.length > 0 || groups.codes.length > 0
}
