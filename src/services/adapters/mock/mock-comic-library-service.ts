import { computed, ref } from "vue"
import { HttpClientError } from "@/api/http-client"
import type {
  ComicCacheStatusDTO,
  LibraryPathStorageStatusDTO,
  ComicReadingPreferencesDTO,
  ComicReadingProgressDTO,
  PutComicReadingPreferencesBody,
  TaskDTO,
} from "@/api/types"
import type {
  ComicBook,
  ComicCacheSettings,
  ComicLibrarySetting,
  ComicPatch,
  ComicReaderSettings,
} from "@/domain/comic/types"
import {
  getMockComicPrefsSnapshot,
  loadMockComicPrefs,
  mergeMockComicPrefsIntoBook,
  upsertMockComicPrefs,
} from "@/lib/mock-comic-prefs-storage"
import type { ComicLibraryService } from "@/services/contracts/comic-library-service"

function mockHttpError(status: number, code: string, message = code): HttpClientError {
  return new HttpClientError(status, {
    code,
    message,
    retryable: false,
  })
}

function nowISO(): string {
  return new Date().toISOString()
}

function samplePages(comicId: string, count: number) {
  return Array.from({ length: count }, (_, index) => {
    const page = String(index + 1).padStart(3, "0")
    return {
      comicId,
      index,
      entryPath: `${page}.jpg`,
      fileName: `${page}.jpg`,
      imageExt: ".jpg",
      width: 1440,
      height: 2048,
      imageUrl: `https://picsum.photos/seed/curated-comic-${comicId}-${page}/1440/2048`,
      thumbUrl: `https://picsum.photos/seed/curated-comic-thumb-${comicId}-${page}/360/512`,
    }
  })
}

function comicSeed(
  id: string,
  title: string,
  tags: string[],
  pageCount: number,
  index: number,
): ComicBook {
  const addedAt = `2026-06-${String(10 + index).padStart(2, "0")}T00:00:00Z`
  return {
    id,
    title,
    tags,
    rating: index === 0 ? 4.6 : null,
    isFavorite: index === 0,
    readStatus: index === 1 ? "reading" : "unread",
    pageCount,
    currentPageIndex: index === 1 ? 6 : 0,
    coverUrl: `https://picsum.photos/seed/curated-comic-cover-${id}/540/760`,
    sourceFileName: `${title.replace(/\s+/g, "-").toLowerCase()}.${index % 2 === 0 ? "cbz" : "zip"}`,
    location: `D:/Comics/${title.replace(/\s+/g, "-")}.${index % 2 === 0 ? "cbz" : "zip"}`,
    addedAt,
    updatedAt: addedAt,
    pages: samplePages(id, pageCount),
  }
}

loadMockComicPrefs()

const comicLibraryEnabledMock = ref(false)
const comicLibraryPathsMock = ref<ComicLibrarySetting[]>([])
const comicLibraryPathStorageStatusesMock = ref<LibraryPathStorageStatusDTO[]>([])
const defaultComicImportLibraryPathIdMock = ref("")
const comicReaderMock = ref<ComicReaderSettings>({
  mode: "page",
  fit: "contain",
  direction: "rtl",
})
const comicCacheMock = ref<ComicCacheSettings>({
  maxBytes: 2 * 1024 * 1024 * 1024,
})

const comicsState = ref<ComicBook[]>(
  [
    comicSeed("mock-comic-1", "Glass City Notebook", ["author:manual", "cyberpunk"], 12, 0),
    comicSeed("mock-comic-2", "Rain Garden Chapter 01", ["series:rain-garden", "slice-of-life"], 18, 1),
    comicSeed("mock-comic-3", "North Pier Sketchbook", ["one-shot", "monochrome"], 9, 2),
  ].map(mergeMockComicPrefsIntoBook),
)

function findComicIndex(comicId: string): number {
  const id = comicId.trim()
  if (!id) return -1
  return comicsState.value.findIndex((comic) => comic.id === id)
}

function replaceComic(comic: ComicBook) {
  comicsState.value = comicsState.value.map((item) => (item.id === comic.id ? comic : item))
}

function progressForComic(comic: ComicBook): ComicReadingProgressDTO {
  const saved = getMockComicPrefsSnapshot()[comic.id]?.progress
  if (saved) {
    return {
      comicId: comic.id,
      pageIndex: saved.pageIndex,
      completed: saved.completed,
      updatedAt: saved.updatedAt,
    }
  }
  return {
    comicId: comic.id,
    pageIndex: comic.currentPageIndex,
    completed: comic.readStatus === "read",
    updatedAt: comic.lastReadAt ?? "",
  }
}

