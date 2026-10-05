import { ref } from "vue"
import type { RouteLocationNormalized, RouteLocationResolved, RouteLocationRaw, LocationQuery } from "vue-router"
import type { DesktopPlaybackSnapshot } from "../../electron/playback-contract"
import { parseResumeSecondsFromQuery, getProgress, hydratePlaybackProgress } from "@/lib/playback-progress-storage"
import { hasPlayedMovie, hydratePlayedMovies } from "@/lib/played-movies-storage"
import { listPlayerPlaylistMovies, resolvePlayerPlaylistSource, readPlaylistAutoAdvance } from "@/lib/player-playlist"
import { resolveInitialLocale } from "@/lib/locale-storage"
import { resolveNavigationBackLink } from "@/lib/navigation-intent"
import { updateActivePlaybackSession } from "@/composables/use-active-playback-session"
import { useLibraryService } from "@/services/library-service"

const snapshot = ref<DesktopPlaybackSnapshot>()
let query: LocationQuery = {}
let stopWeb: (() => void) | undefined
let initialized = false
let onNavigate: ((target: RouteLocationRaw) => Promise<unknown>) | undefined
/** Server UI 的可选桌面适配；只传身份/意图，业务 HTTP 仍由当前引擎所有者处理。 */
export const desktopPlaybackService = {
  snapshot,
  bindWebStop(stop: () => void) { stopWeb = stop; return () => { if (stopWeb === stop) stopWeb = undefined } },
  initialize(navigate: (target: RouteLocationRaw) => Promise<unknown>) {
    onNavigate = navigate
    const bridge = window.javLibrary?.playback
    if (!bridge || initialized) return
    initialized = true
    const receive = (next: DesktopPlaybackSnapshot) => {
      if (snapshot.value && next.revision < snapshot.value.revision) return
      const previous = snapshot.value
      snapshot.value = next
      if (next.sourceQuery) query = { ...next.sourceQuery }
      const movie = next.movie
      if (movie && next.state.durationSec > 0) updateActivePlaybackSession({
        movieId: movie.id, title: movie.title || movie.code, positionSec: next.state.positionSec, durationSec: next.state.durationSec,
        status: next.state.status === "starting" ? "waiting" : next.state.status === "idle" || next.state.status === "stopped" ? "paused" : next.state.status,
        routeQuery: { ...query, ...(next.state.fileId ? { fileId: next.state.fileId } : {}) },
      })
      if (previous?.windowOpen && !next.windowOpen || previous?.sessionId !== next.sessionId && next.movie) {
        void Promise.allSettled([hydratePlaybackProgress(), hydratePlayedMovies()])
      }
      if (next.state.error === "SERVER_LOCKED" && previous?.state.error !== "SERVER_LOCKED") {
        void onNavigate?.({ name: "lock" })
      }
    }
    bridge.subscribe(receive)
    void bridge.snapshot().then(receive).catch(() => {})
    bridge.onWebFallback(input => {
      void onNavigate?.({ name: "player", params: { id: input.movieId },
        query: { ...query, engine: "web", fileId: input.fileId, t: String(input.startSec), autoplay: "1" } })
    })
  },
  async command(action: "pause" | "resume" | "stop" | "focus") {
    const current = snapshot.value
    if (current?.sessionId) await window.javLibrary?.playback?.command(current.sessionId, { action })
  },
  async intercept(to: RouteLocationNormalized | RouteLocationResolved, from: RouteLocationNormalized | RouteLocationResolved): Promise<true | RouteLocationRaw> {
    const bridge = window.javLibrary?.playback
    if (to.name !== "player" || typeof to.params.id !== "string" || !bridge || import.meta.env.VITE_USE_WEB_API !== "true") return true
    const capability = await bridge.capabilities().catch(() => undefined)
    if (capability?.protocol !== 1 || !capability.available || !capability.preferences.preferNative || to.query.engine === "web") {
      const current = snapshot.value
      if (current?.windowOpen) await bridge.command(current.sessionId, { action: "stop" })
      return true
    }
    // 原生选定后才停止旧 Web；不进入 PlayerView，因此没有 HLS prefetch。
    stopWeb?.()
    const library = useLibraryService()
    const source = resolvePlayerPlaylistSource(to.query)
    const queue = listPlayerPlaylistMovies({ source, movies: library.movies.value, trashedMovies: library.trashedMovies.value,
      query: to.query, hasPlayedMovie, getProgress }).map(movie => movie.id).slice(0, 5000)
    const locale = resolveInitialLocale()
    query = { ...to.query }
    try {
      const next = await bridge.open({ movieId: to.params.id, fileId: typeof to.query.fileId === "string" ? to.query.fileId : undefined,
        seekExisting: to.query.back === "curated-frames" || to.query.from === "curated-frames" || from.name === "player",
        startSec: parseResumeSecondsFromQuery(to.query.t), autoplay: to.query.autoplay === "1", queue,
        autoAdvance: readPlaylistAutoAdvance(), locale: locale === "en" ? "en-US" : locale === "ja" ? "ja-JP" : "zh-CN", sourceQuery: to.query })
      snapshot.value = next
      if (next.sourceQuery) query = { ...next.sourceQuery }
      if (from.name && from.name !== "player" && from.name !== "lock") return from.fullPath
      return resolveNavigationBackLink({ name: "player", query: to.query }, to.params.id).to
    } catch (error) {
      const locked = error instanceof Error && error.message.includes("SERVER_LOCKED")
      if (locked) return { name: "lock", query: { redirect: to.fullPath } }
      // 起播失败已由 main 回收；单次 Web 回退不会再次选择 native。
      return { name: "player", params: to.params, query: { ...to.query, engine: "web" } }
    }
  },
}
