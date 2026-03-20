import type { ComputedRef } from "vue"
import type {
  LibrarySetting,
  LibraryStat,
  ScanIntervalOption,
} from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"

export interface LibraryService {
  movies: ComputedRef<readonly Movie[]>
  libraryStats: readonly LibraryStat[]
  libraryPaths: readonly LibrarySetting[]
  scanIntervals: readonly ScanIntervalOption[]
  getMovieById(movieId?: string): Movie | undefined
  getRelatedMovies(movieId: string, limit?: number): Movie[]
  toggleFavorite(movieId: string, nextValue?: boolean): Movie | undefined
}
