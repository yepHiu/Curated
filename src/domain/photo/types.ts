export type PhotoViewerMode = "page" | "scroll"
export type PhotoFitMode = "contain" | "width"
export type PhotoViewingDirection = "ltr" | "rtl"

export interface PhotoPage {
  photoId: string
  index: number
  entryPath: string
  fileName: string
  imageExt?: string
  width?: number
  height?: number
  imageUrl?: string
  thumbUrl?: string
}

export interface PhotoBook {
  id: string
  title: string
  tags: string[]
  rating: number | null
  isFavorite: boolean
  pageCount: number
  currentPageIndex: number
  coverUrl?: string
  sourceFileName: string
  location: string
  addedAt: string
  updatedAt: string
  lastViewedAt?: string
  completedAt?: string
  pages?: PhotoPage[]
}

export interface PhotoListParams {
  q?: string
  limit?: number
  offset?: number
}

export interface PhotoLibrarySetting {
  id: string
  path: string
  title: string
  firstLibraryScanPending?: boolean
}

export interface PhotoViewerSettings {
  mode: PhotoViewerMode
  fit: PhotoFitMode
  direction: PhotoViewingDirection
}

export interface PhotoCacheSettings {
  maxBytes: number
}
