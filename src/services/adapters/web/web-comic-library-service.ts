import { computed, ref, shallowRef, type Ref } from "vue"
import { comicApi } from "@/api/comic-endpoints"
import type {
  ComicBookDetailDTO,
  ComicBookListItemDTO,
  ComicCacheStatusDTO,
  ComicLibraryPathDTO,
  ComicPageDTO,
  ComicReadingPreferencesDTO,
  ComicReadingProgressDTO,
  PatchComicBookBody,
  PutComicReadingPreferencesBody,
  SettingsDTO,
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
import type { ComicLibraryService } from "@/services/contracts/comic-library-service"

const LIST_BATCH_SIZE = 500

const comicsState: Ref<ComicBook[]> = shallowRef([])
const comicsLoadedState = ref(false)
const loadErrorState = ref<string | null>(null)
const comicLibraryEnabledState = ref(false)
const comicLibraryPathsState = ref<ComicLibrarySetting[]>([])
const defaultComicImportLibraryPathIdState = ref("")
const comicReaderState = ref<ComicReaderSettings>({
  mode: "page",
  fit: "contain",
  direction: "rtl",
})
const comicCacheState = ref<ComicCacheSettings>({
  maxBytes: 2 * 1024 * 1024 * 1024,
})

function formatLoadError(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return fallback
}

function mapComicLibraryPath(dto: ComicLibraryPathDTO): ComicLibrarySetting {
  return {
    id: dto.id,
    path: dto.path,
    title: dto.title,
    firstLibraryScanPending: dto.firstLibraryScanPending,
  }
}

function mapComicPage(dto: ComicPageDTO) {
  return {
    comicId: dto.comicId,
    index: dto.index,
    entryPath: dto.entryPath,
    fileName: dto.fileName,
    imageExt: dto.imageExt,
    width: dto.width,
    height: dto.height,
    imageUrl: dto.imageUrl,
    thumbUrl: dto.thumbUrl,
  }
}

function mapComicListItem(dto: ComicBookListItemDTO): ComicBook {
  return {
    id: dto.id,
    title: dto.title,
    tags: [...(dto.tags ?? [])],
    rating: dto.rating ?? null,
    isFavorite: Boolean(dto.isFavorite),
    readStatus: dto.readStatus,
    pageCount: dto.pageCount,
    currentPageIndex: dto.currentPageIndex,
    coverUrl: dto.coverUrl,
    sourceFileName: dto.sourceFileName,
    location: dto.location,
    addedAt: dto.addedAt,
    updatedAt: dto.updatedAt,
    lastReadAt: dto.lastReadAt,
    completedAt: dto.completedAt,
  }
}

function mapComicDetail(dto: ComicBookDetailDTO): ComicBook {
  return {
    ...mapComicListItem(dto),
    pages: (dto.pages ?? []).map(mapComicPage),
  }
}

function applySettingsFromDTO(settings: SettingsDTO) {
  comicLibraryEnabledState.value = Boolean(settings.comicLibraryEnabled)
  comicLibraryPathsState.value = (settings.comicLibraryPaths ?? []).map(mapComicLibraryPath)
  defaultComicImportLibraryPathIdState.value =
    settings.defaultComicImportLibraryPathId?.trim() ?? ""
  comicReaderState.value = {
    mode: settings.comicReader?.mode ?? "page",
    fit: settings.comicReader?.fit ?? "contain",
    direction: settings.comicReader?.direction ?? "rtl",
  }
  comicCacheState.value = {
    maxBytes: Number(settings.comicCache?.maxBytes ?? 2 * 1024 * 1024 * 1024),
  }
}

function mergeComicIntoCache(comic: ComicBook) {
  const idx = comicsState.value.findIndex((item) => item.id === comic.id)
  if (idx >= 0) {
    comicsState.value = comicsState.value.map((item, index) =>
      index === idx ? { ...item, ...comic } : item,
    )
    return
  }
  comicsState.value = [...comicsState.value, comic]
}

function applyProgressToCache(progress: ComicReadingProgressDTO) {
  const id = progress.comicId.trim()
  if (!id) return
  comicsState.value = comicsState.value.map((comic) => {
    if (comic.id !== id) return comic
    return {
      ...comic,
      currentPageIndex: progress.pageIndex,
      readStatus: progress.completed ? "read" : progress.pageIndex > 0 ? "reading" : "unread",
      lastReadAt: progress.updatedAt,
      completedAt: progress.completed ? progress.updatedAt : undefined,
    }
  })
}

function patchToBody(patch: ComicPatch): PatchComicBookBody {
  const body: PatchComicBookBody = {}
  if (patch.title !== undefined) {
    body.title = patch.title
  }
  if (patch.tags !== undefined) {
    body.tags = [...patch.tags]
  }
  if (patch.favorite !== undefined) {
    body.favorite = patch.favorite
  }
  if (patch.rating !== undefined) {
    if (patch.rating === null) {
      body.ratingClear = true
    } else {
      body.ratingSet = true
      body.rating = patch.rating
    }
  }
  return body
}

async function fetchPagedComics(params: ComicListParams = {}): Promise<ComicBook[]> {
  const first = await comicApi.listComics({
    ...params,
    limit: params.limit ?? LIST_BATCH_SIZE,
    offset: params.offset ?? 0,
  })
  const all = first.items.map(mapComicListItem)
  comicsState.value = all
  comicsLoadedState.value = true
  loadErrorState.value = null

  if (params.limit !== undefined) {
    return all
  }

  let offset = all.length
  while (offset < first.total) {
    const page = await comicApi.listComics({
      ...params,
      limit: LIST_BATCH_SIZE,
      offset,
    })
    const batch = page.items.map(mapComicListItem)
    if (batch.length === 0) break
    all.push(...batch)
    comicsState.value = [...all]
    offset += batch.length
  }
  return all
}

function createWebComicLibraryService(): ComicLibraryService {
  const impl: ComicLibraryService = {
    comics: computed(() => comicsState.value),
    comicsLoaded: computed(() => comicsLoadedState.value),
    loadError: computed(() => loadErrorState.value),
    comicLibraryEnabled: computed(() => comicLibraryEnabledState.value),
    comicLibraryPaths: computed(() => comicLibraryPathsState.value),
    defaultComicImportLibraryPathId: computed(() => defaultComicImportLibraryPathIdState.value),
    comicReader: computed(() => comicReaderState.value),
    comicCache: computed(() => comicCacheState.value),

    async refreshSettings() {
      const settings = await comicApi.getSettings()
      applySettingsFromDTO(settings)
    },

    async setComicLibraryEnabled(value: boolean) {
      comicLibraryEnabledState.value = value
      const settings = await comicApi.patchComicSettings({ comicLibraryEnabled: value })
      applySettingsFromDTO(settings)
    },

    async addComicLibraryPath(path: string, title?: string): Promise<TaskDTO | null> {
      const trimmed = path.trim()
      if (!trimmed) return null
      const dto = await comicApi.addComicLibraryPath({
        path: trimmed,
        title: title?.trim() || undefined,
      })
      const nextPath = mapComicLibraryPath(dto)
      comicLibraryPathsState.value = [
        ...comicLibraryPathsState.value.filter((item) => item.id !== nextPath.id),
        nextPath,
      ]
      return null
    },

    async updateComicLibraryPathTitle(id: string, title: string) {
      const dto = await comicApi.updateComicLibraryPathTitle(id.trim(), { title: title.trim() })
      const nextPath = mapComicLibraryPath(dto)
      comicLibraryPathsState.value = comicLibraryPathsState.value.map((item) =>
        item.id === nextPath.id ? nextPath : item,
      )
    },

    async removeComicLibraryPath(id: string) {
      const trimmed = id.trim()
      await comicApi.deleteComicLibraryPath(trimmed)
      comicLibraryPathsState.value = comicLibraryPathsState.value.filter(
        (item) => item.id !== trimmed,
      )
      if (defaultComicImportLibraryPathIdState.value === trimmed) {
        defaultComicImportLibraryPathIdState.value = ""
      }
    },

    async setDefaultComicImportLibraryPathId(id: string) {
      const trimmed = id.trim()
      defaultComicImportLibraryPathIdState.value = trimmed
      const settings = await comicApi.patchComicSettings({
        defaultComicImportLibraryPathId: trimmed,
      })
      applySettingsFromDTO(settings)
    },

    async patchComicReader(patch: Partial<ComicReaderSettings>) {
      const next = { ...comicReaderState.value, ...patch }
      comicReaderState.value = next
      const settings = await comicApi.patchComicSettings({ comicReader: next })
      applySettingsFromDTO(settings)
    },

    async patchComicCache(patch: Partial<ComicCacheSettings>) {
      const next = { ...comicCacheState.value, ...patch }
      comicCacheState.value = next
      const settings = await comicApi.patchComicSettings({ comicCache: next })
      applySettingsFromDTO(settings)
    },

    async reloadComicsFromApi(params?: ComicListParams) {
      try {
        comicsState.value = await fetchPagedComics(params)
        comicsLoadedState.value = true
        loadErrorState.value = null
      } catch (err) {
        loadErrorState.value = formatLoadError(err, "Failed to load comics")
        throw err
      }
    },

    getComicById(comicId?: string) {
      const id = comicId?.trim()
      if (!id) return undefined
      return comicsState.value.find((comic) => comic.id === id)
    },

    async loadComicDetail(comicId: string) {
      const id = comicId.trim()
      if (!id) return undefined
      try {
        const detail = mapComicDetail(await comicApi.getComic(id))
        mergeComicIntoCache(detail)
        loadErrorState.value = null
        return detail
      } catch (err) {
        loadErrorState.value = formatLoadError(err, "Failed to load comic")
        return undefined
      }
    },

    async patchComic(comicId: string, patch: ComicPatch) {
      const id = comicId.trim()
      if (!id) return undefined
      const detail = mapComicDetail(await comicApi.patchComic(id, patchToBody(patch)))
      mergeComicIntoCache(detail)
      return detail
    },

    async deleteComic(comicId: string) {
      const id = comicId.trim()
      if (!id) return
      await comicApi.deleteComic(id)
      comicsState.value = comicsState.value.filter((comic) => comic.id !== id)
    },

    async revealComicSource(comicId: string) {
      const id = comicId.trim()
      if (!id) return
      await comicApi.revealComicSource(id)
    },

    async scanComics(): Promise<TaskDTO | null> {
      return await comicApi.startComicScan()
    },

    async getComicProgress(comicId: string) {
      return await comicApi.getComicProgress(comicId.trim())
    },

    async saveComicProgress(comicId: string, pageIndex: number, completed: boolean) {
      const progress = await comicApi.putComicProgress(comicId.trim(), { pageIndex, completed })
      applyProgressToCache(progress)
      return progress
    },

    async resetComicProgress(comicId: string) {
      const id = comicId.trim()
      if (!id) return
      await comicApi.deleteComicProgress(id)
      comicsState.value = comicsState.value.map((comic) =>
        comic.id === id
          ? {
              ...comic,
              currentPageIndex: 0,
              readStatus: "unread",
              lastReadAt: undefined,
              completedAt: undefined,
            }
          : comic,
      )
    },

    async getComicPreferences(comicId: string): Promise<ComicReadingPreferencesDTO> {
      return await comicApi.getComicPreferences(comicId.trim())
    },

    async saveComicPreferences(
      comicId: string,
      prefs: PutComicReadingPreferencesBody,
    ): Promise<ComicReadingPreferencesDTO> {
      return await comicApi.putComicPreferences(comicId.trim(), prefs)
    },

    async getComicCacheStatus(): Promise<ComicCacheStatusDTO> {
      return await comicApi.getComicCacheStatus()
    },

    async cleanupComicCache(): Promise<ComicCacheStatusDTO> {
      return await comicApi.cleanupComicCache()
    },
  }

  return impl
}

export const webComicLibraryService = createWebComicLibraryService()
