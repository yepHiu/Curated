export type AppPage =
  | "library"
  | "favorites"
  | "recent"
  | "tags"
  | "actors"
  | "history"
  | "curated-frames"
  | "detail"
  | "player"
  | "settings"
  | "not-found"

export type LibraryMode = Extract<AppPage, "library" | "favorites" | "recent" | "tags">
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
}
