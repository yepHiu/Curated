export const WATCH_TIME_HEATMAP_WEEKS = 13
export const WATCH_TIME_HEATMAP_DAYS = WATCH_TIME_HEATMAP_WEEKS * 7

export type WatchTimeLevel = 0 | 1 | 2 | 3 | 4

export interface DailyWatchTimeEntry {
  dayKey: string
  watchedSec: number
}

export interface DailyWatchTimeSummary {
  items: DailyWatchTimeEntry[]
  totalWatchedSec: number
  thisWeekWatchedSec: number
  activeDays: number
  maxDayWatchedSec: number
  longestStreakDays: number
}

export interface WatchTimeHeatmapCell {
  dayKey: string
  date: Date
  watchedSec: number
  level: WatchTimeLevel
  isFuture: boolean
}

export interface WatchTimeMonthLabel {
  weekIndex: number
  date: Date
}

export interface WatchTimeHeatmap {
  cells: WatchTimeHeatmapCell[]
  weeks: WatchTimeHeatmapCell[][]
  monthLabels: WatchTimeMonthLabel[]
  summary: DailyWatchTimeSummary
}

interface WatchTimeDateOptions {
  today?: Date
  days?: number
}

const DAY_KEY_RE = /^\d{4}-\d{2}-\d{2}$/

function startOfLocalDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

function startOfLocalWeek(date: Date): Date {
  const d = startOfLocalDay(date)
  d.setDate(d.getDate() - d.getDay())
  return d
}

function addLocalDays(date: Date, days: number): Date {
  const d = startOfLocalDay(date)
  d.setDate(d.getDate() + days)
  return d
}

export function getLocalDayKey(date: Date = new Date()): string {
  const d = startOfLocalDay(date)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const day = String(d.getDate()).padStart(2, "0")
  return `${y}-${m}-${day}`
}

export function parseLocalDayKey(dayKey: string): Date | null {
  if (!DAY_KEY_RE.test(dayKey)) return null
  const [year, month, day] = dayKey.split("-").map(Number)
  if (!year || !month || !day) return null
  const date = new Date(year, month - 1, day)
  return getLocalDayKey(date) === dayKey ? date : null
}

export function isValidLocalDayKey(dayKey: string): boolean {
  return parseLocalDayKey(dayKey) !== null
}

function normalizeWatchSeconds(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 0
  return value
}

function normalizeEntries(entries: readonly DailyWatchTimeEntry[]): DailyWatchTimeEntry[] {
  const byDay = new Map<string, number>()
  for (const entry of entries) {
    const dayKey = entry.dayKey.trim()
    if (!isValidLocalDayKey(dayKey)) continue
    const watchedSec = normalizeWatchSeconds(entry.watchedSec)
    if (watchedSec <= 0) continue
    byDay.set(dayKey, (byDay.get(dayKey) ?? 0) + watchedSec)
  }
  return Array.from(byDay.entries())
    .map(([dayKey, watchedSec]) => ({ dayKey, watchedSec }))
    .sort((a, b) => (a.dayKey < b.dayKey ? 1 : a.dayKey > b.dayKey ? -1 : 0))
}

export function getWatchTimeLevel(watchedSec: number): WatchTimeLevel {
  if (!Number.isFinite(watchedSec) || watchedSec <= 0) return 0
  if (watchedSec < 30 * 60) return 1
  if (watchedSec < 90 * 60) return 2
  if (watchedSec < 150 * 60) return 3
  return 4
}

export function createEmptyDailyWatchTimeSummary(): DailyWatchTimeSummary {
  return {
    items: [],
    totalWatchedSec: 0,
    thisWeekWatchedSec: 0,
    activeDays: 0,
    maxDayWatchedSec: 0,
    longestStreakDays: 0,
  }
}

