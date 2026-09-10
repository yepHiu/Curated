import { computed, ref, shallowRef, type Ref } from "vue"
import { photoApi } from "@/api/photo-endpoints"
import type {
  PhotoBookDetailDTO,
  PhotoBookListItemDTO,
  PhotoLibraryPathDTO,
  SettingsDTO,
  TaskDTO,
} from "@/api/types"
import type {
  PhotoCacheSettings,
  PhotoBook,
  PhotoListParams,
  PhotoLibrarySetting,
  PhotoPage,
  PhotoViewerSettings,
} from "@/domain/photo/types"
import type { PhotoLibraryService } from "@/services/contracts/photo-library-service"

const photosState: Ref<PhotoBook[]> = shallowRef([])
const photosLoadedState = ref(false)
const loadErrorState = ref<string | null>(null)
const photoLibraryEnabledState = ref(false)
const autoPhotoLibraryWatchState = ref(true)
const photoLibraryPathsState = ref<PhotoLibrarySetting[]>([])
const defaultPhotoImportLibraryPathIdState = ref("")
const photoViewerState = ref<PhotoViewerSettings>({
  mode: "page",
  fit: "contain",
  direction: "ltr",
})
const photoCacheState = ref<PhotoCacheSettings>({
  maxBytes: 5 * 1024 * 1024 * 1024,
})

function mapPhotoLibraryPath(dto: PhotoLibraryPathDTO): PhotoLibrarySetting {
  return {
    id: dto.id,
    path: dto.path,
    title: dto.title,
    firstLibraryScanPending: dto.firstLibraryScanPending,
  }
}

function mapPhotoListItem(dto: PhotoBookListItemDTO): PhotoBook {
  return {
    id: dto.id,
    title: dto.title,
    tags: dto.tags ?? [],
    rating: dto.rating ?? null,
    isFavorite: Boolean(dto.isFavorite),
    pageCount: Number(dto.pageCount ?? 0),
    currentPageIndex: Number(dto.currentPageIndex ?? 0),
    coverUrl: dto.coverUrl,
    sourceFileName: dto.sourceFileName,
    location: dto.location,
    addedAt: dto.addedAt,
    updatedAt: dto.updatedAt,
    lastViewedAt: dto.lastViewedAt,
    completedAt: dto.completedAt,
  }
}

function mapPhotoDetail(dto: PhotoBookDetailDTO): PhotoBook {
  return {
    ...mapPhotoListItem(dto),
    pages: (dto.pages ?? []).map(
      (page): PhotoPage => ({
        photoId: page.photoId,
        index: page.index,
        entryPath: page.entryPath,
        fileName: page.fileName,
        imageExt: page.imageExt,
        width: page.width,
        height: page.height,
        imageUrl: page.imageUrl,
        thumbUrl: page.thumbUrl,
      }),
    ),
  }
}

function mergePhotoIntoCache(photo: PhotoBook) {
  photosState.value = [
    photo,
    ...photosState.value.filter((item) => item.id !== photo.id),
  ]
}

function formatLoadError(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return fallback
}

function applySettingsFromDTO(settings: SettingsDTO) {
  photoLibraryEnabledState.value = Boolean(settings.photoLibraryEnabled)
  autoPhotoLibraryWatchState.value = settings.autoPhotoLibraryWatch ?? true
  photoLibraryPathsState.value = (settings.photoLibraryPaths ?? []).map(mapPhotoLibraryPath)
  defaultPhotoImportLibraryPathIdState.value =
    settings.defaultPhotoImportLibraryPathId?.trim() ?? ""
  photoViewerState.value = {
    mode: settings.photoViewer?.mode ?? "page",
    fit: settings.photoViewer?.fit ?? "contain",
    direction: settings.photoViewer?.direction ?? "ltr",
  }
  photoCacheState.value = {
    maxBytes: Number(settings.photoCache?.maxBytes ?? 5 * 1024 * 1024 * 1024),
  }
}