function preferencesForComic(comicId: string): ComicReadingPreferencesDTO {
  const saved = getMockComicPrefsSnapshot()[comicId]?.preferences
  if (saved) {
    return {
      comicId,
      mode: saved.mode,
      fit: saved.fit,
      direction: saved.direction,
      updatedAt: saved.updatedAt,
    }
  }
  return {
    comicId,
    ...comicReaderMock.value,
  }
}

function patchComicInMemory(comicId: string, patch: ComicPatch): ComicBook | undefined {
  const index = findComicIndex(comicId)
  if (index < 0) return undefined
  const current = comicsState.value[index]
  const next: ComicBook = {
    ...current,
    title: patch.title !== undefined ? patch.title : current.title,
    tags: patch.tags !== undefined ? [...patch.tags] : current.tags,
    isFavorite: patch.favorite !== undefined ? patch.favorite : current.isFavorite,
    rating: patch.rating !== undefined ? patch.rating : current.rating,
    updatedAt: nowISO(),
  }
  replaceComic(next)
  upsertMockComicPrefs(next.id, {
    favorite: next.isFavorite,
    rating: next.rating,
    tags: next.tags,
  })
  return next
}

function updateProgressInMemory(
  comicId: string,
  pageIndex: number,
  completed: boolean,
): ComicReadingProgressDTO {
  const index = findComicIndex(comicId)
  if (index < 0) {
    throw mockHttpError(404, "MOCK_COMIC_NOT_FOUND", "comic not found")
  }
  const comic = comicsState.value[index]
  const boundedPage = Math.min(Math.max(0, pageIndex), Math.max(0, comic.pageCount - 1))
  const updatedAt = nowISO()
  const next: ComicBook = {
    ...comic,
    currentPageIndex: boundedPage,
    readStatus: completed ? "read" : boundedPage > 0 ? "reading" : "unread",
    lastReadAt: updatedAt,
    completedAt: completed ? updatedAt : undefined,
  }
  replaceComic(next)
  const progress = {
    pageIndex: boundedPage,
    completed,
    updatedAt,
  }
  upsertMockComicPrefs(comic.id, { progress })
  return {
    comicId: comic.id,
    ...progress,
  }
}

