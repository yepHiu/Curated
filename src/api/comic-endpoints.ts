import { httpClient } from "./http-client"
import type {
  AddComicLibraryPathBody,
  AddComicLibraryPathResultDTO,
  ComicBookDetailDTO,
  ComicBooksPageDTO,
  ComicCacheStatusDTO,
  ComicImportUploadProgress,
  ComicLibraryPathDTO,
  ComicPageDTO,
  ComicReadingPreferencesDTO,
  ComicReadingProgressDTO,
  ListComicBooksParams,
  PatchComicBookBody,
  PatchComicSettingsBody,
  PutComicProgressBody,
  PutComicReadingPreferencesBody,
  SettingsDTO,
  StartScanBody,
  TaskDTO,
  UpdateComicLibraryPathBody,
} from "./types"

interface ComicImportApiOptions {
  onUploadProgress?: (progress: ComicImportUploadProgress) => void
}

function comicListParamsToQuery(
  params?: ListComicBooksParams,
): Record<string, string | number | undefined> | undefined {
  if (!params) return undefined
  return {
    q: params.q,
    tag: params.tag,
    favorite: params.favorite === undefined ? undefined : String(params.favorite),
    readStatus: params.readStatus,
    limit: params.limit,
    offset: params.offset,
  }
}

function relativePathForFile(file: File): string {
  const candidate = (file as File & { webkitRelativePath?: string }).webkitRelativePath
  return candidate?.trim() || file.name
}

export const comicApi = {
  getSettings(): Promise<SettingsDTO> {
    return httpClient.get<SettingsDTO>("/settings")
  },

  patchComicSettings(body: PatchComicSettingsBody): Promise<SettingsDTO> {
    return httpClient.patch<SettingsDTO>("/settings", body)
  },

  listComicLibraryPaths(): Promise<ComicLibraryPathDTO[]> {
    return httpClient.get<ComicLibraryPathDTO[]>("/library/comics/paths")
  },

  addComicLibraryPath(body: AddComicLibraryPathBody): Promise<AddComicLibraryPathResultDTO> {
    return httpClient.post<AddComicLibraryPathResultDTO>("/library/comics/paths", body)
  },

  updateComicLibraryPathTitle(
    id: string,
    body: UpdateComicLibraryPathBody,
  ): Promise<ComicLibraryPathDTO> {
    return httpClient.patch<ComicLibraryPathDTO>(
      `/library/comics/paths/${encodeURIComponent(id)}`,
      body,
    )
  },

  deleteComicLibraryPath(id: string): Promise<void> {
    return httpClient.delete(`/library/comics/paths/${encodeURIComponent(id)}`)
  },

  startComicScan(body?: StartScanBody): Promise<TaskDTO> {
    return httpClient.post<TaskDTO>("/library/comics/scans", body ?? {})
  },

  importComics(files: File[], options?: ComicImportApiOptions): Promise<TaskDTO> {
    const form = new FormData()
    const totalBytes = files.reduce((sum, file) => sum + file.size, 0)
    if (totalBytes > 0) {
      form.set("totalBytes", String(totalBytes))
    }
    for (const file of files) {
      const relativePath = relativePathForFile(file)
      form.append("relativePath", relativePath)
      form.append("files", file, relativePath)
    }
    return httpClient.postFormWithProgress<TaskDTO>("/import/comics", form, {
      onUploadProgress: options?.onUploadProgress,
    })
  },

  listComics(params?: ListComicBooksParams): Promise<ComicBooksPageDTO> {
    return httpClient.get<ComicBooksPageDTO>("/library/comics", comicListParamsToQuery(params))
  },

  getComic(id: string): Promise<ComicBookDetailDTO> {
    return httpClient.get<ComicBookDetailDTO>(`/library/comics/${encodeURIComponent(id)}`)
  },

  patchComic(id: string, body: PatchComicBookBody): Promise<ComicBookDetailDTO> {
    return httpClient.patch<ComicBookDetailDTO>(
      `/library/comics/${encodeURIComponent(id)}`,
      body,
    )
  },

  deleteComic(id: string): Promise<void> {
    return httpClient.delete(`/library/comics/${encodeURIComponent(id)}`)
  },

  revealComicSource(id: string): Promise<void> {
    return httpClient.post<void>(`/library/comics/books/${encodeURIComponent(id)}/reveal`)
  },

  listComicPages(id: string): Promise<ComicPageDTO[]> {
    return httpClient.get<ComicPageDTO[]>(
      `/library/comics/books/${encodeURIComponent(id)}/pages`,
    )
  },

  getComicProgress(id: string): Promise<ComicReadingProgressDTO> {
    return httpClient.get<ComicReadingProgressDTO>(
      `/library/comics/books/${encodeURIComponent(id)}/progress`,
    )
  },

  putComicProgress(
    id: string,
    body: PutComicProgressBody,
  ): Promise<ComicReadingProgressDTO> {
    return httpClient.put<ComicReadingProgressDTO>(
      `/library/comics/books/${encodeURIComponent(id)}/progress`,
      body,
    )
  },

  deleteComicProgress(id: string): Promise<void> {
    return httpClient.delete(`/library/comics/books/${encodeURIComponent(id)}/progress`)
  },

  getComicPreferences(id: string): Promise<ComicReadingPreferencesDTO> {
    return httpClient.get<ComicReadingPreferencesDTO>(
      `/library/comics/books/${encodeURIComponent(id)}/preferences`,
    )
  },

  putComicPreferences(
    id: string,
    body: PutComicReadingPreferencesBody,
  ): Promise<ComicReadingPreferencesDTO> {
    return httpClient.put<ComicReadingPreferencesDTO>(
      `/library/comics/books/${encodeURIComponent(id)}/preferences`,
      body,
    )
  },

  getComicCacheStatus(): Promise<ComicCacheStatusDTO> {
    return httpClient.get<ComicCacheStatusDTO>("/library/comics/cache/status")
  },

  cleanupComicCache(): Promise<ComicCacheStatusDTO> {
    return httpClient.post<ComicCacheStatusDTO>("/library/comics/cache/cleanup", {})
  },
}
