import type { ComputedRef } from "vue"
import type {
  ComicCacheStatusDTO,
  ComicImportUploadProgress,
  LibraryPathStorageStatusDTO,
  ComicReadingPreferencesDTO,
  ComicReadingProgressDTO,
  BookCommentDTO,
  PutBookCommentBody,
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
  autoComicLibraryWatch: ComputedRef<boolean>
  comicLibraryPaths: ComputedRef<readonly ComicLibrarySetting[]>
  comicLibraryPathStorageStatuses: ComputedRef<readonly LibraryPathStorageStatusDTO[]>
  defaultComicImportLibraryPathId: ComputedRef<string>
  comicReader: ComputedRef<ComicReaderSettings>
  comicCache: ComputedRef<ComicCacheSettings>
  refreshSettings(): Promise<void>
  /** 首次进入补齐设置与全量列表；已加载则跳过。扫描/导入终态仍应调用 reloadComicsFromApi。 */
  ensureComicsLoaded(): Promise<void>
  checkComicLibraryPathStorageStatus(libraryPathIds?: string[]): Promise<void>
  setComicLibraryEnabled(value: boolean): Promise<void>
  setAutoComicLibraryWatch(value: boolean): Promise<void>
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
  scanComics(paths?: string[]): Promise<TaskDTO | null>
  importComics(
    files: File[],
    options?: { onUploadProgress?: (progress: ComicImportUploadProgress) => void },
  ): Promise<TaskDTO | null>
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
  /** Web：GET /library/comics/books/{id}/comment；Mock：localStorage */
  getComicComment(comicId: string): Promise<BookCommentDTO>
  /** Web：PUT /library/comics/books/{id}/comment；Mock：localStorage */
  putComicComment(comicId: string, body: PutBookCommentBody): Promise<BookCommentDTO>
  getComicCacheStatus(): Promise<ComicCacheStatusDTO>
  cleanupComicCache(): Promise<ComicCacheStatusDTO>
}
