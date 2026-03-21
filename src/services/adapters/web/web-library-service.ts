import { computed, ref, type Ref } from "vue"
import type { TaskDTO } from "@/api/types"
import { api } from "@/api/endpoints"
import { moviePlaybackAbsoluteUrl } from "@/api/playback-url"
import type { LibrarySetting, LibraryStat } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import type { LibraryService } from "@/services/contracts/library-service"
import { mapMovieDetail, mapMovieListItem } from "./mappers"

const moviesState: Ref<Movie[]> = ref([])
const libraryPathsState: Ref<LibrarySetting[]> = ref([])
/** 与后端 config.Default() / library-config.cfg 默认一致，避免首屏在 GET 完成前误显示为关 */
const organizeLibraryState = ref(true)
/** 忽略过期的 PATCH 响应，避免快速连点时状态被旧请求写乱 */
let organizeLibrarySaveSeq = 0
let loaded = false

/** 单次请求条数；多页拉取直至 total 或达到上限，避免首屏只看见前 200 条 */
const LIST_BATCH_SIZE = 500
const MAX_MOVIES_PREFETCH = 10_000

function mapLibraryPathsFromSettings(paths: { id: string; path: string; title: string }[]): LibrarySetting[] {
  return paths.map((p) => ({ id: p.id, path: p.path, title: p.title }))
}

async function ensureLoaded() {
  if (loaded) return
  try {
    const first = await api.listMovies({ limit: LIST_BATCH_SIZE, offset: 0 })
    const all: Movie[] = first.items.map(mapMovieListItem)
    let offset = all.length
    const { total } = first

    while (offset < total && all.length < MAX_MOVIES_PREFETCH) {
      const page = await api.listMovies({ limit: LIST_BATCH_SIZE, offset })
      const batch = page.items.map(mapMovieListItem)
      if (batch.length === 0) {
        break
      }
      all.push(...batch)
      offset += batch.length
    }

    moviesState.value = all
    loaded = true
  } catch (err) {
    console.error("[web-library-service] failed to load movies", err)
  }
}

const libraryStats: readonly LibraryStat[] = [
  { label: "Movies Indexed", value: "—", detail: "Connected to backend" },
  { label: "Favorite Picks", value: "—", detail: "Synced from backend" },
  { label: "Metadata Health", value: "—", detail: "Requires real scan" },
]

async function refreshLibraryPathsFromApi() {
  try {
    const settings = await api.getSettings()
    libraryPathsState.value = mapLibraryPathsFromSettings(settings.libraryPaths)
    organizeLibraryState.value = Boolean(settings.organizeLibrary)
  } catch (err) {
    console.error("[web-library-service] failed to load settings", err)
  }
}

function createWebLibraryService(): LibraryService {
  void ensureLoaded()

  return {
    movies: computed(() => moviesState.value),
    libraryStats,
    libraryPaths: computed(() => libraryPathsState.value),
    organizeLibrary: computed(() => organizeLibraryState.value),

    async refreshSettings() {
      await refreshLibraryPathsFromApi()
    },

    async setOrganizeLibrary(value: boolean) {
      const seq = ++organizeLibrarySaveSeq
      organizeLibraryState.value = value
      try {
        const next = await api.patchSettings({ organizeLibrary: value })
        if (seq === organizeLibrarySaveSeq) {
          organizeLibraryState.value = next.organizeLibrary
        }
      } catch (err) {
        if (seq === organizeLibrarySaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // 拉取失败时保留当前 UI 值，由上层错误提示处理
          }
        }
        throw err
      }
    },

    async addLibraryPath(path: string, title?: string) {
      const trimmed = path.trim()
      if (!trimmed) return
      await api.addLibraryPath({ path: trimmed, title: title?.trim() || undefined })
      await refreshLibraryPathsFromApi()
    },

    async updateLibraryPathTitle(id: string, title: string) {
      await api.updateLibraryPathTitle(id, { title: title.trim() })
      await refreshLibraryPathsFromApi()
    },

    async removeLibraryPath(id: string) {
      await api.deleteLibraryPath(id)
      await refreshLibraryPathsFromApi()
    },

    async scanLibraryPaths(paths?: string[]): Promise<TaskDTO | null> {
      return await api.startScan(paths?.length ? { paths } : undefined)
    },

    async refreshMovieMetadata(movieId: string): Promise<TaskDTO | null> {
      return await api.refreshMovieMetadata(movieId)
    },

    getMoviePlaybackUrl(movieId: string): string | null {
      const id = movieId.trim()
      if (!id) return null
      return moviePlaybackAbsoluteUrl(id)
    },

    async ensureMovieCached(movieId: string) {
      await ensureLoaded()
      const trimmed = movieId.trim()
      if (!trimmed) return
      if (moviesState.value.some((m) => m.id === trimmed)) {
        return
      }
      const detail = await loadMovieDetail(trimmed)
      if (!detail) {
        return
      }
      moviesState.value = [...moviesState.value, detail]
    },

    getMovieById(movieId) {
      return moviesState.value.find((m) => m.id === movieId)
    },

    getRelatedMovies(movieId, limit = 6) {
      return moviesState.value.filter((m) => m.id !== movieId).slice(0, limit)
    },

    toggleFavorite(movieId, nextValue) {
      const movie = moviesState.value.find((m) => m.id === movieId)
      if (!movie) return undefined

      const target = typeof nextValue === "boolean" ? nextValue : !movie.isFavorite
      moviesState.value = moviesState.value.map((m) =>
        m.id === movieId ? { ...m, isFavorite: target } : m,
      )
      return moviesState.value.find((m) => m.id === movieId)
    },

    async deleteMovie(movieId: string) {
      await api.deleteMovie(movieId)
      moviesState.value = moviesState.value.filter((m) => m.id !== movieId)
    },
  }
}

export async function loadMovieDetail(movieId: string): Promise<Movie | undefined> {
  try {
    const dto = await api.getMovie(movieId)
    return mapMovieDetail(dto)
  } catch {
    return undefined
  }
}

export const webLibraryService = createWebLibraryService()
