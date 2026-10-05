
function getLocalDayKey(date: Date): string { return [date.getFullYear(), String(date.getMonth() + 1).padStart(2, "0"), String(date.getDate()).padStart(2, "0")].join("-") }

export type WatchTimeDeltaSink = (
  movieId: string,
  dayKey: string,
  watchedSec: number,
) => Promise<void> | void

export interface PlaybackWatchTimeTracker {
  onPlay(mediaTimeSec: number): void
  onTimeUpdate(mediaTimeSec: number): void
  onPause(mediaTimeSec: number): void
  onSeeking(mediaTimeSec: number): void
  flush(mediaTimeSec?: number): Promise<void>
  reset(movieId: string, mediaTimeSec?: number): Promise<void>
}

export interface PlaybackWatchTimeTrackerOptions {
  movieId: string
  now?: () => number
  addDelta: WatchTimeDeltaSink
}

const MAX_SINGLE_SAMPLE_SEC = 30
const MAX_API_DELTA_SEC = 300
const MAX_MEDIA_ADVANCE_SEC = 300
// mpv 可每帧发送 time-pos，30/60fps 的合法推进小于旧 Web 采样阈值。
const MIN_MEDIA_ADVANCE_SEC = 0

function normalizeMediaTime(value: number): number {
  if (!Number.isFinite(value) || value < 0) return 0
  return value
}

function roundSeconds(value: number): number {
  return Math.round(value * 1000) / 1000
}

export function createPlaybackWatchTimeTracker(
  options: PlaybackWatchTimeTrackerOptions,
): PlaybackWatchTimeTracker {
  let movieId = options.movieId.trim()
  const now = options.now ?? (() => Date.now())
  const sink = options.addDelta
  const pendingByDay = new Map<string, number>()
  let playing = false
  let lastWallMs: number | null = null
  let lastMediaSec: number | null = null

  function setSample(mediaTimeSec: number) {
    lastWallMs = now()
    lastMediaSec = normalizeMediaTime(mediaTimeSec)
  }

  function clearSample() {
    lastWallMs = null
    lastMediaSec = null
  }

  function addPending(dayKey: string, watchedSec: number) {
    if (!Number.isFinite(watchedSec) || watchedSec <= 0) return
    pendingByDay.set(dayKey, (pendingByDay.get(dayKey) ?? 0) + watchedSec)
  }

  function sample(mediaTimeSec: number) {
    const wallNowMs = now()
    const mediaNowSec = normalizeMediaTime(mediaTimeSec)
    if (!playing || lastWallMs === null || lastMediaSec === null) {
      lastWallMs = wallNowMs
      lastMediaSec = mediaNowSec
      return
    }

    const wallDeltaSec = (wallNowMs - lastWallMs) / 1000
    const mediaDeltaSec = mediaNowSec - lastMediaSec
    if (
      wallDeltaSec > 0 &&
      mediaDeltaSec > MIN_MEDIA_ADVANCE_SEC &&
      mediaDeltaSec <= MAX_MEDIA_ADVANCE_SEC
    ) {
      let start = Math.max(lastWallMs, wallNowMs - MAX_SINGLE_SAMPLE_SEC * 1000)
      while (start < wallNowMs) {
        const date = new Date(start)
        const midnight = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1).getTime()
        const end = Math.min(midnight, wallNowMs)
        addPending(getLocalDayKey(date), roundSeconds((end - start) / 1000))
        start = end
      }
    }
    lastWallMs = wallNowMs
    lastMediaSec = mediaNowSec
  }

  async function flushPending() {
    if (!movieId || pendingByDay.size === 0) return
    const entries = Array.from(pendingByDay.entries())
    pendingByDay.clear()
    for (let index = 0; index < entries.length; index++) {
      const [dayKey, watchedSec] = entries[index]!
        let remaining = roundSeconds(watchedSec)
      try {
        while (remaining > 0) {
          const chunk = roundSeconds(Math.min(remaining, MAX_API_DELTA_SEC))
          await sink(movieId, dayKey, chunk)
          remaining = roundSeconds(remaining - chunk)
        }
      } catch (error) {
        // 已确认成功的日期/分块不再次补回，失败与尚未发送的增量才重试。
        addPending(dayKey, remaining)
        for (const [pendingDay, pendingSeconds] of entries.slice(index + 1)) addPending(pendingDay, pendingSeconds)
        throw error
      }
    }
  }

  return {
    onPlay(mediaTimeSec: number) {
      playing = true
      setSample(mediaTimeSec)
    },
    onTimeUpdate(mediaTimeSec: number) {
      sample(mediaTimeSec)
    },
    onPause(mediaTimeSec: number) {
      sample(mediaTimeSec)
      playing = false
      clearSample()
    },
    onSeeking(mediaTimeSec: number) {
      setSample(mediaTimeSec)
    },
    async flush(mediaTimeSec?: number) {
      if (mediaTimeSec !== undefined) {
        sample(mediaTimeSec)
      }
      await flushPending()
    },
    async reset(nextMovieId: string, mediaTimeSec?: number) {
      await this.flush(mediaTimeSec)
      movieId = nextMovieId.trim()
      playing = false
      clearSample()
    },
  }
}
