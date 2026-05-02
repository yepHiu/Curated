import { addWatchTimeDelta } from "@/lib/playback-watch-time-storage"
import { getLocalDayKey } from "@/lib/watch-time-heatmap"

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
  addDelta?: WatchTimeDeltaSink
}

const MAX_SINGLE_SAMPLE_SEC = 30
const MAX_API_DELTA_SEC = 300
const MAX_MEDIA_ADVANCE_SEC = 300
const MIN_MEDIA_ADVANCE_SEC = 0.05

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
  const sink = options.addDelta ?? addWatchTimeDelta
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
      addPending(
        getLocalDayKey(new Date(wallNowMs)),
        roundSeconds(Math.min(wallDeltaSec, MAX_SINGLE_SAMPLE_SEC)),
      )
    }
    lastWallMs = wallNowMs
    lastMediaSec = mediaNowSec
  }

  async function flushPending() {
    if (!movieId || pendingByDay.size === 0) return
    const entries = Array.from(pendingByDay.entries())
    pendingByDay.clear()
    try {
      for (const [dayKey, watchedSec] of entries) {
        let remaining = roundSeconds(watchedSec)
        while (remaining > 0) {
          const chunk = roundSeconds(Math.min(remaining, MAX_API_DELTA_SEC))
          await sink(movieId, dayKey, chunk)
          remaining = roundSeconds(remaining - chunk)
        }
      }
    } catch (error) {
      for (const [dayKey, watchedSec] of entries) {
        addPending(dayKey, watchedSec)
      }
      throw error
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
