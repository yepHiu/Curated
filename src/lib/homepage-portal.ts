import type { Movie } from "@/domain/movie/types"
import { compareByAddedAtDesc } from "@/lib/movie-sort"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"
import { hashStringToUint32, mulberry32 } from "@/lib/random-sample"

export interface HomepageReason {
  kind: "actor" | "tag" | "studio" | "favorite" | "rating"
  label: string
}

export interface HomepageRecommendationEntry {
  movie: Movie
  reasons: HomepageReason[]
  score: number
}

export interface HomepageContinueEntry {
  movie: Movie
  progressPercent: number
  remainingMinutes: number
  updatedAt: string
}

export interface HomepageTasteEntry {
  kind: "actor" | "tag" | "studio"
  label: string
  weight: number
}

export interface HomepagePortalModel {
  heroMovies: Movie[]
  recentMovies: Movie[]
  recommendations: HomepageRecommendationEntry[]
  continueWatching: HomepageContinueEntry[]
  tasteRadar: HomepageTasteEntry[]
}

export interface BuildHomepagePortalInput {
  movies: readonly Movie[]
  playbackEntries?: readonly PlaybackProgressEntry[]
  daySeed?: string
  dailyRecommendations?: HomepageDailyRecommendationsSelection
  heroLimit?: number
  recentLimit?: number
  recommendationLimit?: number
  continueLimit?: number
  tasteLimitPerKind?: number
}

export interface HomepageDailyRecommendationsSelection {
  heroMovieIds: readonly string[]
  recommendationMovieIds: readonly string[]
}

function stableDateSeed(daySeed?: string): string {
  if (daySeed && daySeed.trim()) {
    return daySeed.trim()
  }
  return new Date().toISOString().slice(0, 10)
}

function toTimestamp(value: string | undefined): number {
  if (!value) return 0
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? 0 : parsed
}

function buildPlaybackMap(
  playbackEntries: readonly PlaybackProgressEntry[],
): Map<string, PlaybackProgressEntry> {
  const next = new Map<string, PlaybackProgressEntry>()
  for (const row of playbackEntries) {
    const id = row.movieId.trim()
    if (!id) continue
    const current = next.get(id)
    if (!current || toTimestamp(row.updatedAt) > toTimestamp(current.updatedAt)) {
      next.set(id, row)
    }
  }
  return next
}

function seededHeroOrder(movies: readonly Movie[], daySeed: string): Movie[] {
  const rng = mulberry32(hashStringToUint32(daySeed))
  return [...movies]
    .map((movie, index) => {
      const quality =
        (movie.isFavorite ? 24 : 0) +
        (typeof movie.userRating === "number" ? movie.userRating : movie.rating) * 10 +
        (movie.coverUrl || movie.thumbUrl ? 6 : 0) +
        Math.max(0, 2026 - movie.year)
      return {
        movie,
        index,
        score: quality + rng() * 14,
      }
    })
    .sort((left, right) => {
      if (right.score !== left.score) return right.score - left.score
      return left.index - right.index
    })
    .map((entry) => entry.movie)
}

function isFC2MovieCode(code: string | undefined): boolean {
  const normalized = (code ?? "")
    .trim()
    .toUpperCase()
    .replace(/[\s_-]+/g, "")
  return normalized.startsWith("FC2")
}

function fillHeroMovies(movies: readonly Movie[], limit: number): Movie[] {
  if (limit <= 0 || movies.length === 0) return []
  if (movies.length >= limit) return movies.slice(0, limit)

  const filled: Movie[] = []
  while (filled.length < limit) {
    for (const movie of movies) {
      filled.push(movie)
      if (filled.length >= limit) return filled
    }
  }
  return filled
}

