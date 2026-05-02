import { ref } from "vue"
import { api } from "@/api/endpoints"
import {
  WATCH_TIME_HEATMAP_DAYS,
  buildWatchTimeSummary,
  isValidLocalDayKey,
  type DailyWatchTimeEntry,
  type DailyWatchTimeSummary,
} from "@/lib/watch-time-heatmap"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"
const STORAGE_KEY = "curated-playback-watch-time-daily-v1"
const MAX_DELTA_SEC = 300

type StoreShape = Record<string, Record<string, number>>

export const watchTimeRevision = ref(0)

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function normalizeDeltaSec(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 0
  return Math.min(MAX_DELTA_SEC, value)
}

function normalizeStore(value: unknown): StoreShape {
  if (!isObjectRecord(value)) return {}
  const next: StoreShape = {}
  for (const [dayKey, movies] of Object.entries(value)) {
    if (!isValidLocalDayKey(dayKey) || !isObjectRecord(movies)) continue
    const dayStore: Record<string, number> = {}
    for (const [movieId, watchedSec] of Object.entries(movies)) {
      const id = movieId.trim()
      if (!id || typeof watchedSec !== "number" || !Number.isFinite(watchedSec) || watchedSec <= 0) {
        continue
      }
      dayStore[id] = watchedSec
    }
    if (Object.keys(dayStore).length > 0) {
      next[dayKey] = dayStore
    }
  }
  return next
}

function loadStore(): StoreShape {
  if (typeof localStorage === "undefined") return {}
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    return normalizeStore(JSON.parse(raw) as unknown)
  } catch {
    return {}
  }
}

function saveStore(store: StoreShape) {
  if (typeof localStorage === "undefined") return
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(store))
  } catch {
    // Keep the in-memory aggregate even if persistence is unavailable.
  }
}

function aggregateStore(store: StoreShape): DailyWatchTimeEntry[] {
  return Object.entries(store).map(([dayKey, movies]) => ({
    dayKey,
    watchedSec: Object.values(movies).reduce((sum, sec) => sum + sec, 0),
  }))
}

let cache: StoreShape = USE_WEB ? {} : loadStore()

export async function addWatchTimeDelta(
  movieId: string,
  dayKey: string,
  watchedSec: number,
): Promise<void> {
  const id = movieId.trim()
  const day = dayKey.trim()
  const delta = normalizeDeltaSec(watchedSec)
  if (!id || !isValidLocalDayKey(day) || delta <= 0) {
    return
  }

  if (USE_WEB) {
    await api.addPlaybackWatchTimeDaily({
      movieId: id,
      dayKey: day,
      watchedSec: delta,
    })
    watchTimeRevision.value += 1
    return
  }

  const dayStore = cache[day] ?? {}
  dayStore[id] = (dayStore[id] ?? 0) + delta
  cache = {
    ...cache,
    [day]: dayStore,
  }
  saveStore(cache)
  watchTimeRevision.value += 1
}

export async function listDailyWatchTime(
  days: number = WATCH_TIME_HEATMAP_DAYS,
): Promise<DailyWatchTimeSummary> {
  const windowDays = Math.max(1, Math.floor(days))
  if (USE_WEB) {
    const dto = await api.listPlaybackWatchTimeDaily(windowDays)
    return buildWatchTimeSummary(dto.items, { days: windowDays })
  }
  cache = normalizeStore(cache)
  return buildWatchTimeSummary(aggregateStore(cache), { days: windowDays })
}
