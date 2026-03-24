import { computed, ref, type Ref } from "vue"
import type {
  ListActorsParams,
  MetadataRefreshQueuedDTO,
  PatchMovieBody,
  TaskDTO,
} from "@/api/types"
import { HttpClientError } from "@/api/http-client"
import { api } from "@/api/endpoints"
import { moviePlaybackAbsoluteUrl } from "@/api/playback-url"
import type { LibrarySetting } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import { i18n } from "@/i18n"
import { buildSettingsDashboardStats } from "@/lib/library-stats"
import { sampleRandomMovies } from "@/lib/random-sample"
import { playedMovieCount } from "@/lib/played-movies-storage"
import type { LibraryService } from "@/services/contracts/library-service"
import { mapMovieDetail, mapMovieListItem } from "./mappers"

const moviesState: Ref<Movie[]> = ref([])
const trashedMoviesState: Ref<Movie[]> = ref([])
const libraryPathsState: Ref<LibrarySetting[]> = ref([])
/** 与后端 config.Default() / library-config.cfg 默认一致，避免首屏在 GET 完成前误显示为关 */
const organizeLibraryState = ref(true)
/** 与后端默认一致：关，避免误触新库「首次扫描」扩展逻辑 */
const extendedLibraryImportState = ref(false)
const autoLibraryWatchState = ref(true)
const metadataMovieProviderState = ref("")
const metadataMovieProvidersState = ref<string[]>([])
/** 忽略过期的 PATCH 响应，避免快速连点时状态被旧请求写乱 */
let organizeLibrarySaveSeq = 0
let extendedLibraryImportSaveSeq = 0
let autoLibraryWatchSaveSeq = 0
let metadataMovieProviderSaveSeq = 0
let loaded = false
let reloadMoviesDebounce: ReturnType<typeof setTimeout> | null = null

/** 单次请求条数；多页拉取直至 total 或达到上限，避免首屏只看见前 200 条 */
const LIST_BATCH_SIZE = 500
const MAX_MOVIES_PREFETCH = 10_000

function mapLibraryPathsFromSettings(
  paths: { id: string; path: string; title: string; firstLibraryScanPending?: boolean }[],
): LibrarySetting[] {
  return paths.map((p) => ({
    id: p.id,
    path: p.path,
    title: p.title,
    firstLibraryScanPending: p.firstLibraryScanPending,
  }))
}

async function fetchAllMoviesFromApi(): Promise<Movie[]> {
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

  return all
}

async function fetchAllTrashedMoviesFromApi(): Promise<Movie[]> {
  const first = await api.listMovies({ limit: LIST_BATCH_SIZE, offset: 0, mode: "trash" })
  const all: Movie[] = first.items.map(mapMovieListItem)
  let offset = all.length
  const { total } = first

  while (offset < total && all.length < MAX_MOVIES_PREFETCH) {
    const page = await api.listMovies({ limit: LIST_BATCH_SIZE, offset, mode: "trash" })
    const batch = page.items.map(mapMovieListItem)
    if (batch.length === 0) {
      break
    }
    all.push(...batch)
    offset += batch.length
  }

  return all
}

async function reloadMoviesFromApiImmediate() {
  if (reloadMoviesDebounce) {
    clearTimeout(reloadMoviesDebounce)
    reloadMoviesDebounce = null
  }
  try {
    const [active, trashed] = await Promise.all([
      fetchAllMoviesFromApi(),
      fetchAllTrashedMoviesFromApi(),
    ])
    moviesState.value = active
    trashedMoviesState.value = trashed
    loaded = true
  } catch (err) {
    console.error("[web-library-service] failed to reload movies", err)
  }
}

async function ensureLoaded() {
  if (loaded) return
  try {
    const [active, trashed] = await Promise.all([
      fetchAllMoviesFromApi(),
      fetchAllTrashedMoviesFromApi(),
    ])
    moviesState.value = active
    trashedMoviesState.value = trashed
    loaded = true
  } catch (err) {
    console.error("[web-library-service] failed to load movies", err)
  }
}

function mergeMovieIntoListState(movie: Movie) {
  const id = movie.id.trim()
  if (!id) return
  const inTrash = Boolean(movie.trashedAt?.trim())
  if (inTrash) {
    const idx = trashedMoviesState.value.findIndex((m) => m.id === id)
    if (idx >= 0) {
      trashedMoviesState.value = trashedMoviesState.value.map((m, i) =>
        i === idx ? { ...m, ...movie } : m,
      )
    } else {
      trashedMoviesState.value = [...trashedMoviesState.value, movie]
    }
    moviesState.value = moviesState.value.filter((m) => m.id !== id)
    return
  }
  const idx = moviesState.value.findIndex((m) => m.id === id)
  if (idx >= 0) {
    moviesState.value = moviesState.value.map((m, i) => (i === idx ? { ...m, ...movie } : m))
  } else {
    moviesState.value = [...moviesState.value, movie]
  }
  trashedMoviesState.value = trashedMoviesState.value.filter((m) => m.id !== id)
}

