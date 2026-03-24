export type AppPage =
  | "library"
  | "favorites"
  | "recent"
  | "tags"
  | "trash"
  | "actors"
  | "history"
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
  detailKey: string
}

export interface LibrarySetting {
  id: string
  path: string
  title: string
  /** 来自后端：新库根首次扫描完成前为 true */
  firstLibraryScanPending?: boolean
}
