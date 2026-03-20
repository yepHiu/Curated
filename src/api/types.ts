export interface ApiResponse<T> {
  ok: boolean
  data?: T
  error?: ApiError
}

export interface ApiError {
  code: string
  message: string
  retryable: boolean
  details?: Record<string, unknown>
}

export interface HealthDTO {
  name: string
  version: string
  transport: string
  databasePath: string
}

export interface MovieListItemDTO {
  id: string
  title: string
  code: string
  studio: string
  actors: string[]
  tags: string[]
  runtimeMinutes: number
  rating: number
  isFavorite: boolean
  addedAt: string
  location: string
  resolution: string
  year: number
  coverUrl?: string
  thumbUrl?: string
}

export interface MovieDetailDTO extends MovieListItemDTO {
  summary: string
  previewImages?: string[]
  previewVideoUrl?: string
}

export interface MoviesPageDTO {
  items: MovieListItemDTO[]
  total: number
  limit: number
  offset: number
}

export interface LibraryPathDTO {
  id: string
  path: string
  title: string
}

export interface PlayerSettingsDTO {
  hardwareDecode: boolean
}

export interface SettingsDTO {
  libraryPaths: LibraryPathDTO[]
  player: PlayerSettingsDTO
  /** 扫描后整理为 番号/番号.ext 并写入 NFO/资产到番号目录 */
  organizeLibrary: boolean
}

export interface PatchSettingsBody {
  organizeLibrary?: boolean
}

export interface TaskDTO {
  taskId: string
  type: string
  status: "pending" | "running" | "completed" | "partial_failed" | "failed" | "cancelled"
  createdAt: string
  startedAt?: string
  finishedAt?: string
  progress: number
  message?: string
  errorCode?: string
  errorMessage?: string
  metadata?: Record<string, unknown>
}

export interface ListMoviesParams {
  mode?: string
  q?: string
  limit?: number
  offset?: number
}

export interface StartScanBody {
  paths?: string[]
}

export interface AddLibraryPathBody {
  path: string
  title?: string
}

export interface UpdateLibraryPathBody {
  title: string
}
