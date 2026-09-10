export type ComicReadStatus = "unread" | "reading" | "read"
export type ComicReaderMode = "page" | "scroll"
export type ComicFitMode = "contain" | "width"
export type ComicReadingDirection = "ltr" | "rtl"

export interface ComicLibrarySetting {
  id: string
  path: string
  title: string
  firstLibraryScanPending?: boolean
}

export interface ComicPage {
  comicId: string
  index: number
  entryPath: string
  fileName: string
  imageExt?: string
  width?: number
  height?: number
  imageUrl?: string
  thumbUrl?: string
}

export interface ComicBook {
  id: string
  title: string
  tags: string[]
  rating?: number | null
  isFavorite: boolean
  readStatus: ComicReadStatus | string
  pageCount: number
  currentPageIndex: number
  coverUrl?: string
  sourceFileName: string
  location: string
  addedAt: string
  updatedAt: string
  lastReadAt?: string
  completedAt?: string
  pages?: ComicPage[]
}

export interface ComicReaderSettings {
  mode: ComicReaderMode
  fit: ComicFitMode
  direction: ComicReadingDirection
}

export interface ComicCacheSettings {
  maxBytes: number
}

export interface ComicListParams {
  q?: string
  tag?: string
  favorite?: boolean
  readStatus?: ComicReadStatus | string
  limit?: number
  offset?: number
}

export interface ComicPatch {
  title?: string
  tags?: string[]
  favorite?: boolean
  rating?: number | null
}