function createWebPhotoLibraryService(): PhotoLibraryService {
  return {
    async importPhotos(files, options) {
      if (!photoLibraryEnabledState.value) throw new Error("Photo library is disabled")
      return photoApi.importPhotos(files, options)
    },
    photos: computed(() => photosState.value),
    photosLoaded: computed(() => photosLoadedState.value),
    loadError: computed(() => loadErrorState.value),
    photoLibraryEnabled: computed(() => photoLibraryEnabledState.value),
    autoPhotoLibraryWatch: computed(() => autoPhotoLibraryWatchState.value),
    photoLibraryPaths: computed(() => photoLibraryPathsState.value),
    defaultPhotoImportLibraryPathId: computed(() => defaultPhotoImportLibraryPathIdState.value),
    photoViewer: computed(() => photoViewerState.value),
    photoCache: computed(() => photoCacheState.value),

    async refreshSettings() {
      const settings = await photoApi.getSettings()
      applySettingsFromDTO(settings)
    },

    async setPhotoLibraryEnabled(value: boolean) {
      photoLibraryEnabledState.value = value
      const settings = await photoApi.patchPhotoSettings({ photoLibraryEnabled: value })
      applySettingsFromDTO(settings)
    },

    async setAutoPhotoLibraryWatch(value: boolean) {
      autoPhotoLibraryWatchState.value = value
      const settings = await photoApi.patchPhotoSettings({ autoPhotoLibraryWatch: value })
      applySettingsFromDTO(settings)
    },

    async addPhotoLibraryPath(path: string, title?: string): Promise<TaskDTO | null> {
      const trimmed = path.trim()
      if (!trimmed) return null
      const dto = await photoApi.addPhotoLibraryPath({
        path: trimmed,
        title: title?.trim() || undefined,
      })
      const nextPath = mapPhotoLibraryPath(dto)
      photoLibraryPathsState.value = [
        ...photoLibraryPathsState.value.filter((item) => item.id !== nextPath.id),
        nextPath,
      ]
      return dto.scanTask ?? null
    },

    async updatePhotoLibraryPathTitle(id: string, title: string) {
      const dto = await photoApi.updatePhotoLibraryPathTitle(id.trim(), { title: title.trim() })
      const nextPath = mapPhotoLibraryPath(dto)
      photoLibraryPathsState.value = photoLibraryPathsState.value.map((item) =>
        item.id === nextPath.id ? nextPath : item,
      )
    },

    async removePhotoLibraryPath(id: string) {
      const trimmed = id.trim()
      await photoApi.deletePhotoLibraryPath(trimmed)
      photoLibraryPathsState.value = photoLibraryPathsState.value.filter(
        (item) => item.id !== trimmed,
      )
      if (defaultPhotoImportLibraryPathIdState.value === trimmed) {
        defaultPhotoImportLibraryPathIdState.value = ""
      }
    },

    async setDefaultPhotoImportLibraryPathId(id: string) {
      const trimmed = id.trim()
      defaultPhotoImportLibraryPathIdState.value = trimmed
      const settings = await photoApi.patchPhotoSettings({
        defaultPhotoImportLibraryPathId: trimmed,
      })
      applySettingsFromDTO(settings)
    },

    async patchPhotoViewer(patch: Partial<PhotoViewerSettings>) {
      const next = { ...photoViewerState.value, ...patch }
      photoViewerState.value = next
      const settings = await photoApi.patchPhotoSettings({ photoViewer: next })
      applySettingsFromDTO(settings)
    },

    async patchPhotoCache(patch: Partial<PhotoCacheSettings>) {
      const next = { ...photoCacheState.value, ...patch }
      photoCacheState.value = next
      const settings = await photoApi.patchPhotoSettings({ photoCache: next })
      applySettingsFromDTO(settings)
    },

    async reloadPhotosFromApi(params?: PhotoListParams) {
      try {
        const page = await photoApi.listPhotos(params)
        photosState.value = page.items.map(mapPhotoListItem)
        photosLoadedState.value = true
        loadErrorState.value = null
      } catch (err) {
        loadErrorState.value = formatLoadError(err, "Failed to load photos")
        throw err
      }
    },

    getPhotoById(photoId?: string) {
      const id = photoId?.trim()
      if (!id) return undefined
      return photosState.value.find((photo) => photo.id === id)
    },

    async loadPhotoDetail(photoId: string) {
      const id = photoId.trim()
      if (!id) return undefined
      try {
        const detail = mapPhotoDetail(await photoApi.getPhoto(id))
        mergePhotoIntoCache(detail)
        loadErrorState.value = null
        return detail
      } catch (err) {
        loadErrorState.value = formatLoadError(err, "Failed to load photo")
        return undefined
      }
    },

    async replacePhotoTags(photoId: string, tags: string[]) {
      const detail = mapPhotoDetail(await photoApi.replacePhotoTags(photoId.trim(), tags))
      mergePhotoIntoCache(detail)
      return detail
    },

    async scanPhotos(paths?: string[]): Promise<TaskDTO | null> {
      const selected = paths?.map((path) => path.trim()).filter(Boolean) ?? []
      return await photoApi.startPhotoScan(selected.length > 0 ? { paths: selected } : undefined)
    },
  }
}

export const webPhotoLibraryService = createWebPhotoLibraryService()