function buildPreferenceWeights(
  movies: readonly Movie[],
  playbackByMovieId: ReadonlyMap<string, PlaybackProgressEntry>,
): {
  actors: Map<string, number>
  tags: Map<string, number>
  studios: Map<string, number>
} {
  const actors = new Map<string, number>()
  const tags = new Map<string, number>()
  const studios = new Map<string, number>()

  const add = (map: Map<string, number>, key: string, weight: number) => {
    const normalized = key.trim()
    if (!normalized) return
    map.set(normalized, (map.get(normalized) ?? 0) + weight)
  }

  for (const movie of movies) {
    const playback = playbackByMovieId.get(movie.id)
    let weight = 0

    if (movie.isFavorite) weight += 24
    if (typeof movie.userRating === "number") weight += movie.userRating * 12
    weight += movie.rating * 7
    weight += movie.userTags.length * 6
    if (playback) weight += 18

    if (weight <= 0) continue

    for (const actor of movie.actors) add(actors, actor, weight)
    for (const tag of [...movie.tags, ...movie.userTags]) add(tags, tag, weight * 0.72)
    add(studios, movie.studio, weight * 0.8)
  }

  return { actors, tags, studios }
}

function topEntries(
  kind: HomepageTasteEntry["kind"],
  map: ReadonlyMap<string, number>,
  limit: number,
): HomepageTasteEntry[] {
  return [...map.entries()]
    .sort((left, right) => {
      if (right[1] !== left[1]) return right[1] - left[1]
      return left[0].localeCompare(right[0])
    })
    .slice(0, limit)
    .map(([label, weight]) => ({ kind, label, weight }))
}

function pickMoviesBySnapshotIds(
  ids: readonly string[],
  movieById: ReadonlyMap<string, Movie>,
  limit: number,
  excludedIds: Set<string>,
): Movie[] {
  const ordered: Movie[] = []
  for (const rawId of ids) {
    if (ordered.length >= limit) break
    const id = rawId.trim()
    if (!id || excludedIds.has(id)) continue
    const movie = movieById.get(id)
    if (!movie) continue
    ordered.push(movie)
    excludedIds.add(id)
  }
  return ordered
}

function withHomepageDailyRecommendations(
  model: HomepagePortalModel,
  activeMovies: readonly Movie[],
  selection: HomepageDailyRecommendationsSelection | undefined,
): HomepagePortalModel {
  if (!selection) {
    return model
  }

  const movieById = new Map(activeMovies.map((movie) => [movie.id, movie] as const))
  const selectedHeroIds = new Set<string>()
  const heroMovies = pickMoviesBySnapshotIds(
    selection.heroMovieIds,
    movieById,
    model.heroMovies.length,
    selectedHeroIds,
  )

  if (heroMovies.length === 0 && selection.heroMovieIds.length > 0) {
    heroMovies.push(...model.heroMovies)
    for (const movie of model.heroMovies) {
      selectedHeroIds.add(movie.id)
    }
  } else {
    for (const movie of model.heroMovies) {
      if (heroMovies.length >= model.heroMovies.length) break
      if (selectedHeroIds.has(movie.id)) continue
      heroMovies.push(movie)
      selectedHeroIds.add(movie.id)
    }
  }

  const selectedRecommendationIds = new Set(selectedHeroIds)
  const recommendationMovies = pickMoviesBySnapshotIds(
    selection.recommendationMovieIds,
    movieById,
    model.recommendations.length,
    selectedRecommendationIds,
  )

  const fallbackRecommendations = new Map(
    model.recommendations.map((entry) => [entry.movie.id, entry] as const),
  )
  const recommendations: HomepageRecommendationEntry[] = recommendationMovies.map((movie) => (
    fallbackRecommendations.get(movie.id) ?? {
      movie,
      reasons: [],
      score: movie.rating * 10,
    }
  ))

  if (recommendations.length === 0 && selection.recommendationMovieIds.length > 0) {
    recommendations.push(...model.recommendations)
  } else {
    for (const entry of model.recommendations) {
      if (recommendations.length >= model.recommendations.length) break
      if (selectedRecommendationIds.has(entry.movie.id)) continue
      recommendations.push(entry)
      selectedRecommendationIds.add(entry.movie.id)
    }
  }

  return {
    ...model,
    heroMovies,
    recommendations,
  }
}

