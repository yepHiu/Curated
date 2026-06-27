import type { ComputedRef } from "vue"
import type {
  ComicCacheStatusDTO,
  ComicReadingPreferencesDTO,
  ComicReadingProgressDTO,
  PutComicReadingPreferencesBody,
  TaskDTO,
} from "@/api/types"
import type {
  ComicBook,
  ComicCacheSettings,
  ComicLibrarySetting,
  ComicListParams,
  ComicPatch,
  ComicReaderSettings,
} from "@/domain/comic/types"

export interface ComicLibraryService {
  comics: ComputedRef<readonly ComicBook[]>
  comicsLoaded: ComputedRef<boolean>
  loadError: ComputedRef<string | null>
  comicLibraryEnabled: ComputedRef<boolean>
  comicLibraryPaths: ComputedRef<readonly ComicLibrarySetting[]>
  defaultComicImportLibraryPathId: ComputedRef<string>
  comicReader: ComputedRef<ComicReaderSettings>
  comicCache: ComputedRef<ComicCacheSettings>
  refreshSettings(): Promise<void>
  setComicLibraryEnabled(value: boolean): Promise<void>
  addComicLibraryPath(path: string, title?: string): Promise<TaskDTO | null>
  updateComicLibraryPathTitle(id: string, title: string): Promise<void>
  removeComicLibraryPath(id: string): Promise<void>
  setDefaultComicImportLibraryPathId(id: string): Promise<void>
  patchComicReader(patch: Partial<ComicReaderSettings>): Promise<void>
  patchComicCache(patch: Partial<ComicCacheSettings>): Promise<void>
  reloadComicsFromApi(params?: ComicListParams): Promise<void>
  getComicById(comicId?: string): ComicBook | undefined
  loadComicDetail(comicId: string): Promise<ComicBook | undefined>
  patchComic(comicId: string, patch: ComicPatch): Promise<ComicBook | undefined>
  deleteComic(comicId: string): Promise<void>
  revealComicSource(comicId: string): Promise<void>
  scanComics(): Promise<TaskDTO | null>
  getComicProgress(comicId: string): Promise<ComicReadingProgressDTO>
  saveComicProgress(
    comicId: string,
    pageIndex: number,
    completed: boolean,
  ): Promise<ComicReadingProgressDTO>
  resetComicProgress(comicId: string): Promise<void>
  getComicPreferences(comicId: string): Promise<ComicReadingPreferencesDTO>
  saveComicPreferences(
    comicId: string,
    prefs: PutComicReadingPreferencesBody,
  ): Promise<ComicReadingPreferencesDTO>
  getComicCacheStatus(): Promise<ComicCacheStatusDTO>
  cleanupComicCache(): Promise<ComicCacheStatusDTO>
}
