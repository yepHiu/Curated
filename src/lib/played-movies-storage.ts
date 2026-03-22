import { ref } from "vue"
import { api } from "@/api/endpoints"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

const STORAGE_KEY = "jav-library-played-movie-ids"

function loadIdsFromStorage(): Set<string> {
  if (typeof localStorage === "undefined") {
    return new Set()
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return new Set()
    }
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) {
      return new Set()
    }
    return new Set(
      parsed.filter((x): x is string => typeof x === "string" && x.trim() !== "").map((x) => x.trim()),
    )
  } catch {
    return new Set()
  }
}

function saveIdsToStorage(ids: Set<string>) {
  if (typeof localStorage === "undefined") {
    return
  }
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify([...ids]))
  } catch {
    // quota / private mode — ignore
  }
}

const playedIds: Set<string> = USE_WEB ? new Set() : loadIdsFromStorage()

/** 曾进入过播放页的去重影片数（供设置页统计与 computed 依赖） */
export const playedMovieCount = ref(playedIds.size)

/**
 * Web API：启动时从后端拉取已播放 id 集合。
 */
export async function hydratePlayedMovies(): Promise<void> {
  if (!USE_WEB) return
  try {
    const { movieIds } = await api.listPlayedMovies()
    playedIds.clear()
    for (const id of movieIds) {
      const t = id.trim()
      if (t) playedIds.add(t)
    }
    playedMovieCount.value = playedIds.size
  } catch {
    // 保留当前内存（例如离线）
  }
}

/**
 * 记录一部影片曾被播放（进入播放页且解析到有效条目时调用）。
 * 同一 id 只计一次；Mock 用 localStorage，Web API 用 POST 后端 SQLite。
 */
export function recordMoviePlayed(movieId: string) {
  const id = movieId.trim()
  if (!id) {
    return
  }
  if (playedIds.has(id)) {
    return
  }
  playedIds.add(id)
  playedMovieCount.value = playedIds.size
  if (USE_WEB) {
    void api.recordPlayedMovie(id).catch(() => {
      playedIds.delete(id)
      playedMovieCount.value = playedIds.size
    })
  } else {
    saveIdsToStorage(playedIds)
  }
}
