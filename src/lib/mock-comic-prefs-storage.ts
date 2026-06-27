import type { ComicBook, ComicReaderSettings } from "@/domain/comic/types"

export const MOCK_COMIC_PREFS_KEY = "curated-comic-prefs-v1"

export interface MockComicProgress {
  pageIndex: number
  completed: boolean
  updatedAt: string
}

export interface MockComicPrefs {
  favorite?: boolean
  rating?: number | null
  tags?: string[]
  progress?: MockComicProgress
  preferences?: ComicReaderSettings & { updatedAt: string }
}

export type MockComicPrefsMap = Record<string, MockComicPrefs>

function safeParse(raw: string | null): MockComicPrefsMap {
  if (!raw?.trim()) return {}
  try {
    const parsed = JSON.parse(raw) as unknown
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as MockComicPrefsMap
    }
  } catch {
    // ignore malformed localStorage
  }
  return {}
}

let cache: MockComicPrefsMap =
  typeof localStorage !== "undefined" ? safeParse(localStorage.getItem(MOCK_COMIC_PREFS_KEY)) : {}

export function loadMockComicPrefs(): MockComicPrefsMap {
  if (typeof localStorage === "undefined") return {}
  cache = safeParse(localStorage.getItem(MOCK_COMIC_PREFS_KEY))
  return cache
}

export function saveMockComicPrefs(map: MockComicPrefsMap) {
  cache = map
  if (typeof localStorage === "undefined") return
  try {
    localStorage.setItem(MOCK_COMIC_PREFS_KEY, JSON.stringify(map))
  } catch {
    // quota / private mode
  }
}

export function getMockComicPrefsSnapshot(): MockComicPrefsMap {
  return { ...cache }
}

export function upsertMockComicPrefs(comicId: string, patch: MockComicPrefs) {
  const id = comicId.trim()
  if (!id) return
  const prev = cache[id] ?? {}
  const merged: MockComicPrefs = { ...prev }
  if (typeof patch.favorite === "boolean") {
    merged.favorite = patch.favorite
  }
  if ("rating" in patch) {
    merged.rating = patch.rating
  }
  if (Array.isArray(patch.tags)) {
    merged.tags = [...patch.tags]
  }
  if (patch.progress) {
    merged.progress = { ...patch.progress }
  }
  if (patch.preferences) {
    merged.preferences = { ...patch.preferences }
  }
  saveMockComicPrefs({
    ...cache,
    [id]: merged,
  })
}

export function mergeMockComicPrefsIntoBook(book: ComicBook): ComicBook {
  const prefs = cache[book.id]
  if (!prefs) return book
  let next: ComicBook = { ...book }
  if (typeof prefs.favorite === "boolean") {
    next.isFavorite = prefs.favorite
  }
  if ("rating" in prefs) {
    next.rating = prefs.rating
  }
  if (Array.isArray(prefs.tags)) {
    next.tags = [...prefs.tags]
  }
  if (prefs.progress) {
    next = {
      ...next,
      currentPageIndex: prefs.progress.pageIndex,
      readStatus: prefs.progress.completed
        ? "read"
        : prefs.progress.pageIndex > 0
          ? "reading"
          : "unread",
      lastReadAt: prefs.progress.updatedAt,
      completedAt: prefs.progress.completed ? prefs.progress.updatedAt : undefined,
    }
  }
  return next
}
