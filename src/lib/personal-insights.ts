import type {
  PersonalInsightsBreakdownDTO,
  PersonalInsightsBreakdownItemDTO,
  PersonalInsightsDimension,
  PersonalInsightsOverviewDTO,
  PersonalInsightsRange,
} from "@/api/types"
import type { Movie } from "@/domain/movie/types"
import { normalizeActorIdentity } from "@/lib/actor-identity"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"
import type { PlaybackWatchTimeMovieEntry } from "@/lib/playback-watch-time-storage"

const COMPLETION_THRESHOLD = 0.9
const DEFAULT_BREAKDOWN_LIMIT = 10
const MAX_BREAKDOWN_LIMIT = 25

export interface PersonalInsightsSource {
  movies: readonly Movie[]
  progressEntries: readonly PlaybackProgressEntry[]
  watchTimeEntries: readonly PlaybackWatchTimeMovieEntry[]
  range: PersonalInsightsRange
  timezone: string
  now?: Date
}

interface ResolvedInsightsSource {
  moviesById: Map<string, Movie>
  progressByMovieId: Map<string, PlaybackProgressEntry>
  watchedByMovieId: Map<string, number>
  range: PersonalInsightsRange
  timezone: string
  from: string
  to: string
  generatedAt: string
  dataSince: string | null
}

interface MutableBreakdownItem {
  name: string
  watchedSeconds: number
  movieIds: Set<string>
}

function localDayKeyInTimezone(now: Date, timezone: string): string {
  if (!Number.isFinite(now.getTime())) {
    throw new Error("personal insights current time is invalid")
  }
  let formatter: Intl.DateTimeFormat
  try {
    formatter = new Intl.DateTimeFormat("en-CA", {
      timeZone: timezone,
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    })
  } catch {
    throw new Error(`personal insights timezone is invalid: ${timezone}`)
  }
  const parts = formatter.formatToParts(now)
  const year = parts.find((part) => part.type === "year")?.value
  const month = parts.find((part) => part.type === "month")?.value
  const day = parts.find((part) => part.type === "day")?.value
  if (!year || !month || !day) {
    throw new Error(`personal insights timezone is invalid: ${timezone}`)
  }
  return `${year}-${month}-${day}`
}

function subtractCalendarDays(dayKey: string, count: number): string {
  const [year, month, day] = dayKey.split("-").map(Number)
  const date = new Date(Date.UTC(year ?? 0, (month ?? 1) - 1, day ?? 1))
  date.setUTCDate(date.getUTCDate() - count)
  return date.toISOString().slice(0, 10)
}

function rangeDays(range: PersonalInsightsRange): number | null {
  switch (range) {
    case "30d":
      return 30
    case "90d":
      return 90
    case "365d":
      return 365
    case "all":
      return null
  }
}

function normalizeEntityName(value: string): string {
  return value.normalize("NFKC").trim().replace(/\s+/gu, " ")
}

function normalizedEntityKey(value: string): string {
  return normalizeEntityName(value).toUpperCase().toLowerCase()
}

function compareStableNames(left: string, right: string): number {
  const leftFolded = left.toUpperCase().toLowerCase()
  const rightFolded = right.toUpperCase().toLowerCase()
  if (leftFolded < rightFolded) return -1
  if (leftFolded > rightFolded) return 1
  if (left < right) return -1
  if (left > right) return 1
  return 0
}

function resolveSource(source: PersonalInsightsSource): ResolvedInsightsSource {
  const now = source.now ?? new Date()
  const to = localDayKeyInTimezone(now, source.timezone)
  const normalizedRows = source.watchTimeEntries.filter(
    (row) =>
      row.dayKey <= to &&
      row.movieId.trim() !== "" &&
      Number.isFinite(row.watchedSec) &&
      row.watchedSec > 0,
  )
  const dataSince = normalizedRows.reduce<string | null>(
    (earliest, row) => (earliest === null || row.dayKey < earliest ? row.dayKey : earliest),
    null,
  )
  const days = rangeDays(source.range)
  const from = days === null ? (dataSince ?? to) : subtractCalendarDays(to, days - 1)
  const watchedByMovieId = new Map<string, number>()
  for (const row of normalizedRows) {
    if (row.dayKey < from) continue
    const movieId = row.movieId.trim()
    watchedByMovieId.set(movieId, (watchedByMovieId.get(movieId) ?? 0) + row.watchedSec)
  }
  const moviesById = new Map(
    source.movies
      .filter((movie) => movie.id.trim() !== "")
      .map((movie) => [movie.id.trim(), movie] as const),
  )
  for (const movieId of [...watchedByMovieId.keys()]) {
    if (!moviesById.has(movieId)) watchedByMovieId.delete(movieId)
  }
  const progressByMovieId = new Map(
    source.progressEntries
      .filter((entry) => entry.movieId.trim() !== "")
      .map((entry) => [entry.movieId.trim(), entry] as const),
  )
  return {
    moviesById,
    progressByMovieId,
    watchedByMovieId,
    range: source.range,
    timezone: source.timezone,
    from,
    to,
    generatedAt: now.toISOString(),
    dataSince,
  }
}