export const mockComicLibraryService: ComicLibraryService = {
  comics: computed(() => comicsState.value),
  comicsLoaded: computed(() => true),
  loadError: computed(() => null),
  comicLibraryEnabled: computed(() => comicLibraryEnabledMock.value),
  comicLibraryPaths: computed(() => comicLibraryPathsMock.value),
  comicLibraryPathStorageStatuses: computed(() => comicLibraryPathStorageStatusesMock.value),
  defaultComicImportLibraryPathId: computed(() => defaultComicImportLibraryPathIdMock.value),
  comicReader: computed(() => comicReaderMock.value),
  comicCache: computed(() => comicCacheMock.value),

  async refreshSettings() {
    // Mock settings are local state only.
  },

  async checkComicLibraryPathStorageStatus(libraryPathIds?: string[]) {
    const selected = new Set(libraryPathIds?.map((id) => id.trim()).filter(Boolean) ?? [])
    const paths = selected.size > 0
      ? comicLibraryPathsMock.value.filter((path) => selected.has(path.id))
      : comicLibraryPathsMock.value
    const checked = paths.map((path) => {
      const existing = comicLibraryPathStorageStatusesMock.value.find(
        (status) => status.libraryPathId === path.id,
      )
      return existing ?? {
        libraryPathId: path.id,
        path: path.path,
        title: path.title,
        status: "online" as const,
        message: "online",
        checkedAt: nowISO(),
        canRescan: true,
        canImport: true,
      }
    })
    if (selected.size === 0) {
      comicLibraryPathStorageStatusesMock.value = checked
      return
    }
    const next = new Map(comicLibraryPathStorageStatusesMock.value.map((item) => [item.libraryPathId, item]))
    for (const item of checked) {
      next.set(item.libraryPathId, item)
    }
    comicLibraryPathStorageStatusesMock.value = [...next.values()]
  },

  async setComicLibraryEnabled(value: boolean) {
    if (value && comicLibraryPathsMock.value.length === 0) {
      throw mockHttpError(
        400,
        "MOCK_COMIC_PATH_REQUIRED",
        "add a comic library path before enabling comic library",
      )
    }
    comicLibraryEnabledMock.value = value
  },

  async addComicLibraryPath(path: string, title?: string): Promise<TaskDTO | null> {
    const trimmed = path.trim()
    if (!trimmed) return null
    const id = `mock-comic-path-${Date.now()}`
    comicLibraryPathsMock.value = [
      ...comicLibraryPathsMock.value,
      {
        id,
        path: trimmed,
        title: title?.trim() || trimmed,
        firstLibraryScanPending: false,
      },
    ]
    if (!defaultComicImportLibraryPathIdMock.value) {
      defaultComicImportLibraryPathIdMock.value = id
    }
    return null
  },

  async updateComicLibraryPathTitle(id: string, title: string) {
    const trimmed = id.trim()
    comicLibraryPathsMock.value = comicLibraryPathsMock.value.map((item) =>
      item.id === trimmed ? { ...item, title: title.trim() || item.title } : item,
    )
  },

  async removeComicLibraryPath(id: string) {
    const trimmed = id.trim()
    comicLibraryPathsMock.value = comicLibraryPathsMock.value.filter((item) => item.id !== trimmed)
    if (defaultComicImportLibraryPathIdMock.value === trimmed) {
      defaultComicImportLibraryPathIdMock.value = comicLibraryPathsMock.value[0]?.id ?? ""
    }
    if (comicLibraryPathsMock.value.length === 0) {
      comicLibraryEnabledMock.value = false
    }
  },

  async setDefaultComicImportLibraryPathId(id: string) {
    const trimmed = id.trim()
    if (trimmed && !comicLibraryPathsMock.value.some((item) => item.id === trimmed)) {
      throw mockHttpError(400, "MOCK_COMIC_PATH_NOT_FOUND", "comic library path not found")
    }
    defaultComicImportLibraryPathIdMock.value = trimmed
  },

  async patchComicReader(patch: Partial<ComicReaderSettings>) {
    comicReaderMock.value = { ...comicReaderMock.value, ...patch }
  },

  async patchComicCache(patch: Partial<ComicCacheSettings>) {
    comicCacheMock.value = { ...comicCacheMock.value, ...patch }
  },

  async reloadComicsFromApi() {
    // Mock comics are already local.
  },

  getComicById(comicId?: string) {
    const id = comicId?.trim()
    if (!id) return undefined
    return comicsState.value.find((comic) => comic.id === id)
  },

  async loadComicDetail(comicId: string) {
    const comic = this.getComicById(comicId)
    if (!comic) return undefined
    await Promise.resolve()
    return comic
  },

  async patchComic(comicId: string, patch: ComicPatch) {
    return patchComicInMemory(comicId, patch)
  },

  async deleteComic(comicId: string) {
    const id = comicId.trim()
    comicsState.value = comicsState.value.filter((comic) => comic.id !== id)
  },

  async revealComicSource() {
    throw mockHttpError(501, "MOCK_COMIC_REVEAL_NOT_SUPPORTED")
  },

  async scanComics(): Promise<TaskDTO | null> {
    throw mockHttpError(501, "MOCK_COMIC_SCAN_NOT_SUPPORTED")
  },

  async importComics(): Promise<TaskDTO | null> {
    throw mockHttpError(501, "MOCK_COMIC_IMPORT_NOT_SUPPORTED")
  },

  async getComicProgress(comicId: string): Promise<ComicReadingProgressDTO> {
    const comic = this.getComicById(comicId)
    if (!comic) {
      throw mockHttpError(404, "MOCK_COMIC_NOT_FOUND", "comic not found")
    }
    return progressForComic(comic)
  },

  async saveComicProgress(
    comicId: string,
    pageIndex: number,
    completed: boolean,
  ): Promise<ComicReadingProgressDTO> {
    return updateProgressInMemory(comicId, pageIndex, completed)
  },

  async resetComicProgress(comicId: string) {
    updateProgressInMemory(comicId, 0, false)
  },

  async getComicPreferences(comicId: string): Promise<ComicReadingPreferencesDTO> {
    const comic = this.getComicById(comicId)
    if (!comic) {
      throw mockHttpError(404, "MOCK_COMIC_NOT_FOUND", "comic not found")
    }
    return preferencesForComic(comic.id)
  },

  async saveComicPreferences(
    comicId: string,
    prefs: PutComicReadingPreferencesBody,
  ): Promise<ComicReadingPreferencesDTO> {
    const comic = this.getComicById(comicId)
    if (!comic) {
      throw mockHttpError(404, "MOCK_COMIC_NOT_FOUND", "comic not found")
    }
    const merged = {
      ...comicReaderMock.value,
      ...preferencesForComic(comic.id),
      ...prefs,
      comicId: undefined,
      updatedAt: nowISO(),
    }
    upsertMockComicPrefs(comic.id, {
      preferences: {
        mode: merged.mode,
        fit: merged.fit,
        direction: merged.direction,
        updatedAt: merged.updatedAt,
      },
    })
    return {
      comicId: comic.id,
      mode: merged.mode,
      fit: merged.fit,
      direction: merged.direction,
      updatedAt: merged.updatedAt,
    }
  },

  async getComicCacheStatus(): Promise<ComicCacheStatusDTO> {
    return {
      maxBytes: comicCacheMock.value.maxBytes,
      usedBytes: 0,
      entryCount: 0,
    }
  },

  async cleanupComicCache(): Promise<ComicCacheStatusDTO> {
    return await this.getComicCacheStatus()
  },
}
