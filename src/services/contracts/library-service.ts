import type { ComputedRef } from "vue"
import type { TaskDTO } from "@/api/types"
import type { LibrarySetting, LibraryStat } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"

export interface LibraryService {
  movies: ComputedRef<readonly Movie[]>
  libraryStats: readonly LibraryStat[]
  libraryPaths: ComputedRef<readonly LibrarySetting[]>
  refreshSettings(): Promise<void>
  /** 与后端 GET/PATCH /api/settings 同步；mock 为本地状态 */
  organizeLibrary: ComputedRef<boolean>
  setOrganizeLibrary(value: boolean): Promise<void>
  addLibraryPath(path: string, title?: string): Promise<void>
  updateLibraryPathTitle(id: string, title: string): Promise<void>
  removeLibraryPath(id: string): Promise<void>
  /** Returns task when web scan started; mock returns null. */
  scanLibraryPaths(paths?: string[]): Promise<TaskDTO | null>
  getMovieById(movieId?: string): Movie | undefined
  getRelatedMovies(movieId: string, limit?: number): Movie[]
  toggleFavorite(movieId: string, nextValue?: boolean): Movie | undefined
  /** 删除影片（Web：请求后端并从本地列表移除；Mock：仅从内存列表移除） */
  deleteMovie(movieId: string): Promise<void>
}
