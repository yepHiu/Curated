/** Group playback history rows by local calendar day (today / yesterday / formatted date). */

export interface HistoryDayBucket<T> {
  dayKey: string
  label: string
  rows: T[]
}

export interface GroupPlaybackByDayLabels {
  today: string
  yesterday: string
}

export interface GroupPlaybackByDayOptions {
  locale: string
  labels: GroupPlaybackByDayLabels
}

function localDayKey(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const day = String(d.getDate()).padStart(2, "0")
  return `${y}-${m}-${day}`
}

export function groupPlaybackRowsByLocalDay<T extends { updatedAt: string }>(
  rows: T[],
  opts: GroupPlaybackByDayOptions,
): HistoryDayBucket<T>[] {
  const map = new Map<string, T[]>()
  for (const row of rows) {
    const d = new Date(row.updatedAt)
    if (Number.isNaN(d.getTime())) continue
    const key = localDayKey(d)
    const list = map.get(key) ?? []
    list.push(row)
    map.set(key, list)
  }

  const dayKeys = [...map.keys()].sort((a, b) => b.localeCompare(a))

  const now = new Date()
  const todayKey = localDayKey(now)
  const yest = new Date(now)
  yest.setDate(yest.getDate() - 1)
  const yesterdayKey = localDayKey(yest)

  const labelFor = (key: string) => {
    if (key === todayKey) return opts.labels.today
    if (key === yesterdayKey) return opts.labels.yesterday
    const [yy, mm, dd] = key.split("-").map(Number)
    const anchor = new Date(yy, (mm ?? 1) - 1, dd ?? 1, 12, 0, 0)
    return new Intl.DateTimeFormat(opts.locale, {
      year: "numeric",
      month: "long",
      day: "numeric",
    }).format(anchor)
  }

  return dayKeys.map((dayKey) => {
    const bucket = map.get(dayKey) ?? []
    bucket.sort((a, b) => (Date.parse(b.updatedAt) || 0) - (Date.parse(a.updatedAt) || 0))
    return {
      dayKey,
      label: labelFor(dayKey),
      rows: bucket,
    }
  })
}
