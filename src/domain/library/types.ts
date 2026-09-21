export type AppPage =
  | "wishlist"
  | "wishlist-detail"
  | "home"
  | "library"
  | "favorites"
  | "recent"
  | "tags"
  | "trash"
  | "actors"
  | "actor-detail"
  | "comics"
  | "comic-detail"
  | "comic-reader"
  | "photos"
  | "photo-detail"
  | "photo-viewer"
  | "history"
  | "insights"
  | "curated-frames"
  | "detail"
  | "player"
  | "settings"
  | "not-found"

export type LibraryMode = Extract<AppPage, "library" | "favorites" | "recent" | "tags" | "trash">
export type LibraryTab = "all" | "new" | "top-rated"

export interface LibraryStat {
  labelKey: string
  value: string
  /** 概览卡副说明；省略则不渲染说明段落 */
  detailKey?: string
}

export interface LibrarySetting {
  id: string
  path: string
  title: string
  /** 来自后端：新库根首次扫描完成前为 true */
  firstLibraryScanPending?: boolean
}