async function refreshLibraryPathsFromApi() {
  try {
    const settings = await api.getSettings()
    libraryPathsState.value = mapLibraryPathsFromSettings(settings.libraryPaths)
    organizeLibraryState.value = Boolean(settings.organizeLibrary)
    extendedLibraryImportState.value = Boolean(settings.extendedLibraryImport)
    autoLibraryWatchState.value = settings.autoLibraryWatch !== false
    metadataMovieProviderState.value = settings.metadataMovieProvider?.trim() ?? ""
    metadataMovieProvidersState.value = Array.isArray(settings.metadataMovieProviders)
      ? [...settings.metadataMovieProviders]
      : []
  } catch (err) {
    console.error("[web-library-service] failed to load settings", err)
  }
}

function createWebLibraryService(): LibraryService {
  void ensureLoaded()

  const impl: LibraryService = {
    movies: computed(() => moviesState.value),
    trashedMovies: computed(() => trashedMoviesState.value),
    libraryStats: computed(() => {
      const loc = i18n.global.locale.value as string
      return buildSettingsDashboardStats(moviesState.value, playedMovieCount.value, loc)
    }),
    libraryPaths: computed(() => libraryPathsState.value),
    organizeLibrary: computed(() => organizeLibraryState.value),
    extendedLibraryImport: computed(() => extendedLibraryImportState.value),
    autoLibraryWatch: computed(() => autoLibraryWatchState.value),
    metadataMovieProvider: computed(() => metadataMovieProviderState.value),
    metadataMovieProviders: computed(() => metadataMovieProvidersState.value),

    async refreshSettings() {
      await refreshLibraryPathsFromApi()
    },

    async reloadMoviesFromApi() {
      if (reloadMoviesDebounce) {
        clearTimeout(reloadMoviesDebounce)
        reloadMoviesDebounce = null
      }
      reloadMoviesDebounce = setTimeout(() => {
        reloadMoviesDebounce = null
        void reloadMoviesFromApiImmediate()
      }, 450)
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

    async setExtendedLibraryImport(value: boolean) {
      const seq = ++extendedLibraryImportSaveSeq
      extendedLibraryImportState.value = value
      try {
        const next = await api.patchSettings({ extendedLibraryImport: value })
        if (seq === extendedLibraryImportSaveSeq) {
          extendedLibraryImportState.value = Boolean(next.extendedLibraryImport)
        }
      } catch (err) {
        if (seq === extendedLibraryImportSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async setAutoLibraryWatch(value: boolean) {
      const seq = ++autoLibraryWatchSaveSeq
      autoLibraryWatchState.value = value
      try {
        const next = await api.patchSettings({ autoLibraryWatch: value })
        if (seq === autoLibraryWatchSaveSeq) {
          autoLibraryWatchState.value = next.autoLibraryWatch !== false
        }
      } catch (err) {
        if (seq === autoLibraryWatchSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // 拉取失败时保留当前 UI 值，由上层错误提示处理
          }
        }
        throw err
      }
    },

    async setMetadataMovieProvider(name: string) {
      const trimmed = name.trim()
      const seq = ++metadataMovieProviderSaveSeq
      metadataMovieProviderState.value = trimmed
      try {
        const next = await api.patchSettings({ metadataMovieProvider: trimmed })
        if (seq === metadataMovieProviderSaveSeq) {
          metadataMovieProviderState.value = next.metadataMovieProvider?.trim() ?? ""
          if (Array.isArray(next.metadataMovieProviders)) {
            metadataMovieProvidersState.value = [...next.metadataMovieProviders]
          }
        }
      } catch (err) {
        if (seq === metadataMovieProviderSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async addLibraryPath(path: string, title?: string): Promise<TaskDTO | null> {
      const trimmed = path.trim()
      if (!trimmed) return null
      const res = await api.addLibraryPath({ path: trimmed, title: title?.trim() || undefined })
      await refreshLibraryPathsFromApi()
      return res.scanTask ?? null
    },

    async updateLibraryPathTitle(id: string, title: string) {
      await api.updateLibraryPathTitle(id, { title: title.trim() })
      await refreshLibraryPathsFromApi()
    },

    async removeLibraryPath(id: string) {
      await api.deleteLibraryPath(id)
      await refreshLibraryPathsFromApi()
      await reloadMoviesFromApiImmediate()
    },

    async scanLibraryPaths(paths?: string[]): Promise<TaskDTO | null> {
      return await api.startScan(paths?.length ? { paths } : undefined)
    },

    async refreshMovieMetadata(movieId: string): Promise<TaskDTO | null> {
      return await api.refreshMovieMetadata(movieId)
    },

    async refreshMetadataForLibraryPaths(paths: string[]): Promise<MetadataRefreshQueuedDTO> {
      const cleaned = paths.map((p) => p.trim()).filter(Boolean)
      return await api.startMetadataRefreshByPaths({ paths: cleaned })
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
      if (
        moviesState.value.some((m) => m.id === trimmed) ||
        trashedMoviesState.value.some((m) => m.id === trimmed)
      ) {
        return
      }
      await loadMovieDetail(trimmed)
    },

    mergeMovieIntoCache(movie: Movie) {
      mergeMovieIntoListState(movie)
    },

    getMovieById(movieId) {
      const id = movieId?.trim()
      if (!id) return undefined
      return (
        moviesState.value.find((m) => m.id === id) ??
        trashedMoviesState.value.find((m) => m.id === id)
      )
    },

    getRelatedMovies(movieId, limit = 6) {
      const id = movieId.trim()
      const pool = moviesState.value.filter((m) => m.id !== id)
      return sampleRandomMovies(pool, limit, id || "_")
    },

    async patchMovie(movieId: string, body: PatchMovieBody) {
      await ensureLoaded()
      const id = movieId.trim()
      if (!id) return undefined

      let snapshotActive = moviesState.value.map((m) => ({ ...m }))
      let snapshotTrashed = trashedMoviesState.value.map((m) => ({ ...m }))
      let existing =
        moviesState.value.find((m) => m.id === id) ??
        trashedMoviesState.value.find((m) => m.id === id)
      if (!existing) {
        await loadMovieDetail(id)
        snapshotActive = moviesState.value.map((m) => ({ ...m }))
        snapshotTrashed = trashedMoviesState.value.map((m) => ({ ...m }))
        existing =
          moviesState.value.find((m) => m.id === id) ??
          trashedMoviesState.value.find((m) => m.id === id)
      }

      try {
        const dto = await api.patchMovie(id, body)
        const mapped = mapMovieDetail(dto)
        mergeMovieIntoListState(mapped)
        return impl.getMovieById(id)
      } catch (err) {
        moviesState.value = snapshotActive
        trashedMoviesState.value = snapshotTrashed
        throw err
      }
    },

    async toggleFavorite(movieId, nextValue) {
      const id = movieId.trim()
      const movie = moviesState.value.find((m) => m.id === id)
      const target = typeof nextValue === "boolean" ? nextValue : !(movie?.isFavorite ?? false)
      return await impl.patchMovie(id, { isFavorite: target })
    },

    async deleteMovie(movieId: string) {
      const id = movieId.trim()
      await api.deleteMovie(id)
      moviesState.value = moviesState.value.filter((m) => m.id !== id)
      try {
        trashedMoviesState.value = await fetchAllTrashedMoviesFromApi()
      } catch {
        // 忽略回收站刷新失败，主列表已一致
      }
    },

    async restoreMovie(movieId: string) {
      const id = movieId.trim()
      await api.restoreMovie(id)
      await reloadMoviesFromApiImmediate()
    },

    async deleteMoviePermanently(movieId: string) {
      const id = movieId.trim()
      await api.deleteMovie(id, { permanent: true })
      trashedMoviesState.value = trashedMoviesState.value.filter((m) => m.id !== id)
    },

    async listActors(params?: ListActorsParams) {
      return await api.listActors(params)
    },

    async patchActorUserTags(name: string, userTags: string[]) {
      return await api.patchActorUserTags(name.trim(), userTags)
    },
  }

  return impl
}

export async function loadMovieDetail(movieId: string): Promise<Movie | undefined> {
  const id = movieId.trim()
  if (!id) return undefined
  try {
    const dto = await api.getMovie(id)
    const mapped = mapMovieDetail(dto)
    mergeMovieIntoListState(mapped)
    return mapped
  } catch (err) {
    const extra = err instanceof HttpClientError ? ` status=${err.status}` : ""
    console.error(`[web-library-service] loadMovieDetail failed${extra}`, id, err)
    return undefined
  }
}

export const webLibraryService = createWebLibraryService()
