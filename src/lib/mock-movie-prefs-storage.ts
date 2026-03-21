/**
 * Mock 专用：把收藏/用户评分写入 localStorage，整页刷新后仍可读回。
 * Web 模式请勿使用。
 */

import type { Movie } from "@/domain/movie/types"

const PREFS_KEY = "jav-library-movie-prefs"

export type MockMoviePrefs = {
  isFavorite?: boolean
  /** 缺省：不覆盖评分；number：用户分；null：已清除覆盖，用站点分 */
  userRating?: number | null
}

export type MockPrefsMap = Record<string, MockMoviePrefs>

function safeParse(raw: string | null): MockPrefsMap {
  if (!raw?.trim()) return {}
  try {
    const p = JSON.parse(raw) as unknown
    if (p && typeof p === "object" && !Array.isArray(p)) {
      return p as MockPrefsMap
    }
  } catch {
    /* ignore */
  }
  return {}
}

let cache: MockPrefsMap = typeof localStorage !== "undefined" ? safeParse(localStorage.getItem(PREFS_KEY)) : {}

export function loadMockMoviePrefs(): MockPrefsMap {
  if (typeof localStorage === "undefined") return {}
  cache = safeParse(localStorage.getItem(PREFS_KEY))
  return cache
}

export function saveMockMoviePrefs(map: MockPrefsMap) {
  cache = map
  if (typeof localStorage === "undefined") return
  try {
    localStorage.setItem(PREFS_KEY, JSON.stringify(map))
  } catch {
    /* quota / private mode */
  }
}

export function getMockPrefsSnapshot(): MockPrefsMap {
  return { ...cache }
}

/** PATCH 后合并写入某 id 的偏好片段 */
export function upsertMockMoviePrefs(movieId: string, patch: MockMoviePrefs) {
  const id = movieId.trim()
  if (!id) return
  const next: MockPrefsMap = { ...cache }
  const prev = next[id] ?? {}
  const merged: MockMoviePrefs = { ...prev }
  if (typeof patch.isFavorite === "boolean") {
    merged.isFavorite = patch.isFavorite
  }
  if ("userRating" in patch) {
    merged.userRating = patch.userRating
  }
  next[id] = merged
  saveMockMoviePrefs(next)
}

/** 将已持久化的偏好合并进一条 Mock 影片（buildMovie 之后调用） */
export function mergeMockPrefsIntoMovie(movie: Movie): Movie {
  const p = cache[movie.id]
  if (!p) return movie

  const next: Movie = { ...movie }
  if (typeof p.isFavorite === "boolean") {
    next.isFavorite = p.isFavorite
  }
  if ("userRating" in p) {
    if (p.userRating === null) {
      next.userRating = undefined
      next.rating = next.metadataRating ?? movie.rating
    } else if (typeof p.userRating === "number") {
      next.userRating = p.userRating
      next.rating = p.userRating
      if (next.metadataRating === undefined) {
        next.metadataRating = movie.metadataRating ?? movie.rating
      }
    }
  }
  return next
}