export function buildHomepagePortalModel({
  movies,
  playbackEntries = [],
  daySeed,
  dailyRecommendations,
  heroLimit = 8,
  recentLimit = 6,
  recommendationLimit = 6,
  continueLimit = 4,
  tasteLimitPerKind = 3,
}: BuildHomepagePortalInput): HomepagePortalModel {
  const activeMovies = movies.filter((movie) => !movie.trashedAt?.trim())
  const playbackByMovieId = buildPlaybackMap(playbackEntries)
  const heroPool = activeMovies.filter((movie) => !isFC2MovieCode(movie.code))

  const heroMovies = fillHeroMovies(
    seededHeroOrder(heroPool, stableDateSeed(daySeed)),
    heroLimit,
  )

  const recentMovies = [...activeMovies].sort(compareByAddedAtDesc).slice(0, recentLimit)

  const continueWatching = playbackEntries
    .map((entry) => {
      const movie = activeMovies.find((candidate) => candidate.id === entry.movieId)
      if (!movie) return undefined
      if (!Number.isFinite(entry.durationSec) || entry.durationSec <= 0) return undefined
      if (!Number.isFinite(entry.positionSec) || entry.positionSec <= 0) return undefined

      const progressPercent = Math.round((entry.positionSec / entry.durationSec) * 100)
      if (progressPercent >= 95) return undefined

      return {
        movie,
        progressPercent,
        remainingMinutes: Math.max(
          1,
          Math.ceil((entry.durationSec - entry.positionSec) / 60),
        ),
        updatedAt: entry.updatedAt,
      } satisfies HomepageContinueEntry
    })
    .filter((entry): entry is HomepageContinueEntry => Boolean(entry))
    .sort((left, right) => toTimestamp(right.updatedAt) - toTimestamp(left.updatedAt))
    .slice(0, continueLimit)

  const preference = buildPreferenceWeights(activeMovies, playbackByMovieId)
  const heroIds = new Set(heroMovies.map((movie) => movie.id))
  const continueIds = new Set(continueWatching.map((entry) => entry.movie.id))

  const recommendationPool = activeMovies.filter(
    (movie) => !heroIds.has(movie.id) && !continueIds.has(movie.id),
  )
  const fallbackPool = recommendationPool.length > 0 ? recommendationPool : activeMovies

  const recommendations = fallbackPool
    .map((movie) => {
      const actorMatches = movie.actors
        .map((actor) => ({ actor, weight: preference.actors.get(actor.trim()) ?? 0 }))
        .sort((left, right) => right.weight - left.weight)
      const tagMatches = [...movie.tags, ...movie.userTags]
        .map((tag) => ({ tag, weight: preference.tags.get(tag.trim()) ?? 0 }))
        .sort((left, right) => right.weight - left.weight)
      const studioWeight = preference.studios.get(movie.studio.trim()) ?? 0

      let score = movie.rating * 10
      score += actorMatches.reduce((sum, entry) => sum + entry.weight, 0)
      score += tagMatches.reduce((sum, entry) => sum + entry.weight, 0)
      score += studioWeight
      if (movie.isFavorite) score -= 18
      if (playbackByMovieId.has(movie.id)) score -= 12

      const reasons: HomepageReason[] = []
      if (actorMatches[0] && actorMatches[0].weight > 0) {
        reasons.push({ kind: "actor", label: actorMatches[0].actor })
      }
      if (tagMatches[0] && tagMatches[0].weight > 0) {
        reasons.push({ kind: "tag", label: tagMatches[0].tag })
      }
      if (studioWeight > 0) {
        reasons.push({ kind: "studio", label: movie.studio })
      }
      if (movie.isFavorite) {
        reasons.push({ kind: "favorite", label: "favorite" })
      }
      if (movie.rating >= 4.7) {
        reasons.push({ kind: "rating", label: String(movie.rating) })
      }

      return {
        movie,
        score,
        reasons,
      } satisfies HomepageRecommendationEntry
    })
    .sort((left, right) => {
      if (right.score !== left.score) return right.score - left.score
      return compareByAddedAtDesc(left.movie, right.movie)
    })
    .slice(0, recommendationLimit)

  const tasteRadar = [
    ...topEntries("actor", preference.actors, tasteLimitPerKind),
    ...topEntries("tag", preference.tags, tasteLimitPerKind),
    ...topEntries("studio", preference.studios, tasteLimitPerKind),
  ]

  return withHomepageDailyRecommendations({
    heroMovies,
    recentMovies,
    recommendations,
    continueWatching,
    tasteRadar,
  }, activeMovies, dailyRecommendations)
}
