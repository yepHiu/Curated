import type { ComputedRef } from "vue"
import type { PhotoImportUploadProgress, TaskDTO } from "@/api/types"
import type {
  PhotoCacheSettings,
  PhotoBook,
  PhotoListParams,
  PhotoLibrarySetting,
  PhotoViewerSettings,
} from "@/domain/photo/types"

export interface PhotoLibraryService {
  photos: ComputedRef<readonly PhotoBook[]>
  photosLoaded: ComputedRef<boolean>
  loadError: ComputedRef<string | null>
  photoLibraryEnabled: ComputedRef<boolean>
  autoPhotoLibraryWatch: ComputedRef<boolean>
  photoLibraryPaths: ComputedRef<readonly PhotoLibrarySetting[]>
  defaultPhotoImportLibraryPathId: ComputedRef<string>
  photoViewer: ComputedRef<PhotoViewerSettings>
  photoCache: ComputedRef<PhotoCacheSettings>
  refreshSettings(): Promise<void>
  setPhotoLibraryEnabled(value: boolean): Promise<void>
  setAutoPhotoLibraryWatch(value: boolean): Promise<void>
  addPhotoLibraryPath(path: string, title?: string): Promise<TaskDTO | null>
  updatePhotoLibraryPathTitle(id: string, title: string): Promise<void>
  removePhotoLibraryPath(id: string): Promise<void>
  setDefaultPhotoImportLibraryPathId(id: string): Promise<void>
  patchPhotoViewer(patch: Partial<PhotoViewerSettings>): Promise<void>
  patchPhotoCache(patch: Partial<PhotoCacheSettings>): Promise<void>
  reloadPhotosFromApi(params?: PhotoListParams): Promise<void>
  getPhotoById(photoId?: string): PhotoBook | undefined
  loadPhotoDetail(photoId: string): Promise<PhotoBook | undefined>
  replacePhotoTags(photoId: string, tags: string[]): Promise<PhotoBook>
  scanPhotos(paths?: string[]): Promise<TaskDTO | null>
  importPhotos(files: File[], options?: { onUploadProgress?: (progress: PhotoImportUploadProgress) => void }): Promise<TaskDTO | null>
}
