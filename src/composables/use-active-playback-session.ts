import { computed, ref } from "vue"
import type { LocationQuery, RouteLocationRaw } from "vue-router"

export type ActivePlaybackStatus = "playing" | "paused" | "waiting" | "ended" | "error"

export interface UpdateActivePlaybackSessionInput {
  movieId: string
  title: string
  positionSec: number
  durationSec: number
  status: ActivePlaybackStatus
  routeQuery?: LocationQuery
  routeHash?: string
  posterUrl?: string
}

export interface ActivePlaybackSession {
  movieId: string
  title: string
  positionSec: number
  durationSec: number
  progressPercent: number
  status: ActivePlaybackStatus
  updatedAt: string
  resumeRouteTarget: RouteLocationRaw
  posterUrl?: string
}

interface StoredActivePlaybackSession extends ActivePlaybackSession {
  revision: number
}

const MIN_RESUME_POSITION_SEC = 5
const NEAR_END_RATIO = 0.95

let activePlaybackRevision = 0

/**
 * The player publishes on every timeupdate (~4Hz). Sidebar consumers only need
 * coarse position granularity, so skip no-op updates that would otherwise
 * allocate a new snapshot and ripple through every computed each tick.
 */
const ACTIVE_PLAYBACK_POSITION_EPSILON_SEC = 0.75

let lastPublishedSnapshot: {
  movieId: string
  status: ActivePlaybackStatus
  positionSec: number
  durationSec: number
} | null = null

const storedActivePlaybackSession = ref<StoredActivePlaybackSession | null>(null)
const dismissedActivePlayback = ref<{ movieId: string; revision: number } | null>(null)

function normalizeNonNegativeSeconds(value: number): number {
  if (!Number.isFinite(value) || value < 0) return 0
  return value
}

function formatRouteSeconds(value: number): string {
  return String(Math.max(0, Math.round(value)))
}

function progressPercent(positionSec: number, durationSec: number): number {
  if (durationSec <= 0) return 0
  const pct = Math.max(0, Math.min(100, (positionSec / durationSec) * 100))
  return Number(pct.toFixed(1))
}

function shouldHideSession(session: StoredActivePlaybackSession): boolean {
  if (session.status === "ended" || session.status === "error") return true
  if (session.positionSec < MIN_RESUME_POSITION_SEC) return true
  if (session.durationSec > 0 && session.positionSec >= session.durationSec * NEAR_END_RATIO) {
    return true
  }
  return false
}

function buildResumeRouteTarget(input: {
  movieId: string
  positionSec: number
  routeQuery?: LocationQuery
  routeHash?: string
}): RouteLocationRaw {
  return {
    name: "player",
    params: { id: input.movieId },
    query: {
      ...(input.routeQuery ?? {}),
      autoplay: "1",
      t: formatRouteSeconds(input.positionSec),
    },
    hash: input.routeHash,
  }
}

export const activePlaybackSession = computed<ActivePlaybackSession | null>(() => {
  const session = storedActivePlaybackSession.value
  if (!session || shouldHideSession(session)) return null
  const dismissed = dismissedActivePlayback.value
  if (dismissed?.movieId === session.movieId && dismissed.revision === session.revision) {
    return null
  }

  return {
    movieId: session.movieId,
    title: session.title,
    positionSec: session.positionSec,
    durationSec: session.durationSec,
    progressPercent: session.progressPercent,
    status: session.status,
    updatedAt: session.updatedAt,
    resumeRouteTarget: session.resumeRouteTarget,
    posterUrl: session.posterUrl,
  }
})

export function updateActivePlaybackSession(input: UpdateActivePlaybackSessionInput) {
  const movieId = input.movieId.trim()
  if (!movieId) return

  const positionSec = normalizeNonNegativeSeconds(input.positionSec)
  const durationSec = normalizeNonNegativeSeconds(input.durationSec)
  const last = lastPublishedSnapshot
  if (
    last &&
    last.movieId === movieId &&
    last.status === input.status &&
    last.durationSec === durationSec &&
    Math.abs(last.positionSec - positionSec) < ACTIVE_PLAYBACK_POSITION_EPSILON_SEC
  ) {
    return
  }
  lastPublishedSnapshot = {
    movieId,
    status: input.status,
    positionSec,
    durationSec,
  }

  const revision = ++activePlaybackRevision

  storedActivePlaybackSession.value = {
    movieId,
    title: input.title.trim() || movieId,
    positionSec,
    durationSec,
    progressPercent: progressPercent(positionSec, durationSec),
    status: input.status,
    updatedAt: new Date().toISOString(),
    resumeRouteTarget: buildResumeRouteTarget({
      movieId,
      positionSec,
      routeQuery: input.routeQuery,
      routeHash: input.routeHash,
    }),
    posterUrl: input.posterUrl?.trim() || undefined,
    revision,
  }
}

export function clearActivePlaybackSession(movieId?: string) {
  const id = movieId?.trim()
  if (id && storedActivePlaybackSession.value?.movieId !== id) return
  storedActivePlaybackSession.value = null
  dismissedActivePlayback.value = null
  lastPublishedSnapshot = null
}

export function dismissActivePlaybackSession(movieId?: string) {
  const session = storedActivePlaybackSession.value
  if (!session) return
  const id = movieId?.trim()
  if (id && session.movieId !== id) return
  dismissedActivePlayback.value = {
    movieId: session.movieId,
    revision: session.revision,
  }
}

export function useActivePlaybackSession() {
  return {
    activePlaybackSession,
    dismissActivePlaybackSession,
  }
}