export function buildPersonalInsightsOverview(source: PersonalInsightsSource): PersonalInsightsOverviewDTO {
  const resolved = resolveSource(source)
  let watchedSeconds = 0
  let completedMovies = 0
  const ratings: number[] = []
  for (const [movieId, watched] of resolved.watchedByMovieId) {
    watchedSeconds += watched
    const progress = resolved.progressByMovieId.get(movieId)
    if (
      progress &&
      Number.isFinite(progress.durationSec) &&
      progress.durationSec > 0 &&
      Number.isFinite(progress.positionSec) &&
      progress.positionSec >= progress.durationSec * COMPLETION_THRESHOLD
    ) {
      completedMovies += 1
    }
    const rating = resolved.moviesById.get(movieId)?.userRating
    if (typeof rating === "number" && Number.isFinite(rating)) ratings.push(rating)
  }
  const startedMovies = resolved.watchedByMovieId.size
  const averageUserRating = ratings.length > 0
    ? ratings.reduce((sum, rating) => sum + rating, 0) / ratings.length
    : null
  return {
    range: resolved.range,
    from: resolved.from,
    to: resolved.to,
    timezone: resolved.timezone,
    generatedAt: resolved.generatedAt,
    dataSince: resolved.dataSince,
    watchedSeconds,
    startedMovies,
    completedMovies,
    completionRate: startedMovies > 0 ? completedMovies / startedMovies : null,
    completionThreshold: COMPLETION_THRESHOLD,
    ratedMovies: ratings.length,
    averageUserRating,
  }
}

function entitiesForMovie(movie: Movie, dimension: PersonalInsightsDimension): Array<{ key: string; name: string }> {
  const rawValues = dimension === "actor"
    ? movie.actors
    : dimension === "studio"
      ? [movie.studio]
      : [...movie.tags, ...movie.userTags]
  const entities = new Map<string, string>()
  for (const rawValue of rawValues) {
    const name = normalizeEntityName(rawValue)
    if (!name) continue
    const key = dimension === "actor" ? normalizeActorIdentity(name) : normalizedEntityKey(name)
    if (!key) continue
    const current = entities.get(key)
    if (current === undefined || compareStableNames(name, current) < 0) entities.set(key, name)
  }
  return [...entities].map(([key, name]) => ({ key, name }))
}

export function buildPersonalInsightsBreakdown(
  source: PersonalInsightsSource & { dimension: PersonalInsightsDimension; limit?: number },
): PersonalInsightsBreakdownDTO {
  const resolved = resolveSource(source)
  const limit = source.limit ?? DEFAULT_BREAKDOWN_LIMIT
  if (!Number.isInteger(limit) || limit < 1 || limit > MAX_BREAKDOWN_LIMIT) {
    throw new Error(`personal insights limit must be between 1 and ${MAX_BREAKDOWN_LIMIT}`)
  }
  const aggregates = new Map<string, MutableBreakdownItem>()
  let totalWatchedSeconds = 0
  for (const [movieId, watchedSeconds] of resolved.watchedByMovieId) {
    totalWatchedSeconds += watchedSeconds
    const movie = resolved.moviesById.get(movieId)
    if (!movie) continue
    for (const entity of entitiesForMovie(movie, source.dimension)) {
      const current = aggregates.get(entity.key)
      if (current) {
        current.watchedSeconds += watchedSeconds
        current.movieIds.add(movieId)
        if (compareStableNames(entity.name, current.name) < 0) current.name = entity.name
      } else {
        aggregates.set(entity.key, {
          name: entity.name,
          watchedSeconds,
          movieIds: new Set([movieId]),
        })
      }
    }
  }
  const items: PersonalInsightsBreakdownItemDTO[] = [...aggregates.values()]
    .map((item) => ({
      name: item.name,
      watchedSeconds: item.watchedSeconds,
      movieCount: item.movieIds.size,
      shareOfTotal: totalWatchedSeconds > 0 ? item.watchedSeconds / totalWatchedSeconds : 0,
    }))
    .filter((item) => Number.isFinite(item.shareOfTotal))
    .sort(
      (left, right) =>
        right.watchedSeconds - left.watchedSeconds ||
        right.movieCount - left.movieCount ||
        compareStableNames(left.name, right.name),
    )
    .slice(0, limit)
  return {
    range: resolved.range,
    dimension: source.dimension,
    from: resolved.from,
    to: resolved.to,
    timezone: resolved.timezone,
    generatedAt: resolved.generatedAt,
    dataSince: resolved.dataSince,
    totalWatchedSeconds,
    attribution: "full-per-entity",
    items,
    limit,
  }
}
