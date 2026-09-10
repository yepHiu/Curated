import { computed, ref } from "vue"
import type { TaskDTO } from "@/api/types"
import type { PhotoLibraryService } from "@/services/contracts/photo-library-service"
import type {
  PhotoCacheSettings,
  PhotoBook,
  PhotoLibrarySetting,
  PhotoViewerSettings,
} from "@/domain/photo/types"

function samplePages(photoId: string, count: number) {
  return Array.from({ length: count }, (_, index) => {
    const page = String(index + 1).padStart(3, "0")
    return {
      photoId,
      index,
      entryPath: `${page}.jpg`,
      fileName: `${page}.jpg`,
      imageExt: ".jpg",
      width: 1440,
      height: 2048,
      imageUrl: `https://picsum.photos/seed/curated-photo-${photoId}-${page}/1440/2048`,
      thumbUrl: `https://picsum.photos/seed/curated-photo-thumb-${photoId}-${page}/360/512`,
    }
  })
}

function photoSeed(
  id: string,
  title: string,
  tags: string[],
  pageCount: number,
  index: number,
): PhotoBook {
  const addedAt = `2026-07-${String(1 + index).padStart(2, "0")}T00:00:00Z`
  return {
    id,
    title,
    tags,
    rating: index === 0 ? 4.4 : null,
    isFavorite: index === 0,
    pageCount,
    currentPageIndex: index === 1 ? 4 : 0,
    coverUrl: `https://picsum.photos/seed/curated-photo-cover-${id}/540/760`,
    sourceFileName: `${title.replace(/\s+/g, "-").toLowerCase()}.${index % 2 === 0 ? "cbz" : "zip"}`,
    location: `D:/Photos/${title.replace(/\s+/g, "-")}.${index % 2 === 0 ? "cbz" : "zip"}`,
    addedAt,
    updatedAt: addedAt,
    pages: samplePages(id, pageCount),
  }
}

const photoLibraryEnabled = ref(false)
const autoPhotoLibraryWatch = ref(true)
const photoLibraryPaths = ref<PhotoLibrarySetting[]>([])
const defaultPhotoImportLibraryPathId = ref("")
const photoViewer = ref<PhotoViewerSettings>({
  mode: "page",
  fit: "contain",
  direction: "ltr",
})
const photoCache = ref<PhotoCacheSettings>({
  maxBytes: 5 * 1024 * 1024 * 1024,
})
const photosState = ref<PhotoBook[]>([
  photoSeed("mock-photo-1", "Summer Frame", ["portrait", "outdoor"], 36, 0),
  photoSeed("mock-photo-2", "Night Portrait", ["studio", "monochrome"], 28, 1),
  photoSeed("mock-photo-3", "Quiet Window", ["soft-light", "set:window"], 42, 2),
])

const photoTagsStorageKey = "curated-mock-photo-tags"
try {
  const saved: unknown = JSON.parse(localStorage.getItem(photoTagsStorageKey) ?? "{}")
  if (saved && typeof saved === "object") {
    for (const photo of photosState.value) {
      const tags: unknown = (saved as Record<string, unknown>)[photo.id]
      if (Array.isArray(tags) && tags.length <= 64 && tags.every(tag => typeof tag === "string" && [...tag].length <= 64)) {
        photo.tags = [...new Set(tags.map(tag => tag.trim()).filter(Boolean))]
      }
    }
  }
} catch { /* Missing or invalid local preferences leave the sample tags intact. */ }

export const mockPhotoLibraryService: PhotoLibraryService = {
  /** Mock acknowledges an upload without copying files to the local filesystem. */
  async importPhotos(files, options) {
    if (!photoLibraryEnabled.value) throw new Error("Photo library is disabled")
    if (!defaultPhotoImportLibraryPathId.value) throw new Error("Default photo path is not configured")
    const total = files.reduce((sum, file) => sum + file.size, 0)
    options?.onUploadProgress?.({ loaded: total, total, percent: 100 })
    return null
  },
  photos: computed(() => photosState.value),
  photosLoaded: computed(() => true),
  loadError: computed(() => null),
  photoLibraryEnabled: computed(() => photoLibraryEnabled.value),
  autoPhotoLibraryWatch: computed(() => autoPhotoLibraryWatch.value),
  photoLibraryPaths: computed(() => photoLibraryPaths.value),
  defaultPhotoImportLibraryPathId: computed(() => defaultPhotoImportLibraryPathId.value),
  photoViewer: computed(() => photoViewer.value),
  photoCache: computed(() => photoCache.value),
  async refreshSettings() {},
  async setPhotoLibraryEnabled(value: boolean) {
    photoLibraryEnabled.value = value
  },
  async setAutoPhotoLibraryWatch(value: boolean) {
    autoPhotoLibraryWatch.value = value
  },
  async addPhotoLibraryPath(path: string, title?: string): Promise<TaskDTO | null> {
    const trimmed = path.trim()
    if (!trimmed) return null
    const dto: PhotoLibrarySetting = {
      id: `photo-library-${Date.now()}`,
      path: trimmed,
      title: title?.trim() || trimmed,
      firstLibraryScanPending: true,
    }
    photoLibraryPaths.value = [...photoLibraryPaths.value, dto]
    return null
  },
  async updatePhotoLibraryPathTitle(id: string, title: string) {
    const trimmed = id.trim()
    const nextTitle = title.trim()
    photoLibraryPaths.value = photoLibraryPaths.value.map((path) =>
      path.id === trimmed ? { ...path, title: nextTitle || path.path } : path,
    )
  },
  async removePhotoLibraryPath(id: string) {
    const trimmed = id.trim()
    photoLibraryPaths.value = photoLibraryPaths.value.filter((path) => path.id !== trimmed)
    if (defaultPhotoImportLibraryPathId.value === trimmed) {
      defaultPhotoImportLibraryPathId.value = ""
    }
  },
  async setDefaultPhotoImportLibraryPathId(id: string) {
    defaultPhotoImportLibraryPathId.value = id.trim()
  },
  async patchPhotoViewer(patch: Partial<PhotoViewerSettings>) {
    photoViewer.value = { ...photoViewer.value, ...patch }
  },
  async patchPhotoCache(patch: Partial<PhotoCacheSettings>) {
    photoCache.value = { ...photoCache.value, ...patch }
  },
  async reloadPhotosFromApi() {
    // Mock photo books are already local.
  },
  getPhotoById(photoId?: string) {
    const id = photoId?.trim()
    if (!id) return undefined
    return photosState.value.find((photo) => photo.id === id)
  },
  async loadPhotoDetail(photoId: string) {
    const id = photoId.trim()
    if (!id) return undefined
    return photosState.value.find((photo) => photo.id === id)
  },
  async replacePhotoTags(photoId: string, raw: string[]) {
    const photo = photosState.value.find(item => item.id === photoId.trim())
    if (!photo) throw new Error("Photo not found")
    if (raw.length > 64 || raw.some(tag => [...tag.trim()].length > 64)) throw new Error("Invalid photo tags")
    const tags = [...new Set(raw.map(tag => tag.trim()).filter(Boolean))]
    const saved = Object.fromEntries(photosState.value.map(item => [item.id, item.id === photo.id ? tags : item.tags]))
    localStorage.setItem(photoTagsStorageKey, JSON.stringify(saved))
    const updated = { ...photo, tags, updatedAt: new Date().toISOString() }
    photosState.value = photosState.value.map(item => item.id === photo.id ? updated : item)
    return updated
  },
  async scanPhotos() {
    return null
  },
}
