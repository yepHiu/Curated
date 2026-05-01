/**
 * 播放进度：Mock 模式用 localStorage；VITE_USE_WEB_API 时用后端 SQLite（与资料库同库）。
 */

import { ref } from "vue"
import { api } from "@/api/endpoints"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

const STORAGE_KEY = "jav-library-playback-progress-v1"

/** Bump on save/remove/hydrate so Vue computeds（历史页、侧栏数量等）保持更新。 */
export const playbackProgressRevision = ref(0)

export interface PlaybackProgressEntry {
  movieId: string
  positionSec: number
  durationSec: number
  updatedAt: string
}

type StoreShape = Record<string, PlaybackProgressEntry>

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function isNonNegativeFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value >= 0
}

function isPlaybackProgressEntry(value: unknown): value is PlaybackProgressEntry {
  if (!isObjectRecord(value)) {
    return false
  }
  return (
    typeof value.movieId === "string" &&
    value.movieId.trim() !== "" &&
    isNonNegativeFiniteNumber(value.positionSec) &&
    isNonNegativeFiniteNumber(value.durationSec) &&
    typeof value.updatedAt === "string" &&
    value.updatedAt.trim() !== ""
  )
}

function normalizeEntry(row: PlaybackProgressEntry): PlaybackProgressEntry {
  return {
    movieId: row.movieId.trim(),
    positionSec: row.positionSec,
    durationSec: row.durationSec,
    updatedAt: row.updatedAt,
  }
}

function normalizeStore(value: unknown): StoreShape {
  if (!isObjectRecord(value)) {
    return {}
  }
  const next: StoreShape = {}
  for (const row of Object.values(value)) {
    if (!isPlaybackProgressEntry(row)) {
      continue
    }
    const normalized = normalizeEntry(row)
    next[normalized.movieId] = normalized
  }
  return next
}

function loadStore(): StoreShape {
  if (typeof localStorage === "undefined") {
    return {}
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as unknown
    return normalizeStore(parsed)
  } catch {
    return {}
  }
}

function saveStore(store: StoreShape) {
  if (typeof localStorage === "undefined") {
    return
  }
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(store))
  } catch {
    // quota / private mode
  }
}

let cache: StoreShape = USE_WEB ? {} : loadStore()

function normalizeSeconds(value: number): number {
  if (!Number.isFinite(value) || value < 0) return 0
  return value
}

/**
 * Web API 模式：启动时从后端拉取全量进度，写入内存缓存。
 */
export async function hydratePlaybackProgress(): Promise<void> {
  if (!USE_WEB) return
  try {
    const { items } = await api.listPlaybackProgress()
    const next: StoreShape = {}
    for (const row of items) {
      const id = row.movieId.trim()
      if (!id) continue
      next[id] = {
        movieId: id,
        positionSec: row.positionSec,
        durationSec: row.durationSec,
        updatedAt: row.updatedAt,
      }
    }
    cache = next
    playbackProgressRevision.value += 1
  } catch {
    // 保留当前 cache（例如离线）；首次为空即可
  }
}

/**
 * Parse `t` from route query: positive seconds, including fractional seek positions.
 */
export function parseResumeSecondsFromQuery(t: unknown): number | undefined {
  if (typeof t !== "string" || !t.trim()) return undefined
  const n = Number(t)
  if (!Number.isFinite(n) || n < 0) return undefined
  return n
}

export function getProgress(movieId: string): PlaybackProgressEntry | undefined {
  const id = movieId.trim()
  if (!id) return undefined
  const row = cache[id]
  if (!isPlaybackProgressEntry(row)) return undefined
  return normalizeEntry(row)
}

export function listSortedByUpdatedDesc(): PlaybackProgressEntry[] {
  return Object.values(cache)
    .filter(isPlaybackProgressEntry)
    .map(normalizeEntry)
    .sort((a, b) => {
      const ta = Date.parse(a.updatedAt) || 0
      const tb = Date.parse(b.updatedAt) || 0
      return tb - ta
    })
}

export function saveProgress(movieId: string, positionSec: number, durationSec: number) {
  const id = movieId.trim()
  if (!id) return

  let pos = normalizeSeconds(positionSec)
  const dur = normalizeSeconds(durationSec)

  if (dur > 0) {
    pos = Math.min(pos, dur)
  }

  const updatedAt = new Date().toISOString()
  cache[id] = {
    movieId: id,
    positionSec: pos,
    durationSec: dur,
    updatedAt,
  }
  if (!USE_WEB) {
    saveStore(cache)
  } else {
    void api.putPlaybackProgress(id, { positionSec: pos, durationSec: dur }).catch(() => {
      // 内存已更新；同步失败时下次 hydrate 或重试可收敛
    })
  }
  playbackProgressRevision.value += 1
}

export function removeProgress(movieId: string) {
  const id = movieId.trim()
  if (!id) return
  if (!cache[id]) return
  delete cache[id]
  if (!USE_WEB) {
    saveStore(cache)
  } else {
    void api.deletePlaybackProgress(id).catch(() => {})
  }
  playbackProgressRevision.value += 1
}

/** Seconds to pass as `t` when opening the player from detail/library (skip tiny / near-end). */
export function getResumeSecondsForOpenPlayer(movieId: string): number | undefined {
  const row = getProgress(movieId)
  if (!row) return undefined
  const pos = row.positionSec
  const dur = row.durationSec
  if (pos < 5) return undefined
  if (dur > 0 && pos >= dur * 0.95) return undefined
  return Math.floor(pos)
}
