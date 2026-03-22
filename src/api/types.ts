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
  /** 元数据/刮削标签 */
  tags: string[]
  /** 用户本地标签（与 tags 独立，刮削不覆盖） */
  userTags?: string[]
  runtimeMinutes: number
  rating: number
  isFavorite: boolean
  addedAt: string
  location: string
  resolution: string
  year: number
  /** 发行日 YYYY-MM-DD，无则省略 */
  releaseDate?: string
  coverUrl?: string
  thumbUrl?: string
}

export interface MovieDetailDTO extends MovieListItemDTO {
  summary: string
  previewImages?: string[]
  previewVideoUrl?: string
  /** 刮削/站点评分（movies.rating） */
  metadataRating: number
  /** 用户本地评分（movies.user_rating），无覆盖时省略 */
  userRating?: number | null
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

/** POST /library/metadata-scrape — 仅允许已配置的库根路径 */
export interface MetadataScrapeByPathsBody {
  paths: string[]
}

export interface MetadataRefreshQueuedDTO {
  queued: number
  skipped: number
  invalidPaths: string[]
}

export interface AddLibraryPathBody {
  path: string
  title?: string
}

export interface UpdateLibraryPathBody {
  title: string
}

/** PATCH /library/movies/{id}；rating 为 null 表示清除用户评分；userTags / metadataTags 出现时整表替换对应标签（NFO 与「我的标签」互不影响） */
export interface PatchMovieBody {
  isFavorite?: boolean
  rating?: number | null
  userTags?: string[]
  /** 元数据/NFO 类标签整表替换；空数组表示清空本地 NFO 标签（下次刮削会再写入） */
  metadataTags?: string[]
}

/** GET /playback/progress */
export interface PlaybackProgressItemDTO {
  movieId: string
  positionSec: number
  durationSec: number
  updatedAt: string
}

export interface PlaybackProgressListDTO {
  items: PlaybackProgressItemDTO[]
}

/** PUT /playback/progress/{movieId} */
export interface PutPlaybackProgressBody {
  positionSec: number
  durationSec: number
}

/** GET /curated-frames（无图像字节，图用 GET /curated-frames/{id}/image） */
export interface CuratedFrameItemDTO {
  id: string
  movieId: string
  title: string
  code: string
  actors: string[]
  positionSec: number
  capturedAt: string
  tags: string[]
}

export interface CuratedFramesListDTO {
  items: CuratedFrameItemDTO[]
}

/** POST /curated-frames */
export interface CreateCuratedFrameBody {
  id: string
  movieId: string
  title: string
  code: string
  actors: string[]
  positionSec: number
  capturedAt: string
  tags?: string[]
  imageBase64: string
}

/** PATCH /curated-frames/{id}/tags */
export interface PatchCuratedFrameTagsBody {
  tags: string[]
}

/** GET /library/played-movies */
export interface PlayedMoviesListDTO {
  movieIds: string[]
}
