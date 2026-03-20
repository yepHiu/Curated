export type AppPage =
  | "library"
  | "favorites"
  | "recent"
  | "tags"
  | "detail"
  | "player"
  | "settings"
  | "not-found"

export type LibraryMode = Extract<AppPage, "library" | "favorites" | "recent" | "tags">
export type LibraryTab = "all" | "new" | "favorites" | "top-rated"

export interface LibraryStat {
  label: string
  value: string
  detail: string
}

export interface LibrarySetting {
  id: string
  path: string
  title: string
}
