import { computed, ref } from "vue"
import type { PutBookCommentBody, TaskDTO } from "@/api/types"
import type { PhotoLibraryService } from "@/services/contracts/photo-library-service"
import type {
  PhotoCacheSettings,
  PhotoBook,
  PhotoLibrarySetting,
  PhotoPatch,
  PhotoViewerSettings,
} from "@/domain/photo/types"
import {
  MOCK_PHOTO_COMMENTS_KEY,
  getLocalBookComment,
  putLocalBookComment,
} from "@/lib/book-comment-local-storage"

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

const photoFavoritesStorageKey = "curated-mock-photo-favorites-v1"
const photoDeletedStorageKey = "curated-mock-photo-deleted-v1"
try {
  const favorites: unknown = JSON.parse(localStorage.getItem(photoFavoritesStorageKey) ?? "{}")
  const deleted: unknown = JSON.parse(localStorage.getItem(photoDeletedStorageKey) ?? "[]")
  if (favorites && typeof favorites === "object" && !Array.isArray(favorites)) {
    for (const photo of photosState.value) {
      const value = (favorites as Record<string, unknown>)[photo.id]
      if (typeof value === "boolean") photo.isFavorite = value
    }
  }
  if (Array.isArray(deleted)) {
    const ids = new Set(deleted.filter((id): id is string => typeof id === "string"))
    photosState.value = photosState.value.filter((photo) => !ids.has(photo.id))
  }
} catch { /* Invalid local preferences leave sample photos intact. */ }

const photoRatingsStorageKey = "curated-mock-photo-ratings-v1"
const photoTitlesStorageKey = "curated-mock-photo-titles-v1"

/** 从 localStorage 读出写真评分覆盖。 */
function readMockPhotoRatings(): Record<string, number | null> {
  try {
    const saved: unknown = JSON.parse(localStorage.getItem(photoRatingsStorageKey) ?? "{}")
    if (!saved || typeof saved !== "object") return {}
    const out: Record<string, number | null> = {}
    for (const [id, value] of Object.entries(saved as Record<string, unknown>)) {
      if (value === null) {
        out[id] = null
        continue
      }
      if (typeof value === "number" && Number.isFinite(value) && value >= 0 && value <= 5) {
        out[id] = value
      }
    }
    return out
  } catch {
    return {}
  }
}

const savedRatings = readMockPhotoRatings()
for (const photo of photosState.value) {
  if (Object.prototype.hasOwnProperty.call(savedRatings, photo.id)) {
    photo.rating = savedRatings[photo.id] ?? null
  }
}

/** 从 localStorage 读出写真展示标题覆盖。 */
function readMockPhotoTitles(): Record<string, string> {
  try {
    const saved: unknown = JSON.parse(localStorage.getItem(photoTitlesStorageKey) ?? "{}")
    if (!saved || typeof saved !== "object") return {}
    const out: Record<string, string> = {}
    for (const [id, value] of Object.entries(saved as Record<string, unknown>)) {
      if (typeof value === "string" && value.trim()) {
        out[id] = value.trim()
      }
    }
    return out
  } catch {
    return {}
  }
}

const savedTitles = readMockPhotoTitles()
for (const photo of photosState.value) {
  if (savedTitles[photo.id]) {
    photo.title = savedTitles[photo.id]
  }
}

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
  /** Mock 列表已在内存中，暖页短路无需请求。 */
  async ensurePhotosLoaded() {},
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
  /** Mock 更新一本写真的本地评分或展示标题并写入 localStorage。 */
  async patchPhoto(photoId: string, patch: PhotoPatch) {
    const photo = photosState.value.find((item) => item.id === photoId.trim())
    if (!photo) throw new Error("Photo not found")
    if (patch.title !== undefined && !patch.title.trim()) {
      throw new Error("title is required")
    }
    if (patch.rating !== undefined && patch.rating !== null && (patch.rating < 0 || patch.rating > 5)) {
      throw new Error("Photo rating must be between 0 and 5")
    }
    const rating = patch.rating !== undefined ? patch.rating : photo.rating
    const title = patch.title !== undefined ? patch.title.trim() : photo.title
    const updated = { ...photo, title, rating, isFavorite: patch.favorite ?? photo.isFavorite, updatedAt: new Date().toISOString() }
    photosState.value = photosState.value.map((item) => (item.id === photo.id ? updated : item))
    const saved = readMockPhotoRatings()
    if (rating === null) {
      saved[photo.id] = null
    } else if (typeof rating === "number") {
      saved[photo.id] = rating
    }
    localStorage.setItem(photoRatingsStorageKey, JSON.stringify(saved))
    if (patch.title !== undefined) {
      const titles = readMockPhotoTitles()
      titles[photo.id] = title
      localStorage.setItem(photoTitlesStorageKey, JSON.stringify(titles))
    }
    if (patch.favorite !== undefined) {
      const favorites = Object.fromEntries(photosState.value.map((item) => [item.id, item.isFavorite]))
      localStorage.setItem(photoFavoritesStorageKey, JSON.stringify(favorites))
    }
    return updated
  },
  async deletePhoto(photoId: string) {
    const id = photoId.trim()
    if (!photosState.value.some((photo) => photo.id === id)) throw new Error("Photo not found")
    photosState.value = photosState.value.filter((photo) => photo.id !== id)
    let deleted: unknown = []
    try { deleted = JSON.parse(localStorage.getItem(photoDeletedStorageKey) ?? "[]") } catch { /* reset invalid saved IDs */ }
    const ids = new Set(Array.isArray(deleted) ? deleted.filter((item): item is string => typeof item === "string") : [])
    ids.add(id)
    localStorage.setItem(photoDeletedStorageKey, JSON.stringify([...ids]))
  },
  /** Mock 读取一本写真的本地备注。 */
  async getPhotoComment(photoId: string) {
    const photo = this.getPhotoById(photoId)
    if (!photo) throw new Error("Photo not found")
    return getLocalBookComment(MOCK_PHOTO_COMMENTS_KEY, photo.id)
  },
  /** Mock 覆盖保存一本写真的本地备注。 */
  async putPhotoComment(photoId: string, body: PutBookCommentBody) {
    const photo = this.getPhotoById(photoId)
    if (!photo) throw new Error("Photo not found")
    return putLocalBookComment(MOCK_PHOTO_COMMENTS_KEY, photo.id, body.body.trim())
  },
  async scanPhotos() {
    return null
  },
}
