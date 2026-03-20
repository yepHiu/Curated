import { computed, ref, type Ref } from "vue"
import type { TaskDTO } from "@/api/types"
import { api } from "@/api/endpoints"
import type { LibrarySetting, LibraryStat } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import type { LibraryService } from "@/services/contracts/library-service"
import { mapMovieDetail, mapMovieListItem } from "./mappers"

const moviesState: Ref<Movie[]> = ref([])
const libraryPathsState: Ref<LibrarySetting[]> = ref([])
const organizeLibraryState = ref(false)
let loaded = false

function mapLibraryPathsFromSettings(paths: { id: string; path: string; title: string }[]): LibrarySetting[] {
  return paths.map((p) => ({ id: p.id, path: p.path, title: p.title }))
}

async function ensureLoaded() {
  if (loaded) return
  try {
    const page = await api.listMovies({ limit: 200, offset: 0 })
    moviesState.value = page.items.map(mapMovieListItem)
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
  ensureLoaded()

  return {
    movies: computed(() => moviesState.value),
    libraryStats,
    libraryPaths: computed(() => libraryPathsState.value),
    organizeLibrary: computed(() => organizeLibraryState.value),

    async refreshSettings() {
      await refreshLibraryPathsFromApi()
    },

    async setOrganizeLibrary(value: boolean) {
      const next = await api.patchSettings({ organizeLibrary: value })
      organizeLibraryState.value = next.organizeLibrary
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
