import { computed, ref, type Ref } from "vue"
import { api } from "@/api/endpoints"
import type { LibrarySetting, LibraryStat, ScanIntervalOption } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import type { LibraryService } from "@/services/contracts/library-service"
import { mapMovieDetail, mapMovieListItem } from "./mappers"

const moviesState: Ref<Movie[]> = ref([])
let loaded = false

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

const scanIntervals: readonly ScanIntervalOption[] = [
  { label: "Every 30 minutes", value: "1800" },
  { label: "Every hour", value: "3600" },
  { label: "Every 6 hours", value: "21600" },
  { label: "Daily", value: "86400" },
]

function createWebLibraryService(): LibraryService {
  ensureLoaded()

  return {
    movies: computed(() => moviesState.value),
    libraryStats,
    libraryPaths: [] as LibrarySetting[],
    scanIntervals,

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