export function buildWatchTimeSummary(
  entries: readonly DailyWatchTimeEntry[],
  options: WatchTimeDateOptions = {},
): DailyWatchTimeSummary {
  const today = startOfLocalDay(options.today ?? new Date())
  const todayKey = getLocalDayKey(today)
  const days = Math.max(1, Math.floor(options.days ?? WATCH_TIME_HEATMAP_DAYS))
  const rangeEnd = addLocalDays(startOfLocalWeek(today), 6)
  const rangeStartKey = getLocalDayKey(addLocalDays(rangeEnd, -(days - 1)))
  const weekStartKey = getLocalDayKey(startOfLocalWeek(today))
  const items = normalizeEntries(entries).filter(
    (entry) => entry.dayKey >= rangeStartKey && entry.dayKey <= todayKey,
  )

  let totalWatchedSec = 0
  let thisWeekWatchedSec = 0
  let maxDayWatchedSec = 0
  const activeDays = new Set<string>()

  for (const item of items) {
    totalWatchedSec += item.watchedSec
    maxDayWatchedSec = Math.max(maxDayWatchedSec, item.watchedSec)
    if (item.watchedSec > 0) {
      activeDays.add(item.dayKey)
    }
    if (item.dayKey >= weekStartKey && item.dayKey <= todayKey) {
      thisWeekWatchedSec += item.watchedSec
    }
  }

  let longestStreakDays = 0
  let currentStreakDays = 0
  for (
    let d = startOfLocalDay(parseLocalDayKey(rangeStartKey) ?? today);
    getLocalDayKey(d) <= todayKey;
    d = addLocalDays(d, 1)
  ) {
    if (activeDays.has(getLocalDayKey(d))) {
      currentStreakDays += 1
      longestStreakDays = Math.max(longestStreakDays, currentStreakDays)
    } else {
      currentStreakDays = 0
    }
  }

  return {
    items,
    totalWatchedSec,
    thisWeekWatchedSec,
    activeDays: activeDays.size,
    maxDayWatchedSec,
    longestStreakDays,
  }
}

export function buildWatchTimeHeatmap(
  entries: readonly DailyWatchTimeEntry[],
  options: WatchTimeDateOptions = {},
): WatchTimeHeatmap {
  const today = startOfLocalDay(options.today ?? new Date())
  const todayKey = getLocalDayKey(today)
  const end = addLocalDays(startOfLocalWeek(today), 6)
  const start = addLocalDays(end, -(WATCH_TIME_HEATMAP_DAYS - 1))
  const normalized = new Map(normalizeEntries(entries).map((entry) => [entry.dayKey, entry.watchedSec]))
  const cells: WatchTimeHeatmapCell[] = []

  for (let i = 0; i < WATCH_TIME_HEATMAP_DAYS; i += 1) {
    const date = addLocalDays(start, i)
    const dayKey = getLocalDayKey(date)
    const isFuture = dayKey > todayKey
    const watchedSec = isFuture ? 0 : (normalized.get(dayKey) ?? 0)
    cells.push({
      dayKey,
      date,
      watchedSec,
      level: getWatchTimeLevel(watchedSec),
      isFuture,
    })
  }

  const weeks: WatchTimeHeatmapCell[][] = []
  for (let i = 0; i < WATCH_TIME_HEATMAP_WEEKS; i += 1) {
    weeks.push(cells.slice(i * 7, i * 7 + 7))
  }

  const monthLabels: WatchTimeMonthLabel[] = []
  weeks.forEach((week, weekIndex) => {
    const firstOfMonth = week.find((cell) => cell.date.getDate() === 1)
    if (firstOfMonth) {
      monthLabels.push({ weekIndex, date: firstOfMonth.date })
    }
  })

  return {
    cells,
    weeks,
    monthLabels,
    summary: buildWatchTimeSummary(entries, {
      today,
      days: options.days ?? WATCH_TIME_HEATMAP_DAYS,
    }),
  }
}

export function formatWatchTimeDuration(watchedSec: number): string {
  if (!Number.isFinite(watchedSec) || watchedSec <= 0) return "0m"
  const totalMinutes = Math.max(1, Math.round(watchedSec / 60))
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours <= 0) return `${minutes}m`
  if (minutes <= 0) return `${hours}h`
  return `${hours}h ${minutes}m`
}
