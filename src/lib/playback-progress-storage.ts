/**
 * 播放进度：Mock 模式用 localStorage；VITE_USE_WEB_API 时用后端 SQLite（与资料库同库）。
 */

import { ref } from "vue"
import { api } from "@/api/endpoints"
import { isNearPlaybackEnd } from "@/lib/playback-targets"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

const STORAGE_KEY = "jav-library-playback-progress-v1"

/** Bump on save/remove/hydrate so Vue computeds（历史页、侧栏数量等）保持更新。 */
export const playbackProgressRevision = ref(0)

export interface PlaybackProgressEntry {
  movieId: string
  fileId?: string
  positionSec: number
  durationSec: number
  updatedAt: string
}

type StoreShape = Record<string, PlaybackProgressEntry>

/** JSON 元组避免电影 ID 和文件 ID 的分隔符碰撞。 */
function fileProgressKey(movieId: string, fileId: string): string {
 return JSON.stringify([movieId, fileId])
}

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function isNonNegativeFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value >= 0
}

/** 校验进度字段及可选文件标识。 */
function isPlaybackProgressEntry(value: unknown): value is PlaybackProgressEntry {
  if (!isObjectRecord(value)) {
    return false
  }
  return (
    (value.fileId === undefined || typeof value.fileId === "string") &&
    typeof value.movieId === "string" &&
    value.movieId.trim() !== "" &&
    isNonNegativeFiniteNumber(value.positionSec) &&
    isNonNegativeFiniteNumber(value.durationSec) &&
    typeof value.updatedAt === "string" &&
    value.updatedAt.trim() !== ""
  )
}

/** 规范作品和文件标识，保留片内位置。 */
function normalizeEntry(row: PlaybackProgressEntry): PlaybackProgressEntry {
  return {
    movieId: row.movieId.trim(),
    ...(row.fileId?.trim() ? { fileId: row.fileId.trim() } : {}),
    positionSec: row.positionSec,
    durationSec: row.durationSec,
    updatedAt: row.updatedAt,
  }
}

/** 重建每片进度及每部作品唯一的最后播放快照。 */
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
    if (normalized.fileId) next[fileProgressKey(normalized.movieId, normalized.fileId)] = normalized
    const previous = next[normalized.movieId]
    if (!previous || normalized.updatedAt >= previous.updatedAt) next[normalized.movieId] = normalized
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
      const entry: PlaybackProgressEntry = {
        movieId: id,
        ...(row.fileId ? { fileId: row.fileId } : {}),
        positionSec: row.positionSec,
        durationSec: row.durationSec,
        updatedAt: row.updatedAt,
      }
      next[id] = entry
      if (entry.fileId) next[fileProgressKey(id, entry.fileId)] = entry
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

/** 文件进度独立寻址；省略 fileId 时返回作品最后一次播放，供历史与续播入口使用。 */
export function getProgress(movieId: string, fileId?: string): PlaybackProgressEntry | undefined {
  const id = movieId.trim()
  if (!id) return undefined
  const row = cache[fileId ? fileProgressKey(id, fileId) : id]
  if (!isPlaybackProgressEntry(row)) return undefined
  return normalizeEntry(row)
}

/** 只列作品历史投影，隐藏独立分片缓存条目。 */
export function listSortedByUpdatedDesc(): PlaybackProgressEntry[] {
  return Object.entries(cache)
    .filter(/* 从文件缓存中提取每部作品唯一的历史投影。 */ ([key, entry]) => key === entry.movieId)
    .map(/* 从文件缓存中提取每部作品唯一的历史投影。 */ ([, entry]) => entry)
    .filter(isPlaybackProgressEntry)
    .map(normalizeEntry)
    .sort((a, b) => {
      const ta = Date.parse(a.updatedAt) || 0
      const tb = Date.parse(b.updatedAt) || 0
      return tb - ta
    })
}

/** 同步当前分片，并维护每部作品唯一的历史入口。 */
export function saveProgress(movieId: string, positionSec: number, durationSec: number, fileId?: string) {
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
    ...(fileId ? { fileId } : {}),
    positionSec: pos,
    durationSec: dur,
    updatedAt,
  }
  if (fileId) cache[fileProgressKey(id, fileId)] = cache[id]!
  if (!USE_WEB) {
    saveStore(cache)
  } else {
    void api.putPlaybackProgress(id, { positionSec: pos, durationSec: dur, ...(fileId ? { fileId } : {}) }).catch(() => {
      // 内存已更新；同步失败时下次 hydrate 或重试可收敛
    })
  }
  playbackProgressRevision.value += 1
}

/** 清除作品历史及所有分片进度。 */
export function removeProgress(movieId: string) {
  const id = movieId.trim()
  if (!id) return
  if (!cache[id]) return
  for (const [key, row] of Object.entries(cache)) {
    if (row.movieId === id) delete cache[key]
  }
  if (!USE_WEB) {
    saveStore(cache)
  } else {
    void api.deletePlaybackProgress(id).catch(() => {})
  }
  playbackProgressRevision.value += 1
}

/** Seconds to pass as `t` when opening the player from detail/library (skip tiny / near-end). */
export function getResumeSecondsForOpenPlayer(movieId: string, fileId?: string): number | undefined {
  const row = getProgress(movieId, fileId)
  if (!row) return undefined
  const pos = row.positionSec
  const dur = row.durationSec
  if (pos < 5) return undefined
  if (isNearPlaybackEnd(pos, dur)) return undefined
  return Math.floor(pos)
}
