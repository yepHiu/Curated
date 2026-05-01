import type { HealthDTO, MovieDetailDTO, MovieListItemDTO, MoviesPageDTO } from "./types"

export class InvalidApiResponseError extends Error {
  readonly endpoint: string
  readonly value: unknown

  constructor(endpoint: string, value: unknown) {
    super(`Invalid API response for ${endpoint}`)
    this.name = "InvalidApiResponseError"
    this.endpoint = endpoint
    this.value = value
  }
}

type Guard<T> = (value: unknown) => value is T

export function assertApiResponse<T>(
  endpoint: string,
  value: unknown,
  guard: Guard<T>,
): T {
  if (!guard(value)) {
    throw new InvalidApiResponseError(endpoint, value)
  }
  return value
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function isString(value: unknown): value is string {
  return typeof value === "string"
}

function isBoolean(value: unknown): value is boolean {
  return typeof value === "boolean"
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value)
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every(isString)
}

function isOptionalString(value: unknown): boolean {
  return value === undefined || isString(value)
}

function isOptionalStringArray(value: unknown): boolean {
  return value === undefined || isStringArray(value)
}

function isOptionalNullableNumber(value: unknown): boolean {
  return value === undefined || value === null || isFiniteNumber(value)
}

function isOptionalStringRecord(value: unknown): boolean {
  if (value === undefined) {
    return true
  }
  if (!isRecord(value)) {
    return false
  }
  return Object.values(value).every(isString)
}

export function isHealthDTO(value: unknown): value is HealthDTO {
  return (
    isRecord(value) &&
    isString(value.name) &&
    isString(value.version) &&
    isString(value.transport) &&
    isString(value.databasePath) &&
    isOptionalString(value.channel) &&
    isOptionalString(value.installerVersion)
  )
}

export function isMovieListItemDTO(value: unknown): value is MovieListItemDTO {
  return (
    isRecord(value) &&
    isString(value.id) &&
    isString(value.title) &&
    isString(value.code) &&
    isString(value.studio) &&
    isStringArray(value.actors) &&
    isStringArray(value.tags) &&
    isOptionalStringArray(value.userTags) &&
    isFiniteNumber(value.runtimeMinutes) &&
    isFiniteNumber(value.rating) &&
    isBoolean(value.isFavorite) &&
    isString(value.addedAt) &&
    isString(value.location) &&
    isString(value.resolution) &&
    isFiniteNumber(value.year) &&
    isOptionalString(value.releaseDate) &&
    isOptionalString(value.coverUrl) &&
    isOptionalString(value.thumbUrl) &&
    isOptionalString(value.trashedAt)
  )
}

export function isMovieDetailDTO(value: unknown): value is MovieDetailDTO {
  if (!isMovieListItemDTO(value) || !isRecord(value)) {
    return false
  }
  return (
    isString(value.summary) &&
    isFiniteNumber(value.metadataRating) &&
    isOptionalNullableNumber(value.userRating) &&
    isOptionalStringArray(value.previewImages) &&
    isOptionalString(value.previewVideoUrl) &&
    isOptionalStringRecord(value.actorAvatarUrls)
  )
}

export function isMoviesPageDTO(value: unknown): value is MoviesPageDTO {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(isMovieListItemDTO) &&
    isFiniteNumber(value.total) &&
    isFiniteNumber(value.limit) &&
    isFiniteNumber(value.offset)
  )
}
